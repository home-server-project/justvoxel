package main

// These transport schemas document common requests; authoritative operation
// handlers still validate every field and all plan/ownership/safety conditions.
// The remaining workflows retain their existing typed requests as documented in
// the Management API, rather than introducing another business model.
func gatewayRequestSchema(name, method string) map[string]any {
	if method == "GET" {
		return nil
	}
	object := func(properties map[string]any, required ...string) map[string]any {
		schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	}
	stringField := map[string]any{"type": "string"}
	switch name {
	case "systemd":
		return object(map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{"status", "start", "stop", "restart", "enable", "disable", "mask", "unmask", "daemon-reload"}},
			"unit":   map[string]any{"type": "string", "maxLength": 210, "description": "One validated unit name. Omit for daemon-reload."},
		}, "action")
	case "instance_start", "instance_restart", "instance_stop":
		return object(map[string]any{"confirm_players": map[string]any{"type": "boolean"}})
	case "whitelist_change", "minecraft_network_whitelist":
		return object(map[string]any{"platform": map[string]any{"type": "string", "enum": []string{"java", "bedrock"}}, "action": map[string]any{"type": "string", "enum": []string{"add", "remove"}}, "name": stringField}, "platform", "action", "name")
	case "minecraft_network_configure":
		return object(map[string]any{"enabled": map[string]any{"type": "boolean"}, "members": map[string]any{"type": "array", "items": stringField}, "java_port": map[string]any{"type": "integer"}, "bedrock_port": map[string]any{"type": "integer"}, "welcome_message": stringField, "whitelist_enforced": map[string]any{"type": "boolean"}}, "enabled", "members", "java_port", "bedrock_port", "welcome_message")
	case "minecraft_network_recover":
		return object(map[string]any{})
	case "instance_rename":
		return object(map[string]any{"display_name": stringField}, "display_name")
	case "instance_configure":
		return object(map[string]any{"runtime": map[string]any{"type": "object", "description": "Public instance runtime settings. Existing runtime validation is authoritative."}}, "runtime")
	default:
		return map[string]any{"type": "object", "description": "Existing typed appliance-operation request. Retain discovery/plan fields and opaque selectors; unknown fields are rejected by the authoritative handler."}
	}
}
