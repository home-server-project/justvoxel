package main

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestRouterForwardingTransactions(t *testing.T) {
	const scope = "jv-family1"
	desired := routerOSRow{"chain": "dstnat", "action": "dst-nat", "protocol": "tcp", "dst-port": "25565", "to-addresses": "192.0.2.9", "to-ports": "25565", "in-interface-list": "uplinks", "comment": "JustVoxel | jv-family1 | java"}
	owned := maps.Clone(desired)
	owned[".id"] = "*1"
	drift := maps.Clone(owned)
	drift["to-ports"] = "25566"
	extra := maps.Clone(owned)
	extra[".id"] = "*2"
	extra["comment"] = routerOwnerComment(scope, "bedrock-tcp")
	foreign := maps.Clone(owned)
	foreign[".id"], foreign["comment"] = "*3", "manual"
	changed := maps.Clone(drift)
	changed["comment"] = routerOwnerComment(scope, "bedrock-udp")
	wrongID := maps.Clone(drift)
	wrongID[".id"] = "*A"
	dynamic := maps.Clone(drift)
	dynamic["dynamic"] = "true"
	wrongChain := maps.Clone(drift)
	wrongChain["chain"] = "srcnat"
	conflict := maps.Clone(foreign)
	conflict["to-addresses"] = "192.0.2.10"
	unknown := maps.Clone(conflict)
	unknown["src-address-list"] = "limited"
	type request struct {
		method, id string
		status     int
		response   any
		fields     routerOSRow
	}
	table := func(rows ...routerOSRow) request {
		return request{"GET", "", 200, append([]routerOSRow{}, rows...), nil}
	}
	record := func(row routerOSRow) request { return request{"GET", row[".id"], 200, row, nil} }
	patch := request{"PATCH", "*1", 200, owned, routerOSRow{"to-ports": "25565"}}
	remove := func(id string) request { return request{"DELETE", id, 204, nil, nil} }
	put := request{"PUT", "", 201, owned, desired}
	for _, tc := range []struct {
		name, action string
		steps        []request
		code         routerOSError
		state        string
	}{
		{"enable", "enable", []request{table(), put, table(owned)}, "", "open"},
		{"repeat enable", "enable", []request{table(owned), table(owned)}, "", "open"},
		{"repair and obsolete", "repair", []request{table(extra, drift), record(drift), patch, record(extra), remove("*2"), table(owned)}, "", "open"},
		{"disable preserves foreign", "disable", []request{table(owned, foreign), record(owned), remove("*1"), table(foreign)}, "", "open"},
		{"foreign conflict", "disable", []request{table(owned, conflict)}, "forwarding_conflict", ""},
		{"unknown", "enable", []request{table(unknown)}, "forwarding_unknown", ""},
		{"ownership changed before patch", "repair", []request{table(drift, extra), record(changed), table(changed, extra)}, routerChanged, ""},
		{"ownership changed before delete", "disable", []request{table(owned, extra), record(changed), table(changed, extra)}, routerChanged, ""},
		{"record missing", "disable", []request{table(owned, extra), {"GET", "*1", 404, nil, nil}, table(extra)}, routerChanged, ""},
		{"id mismatch", "disable", []request{table(owned), {"GET", "*1", 200, wrongID, nil}, table(owned)}, routerChanged, ""},
		{"dynamic record", "disable", []request{table(owned), record(dynamic), table(dynamic)}, routerChanged, ""},
		{"chain changed", "disable", []request{table(owned), record(wrongChain), table(wrongChain)}, routerChanged, ""},
		{"ambiguous record", "disable", []request{table(owned), {"GET", "*1", 200, json.RawMessage(`{".id":"*1","chain":"dstnat","comment":"manual","comment":"JustVoxel | jv-family1 | java"}`), nil}, table(owned)}, routerChanged, ""},
		{"write failure", "repair", []request{table(drift, extra), record(drift), {"PATCH", "*1", 500, nil, patch.fields}, table(drift, extra)}, "forwarding_incomplete", ""},
		{"partial repair", "repair", []request{table(drift, extra), record(drift), patch, {"GET", "*2", 404, nil, nil}, table(owned)}, routerChanged, ""},
		{"readback failed", "enable", []request{table(), put, {"GET", "", 500, nil, nil}}, "forwarding_unknown", ""},
		{"readback missing ownership", "enable", []request{table(), put, table(foreign)}, "forwarding_incomplete", ""},
		{"readback missing rule", "enable", []request{table(), put, table()}, "forwarding_incomplete", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			next := 0
			router := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if next >= len(tc.steps) {
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
					return
				}
				step := tc.steps[next]
				next++
				path := routerNATPath
				if step.id != "" {
					path += "/" + step.id
				}
				user, pw, ok := r.BasicAuth()
				if r.Method != step.method || r.URL.Path != path || r.URL.RawQuery != "" || !ok || user != "justvoxel" || pw != testRouterPassword {
					t.Errorf("step %d: unexpected %s %s or authentication", next, r.Method, r.URL.Path)
				}
				body, err := io.ReadAll(r.Body)
				var fields routerOSRow
				if err != nil || (len(body) != 0 && json.Unmarshal(body, &fields) != nil) || !maps.Equal(fields, step.fields) {
					t.Errorf("step %d: wrong NAT fields", next)
				}
				w.WriteHeader(step.status)
				if step.response != nil {
					json.NewEncoder(w).Encode(step.response)
				}
			}))
			defer router.Close()
			c := trustTestRouter(t, router)
			d := routerOSDiscovery{interfaces: map[string]routerOSRow{"ether1": {"name": "ether1"}}, lists: map[string]routerOSRow{"uplinks": {"name": "uplinks"}}, members: map[string]map[string]bool{"uplinks": {"ether1": true}}}
			out := routerOSStatus{ForwardingScope: scope, Endpoints: []routerMinecraftEndpoint{{ID: scope, Edition: "java", Protocols: []string{"tcp"}, Port: 25565, LastPort: 25565}}}
			got, code := c.forwardNAT(context.Background(), testRouterConfiguration(router.URL), d, out, tc.action)
			mu.Lock()
			defer mu.Unlock()
			if next != len(tc.steps) || code != tc.code || got.ForwardingState != tc.state {
				t.Fatalf("received %d/%d requests, result %s %s", next, len(tc.steps), code, got.ForwardingState)
			}
			if tc.action == "disable" && tc.state == "open" && (!got.ExistingForwarding || got.ForwardingAction != "") {
				t.Fatal("surviving foreign rule must stay open without a managed action")
			}
		})
	}
}

func TestRouterNATRejectsUnsupportedRequests(t *testing.T) {
	// A zero client will panic if an invalid request reaches the transport.
	c := routerOSClient{}
	cred := testRouterConfiguration("https://router.example").routerOSCredentials
	ctx := context.Background()
	for _, method := range []string{"POST", "GET", "PATCH", "DELETE", "PUT"} {
		change := routerNATChange{method: method, id: "*1/../../system", role: "java"}
		if c.applyNAT(ctx, cred, "jv-family1", change) == nil {
			t.Fatal(method)
		}
	}
	for _, method := range []string{"POST", "MOVE"} {
		if c.applyNAT(ctx, cred, "jv-family1", routerNATChange{method: method, id: "*1", role: "java"}) == nil {
			t.Fatal(method)
		}
	}
	for _, id := range []string{"/rest/system/resource", "*A?x", "*1/2", ""} {
		if _, err := c.natRecord(ctx, cred, id); err == nil {
			t.Fatal(id)
		}
	}
}
