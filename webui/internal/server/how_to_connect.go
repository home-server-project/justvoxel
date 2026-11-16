package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

// Only connection presentation data reaches this page, never provider auth data.
type connectionEndpoint struct {
	Edition     string
	Address     string
	Port        string
	JoinAddress string
}
type connectionMethod struct {
	ID        string
	Name      string
	Note      string
	Status    string
	Available bool
	Default   bool
	Endpoints []connectionEndpoint
}
type connectionGuide struct {
	Configured  bool
	Server      string
	ServerName  string
	Bedrock     bool
	JavaPort    int
	BedrockPort int
	Methods     []connectionMethod
}
type connectionStatusAPI interface {
	RemoteAccessStatus(context.Context, string) (api.RemoteAccessStatus, error)
}

func (a *App) howToConnect(w http.ResponseWriter, r *http.Request) {
	if mustChange(r) {
		http.Redirect(w, r, "/password", http.StatusSeeOther)
		return
	}
	session, ok := sessionFromRequest(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	client, ok := a.api.(sessionIdentityAPI)
	if !ok {
		http.Error(w, ui(w, "ui.db238a1fe47d"), http.StatusServiceUnavailable)
		return
	}
	identity, err := client.Session(r.Context(), session)
	if err != nil {
		if errors.Is(err, api.ErrPasswordChangeRequired) {
			a.handleDashboardError(w, r, err)
		} else {
			a.handleRolePageError(w, r, err)
		}
		return
	}
	if identity.MustChange {
		http.Redirect(w, r, "/password", http.StatusSeeOther)
		return
	}
	switch identity.Role {
	case "administrator", "operator", "viewer":
	default:
		http.Error(w, ui(w, "ui.cc11d415d932"), http.StatusForbidden)
		return
	}
	status, err := a.api.Status(r.Context(), session)
	if err != nil {
		a.handleDashboardError(w, r, err)
		return
	}
	instances, err := a.independentServerInventory(r.Context(), session)
	if err != nil {
		a.handleDashboardError(w, r, err)
		return
	}
	selected := selectedServerContext{}
	values, explicit := r.URL.Query()["server"]
	if len(instances) == 0 && len(values) == 1 && values[0] == "" {
		explicit = false
	}
	if explicit || len(instances) == 1 {
		var valid bool
		selected, valid = a.selectedMinecraft(w, r, session)
		if !valid {
			return
		}
	}
	choices := []selectedServerContext{}
	for _, instance := range instances {
		name := strings.TrimSpace(instance.DisplayName)
		if name == "" {
			name = ui(w, "ui.93383944c4b5")
		}
		choices = append(choices, selectedServerContext{ID: instance.ID, Name: name})
	}
	status.Minecraft = (api.Status{}).Minecraft
	if selected.ID != "" {
		client, ok := a.api.(interface {
			MinecraftInstanceConfiguration(context.Context, string, string) (api.InstanceRuntimeSettings, error)
		})
		if ok {
			configuration, configErr := client.MinecraftInstanceConfiguration(r.Context(), session, selected.ID)
			if configErr == nil {
				configuration, err = a.minecraftConnectionSettings(r.Context(), session, selected.ID, configuration)
				if err != nil {
					a.handleDashboardError(w, r, err)
					return
				}
				status.Minecraft = instancePresentation(configuration).Minecraft
			} else if errors.Is(configErr, api.ErrUnauthorized) || errors.Is(configErr, api.ErrPasswordChangeRequired) {
				a.handleDashboardError(w, r, configErr)
				return
			}
		}
	}
	var remote api.RemoteAccessStatus
	if status.Minecraft.Configured {
		if client, ok := a.api.(connectionStatusAPI); ok {
			remote, err = client.RemoteAccessStatus(r.Context(), session)
			if err != nil {
				a.handleDashboardError(w, r, err)
				return
			}
		}
	}
	a.render(w, "how_to_connect.html", pageData{Title: ui(w, "ui.2198ec8ff357"), SelectedMinecraftID: selected.ID, ConnectionServers: choices, ConnectionGuide: newConnectionGuide(status, remote, selected.Name)})
}

func usableConnectionIP(value string, ipv4 bool) bool {
	ip, err := netip.ParseAddr(value)
	return err == nil && (!ipv4 || ip.Is4()) && ip.Zone() == "" && ip.IsGlobalUnicast() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}

func validConnectionPort(port int) bool { return port > 0 && port <= 65535 }

var connectionNumericHost = regexp.MustCompile(`^[0-9.]+$`)
var connectionHostname = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)

