package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGatewayMinecraftNetworkCatalogLevelsAndApplianceScope(t *testing.T) {
	for level := 1; level <= 4; level++ {
		store, _ := openTestWebUIStore(t)
		s := &server{store: store}
		c, token := createTestGatewayConnection(t, store, level)
		for _, all := range []bool{true, false} {
			c.AllInstances = all
			c.InstanceIDs = nil
			if !all {
				c.InstanceIDs = []string{"jv-abcdef"}
			}
			if _, _, err := store.gatewaySave(c, false, true, gatewayTestActor); err != nil {
				t.Fatal(err)
			}
			catalog := gatewayTestCatalog(t, s, token)
			visible := map[string]bool{}
			for _, op := range catalog {
				visible[op.Name] = true
			}
			if !visible["minecraft_network_status"] {
				t.Fatal("sanitized status missing", level, all)
			}
			for _, name := range []string{"minecraft_network_configure", "minecraft_network_whitelist", "minecraft_network_recover", "minecraft_network_whitelist_status"} {
				if visible[name] != (level >= 3 && all) {
					t.Fatal("network catalog scope", level, all, name)
				}
				if level < 3 || !all {
					rr := gatewayTestRequest(s, token, gatewayCall{Operation: name, Body: json.RawMessage(`{}`)})
					if rr.Code != 403 {
						t.Fatal("denied Network handler was reachable", level, all, name, rr.Code)
					}
				}
			}
		}
	}
}
func TestGatewayMinecraftNetworkProjectionOmitsUnauthorizedInstancesAndSecrets(t *testing.T) {
	config := networkDefaultState().Config
	config.Enabled = true
	config.Members = []string{"jv-abcdef", "jv-fedcba"}
	scoped := gatewayConnection{Level: 1, InstanceIDs: []string{"jv-abcdef"}}
	servers := []map[string]any{{"id": "jv-abcdef", "software": "paper", "eligible": true, "state": "running", "forwarding_secret": "private"}, {"id": "jv-fedcba", "software": "purpur", "eligible": true, "state": "running"}}
	raw, _ := json.Marshal(gatewayProjectNetwork(scoped, config, servers, false, true))
	for _, forbidden := range []string{"jv-fedcba", "forwarding_secret", "private", "whitelist", "welcome_message"} {
		// whitelist_enforced is an intentional public boolean.
		if forbidden == "whitelist" {
			if strings.Contains(string(raw), `"whitelist":`) {
				t.Fatal("network identities disclosed")
			}
			continue
		}
		if strings.Contains(string(raw), forbidden) {
			t.Fatal("projection leaked unauthorized field", forbidden)
		}
	}
	if !strings.Contains(string(raw), "jv-abcdef") || !strings.Contains(string(raw), `"eligible":true`) {
		t.Fatal("authorized member missing")
	}
}

func TestGatewayNetworkFieldsDoNotExpandExistingResponsePolicy(t *testing.T) {
	value := map[string]any{"config": map[string]any{"members": []any{"jv-abcdef"}, "forwarding_secret": "private"}, "servers": []any{map[string]any{"id": "jv-abcdef", "eligible": true, "token": "private"}}}
	old, _ := json.Marshal(gatewayPublicValue(value))
	if string(old) != "{}" {
		t.Fatal("Network response fields changed established gateway projection", string(old))
	}
	network, _ := json.Marshal(gatewayNetworkPublicValue(value))
	if !strings.Contains(string(network), "jv-abcdef") || strings.Contains(string(network), "private") {
		t.Fatal("Network projection missing public fields or exposing private fields", string(network))
	}
}
