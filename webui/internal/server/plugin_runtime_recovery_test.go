package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type pluginRuntimeRecoveryTestAPI struct {
	*serverInventoryTestAPI
	request api.PluginInstallRestoreRuntimeRequest
	target  string
	calls   int
}

func (f *pluginRuntimeRecoveryTestAPI) AdminPluginInstallAcknowledge(ctx context.Context, _ string, req api.PluginInstallAcknowledgeRequest) (api.PersistentOperationResponse, error) {
	f.target = api.MinecraftMaintenanceInstance(ctx)
	return api.PersistentOperationResponse{}, &api.ResponseError{StatusCode: 409, PluginRecovery: &api.PluginRuntimeRecovery{
		ID: f.target, RecoveryRequired: true, RecoveryKind: "plugin_install", RecoveryOperationID: req.OperationID,
		RecoveryAction: "stop", Message: "Minecraft was stopped before this operation but is currently running.",
	}}
}

func (f *pluginRuntimeRecoveryTestAPI) AdminPluginInstallRestoreRuntime(ctx context.Context, _ string, req api.PluginInstallRestoreRuntimeRequest) (api.PersistentOperationResponse, error) {
	f.calls++
	f.target, f.request = api.MinecraftMaintenanceInstance(ctx), req
	if !req.ConfirmPlayers {
		return api.PersistentOperationResponse{}, &api.ResponseError{StatusCode: 409, PluginRecovery: &api.PluginRuntimeRecovery{
			ID: f.target, RecoveryRequired: true, RecoveryKind: "plugin_install", RecoveryOperationID: req.OperationID,
			RecoveryAction: "stop", ConfirmationRequired: true, Online: 1, Players: []string{"Alex"}, Message: "Minecraft was stopped before this operation but is currently running.",
		}}
	}
	return api.PersistentOperationResponse{Operation: &api.PersistentOperation{InstanceID: f.target, OperationType: "plugin_install",
		OperationID: req.OperationID, State: "resolved", Status: "Plugin state verified"}}, nil
}

func TestPluginWorkspaceRuntimeRecoveryContracts(t *testing.T) {
	const operationID = "12345678-1234-4123-8123-123456789abc"
	_, base := newServerInventoryApp(t)
	f := &pluginRuntimeRecoveryTestAPI{serverInventoryTestAPI: base}
	app, err := New(f, Config{})
	if err != nil {
		t.Fatal(err)
	}
	request := func(route, selection, form string) (int, string) {
		t.Helper()
		rr := httptestResponse(app, authenticatedAdminRequest(http.MethodPost, "/api/plugins/workspace/install/"+route+selection, form))
		return rr.Code, rr.Body.String()
	}
	const selected = "?server=jv-kids001"
	const form = "csrf=csrf-token&operation_id=" + operationID
	const explicit = form + "&restore_runtime=true"
	for _, tc := range []struct {
		route, selection, form, role string
		want                         int
	}{
		{"acknowledge", selected, explicit, "administrator", 400},
		{"restore-runtime", selected, form, "administrator", 400},
		{"restore-runtime", selected, explicit + "&action=stop", "administrator", 400},
		{"restore-runtime", selected, explicit + "&unit=minecraft.service", "administrator", 400},
		{"restore-runtime", selected, explicit + "&container=minecraft", "administrator", 400},
		{"restore-runtime", selected, explicit + "&data_path=/private", "administrator", 400},
		{"restore-runtime", selected, explicit + "&confirm_players=1", "administrator", 400},
		{"restore-runtime", selected, explicit + "&confirm_players=true&confirm_players=false", "administrator", 400},
		{"restore-runtime", selected, "csrf=wrong&operation_id=" + operationID + "&restore_runtime=true", "administrator", 403},
		{"restore-runtime", "?server=jv-missing1", explicit, "administrator", 404},
		{"restore-runtime", selected, explicit, "operator", 403},
	} {
		base.role = tc.role
		if code, body := request(tc.route, tc.selection, tc.form); code != tc.want || f.calls != 0 {
			t.Fatal(tc, code, body, f.calls)
		}
	}
	base.role = "administrator"
	code, body := request("acknowledge", selected, form)
	if code != 409 || !strings.Contains(body, `"recovery_action":"stop"`) || f.calls != 0 || f.target != "jv-kids001" {
		t.Fatal("re-check lost the stop recovery or dispatched it", code, body, f.calls, f.target)
	}
	code, body = request("restore-runtime", selected, explicit)
	var stop api.PluginRuntimeRecovery
	if code != 409 || json.Unmarshal([]byte(body), &stop) != nil || !stop.ConfirmationRequired ||
		stop.ID != "jv-kids001" || stop.RecoveryOperationID != operationID || stop.Online != 1 || len(stop.Players) != 1 || stop.Players[0] != "Alex" {
		t.Fatal("stop confirmation lost", code, body)
	}
	if f.calls != 1 || f.request != (api.PluginInstallRestoreRuntimeRequest{OperationID: operationID, RestoreRuntime: true}) || f.target != "jv-kids001" {
		t.Fatal("unconfirmed request changed", f.calls, f.request, f.target)
	}
	code, body = request("restore-runtime", selected, explicit+"&confirm_players=true")
	if code != 200 || !strings.Contains(body, `"state":"resolved"`) || f.calls != 2 || !f.request.ConfirmPlayers {
		t.Fatal("confirmed request lost", code, body, f.calls, f.request)
	}
}

