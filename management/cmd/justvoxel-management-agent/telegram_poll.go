package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type telegramAPIError struct{ Code int }

func (e *telegramAPIError) Error() string { return "Telegram request failed" }

// Never return the HTTP client's URL-bearing errors or Telegram descriptions.
// Redirects are forbidden so a bot credential cannot be forwarded to another host.
var telegramHTTP = &http.Client{Timeout: 35 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
var telegramRequest = func(ctx context.Context, token, method string, input, output any) error {
	switch method {
	case "getMe", "getWebhookInfo", "getUpdates", "sendMessage", "answerCallbackQuery", "setMyCommands":
	default:
		return errors.New("Telegram method denied")
	}
	if !telegramTokenPattern.MatchString(token) {
		return &telegramAPIError{Code: 401}
	}
	body, err := json.Marshal(input)
	if err != nil {
		return errors.New("Telegram request invalid")
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return errors.New("Telegram request invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := telegramHTTP.Do(req)
	if err != nil {
		return errors.New("Telegram connection unavailable")
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil || len(raw) > 1<<20 {
		return errors.New("Telegram response unavailable")
	}
	var envelope struct {
		OK        bool            `json:"ok"`
		Result    json.RawMessage `json:"result"`
		ErrorCode int             `json:"error_code"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return errors.New("Telegram response invalid")
	}
	if !envelope.OK || response.StatusCode != 200 {
		code := envelope.ErrorCode
		if code == 0 {
			code = response.StatusCode
		}
		return &telegramAPIError{Code: code}
	}
	if output != nil && json.Unmarshal(envelope.Result, output) != nil {
		return errors.New("Telegram response invalid")
	}
	return nil
}

type telegramIdentity struct {
	ID    int64 `json:"id"`
	IsBot bool  `json:"is_bot"`
}
type telegramMessage struct {
	From telegramIdentity `json:"from"`
	Chat struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"chat"`
	Text string `json:"text"`
}
type telegramUpdate struct {
	ID       int64            `json:"update_id"`
	Message  *telegramMessage `json:"message"`
	Callback *struct {
		ID      string           `json:"id"`
		From    telegramIdentity `json:"from"`
		Message *telegramMessage `json:"message"`
		Data    string           `json:"data"`
	} `json:"callback_query"`
}

func (s *server) telegramResetAllowed() bool {
	if s.operations == nil {
		return true
	}
	current, err := s.operations.currentFactoryReset()
	return err == nil && (current == nil || !operationIsCurrent(current.State))
}

// One poller for the lifetime of the existing Management Agent; an OS lock also
// prevents a second local Agent process from consuming updates for the bot.
func (s *server) startTelegram(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() { defer close(done); s.telegramLoop(ctx) }()
	return done
}
func telegramPause(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func (s *server) telegramLoop(ctx context.Context) {
	lock, err := os.OpenFile(filepath.Join("/run/justvoxel", "telegram.lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return
	}
	defer lock.Close()
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return
	}
	failures := 0
	lastGeneration := int64(-1)
	registered := int64(-1)
	nextObservation := time.Time{}
	for ctx.Err() == nil {
		s.telegramMu.Lock()
		v, err := s.store.telegramRead()
		if err != nil || !v.Enabled || !s.telegramResetAllowed() {
			s.telegramMu.Unlock()
			if !telegramPause(ctx, time.Second) {
				return
			}
			continue
		}
		generation, token, offset := v.Generation, v.Token, v.Offset
		if generation != lastGeneration {
			failures = 0
			lastGeneration = generation
		}
		pollCtx, cancel := context.WithCancel(ctx)
		s.telegramPollCancel = cancel
		s.telegramMu.Unlock()
		if registered != generation {
			commands := []map[string]string{{"command": "start", "description": "Open JustVoxel"}, {"command": "menu", "description": "Servers and appliance"}, {"command": "status", "description": "Appliance status"}, {"command": "notifications", "description": "Notification preference"}, {"command": "help", "description": "Access and help"}}
			err = telegramRequest(pollCtx, token, "setMyCommands", map[string]any{"commands": commands}, nil)
			if err == nil {
				registered = generation
			}
		}
		var updates []telegramUpdate
		if err == nil {
			err = telegramRequest(pollCtx, token, "getUpdates", map[string]any{"offset": offset, "timeout": 25, "limit": 10, "allowed_updates": []string{"message", "callback_query"}}, &updates)
		}
		cancelled := pollCtx.Err() != nil
		cancel()
		s.telegramMu.Lock()
		s.telegramPollCancel = nil
		current, readErr := s.store.telegramRead()
		if readErr != nil || current.Generation != generation || !current.Enabled || cancelled {
			s.telegramMu.Unlock()
			continue
		}
		if err != nil {
			failures++
			current.Status = "connection_retry"
			var apiError *telegramAPIError
			terminal := errors.As(err, &apiError) && (apiError.Code == 409 || apiError.Code == 401)
			if terminal || failures >= 8 {
				current.Enabled = false
				current.Status = "connection_paused"
				current.invalidate()
			}
			saveErr := s.store.telegramSave(current, nil, "")
			s.telegramMu.Unlock()
			if saveErr != nil {
				return
			}
			delay := time.Second << min(failures, 6)
			if !telegramPause(ctx, delay) {
				return
			}
			continue
		}
		failures = 0
		current.Status = "connected"
		if err = s.store.telegramSave(current, nil, ""); err != nil {
			s.telegramMu.Unlock()
			return
		}
		s.telegramMu.Unlock()
		for _, update := range updates {
			if ctx.Err() != nil {
				return
			}
			s.telegramProcessUpdate(ctx, generation, update)
		}
		if time.Now().After(nextObservation) {
			s.telegramObserve(ctx, generation)
			nextObservation = time.Now().Add(time.Minute)
		}
	}
}

func (s *server) telegramProcessUpdate(ctx context.Context, generation int64, update telegramUpdate) {
	s.telegramExecutionMu.RLock()
	defer s.telegramExecutionMu.RUnlock()
	s.telegramMu.Lock()
	defer s.telegramMu.Unlock()
	v, err := s.store.telegramRead()
	if err != nil || !v.Enabled || v.Generation != generation || update.ID < v.Offset || !s.telegramResetAllowed() {
		return
	}
	// Persist consumption before dispatch. An interrupted request is never replayed.
	v.Offset = update.ID + 1
	if s.store.telegramSave(v, nil, "") != nil {
		return
	}
	var userID int64
	var message *telegramMessage
	callback := ""
	if update.Callback != nil {
		userID, message, callback = update.Callback.From.ID, update.Callback.Message, update.Callback.Data
		if update.Callback.From.IsBot {
			return
		}
	} else if update.Message != nil {
		userID, message = update.Message.From.ID, update.Message
		if update.Message.From.IsBot {
			return
		}
	}
	if userID <= 0 || message == nil || message.Chat.Type != "private" || message.Chat.ID != userID {
		return
	}
	if update.Callback != nil {
		_ = telegramRequest(ctx, v.Token, "answerCallbackQuery", map[string]any{"callback_query_id": update.Callback.ID}, nil)
	}
	if _, approved := v.user(userID); !approved {
		if update.Callback == nil && telegramClaimPair(&v, userID, message.Text, s.store.now()) {
			if s.store.telegramSave(v, nil, "") == nil {
				telegramPublicReply(ctx, v.Token, userID, "Pairing request received. The primary administrator must approve your numeric Telegram ID in JustVoxel.")
			}
		}
		return
	}
	if callback != "" {
		cb, allowed := v.consumeCallback(callback, userID, s.store.now())
		if s.store.telegramSave(v, nil, "") != nil {
			return
		}
		if !allowed {
			actor := telegramActor(userID)
			_ = s.store.recordAuditEvent(actor, "telegram_denied", "telegram", false, "expired or unauthorized callback")
			s.telegramSend(ctx, &v, userID, "This button expired or access changed. Open /menu for current choices.", nil)
			return
		}
		s.telegramDispatch(ctx, &v, userID, cb)
	} else {
		s.telegramCommand(ctx, &v, userID, message.Text)
	}
}

func telegramPublicReply(ctx context.Context, token string, id int64, text string) {
	replyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = telegramRequest(replyCtx, token, "sendMessage", map[string]any{"chat_id": id, "text": text, "protect_content": true}, nil)
}