func validConnectionHost(host string) bool {
	if usableConnectionIP(host, false) {
		return true
	}
	if connectionNumericHost.MatchString(host) || len(host) > 253 || strings.ContainsAny(host, ":/@\\ \t\r\n") {
		return false
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if !connectionHostname.MatchString(label) {
			return false
		}
	}
	return host != ""
}

// The producer forwards Playit display_address unchanged. An explicit public
// port is split for Bedrock; a bare hostname remains bare and gains no local port.
func playitConnectionEndpoint(value, edition string) (connectionEndpoint, bool) {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		if !validConnectionHost(value) {
			return connectionEndpoint{}, false
		}
		return connectionEndpoint{Edition: edition, Address: value, JoinAddress: value}, true
	}
	number, err := strconv.Atoi(port)
	if err != nil || !validConnectionPort(number) || !validConnectionHost(host) {
		return connectionEndpoint{}, false
	}
	return connectionEndpoint{Edition: edition, Address: host, Port: port, JoinAddress: value}, true
}
func localConnectionEndpoints(host string, status api.Status) []connectionEndpoint {
	endpoints := []connectionEndpoint{}
	add := func(edition string, port int) {
		if validConnectionPort(port) {
			endpoints = append(endpoints, connectionEndpoint{Edition: edition, Address: host, Port: strconv.Itoa(port), JoinAddress: net.JoinHostPort(host, strconv.Itoa(port))})
		}
	}
	add("Java", status.Minecraft.JavaPort)
	if status.Minecraft.Bedrock {
		add("Bedrock", status.Minecraft.BedrockPort)
	}
	return endpoints
}
func newConnectionGuide(status api.Status, remote api.RemoteAccessStatus, serverName string) *connectionGuide {
	mc := status.Minecraft
	software := mc.ServerSoftware
	if software == "" {
		software = mc.ServerType
	}
	serverName = strings.TrimSpace(serverName)
	if serverName == "" {
		serverName = uiText("ui.93383944c4b5")
	}
	guide := &connectionGuide{Configured: mc.Configured, Server: serverName, ServerName: serverName, Bedrock: mc.Bedrock, JavaPort: mc.JavaPort, BedrockPort: mc.BedrockPort}
	if !mc.Configured {
		return guide
	}
	if label := strings.TrimSpace(software + " " + mc.Version); label != "" {
		guide.Server += " — " + label
	}
	guide.Methods = []connectionMethod{
		{ID: "custom", Name: uiText("ui.d7d4c1e416bc"), Note: uiText("ui.d3d4c5b7fa99"), Available: validConnectionPort(mc.JavaPort), Status: uiText("ui.5a1393ad3611")},
		{ID: "local", Name: uiText("ui.017d5675a6d9"), Note: uiText("ui.49ced9597e88"), Status: uiText("ui.ca1844969742")},
		{ID: "tailscale", Name: "Tailscale", Note: uiText("ui.0616790ce939"), Status: uiText("ui.ca1844969742")},
		{ID: "netbird", Name: "NetBird", Note: uiText("ui.dd8be4917318"), Status: uiText("ui.ca1844969742")},
		{ID: "playit", Name: "Playit.gg", Note: uiText("ui.50f2025719ef"), Status: uiText("ui.ca1844969742")},
		{ID: "nordvpn", Name: uiText("ui.efbb9e0df755"), Note: uiText("ui.a2384f9ffab8"), Status: uiText("ui.ca1844969742")},
	}
	local := &guide.Methods[1]
	if usableConnectionIP(status.System.IPv4, true) {
		local.Endpoints = localConnectionEndpoints(status.System.IPv4, status)
		local.Available = len(local.Endpoints) > 0
		local.Default = local.Available
		if local.Available {
			local.Status = uiText("ui.e674447337e8")
		}
	}
	for i := 2; i < len(guide.Methods); i++ {
		method := &guide.Methods[i]
		for _, provider := range remote.Providers {
			if provider.ID != method.ID {
				continue
			}
			// Friendly producer states only; never echo arbitrary setup/auth text.
			switch provider.Summary {
			case "Connected", "Not connected", "Stopped", "Connecting", "Stopping", "Not configured", "Unavailable", "Disabled", "No tunnels", "Running", "Setup pending", "Awaiting approval", "Starting":
				method.Status = provider.Summary
			}
			if !provider.Connected {
				break
			}
			if method.ID != "playit" {
				if usableConnectionIP(provider.IP, false) && (method.ID != "nordvpn" || validConnectionPort(mc.JavaPort)) {
					method.Endpoints = localConnectionEndpoints(provider.IP, status)
				}
			} else if provider.ServiceActive {
				for _, tunnel := range provider.Tunnels {
					if tunnel.Disabled {
						continue
					}
					host, port, err := net.SplitHostPort(tunnel.Destination)
					if err != nil {
						continue
					}
					// Known local destination format: host:port. Do not match remote hosts
					// or ambiguous equal Java/Bedrock ports without protocol evidence.
					if host != "127.0.0.1" && host != "::1" && !(usableConnectionIP(status.System.IPv4, true) && host == status.System.IPv4) {
						continue
					}
					number, err := strconv.Atoi(port)
					if err != nil || !validConnectionPort(number) {
						continue
					}
					edition := ""
					if number == mc.JavaPort {
						edition = "Java"
					}
					if mc.Bedrock && number == mc.BedrockPort {
						if edition != "" {
							continue
						}
						edition = "Bedrock"
					}
					if edition == "" {
						continue
					}
					if endpoint, ok := playitConnectionEndpoint(tunnel.DisplayAddress, edition); ok {
						method.Endpoints = append(method.Endpoints, endpoint)
					}
				}
			}
			method.Available = len(method.Endpoints) > 0
			if method.Available {
				method.Status = uiText("ui.e674447337e8")
			} else if provider.Connected {
				method.Status = uiText("ui.7cf298fe8158")
			}
			break
		}
	}
	return guide
}