func TestMinecraftServerCardPluginRecoveryDialog(t *testing.T) {
	raw, err := englishPresentationAsset("static/minecraft-servers.js")
	if err != nil {
		t.Fatal(err)
	}
	program := `
const assert = require("node:assert/strict");
const operationID = "12345678-1234-4123-8123-123456789abc", id = "jv-kids001";
const otherOperationID = "12345678-1234-4123-8123-123456789def", otherID = "jv-family1";
class Node {
  constructor(tag) { Object.assign(this, {tag, children: [], events: {}, dataset: {}, checked: false, disabled: false}); }
  addEventListener(name, fn) { this.events[name] = fn; }
  setAttribute() {}
  append(...nodes) { nodes.forEach(node => { node.parent = this; this.children.push(node); }); }
  appendChild(node) { this.append(node); }
  remove() { this.parent.children = this.parent.children.filter(node => node !== this); }
  showModal() { this.open = true; }
  close() { this.open = false; this.events.close(); }
}
const document = { body: new Node("body"), createElement: tag => new Node(tag), querySelector: () => null };
const window = {addEventListener() {}, removeEventListener() {}};
const setInterval = () => 1, clearInterval = () => {};
const response = (status, payload) => ({ok: status < 400, status, json: async () => payload});
let reply, requests, refreshes;
const fetch = async (url, options) => { requests.push([url, Object.fromEntries(options.body)]); return reply.shift(); };
` + string(raw) + `
(async () => {
  for (const scenario of ["start", "stop", "ordinary refusal", "other instance", "invalid operation", "wrong operation on restore", "wrong instance on resolution"]) {
    requests = []; refreshes = 0;
    const running = new Node("input"), feedback = new Node("p"), status = new Node("p");
    const stop = scenario === "stop";
    running.checked = stop;
    const inventory = {dataset: {serverCsrf: "csrf-token"}};
    const selectors = {
      "[data-server-running]": running, ".minecraft-server-state": status,
      "[data-server-move-earlier]": new Node("button"), "[data-server-move-later]": new Node("button"),
    };
    const card = {dataset: {serverId: id}, querySelectorAll: selector => selector === "input, button" ? [running] : [],
                  querySelector: selector => selectors[selector] || null};
    const nodes = {
      ".minecraft-server-entries": {isConnected: false, querySelectorAll: () => [card]},
      "[data-server-arrange]": new Node("button"), "[data-server-arrange-done]": new Node("button"),
      "[data-server-csrf]": inventory, "[data-server-feedback]": feedback,
    };
    const root = {querySelector: selector => nodes[selector] || null,
                  justVoxelRefreshServers: async () => { running.checked = ++refreshes === 1 ? stop : !stop; }};
    globalThis.justVoxelMinecraftServers(root);
    const recovery = {id, recovery_required: true, recovery_kind: "plugin_install", recovery_operation_id: operationID, recovery_action: stop ? "stop" : "start"};
    const resolved = response(200, {operation: {instance_id: id, operation_id: operationID, state: "resolved"}});
    const refusal = "Maintenance requires attention.";
    const refusals = {
      "ordinary refusal": {},
      "other instance": {...recovery, id: otherID},
      "invalid operation": {...recovery, recovery_operation_id: "invalid"},
    };
    const confirmation = {...recovery, confirmation_required: true, online: 1, players: ["Alex"]};
    const restore = {
      "stop": [response(409, confirmation), resolved],
      "wrong operation on restore": [response(409, {...confirmation, recovery_action: "stop", recovery_operation_id: otherOperationID, error: refusal})],
      "wrong instance on resolution": [response(200, {operation: {instance_id: otherID, operation_id: operationID, state: "resolved"}})],
    };
    reply = refusals[scenario] ? [response(409, {...refusals[scenario], error: refusal})] :
      [response(409, recovery), ...(restore[scenario] || [resolved])];
    running.checked = !stop;
    running.events.change();
    await new Promise(resolve => setImmediate(resolve));
    assert.equal(refreshes, 1);
    assert.equal(running.checked, stop);
    assert.deepEqual(requests, [["/api/minecraft/servers/lifecycle?server=" + id, {csrf: "csrf-token", action: stop ? "stop" : "start"}]]);
    if (refusals[scenario]) {
      assert.equal(document.body.children.length, 0);
      assert.equal(feedback.textContent, refusal);
      continue;
    }
    const dialog = document.body.children[0];
    assert.equal(dialog.open, true);
    assert.equal(dialog.children[0].textContent, "Minecraft recovery required");
    const [apply, details, close] = dialog.children[3].children;
    assert.equal(apply.textContent, (stop ? "Stop" : "Start") + " Minecraft and re-check");
    assert.equal(details.href, "/?workspace=plugins&server=" + id);
    assert.equal(details.textContent, "Open recovery details");
    assert.equal(close.textContent, "Close");
    const first = apply.events.click();
    assert.equal(apply.disabled, true);
    await apply.events.click();
    await first;
    assert.deepEqual(requests[1], ["/api/plugins/workspace/install/restore-runtime?server=" + id, {csrf: "csrf-token", operation_id: operationID, restore_runtime: "true"}]);
    if (scenario.startsWith("wrong ")) {
      assert.equal(dialog.open, true);
      assert.equal(apply.textContent, "Start Minecraft and re-check");
      assert.notEqual(dialog.children[2].textContent, "");
      assert.equal(refreshes, 1);
      assert.equal(requests.length, 2);
      close.events.click();
      assert.equal(document.body.children.length, 0);
      continue;
    }
    if (stop) {
      assert.equal(dialog.open, true);
      assert.equal(apply.textContent, "Confirm stop and re-check");
      assert.match(dialog.children[2].textContent, /Players online \(1\).*Alex/);
      assert.equal(requests.length, 2);
      await apply.events.click();
      assert.deepEqual(requests[2], ["/api/plugins/workspace/install/restore-runtime?server=" + id, {csrf: "csrf-token", operation_id: operationID, restore_runtime: "true", confirm_players: "true"}]);
    } else assert.equal(requests.length, 2);
    assert.equal(refreshes, 2);
    assert.equal(running.checked, !stop);
    assert.equal(dialog.open, false);
    assert.equal(document.body.children.length, 0);
  }
})().catch(error => {console.error(error); process.exitCode = 1;});
`
	cmd := nodeTestCommand()
	cmd.Stdin = strings.NewReader(program)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("server card recovery: %v\n%s", err, output)
	}
}

