package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type gatewayScopeKey struct{}

// The existing administrator status handler remains authoritative. This curated
// read grants no mutation rights and projects only explicit public fields.
func (s *server) gatewayMinecraftNetworkStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := r.Context().Value(gatewayPrincipalKey{}).(session)
	c, scoped := r.Context().Value(gatewayScopeKey{}).(gatewayConnection)
	if !ok || !scoped {
		writeError(w, 403, "connection scope required")
		return
	}
	actor.Role = roleAdministrator
	capture := &gatewayResponse{header: make(http.Header), status: 200}
	s.minecraftNetworkStatus(capture, r.WithContext(context.WithValue(r.Context(), gatewayPrincipalKey{}, actor)))
	if capture.status != 200 || capture.overflow {
		writeError(w, 503, "Minecraft Network status unavailable")
		return
	}
	var status struct {
		Config           minecraftNetworkConfig `json:"config"`
		Servers          []map[string]any       `json:"servers"`
		RecoveryRequired bool                   `json:"recovery_required"`
		RuntimeReady     bool                   `json:"runtime_ready"`
	}
	if json.Unmarshal(capture.body.Bytes(), &status) != nil {
		writeError(w, 503, "Minecraft Network status unavailable")
		return
	}
	writeJSON(w, 200, gatewayProjectNetwork(c, status.Config, status.Servers, status.RecoveryRequired, status.RuntimeReady))
}

func gatewayProjectNetwork(c gatewayConnection, config minecraftNetworkConfig, servers []map[string]any, recovery, ready bool) map[string]any {
	members := []string{}
	for _, id := range config.Members {
		if c.allowsInstance(id) {
			members = append(members, id)
		}
	}
	visible := []map[string]any{}
	for _, server := range servers {
		id, _ := server["id"].(string)
		if !c.allowsInstance(id) {
			continue
		}
		item := map[string]any{"id": id, "network_member": config.Enabled && networkContains(members, id)}
		for _, key := range []string{"software", "eligible", "compatibility_status", "compatibility_reason", "state"} {
			if value, ok := server[key]; ok {
				item[key] = value
			}
		}
		visible = append(visible, item)
	}
	return map[string]any{"config": map[string]any{"enabled": config.Enabled, "java_port": config.JavaPort, "bedrock_port": config.BedrockPort, "members": members, "whitelist_enforced": config.whitelistEnforced()}, "servers": visible, "recovery_required": recovery, "runtime_ready": ready}
}

// Network's response fields are local to these new reads. Adding them must not
// expand the public field policy of any established gateway operation.
var gatewayNetworkPublicFields = func() map[string]bool {
	out := map[string]bool{}
	for _, key := range strings.Fields("config enabled java_port bedrock_port members whitelist_enforced servers id software eligible compatibility_status compatibility_reason state recovery_required runtime_ready whitelist name platform welcome_message network_member ok error") {
		out[key] = true
	}
	return out
}()

func gatewayNetworkPublicValue(value any) any {
	switch item := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, child := range item {
			if gatewayNetworkPublicFields[key] {
				out[key] = gatewayNetworkPublicValue(child)
			}
		}
		return out
	case []any:
		out := make([]any, 0, len(item))
		for _, child := range item {
			out = append(out, gatewayNetworkPublicValue(child))
		}
		return out
	default:
		return gatewayPublicValue(value)
	}
}