// Only the current guide's default method contributes to a direct invitation.
func defaultConnectionInvitation(guide *connectionGuide, locales ...string) string {
	locale := "en"
	if len(locales) > 0 && supportedLocale(locales[0]) {
		locale = locales[0]
	}
	methods := []string{}
	for _, method := range guide.Methods {
		if !method.Default || !method.Available {
			continue
		}
		lines := []string{translateSource(locale, method.Name)}
		for _, endpoint := range method.Endpoints {
			if endpoint.Edition == "Java" && endpoint.JoinAddress != "" {
				lines = append(lines, "Java: "+endpoint.JoinAddress)
			}
			if endpoint.Edition == "Bedrock" && endpoint.Address != "" && endpoint.Port != "" {
				lines = append(lines, "Bedrock: "+net.JoinHostPort(endpoint.Address, endpoint.Port))
			}
		}
		if len(lines) > 1 {
			methods = append(methods, strings.Join(lines, "\n"))
		}
	}
	if len(methods) == 0 {
		return ""
	}
	return translate(locale, "ui.2cd990a0a03f", "name", guide.ServerName) + "\n\n" + strings.Join(methods, "\n\n")
}

func (a *App) minecraftServersInvitation(w http.ResponseWriter, r *http.Request) {
	// Provider/discovery errors may contain internal details. Only auth errors and
	// fixed presentation messages cross this read-only boundary.
	failure := func(err error, message string) {
		if errors.Is(err, api.ErrUnauthorized) || errors.Is(err, api.ErrPasswordChangeRequired) {
			a.writeMinecraftWorkspaceAPIError(w, err, message)
		} else {
			writeMinecraftWorkspaceError(w, 502, message)
		}
	}
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator", "operator", "viewer")
	if !ok {
		return
	}
	selected, ok := a.explicitCardServer(w, r, session)
	if !ok {
		return
	}
	client, ok := a.api.(minecraftServerInventoryAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.54c7a930a635"))
		return
	}
	configuration, err := client.MinecraftInstanceConfiguration(r.Context(), session, selected.ID)
	if err != nil {
		failure(err, ui(w, "ui.5f084d0e0517"))
		return
	}
	status, err := a.api.Status(r.Context(), session)
	if err != nil {
		failure(err, ui(w, "ui.54c7a930a635"))
		return
	}
	configuration, err = a.minecraftConnectionSettings(r.Context(), session, selected.ID, configuration)
	if err != nil {
		failure(err, ui(w, "ui.54c7a930a635"))
		return
	}
	status.Minecraft = instancePresentation(configuration).Minecraft
	var remote api.RemoteAccessStatus
	if status.Minecraft.Configured {
		if client, ok := a.api.(connectionStatusAPI); ok {
			remote, err = client.RemoteAccessStatus(r.Context(), session)
			if err != nil {
				failure(err, ui(w, "ui.54c7a930a635"))
				return
			}
		}
	}
	guide := newConnectionGuide(status, remote, selected.Name)
	invitation := defaultConnectionInvitation(guide, requestLocale(r))
	if invitation == "" {
		writeMinecraftWorkspaceError(w, 409, ui(w, "ui.c6372025b3ab"))
		return
	}
	writeMinecraftWorkspaceJSON(w, 200, struct {
		Name       string `json:"name"`
		Invitation string `json:"invitation"`
	}{guide.ServerName, invitation})
}