func TestPluginsWorkspaceExplicitRuntimeRecovery(t *testing.T) {
	raw, err := englishPresentationAsset("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	script := string(raw)
	request := pluginScriptSection(t, script, "  const pluginRequest =", "  const pluginDiagnosticLink =")
	operation := pluginScriptSection(t, script, "  const showOperation =", "  const pollOperation =")
	program := `
const assert = require("node:assert/strict");
const id = "jv-kids001", operationID = "12345678-1234-4123-8123-123456789abc";
const persisted = {instance_id: id, operation_id: operationID, operation_type: "plugin_install", state: "needs_attention", status: "Recovery needs verification."};
globalThis.justVoxelSelectionGeneration = {plugins: 1};
globalThis.justVoxelSelectedURL = path => path + "?server=" + id;
const document = {querySelector: () => ({value: "csrf-token"})};
const acceptedServer = {content_kind: "mods"}, contentSingular = () => "mod";
const node = (textContent = "") => ({textContent, children: [], focus() {}, setAttribute() {},
  append(...children) { this.children.push(...children); }, replaceChildren(...children) { this.children = children; }});
const panel = {hidden: true}, panelBody = node();
let operationViewID, operationIdentity, installGeneration = 1, installController, requests, replies, heading, footer, loads;
const showPanel = () => { panel.hidden = false; panelBody.replaceChildren(); };
const overlayHeader = value => { heading = value; };
const panelAction = (label, click) => Object.assign(node(label), {click});
const overlayActions = (...buttons) => { footer = node(); footer.append(...buttons); return footer; };
const text = (_, value) => node(value);
const load = () => { loads++; }, closePanel = () => {}, pluginDiagnosticLink = () => {};
panelBody.querySelector = () => ({set textContent(value) { heading = value; }});
const fetch = async (url, options) => {
  requests.push([url, Object.fromEntries(options.body)]);
  const [status, payload] = replies.shift();
  return {ok: status < 400, status, json: async () => payload};
};
` + request + operation + `
(async () => {
  for (const action of ["start", "stop"]) {
    panel.hidden = true; requests = []; loads = 0;
    const recovery = {id, recovery_required: true, recovery_kind: "plugin_install", recovery_operation_id: operationID, recovery_action: action};
    replies = [[409, recovery], ...(action === "stop" ? [[409, {...recovery, confirmation_required: true, online: 1, players: ["Alex"]}]] : []),
      [200, {operation: {...persisted, state: "resolved", status: "Verified."}}]];
    showOperation(persisted);
    assert.equal(heading, "Administrator attention required");
    assert.equal(footer.children[0].textContent, "Re-check recovery");
    await footer.children[0].click();
    assert.deepEqual(requests, [["/api/plugins/workspace/install/acknowledge?server=" + id, {csrf: "csrf-token", operation_id: operationID}]]);
    assert.equal(footer.children[0].textContent, (action === "start" ? "Start" : "Stop") + " Minecraft and re-check");
    assert.match(panelBody.children.at(-1).textContent, action === "start" ? /was running.*currently stopped/ : /was stopped.*currently running/);
    await footer.children[0].click();
    assert.deepEqual(requests[1], ["/api/plugins/workspace/install/restore-runtime?server=" + id, {csrf: "csrf-token", operation_id: operationID, restore_runtime: "true"}]);
    if (action === "stop") {
      assert.equal(footer.children[0].textContent, "Confirm stop and re-check");
      assert.match(panelBody.children.at(-1).textContent, /Players online \(1\).*Alex/);
      assert.equal(requests.length, 2);
      await footer.children[0].click();
      assert.equal(requests[2][1].confirm_players, "true");
    }
    assert.equal(loads, 1);
    assert.equal(heading, "Plugin state verified");
  }
})().catch(error => {console.error(error); process.exitCode = 1;});
`
	cmd := nodeTestCommand()
	cmd.Stdin = strings.NewReader(program)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Plugins recovery: %v\n%s", err, output)
	}
}
