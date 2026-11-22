package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"time"
)

// RouterOS omits dynamic on some static records. An explicitly unknown value is ambiguous.
func routerStaticRecord(row routerOSRow) bool {
	v, set := row["dynamic"]
	return !set || v == "false" || v == "no"
}

// Reject duplicate keys and non-string values before using a record as ownership evidence.
func decodeRouterNATRow(data []byte) (routerOSRow, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errRouterOSResponse
	}
	row := routerOSRow{}
	for d.More() {
		key, err := d.Token()
		name, ok := key.(string)
		if err != nil || !ok {
			return nil, errRouterOSResponse
		}
		if _, exists := row[name]; exists {
			return nil, errRouterOSResponse
		}
		token, err = d.Token()
		value, ok := token.(string)
		if err != nil || !ok {
			return nil, errRouterOSResponse
		}
		row[name] = value
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') || len(row) == 0 || d.Decode(new(any)) != io.EOF {
		return nil, errRouterOSResponse
	}
	return row, nil
}

const routerNATPath = "/rest/ip/firewall/nat"

func (c routerOSClient) natRecord(ctx context.Context, cred routerOSCredentials, id string) (routerOSRow, error) {
	if !routerRecordID.MatchString(id) || !cred.normalize() {
		return nil, errRouterOSResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cred.Endpoint+routerNATPath+"/"+id, nil)
	if err != nil {
		return nil, errRouterOSResponse
	}
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(cred.Username, cred.Password)
	data, err := routerOSResponse(c.http.Do(req))
	if err != nil {
		return nil, err
	}
	return decodeRouterNATRow(data)
}

const routerChanged routerOSError = "router_changed"

