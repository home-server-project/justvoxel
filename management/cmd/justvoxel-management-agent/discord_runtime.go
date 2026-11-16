package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/disgoorg/disgo/gateway"
	"github.com/gorilla/websocket"
)

const discordGatewayIntents = gateway.IntentGuilds
const discordGuildLimit = 200

// All fields are private, volatile, and guarded by discordMu. A new runtime
// pointer also fences callbacks from failed attempts within the same generation.
type discordRuntime struct {
	generation        int64
	status            string
	guilds            map[string]discordGuild
	pending           map[string]bool
	ready             bool
	discoveryDeadline time.Time
	cancel            context.CancelFunc
	client            discordGatewayClient
	api               discordREST
	adminResponses    map[string]discordAdminResponse
	actions           map[string]discordPendingAction
	accessApprovals   map[string]bool
	syncing           bool
	syncedGuild       string
	syncAfter         time.Time
	workers           int
}
type discordGuild struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"-"`
}
type discordGatewayClient interface {
	Open(context.Context) error
	Close(context.Context)
	Status() gateway.Status
}

// This is the only client-creation seam. Tests inject an in-memory client.
type discordGatewayFactory func(context.Context, string, gateway.EventHandlerFunc, func()) discordGatewayClient

func newDiscordGateway(ctx context.Context, token string, handler gateway.EventHandlerFunc, closed func()) discordGatewayClient {
	// No library diagnostics may reach logs, including error-level payloads.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second
	dialer.NetDialContext = func(callCtx context.Context, network, address string) (net.Conn, error) {
		// DisGo's automatic reconnect uses a background context. Fence its dials
		// with this client's lifetime as well. A normal close error terminates its
		// internal retry loop after cancellation, without another authentication.
		if ctx.Err() != nil {
			return nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}
		}
		dialCtx, cancel := context.WithCancel(callCtx)
		stop := context.AfterFunc(ctx, cancel)
		defer stop()
		defer cancel()
		conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(dialCtx, network, address)
		if ctx.Err() != nil {
			if conn != nil {
				_ = conn.Close()
			}
			return nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}
		}
		if err != nil {
			return nil, err
		}
		stopConn := context.AfterFunc(ctx, func() { _ = conn.Close() })
		return &discordGatewayConn{Conn: conn, stop: stopConn}, nil
	}
	return gateway.New(token, handler, gateway.WithIntents(discordGatewayIntents), gateway.WithLogger(logger), gateway.WithDialer(&dialer), gateway.WithDefaultRateLimiterConfigOpts(gateway.WithRateLimiterLogger(logger)), gateway.WithCloseHandler(func(_ gateway.Gateway, _ error, _ bool) { closed() }))
}

type discordGatewayConn struct {
	net.Conn
	stop func() bool
}

func (c *discordGatewayConn) Close() error { c.stop(); return c.Conn.Close() }

func (s *server) discordResetAllowed() bool {
	if s.operations == nil {
		return true
	}
	current, err := s.operations.currentFactoryReset()
	return err == nil && (current == nil || !operationIsCurrent(current.State))
}

// Called with discordMu held. Closing never waits on callbacks: DisGo's
// synchronous event delivery may itself be waiting for discordMu.
func (s *server) discordStopLocked() {
	rt := s.discordRuntime
	s.discordRuntime = nil
	if rt == nil {
		return
	}
	rt.adminResponses = nil
	if rt.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// Let DisGo send its normal close frame first. If a dial, handshake or
		// write is stuck, lifetime cancellation also closes the underlying socket.
		stop := context.AfterFunc(ctx, func() {
			if rt.cancel != nil {
				rt.cancel()
			}
		})
		rt.client.Close(ctx)
		stop()
		cancel()
	}
	if rt.cancel != nil {
		rt.cancel()
	}
}

func (s *server) startDiscord(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		lock, err := os.OpenFile("/run/justvoxel/discord.lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
		if err != nil {
			s.discordPauseUnavailable()
			return
		}
		defer lock.Close()
		if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
			s.discordPauseUnavailable()
			return
		}
		s.discordLoop(ctx)
	}()
	return done
}

func (s *server) discordPauseUnavailable() {
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	s.discordUnavailable = true
}

// A configuration change interrupts backoff without starting a second loop.
func (s *server) discordWakeLocked() {
	select {
	case s.discordWake <- struct{}{}:
	default:
	}
}

func (s *server) discordWait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-s.discordWake:
		return true
	case <-timer.C:
		return true
	}
}

func (s *server) discordLoop(ctx context.Context) {
	s.discordMu.Lock()
	if s.discordWake == nil {
		s.discordWake = make(chan struct{}, 1)
	}
	s.discordMu.Unlock()
	defer func() { s.discordMu.Lock(); s.discordStopLocked(); s.discordMu.Unlock() }()
	lastGeneration := int64(-1)
	failures := 0
	for ctx.Err() == nil {
		s.discordMu.Lock()
		v, err := s.store.discordRead()
		if err != nil || !v.Enabled || !v.Verified || v.Token == "" || !s.discordResetAllowed() {
			s.discordStopLocked()
			s.discordMu.Unlock()
			if !s.discordWait(ctx, 250*time.Millisecond) {
				return
			}
			continue
		}
		if lastGeneration != v.Generation {
			failures = 0
			lastGeneration = v.Generation
		}
		if s.discordRuntime != nil {
			rt := s.discordRuntime
			if rt.generation == v.Generation && (rt.client != nil || rt.status == "connection_paused") {
				if rt.client != nil && rt.status != "connection_paused" {
					switch rt.client.Status() {
					case gateway.StatusReady:
						rt.status = "connected"
					default:
						rt.status = "connection_retry"
					}
					// READY lists every guild, including unavailable guilds for which
					// Discord may send no GUILD_CREATE. Finish that initial view boundedly.
					if !rt.ready && !rt.discoveryDeadline.IsZero() && time.Now().After(rt.discoveryDeadline) {
						rt.ready = true
						rt.pending = nil
					}
				}
				discordPruneAdminResponses(rt, time.Now())
				s.discordSyncLocked(rt, v)
				s.discordReconcileRebootLocked(rt, v)
				s.discordMu.Unlock()
				if !s.discordWait(ctx, 250*time.Millisecond) {
					return
				}
				continue
			}
			s.discordStopLocked()
		}
		runCtx, cancel := context.WithCancel(ctx)
		rt := &discordRuntime{generation: v.Generation, status: "connecting", guilds: map[string]discordGuild{}, cancel: cancel}
		if failures > 0 {
			rt.status = "connection_retry"
		}
		s.discordRuntime = rt
		rt.api = s.discordRESTClient(v.Token)
		factory := s.discordNewGateway
		if factory == nil {
			factory = newDiscordGateway
		}
		// Creation is local only; gateway.New performs no network requests.
		rt.client = factory(runCtx, v.Token, func(_ gateway.Gateway, kind gateway.EventType, _ int, event gateway.EventData) {
			s.discordEvent(rt, kind, event)
		}, func() {
			s.discordMu.Lock()
			defer s.discordMu.Unlock()
			if s.discordRuntime == rt && runCtx.Err() == nil {
				rt.status = "connection_paused"
			}
		})
		s.discordMu.Unlock()
		// DisGo handles initial exponential backoff inside Open. Bound each initial
		// window; after three failures pause until a new configuration generation.
		openCtx, openCancel := context.WithTimeout(runCtx, 20*time.Second)
		// Also fence the socket if Open's timeout path is itself closing a
		// stalled connection. Stop this hook before canceling a successful Open.
		stopOpen := context.AfterFunc(openCtx, cancel)
		err = rt.client.Open(openCtx)
		stopOpen()
		openCancel()
		s.discordMu.Lock()
		if s.discordRuntime != rt || ctx.Err() != nil {
			s.discordMu.Unlock()
			continue
		}
		if err == nil && runCtx.Err() == nil {
			rt.status = "connected"
			s.discordMu.Unlock()
			continue
		}
		failures++
		s.discordStopLocked()
		s.discordRuntime = &discordRuntime{generation: v.Generation, status: "connection_retry"}
		if failures >= 3 {
			s.discordRuntime = &discordRuntime{generation: v.Generation, status: "connection_paused"}
		}
		s.discordMu.Unlock()
		if !s.discordWait(ctx, time.Second*time.Duration(1<<min(failures, 3))) {
			return
		}
	}
}

// DisGo's low-level READY/GUILD_CREATE/UPDATE/DELETE events provide the same
// discovery, join, available, rename, unavailable and leave semantics without
// installing handlers for messages, members, channels or roles.
func (s *server) discordEvent(rt *discordRuntime, _ gateway.EventType, event gateway.EventData) {
	if e, ok := event.(gateway.EventInteractionCreate); ok {
		s.discordInteraction(rt, e.Interaction)
		return
	}
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	v, err := s.store.discordRead()
	if s.discordRuntime != rt || err != nil || v.Generation != rt.generation || !v.Enabled || !v.Verified || !s.discordResetAllowed() {
		return
	}
	switch e := event.(type) {
	case gateway.EventReady:
		rt.status = "connected"
		rt.syncedGuild = ""
		rt.syncAfter = time.Time{}
		rt.guilds = map[string]discordGuild{}
		rt.pending = map[string]bool{}
		for _, guild := range e.Guilds {
			id := guild.ID.String()
			rt.guilds[id] = discordGuild{ID: id}
			rt.pending[id] = true
		}
		rt.ready = len(rt.pending) == 0
		rt.discoveryDeadline = time.Now().Add(15 * time.Second)
	case gateway.EventGuildCreate:
		id := e.ID.String()
		rt.guilds[id] = discordGuild{ID: id, Name: discordGuildName(e.Name), Available: !e.Unavailable}
		delete(rt.pending, id)
		if len(rt.pending) == 0 {
			rt.ready = true
		}
	case gateway.EventGuildUpdate:
		id := e.ID.String()
		if guild, ok := rt.guilds[id]; ok {
			guild.Name = discordGuildName(e.Name)
			rt.guilds[id] = guild
		}
	case gateway.EventGuildDelete:
		id := e.ID.String()
		if e.Unavailable {
			guild := rt.guilds[id]
			guild.ID = id
			guild.Available = false
			rt.guilds[id] = guild
		} else {
			delete(rt.guilds, id)
		}
		delete(rt.pending, id)
		if len(rt.pending) == 0 {
			rt.ready = true
		}
	case gateway.EventResumed:
		rt.status = "connected"
	}
}

func discordGuildName(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	return string([]rune(name)[:min(len([]rune(name)), 100)])
}

// Caller holds discordMu. No library objects are ever serialized.
func (s *server) discordSnapshot(v discordState) map[string]any {
	out := discordPublicState(v)
	if !v.Enabled {
		return out
	}
	if s.discordUnavailable {
		out["runtime_status"] = "connection_paused"
		return out
	}
	rt := s.discordRuntime
	if rt == nil {
		return out
	}
	if rt.generation != v.Generation {
		return out
	}
	out["runtime_status"] = rt.status
	connected := rt.status == "connected" && rt.client != nil && rt.client.Status() == gateway.StatusReady
	if rt.status == "connected" && !connected {
		out["runtime_status"] = "connection_retry"
	}
	out["guilds_ready"] = rt.ready && connected
	guilds := []discordGuild{}
	if connected && rt.ready {
		for _, guild := range rt.guilds {
			if guild.Available {
				guilds = append(guilds, guild)
			}
		}
	}
	sort.Slice(guilds, func(i, j int) bool {
		if guilds[i].Name == guilds[j].Name {
			return guilds[i].ID < guilds[j].ID
		}
		return guilds[i].Name < guilds[j].Name
	})
	if len(guilds) > discordGuildLimit {
		guilds = guilds[:discordGuildLimit]
	}
	out["guilds"] = guilds
	bound, ok := rt.guilds[v.GuildID]
	out["bound_guild_available"] = connected && rt.ready && ok && bound.Available
	if ok && bound.Name != "" {
		out["guild_name"] = bound.Name
	}
	return out
}

var errDiscordGuild = errors.New("Choose an available Discord server after discovery is ready.")

func (s *server) discordBind(v *discordState, id string) error {
	rt := s.discordRuntime
	if !v.Verified || !v.Enabled || !validDiscordSnowflake(id) || rt == nil || rt.generation != v.Generation || !rt.ready || rt.client == nil || rt.client.Status() != gateway.StatusReady {
		return errDiscordGuild
	}
	guild, ok := rt.guilds[id]
	if !ok || !guild.Available {
		return errDiscordGuild
	}
	if v.GuildID != guild.ID {
		v.Pending = nil
		v.Administrators = nil
		v.Generation++
	}
	v.GuildID, v.GuildName = guild.ID, guild.Name
	return nil
}