func (c routerOSClient) applyNAT(ctx context.Context, cred routerOSCredentials, scope string, change routerNATChange) error {
	if !cred.normalize() || !routerScopeValid(scope) {
		return errRouterOSResponse
	}
	if !slices.Contains(routerRoles[:], change.role) {
		return errRouterOSResponse
	}
	// Only the planner supplies fields. Restrict its wire format independently of REST paths.
	for field, value := range change.fields {
		switch field {
		case "protocol", "dst-port", "to-addresses", "to-ports", "in-interface", "in-interface-list":
		case "action":
			if value != "dst-nat" {
				return errRouterOSResponse
			}
		case "chain":
			if value != "dstnat" {
				return errRouterOSResponse
			}
		case "comment":
			if change.method != http.MethodPut || value != routerOwnerComment(scope, change.role) {
				return errRouterOSResponse
			}
		case "disabled":
			if change.method != http.MethodPatch || value != "false" {
				return errRouterOSResponse
			}
		default:
			return errRouterOSResponse
		}
	}
	path := routerNATPath
	switch change.method {
	case http.MethodPut:
		if change.id != "" || change.fields["chain"] != "dstnat" || change.fields["action"] != "dst-nat" || change.fields["comment"] != routerOwnerComment(scope, change.role) {
			return errRouterOSResponse
		}
	case http.MethodPatch, http.MethodDelete:
		if !routerRecordID.MatchString(change.id) || (change.method == http.MethodPatch) != (len(change.fields) > 0) {
			return errRouterOSResponse
		}
		row, err := c.natRecord(ctx, cred, change.id)
		if err != nil || row[".id"] != change.id || row["chain"] != "dstnat" || !routerStaticRecord(row) || row["comment"] != routerOwnerComment(scope, change.role) {
			return routerChanged
		}
		path += "/" + change.id
	default:
		return errRouterOSResponse
	}
	var body io.Reader
	if change.method != http.MethodDelete {
		data, err := json.Marshal(change.fields)
		if err != nil {
			return errRouterOSResponse
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, change.method, cred.Endpoint+path, body)
	if err != nil {
		return errRouterOSResponse
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.SetBasicAuth(cred.Username, cred.Password)
	data, err := routerOSResponse(c.http.Do(req))
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(data)) > 0 {
		row, err := decodeRouterNATRow(data)
		if err != nil || row["error"] != "" || !routerRecordID.MatchString(row[".id"]) || (change.id != "" && row[".id"] != change.id) {
			return errRouterOSResponse
		}
	}
	return nil
}

func (s *server) routerOSForwarding(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := s.requirePrimaryAdministrator(w, r)
	if !ok {
		return
	}
	var input struct {
		Scope  string `json:"scope"`
		Action string `json:"action"`
	}
	if r.URL.ForceQuery || r.URL.RawQuery != "" || decodeRouterOSRequest(r.Body, &input) != nil || !routerScopeValid(input.Scope) || (input.Action != "enable" && input.Action != "disable" && input.Action != "repair") {
		_ = s.store.recordAuditEvent(actor, "routeros_forwarding", "mikrotik", false, "invalid_request")
		writeError(w, 400, "invalid_request")
		return
	}
	s.routerOSMu.Lock()
	defer s.routerOSMu.Unlock()
	audit := func(code string) bool {
		return s.store.recordAuditEvent(actor, "routeros_forwarding_"+input.Action, input.Scope, code == "", code) == nil
	}
	if !s.allowRouterOSSave(w) {
		audit("factory_reset_pending")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	out, code := s.mutateForwarding(ctx, input.Scope, input.Action)
	if !audit(string(code)) {
		writeError(w, 503, "forwarding_unknown")
		return
	}
	if code != "" {
		writeError(w, 409, string(code))
		return
	}
	writeJSON(w, 200, out)
}

func (s *server) mutateForwarding(ctx context.Context, scope, action string) (routerOSStatus, routerOSError) {
	var out routerOSStatus
	config, err := s.store.readRouterOS(ctx)
	if err != nil || config == nil {
		return out, "configuration_error"
	}
	client, _, closeIdle, err := routerOSConnection(config.CAPEM)
	if err != nil {
		return out, routerOSErrorCode(err)
	}
	defer closeIdle()
	d, err := client.discover(ctx, config.routerOSCredentials)
	if err != nil {
		return out, routerOSErrorCode(err)
	}
	if !d.has(config.Ingress) {
		return out, "ingress_unavailable"
	}
	snapshot, err := routerNetworkSnapshot(ctx)
	if err != nil {
		return out, "target_unavailable"
	}
	candidates, _ := routerForwardingCandidates(config.Endpoint, snapshot)
	if !containsRouterTarget(candidates, config.Target) {
		return out, "target_unavailable"
	}
	if out.inventory(scope) != nil || out.ForwardingScope != scope || len(out.Endpoints) == 0 {
		return out, "inventory_error"
	}
	return client.forwardNAT(ctx, *config, d, out, action)
}

func (c routerOSClient) forwardNAT(ctx context.Context, config routerOSConfiguration, d routerOSDiscovery, out routerOSStatus, action string) (routerOSStatus, routerOSError) {
	scope := out.ForwardingScope
	readPlan := func(ctx context.Context, disable bool) (routerNATPlan, error) {
		rows, err := c.readList(ctx, config.routerOSCredentials, routerOSNAT)
		if err != nil {
			return routerNATPlan{}, err
		}
		return planRouterNAT(scope, out.Endpoints, config, rows, d, disable), nil
	}
	disable := action == "disable"
	plan, err := readPlan(ctx, disable)
	if err != nil {
		return out, routerOSErrorCode(err)
	}
	if plan.state == routerForwardConflict {
		return out, "forwarding_conflict"
	}
	if plan.state == routerForwardUnknown {
		return out, "forwarding_unknown"
	}
	if !disable && len(plan.changes) > 0 && plan.action != action {
		return out, routerChanged
	}
	expectedOwned := plan.owned
	for _, change := range plan.changes {
		if err := c.applyNAT(ctx, config.routerOSCredentials, scope, change); err != nil {
			// Even an unsuccessful write response may follow a committed router change.
			probe, cancel := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
			actual, readErr := readPlan(probe, false)
			cancel()
			if err == routerChanged {
				return out, routerChanged
			}
			if readErr != nil || actual.state == routerForwardUnknown {
				return out, "forwarding_unknown"
			}
			return out, "forwarding_incomplete"
		}
		switch change.method {
		case http.MethodPut:
			expectedOwned++
		case http.MethodDelete:
			expectedOwned--
		}
	}
	actual, err := readPlan(ctx, false)
	if err != nil || actual.state == routerForwardUnknown {
		return out, "forwarding_unknown"
	}
	if actual.state == routerForwardConflict || actual.owned != expectedOwned || (disable && actual.owned != 0) || (!disable && (actual.state != routerForwardOpen || len(actual.changes) != 0)) {
		return out, "forwarding_incomplete"
	}
	out.Configured, out.Status = true, "connected"
	out.ForwardingAction, out.ForwardingState, out.ExistingForwarding = actual.action, actual.state, actual.existing
	out.Endpoints = actual.endpoints
	return out, ""
}
