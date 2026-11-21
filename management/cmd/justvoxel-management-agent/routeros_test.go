package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/home-server-project/justvoxel/management/internal/networking"
)

const testRouterPassword = "connector-password-unique"

func testRouterConfiguration(endpoint string) routerOSConfiguration {
	return routerOSConfiguration{
		routerOSCredentials: routerOSCredentials{Endpoint: endpoint, Username: "justvoxel", Password: testRouterPassword},
		Target:              "192.0.2.9",
		Ingress:             routerOSSelector{Type: "list", Name: "uplinks"},
	}
}

func trustTestRouter(t *testing.T, router *httptest.Server) routerOSClient {
	t.Helper()
	client := newRouterOSClient()
	client.transport.TLSClientConfig.RootCAs = x509.NewCertPool()
	client.transport.TLSClientConfig.RootCAs.AddCert(router.Certificate())
	t.Cleanup(client.transport.CloseIdleConnections)
	return client
}

func TestRouterOSHTTPSBoundary(t *testing.T) {
	ctx := context.Background()
	var redirected atomic.Bool
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected.Store(true)
	}))
	defer redirect.Close()
	var mode atomic.Int32
	router := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "justvoxel" || password != testRouterPassword || r.Method != http.MethodGet ||
			r.URL.Path != "/rest/system/identity" || r.URL.RawQuery != "" {
			t.Error("unexpected RouterOS method, credentials or resource")
		}
		switch mode.Load() {
		case 1:
			http.Redirect(w, r, redirect.URL, http.StatusTemporaryRedirect)
		case 2:
			fmt.Fprint(w, "["+strings.Repeat(" ", routerOSResponseLimit)+"]")
		case 3:
			fmt.Fprint(w, `[{"name":"lab","number":1}]`)
		default:
			fmt.Fprint(w, `[{"name":"lab","number":"1","flag":"true"}]`)
		}
	}))
	defer router.Close()
	c := testRouterConfiguration(router.URL).routerOSCredentials
	production := newRouterOSClient()
	if production.transport.Proxy != nil || production.transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("RouterOS transport uses proxy or bypasses certificates")
	}
	if _, err := production.read(ctx, c, routerOSIdentity); err != errRouterOSUnavailable {
		t.Fatal("accepted an untrusted certificate")
	}
	client := trustTestRouter(t, router)
	for _, endpoint := range []string{"http://router.example", "https://name:secret@router.example", "https://router.example?", "https://router.example#", "https://router.example/rest/system/identity"} {
		invalid := c
		invalid.Endpoint = endpoint
		if _, err := client.read(ctx, invalid, routerOSIdentity); err != errRouterOSUnavailable {
			t.Errorf("accepted invalid endpoint %q", endpoint)
		}
	}
	if _, err := client.read(ctx, c, routerOSResource(100)); err != errRouterOSUnavailable {
		t.Fatal("accepted an unlisted REST resource")
	}
	rows, err := client.read(ctx, c, routerOSIdentity)
	if err != nil || len(rows) != 1 || rows[0]["name"] != "lab" || rows[0]["number"] != "1" || !routerOSTrue(rows[0]["flag"]) {
		t.Fatal("failed to read RouterOS string values")
	}
	for _, failure := range []int32{1, 2, 3} {
		mode.Store(failure)
		if _, err := client.read(ctx, c, routerOSIdentity); err != errRouterOSUnavailable {
			t.Error("accepted a redirect, oversized body or invalid response")
		}
	}
	if redirected.Load() {
		t.Fatal("followed a redirect")
	}
}

func TestRouterOSMigrationPersistenceAndRevocation(t *testing.T) {
	store, path := openTestWebUIStore(t)
	if _, err := store.db.Exec(`DROP TABLE routeros_state; DELETE FROM schema_migrations WHERE version=12`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.createWebUser("previous-user", webUserRoleOperator, "previous-password"); err != nil {
		t.Fatal(err)
	}
	if err := applyWebUIStoreMigrations(store.db); err != nil {
		t.Fatal(err)
	}
	if users, err := store.listWebUsers(); err != nil || len(users) != 1 {
		t.Fatal("additive migration changed existing WebUI users", err)
	}
	ctx := context.Background()
	c := testRouterConfiguration("https://router.example")
	actor := administratorSession(time.Now())
	if err := store.writeRouterOS(ctx, &c, actor, "routeros_save"); err != nil {
		t.Fatal(err)
	}
	reopened, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	got, err := reopened.readRouterOS(ctx)
	if err != nil || got == nil || *got != c {
		t.Fatal("stored connector details did not survive reopening")
	}
	events, err := reopened.listAuditEvents(20)
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := json.Marshal(events)
	if err != nil || !bytes.Contains(eventJSON, []byte("routeros_save")) || bytes.Contains(eventJSON, []byte(testRouterPassword)) {
		t.Fatal("missing sanitized connector audit event")
	}
	if _, err := reopened.db.Exec(`UPDATE routeros_state SET endpoint='http://router.example' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if config, err := reopened.readRouterOS(ctx); config != nil || err != errRouterOSState {
		t.Fatal("malformed connector state was accepted")
	}
	s := &server{store: reopened}
	if err := s.routerOSFactoryRevoke(); err != nil {
		t.Fatal(err)
	}
	if config, err := reopened.readRouterOS(ctx); config != nil || err != nil {
		t.Fatal("malformed saved credentials survived Factory Reset revocation")
	}
	for _, file := range []string{path, path + "-wal"} {
		data, err := os.ReadFile(file)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(testRouterPassword)) {
			t.Fatal("revoked credential remains in SQLite data")
		}
	}
}

func TestRouterOSFactoryResetRevokesBeforeCleanup(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			useEmptyFactoryResetRuntime(t)
			useTestMinecraftInstances(t)
			s := surfaceTestServer(t, roleAdministrator)
			operations := openTestOperationStore(t)
			attachTestOperationStore(t, s, operations)
			ctx := context.Background()
			c := testRouterConfiguration("https://router.example")
			if err := s.store.writeRouterOS(ctx, &c, administratorSession(time.Now()), "routeros_save"); err != nil {
				t.Fatal(err)
			}
			if failure {
				_, err := s.store.db.Exec(`CREATE TRIGGER fail_routeros_revocation BEFORE DELETE ON routeros_state BEGIN SELECT RAISE(ABORT,'private storage failure'); END`)
				if err != nil {
					t.Fatal(err)
				}
			}
			original := runAdminFactoryResetHelper
			t.Cleanup(func() { runAdminFactoryResetHelper = original })
			cleanupReached := false
			runAdminFactoryResetHelper = func(_ context.Context, args ...string) ([]byte, error) {
				if slices.Equal(args, []string{"plan"}) {
					return []byte(validFactoryResetPlanHelper), nil
				}
				cleanupReached = true
				if stored, err := s.store.readRouterOS(ctx); stored != nil || err != nil {
					t.Error("destructive cleanup reached before connector revocation")
				}
				return nil, fmt.Errorf("simulated cleanup failure")
			}
			fingerprint := exactFactoryResetFingerprint(t)
			operation, _, err := operations.beginFactoryReset(fingerprint)
			if err != nil {
				t.Fatal(err)
			}
			if err := executeFactoryReset(ctx, s, operation.OperationID, fingerprint); err == nil || cleanupReached == failure {
				t.Fatal("incorrect RouterOS revocation boundary")
			}
			if failure {
				current, err := operations.get(operation.OperationID)
				if err != nil || current.Stage != "routeros_revocation_failed" || current.State != operationNeedsAttention {
					t.Fatal("connector revocation failure did not preserve reset recovery", err)
				}
			}
		})
	}
}

func TestRouterOSLocalRoutes(t *testing.T) {
	useEmptyFactoryResetRuntime(t)
	useTestMinecraftInstances(t)
	var requests atomic.Int32
	router := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		username, password, ok := r.BasicAuth()
		if r.Method != http.MethodGet || r.URL.RawQuery != "" || !ok || username != "justvoxel" || password != testRouterPassword {
			t.Error("unexpected RouterOS request")
		}
		switch r.URL.Path {
		case "/rest/system/identity":
			fmt.Fprint(w, `[{"name":"router-lab","ignored":"`+testRouterPassword+`"}]`)
		case "/rest/system/resource":
			fmt.Fprint(w, `[{"version":"7.18.0","board-name":"test-board"}]`)
		case "/rest/interface":
			fmt.Fprint(w, `[{"name":"ethernet"},{"name":"vlan"},{"name":"`+testRouterPassword+`"}]`)
		case "/rest/interface/list":
			fmt.Fprint(w, `[{"name":"uplinks","include":"none"}]`)
		case "/rest/interface/list/member":
			fmt.Fprint(w, `[{"list":"uplinks","interface":"ethernet","disabled":"false"}]`)
		case "/rest/ip/firewall/nat":
			fmt.Fprint(w, `[{"chain":"dstnat","action":"dst-nat","comment":"private-router-rule"}]`)
		default:
			t.Error("unexpected RouterOS REST path")
			w.WriteHeader(404)
		}
	}))
	defer router.Close()
	oldClient, oldNetwork := liveRouterOS, openNetworkClient
	t.Cleanup(func() { liveRouterOS, openNetworkClient = oldClient, oldNetwork })
	liveRouterOS = trustTestRouter(t, router)
	openNetworkClient = func(context.Context) (networkClient, error) {
		return &fakeNetworkClient{snapshot: networking.Snapshot{Devices: []networking.Device{
			{IPv4: networking.IPConfig{Addresses: []networking.IPAddress{{Address: "192.0.2.9"}, {Address: "127.0.0.1"}, {Address: "2001:db8::1"}}}},
		}}}, nil
	}
	s := surfaceTestServer(t, roleAdministrator)
	now := time.Now()
	s.sessions["primary"] = administratorSession(now)
	s.sessions["secondary"] = session{Username: "admin-voxel", Role: roleAdministrator, Created: now, LastSeen: now}
	s.sessions["viewer"] = session{Username: "viewer", Role: roleViewer, Created: now, LastSeen: now}
	mux := http.NewServeMux()
	registerRouterOSRoutes(mux, s)
	call := func(method, suffix, actor string, body any, want int) *httptest.ResponseRecorder {
		t.Helper()
		var payload []byte
		if body != nil {
			payload, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, routerOSRoute+suffix, bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer "+actor)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != want || strings.Contains(rr.Body.String(), testRouterPassword) || strings.Contains(rr.Body.String(), "private-router-rule") {
			t.Fatalf("%s %s by %s: unexpected status or private response (%d)", method, suffix, actor, rr.Code)
		}
		return rr
	}
	c := testRouterConfiguration(router.URL)
	call(http.MethodGet, "", "viewer", nil, 403)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		suffix := ""
		if method == http.MethodPost {
			suffix = "/test"
		}
		call(method, suffix, "secondary", nil, 403)
	}
	call(http.MethodGet, "?port=22", "secondary", nil, 400)
	call(http.MethodGet, "/instances/invalid", "secondary", nil, 400)
	call(http.MethodGet, "", "secondary", nil, 200)
	status := call(http.MethodPost, "/test", "primary", c.routerOSCredentials, 200)
	for _, want := range []string{"router-lab", "7.18.0", "test-board", "uplinks", "ethernet", "vlan", "192.0.2.9"} {
		if !strings.Contains(status.Body.String(), want) {
			t.Errorf("test response omitted %s", want)
		}
	}
	if stored, err := s.store.readRouterOS(context.Background()); stored != nil || err != nil {
		t.Fatal("testing saved a candidate password")
	}
	before := requests.Load()
	unassigned := c
	unassigned.Target = "192.0.2.10"
	call(http.MethodPut, "", "primary", unassigned, 400)
	if requests.Load() != before {
		t.Fatal("probed RouterOS after receiving a non-local target address")
	}
	missing := c
	missing.Ingress.Name = "missing"
	call(http.MethodPut, "", "primary", missing, 400)
	call(http.MethodPut, "", "primary", c, 200)
	status = call(http.MethodGet, "", "secondary", nil, 200)
	if !strings.Contains(status.Body.String(), `"status":"connected"`) || strings.Contains(status.Body.String(), `"can_configure":true`) {
		t.Fatal("read-only administrator status is incorrect")
	}
	router.Close()
	out := routerOSStatus{Endpoints: []routerMinecraftEndpoint{{Port: 25570, LastPort: 25570, Protocols: []string{"tcp"}, State: routerForwardUnknown}}}
	out.inspect(context.Background(), c, true)
	if out.Status != "unavailable" || out.Endpoints[0].State != routerForwardUnknown {
		t.Fatal("unreachable router did not produce Unknown")
	}
	before = requests.Load()
	call(http.MethodDelete, "", "primary", nil, 200)
	if requests.Load() != before {
		t.Fatal("forgetting credentials contacted RouterOS")
	}
	if stored, err := s.store.readRouterOS(context.Background()); stored != nil || err != nil {
		t.Fatal("forgotten connector credentials remain in the store")
	}
}

func TestReadOnlyRouterOSNAT(t *testing.T) {
	d := routerOSDiscovery{
		interfaces: map[string]routerOSRow{"wan-port": {"name": "wan-port"}, "lan-port": {"name": "lan-port"}},
		lists: map[string]routerOSRow{
			"base": {"name": "base"}, "external": {"name": "external", "include": "base"},
			"internal": {"name": "internal"},
		},
		members: map[string]map[string]bool{"base": {"wan-port": true}, "internal": {"lan-port": true}},
	}
	const target = "192.0.2.9"
	base := func() routerOSRow {
		return routerOSRow{"chain": "dstnat", "action": "dst-nat", "protocol": "6", "dst-port": "123,25560-25580", "in-interface-list": "external", "to-addresses": target}
	}
	for _, selector := range []routerOSSelector{{Type: "interface", Name: "wan-port"}, {Type: "list", Name: "external"}} {
		ingress, ok := d.ingress(selector)
		if !ok || len(ingress) != 1 {
			t.Fatal("could not resolve selected ingress")
		}
		for _, tc := range []struct {
			key, value string
			want       string
		}{
			{"", "", routerForwardOpen},
			{"to-ports", "25570", routerForwardOpen},
			{"dst-port", "25560-25569,25571", routerForwardClosed},
			{"to-addresses", "192.0.2.22", routerForwardConflict},
			{"to-ports", "25571", routerForwardConflict},
			{"disabled", "true", routerForwardClosed},
			{"in-interface-list", "internal", routerForwardClosed},
			{"in-interface", "lan-port", routerForwardClosed},
			{"chain", "srcnat", routerForwardClosed},
			{"protocol", "udp", routerForwardClosed},
			{"dst-port", "25580-25560", routerForwardUnknown},
			{"dst-port", strings.Repeat("99,", 128) + "25570", routerForwardUnknown},
			{"src-address-list", "limited-sources", routerForwardUnknown},
		} {
			rule := base()
			if tc.key != "" {
				rule[tc.key] = tc.value
			}
			rules, valid := routerOSNATRules([]routerOSRow{rule})
			budget := 1000
			if got := routerOSPortState(rules, d, &ingress[0], target, "tcp", 25570, &budget); !valid || got != tc.want {
				t.Errorf("%s %s=%s: got %s want %s", selector.Type, tc.key, tc.value, got, tc.want)
			}
		}
		budget := 1000
		if got := routerOSPortState(nil, d, &ingress[0], target, "tcp", 25570, &budget); got != routerForwardClosed {
			t.Fatal("empty table is not Closed")
		}
		rule := base()
		rule["in-interface-list"], rule["in-interface"] = "", "wan-port"
		rules, _ := routerOSNATRules([]routerOSRow{rule})
		if got := routerOSPortState(rules, d, &ingress[0], target, "tcp", 25570, &budget); got != routerForwardOpen {
			t.Fatal("direct incoming interface did not match")
		}
	}
}

func TestRouterMinecraftEndpointProtocols(t *testing.T) {
	record := instanceRuntimeRecord{Runtime: minecraftInstanceRuntimeConfig{ID: "jv-a1b2c3", JavaPort: 25570, BedrockEnabled: true, BedrockPort: 19135}}
	network := minecraftNetworkConfig{Enabled: true, JavaPort: 25565, BedrockPort: 19132}
	standalone := routerMinecraftPorts(record, network, "Fabric", false)
	if len(standalone) != 2 || standalone[0].ID != record.Runtime.ID || standalone[0].Port != 25570 ||
		!slices.Equal(standalone[0].Protocols, []string{"tcp"}) || standalone[1].Port != 19135 ||
		!slices.Equal(standalone[1].Protocols, []string{"tcp", "udp"}) {
		t.Fatal("standalone inventory diverges from runtime publication")
	}
	record.Runtime.BedrockEnabled = false
	if ports := routerMinecraftPorts(record, network, "", false); len(ports) != 1 {
		t.Fatal("disabled cross-play appeared in inventory")
	}
	record.NetworkMember = true
	member := effectiveMinecraftConnection(record, network)
	shared := routerMinecraftPorts(instanceRuntimeRecord{NetworkMember: true}, network, "", true)
	if !member.networkMember || len(shared) != 2 || shared[0].ID != "" || !shared[0].Network ||
		shared[0].Port != member.javaPort || shared[1].Port != member.bedrockPort {
		t.Fatal("Velocity ports do not represent the shared connection")
	}
	bedrock := instanceRuntimeRecord{Runtime: minecraftInstanceRuntimeConfig{ID: "jv-b1b2c3", Software: "bedrock", BedrockTCPPort: 19200, BedrockUDPStart: 19201, BedrockUDPEnd: 19210}}
	native := routerMinecraftPorts(bedrock, network, "", false)
	if len(native) != 2 || native[0].Port != 19200 || !slices.Equal(native[0].Protocols, []string{"tcp"}) ||
		native[1].Port != 19201 || native[1].LastPort != 19210 || !slices.Equal(native[1].Protocols, []string{"udp"}) {
		t.Fatal("native Bedrock signaling or gameplay ports are incorrect")
	}
}

func TestRouterOSInventoryUsesOneVelocityEntrance(t *testing.T) {
	f := newInstanceActivationFixture(t)
	oldRuntime, oldNames := instanceRuntimes, minecraftInstances
	t.Cleanup(func() { instanceRuntimes, minecraftInstances = oldRuntime, oldNames })
	instanceRuntimes, minecraftInstances = f.backend, f.backend.identities()
	f.backend.networkRoot = filepath.Join(t.TempDir(), "network")
	dir, err := f.backend.privateDirectory(f.backend.networkRoot, true)
	if err != nil {
		t.Fatal(err)
	}
	dir.Close()
	n := networkDefaultState()
	n.Config.Enabled = true
	n.ForwardingSecret = strings.Repeat("a", 32)
	n.FloodgateKey = make([]byte, 16)
	for _, config := range f.configs {
		record, err := f.backend.readRecord(config.ID)
		if err != nil {
			t.Fatal(err)
		}
		n.Config.Members = append(n.Config.Members, config.ID)
		record.NetworkMember = true
		dir, _, _ := f.backend.paths(config.ID)
		if err := f.backend.writeJSON(filepath.Join(dir, "config.json"), record, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.backend.writeJSON(filepath.Join(f.backend.networkRoot, "state.json"), n, false); err != nil {
		t.Fatal(err)
	}
	var all routerOSStatus
	if err := all.inventory(""); err != nil || len(all.Endpoints) != 2 {
		t.Fatal("Velocity inventory should contain Java and Bedrock exactly once", err)
	}
	for _, endpoint := range all.Endpoints {
		if !endpoint.Network || endpoint.ID != "" {
			t.Fatal("member backend appeared as a public endpoint")
		}
	}
	for _, config := range f.configs {
		var selected routerOSStatus
		if err := selected.inventory(config.ID); err != nil || !selected.NetworkMember ||
			selected.ServerName == "" || len(selected.Endpoints) != 2 || selected.Endpoints[0].Port != n.Config.JavaPort {
			t.Fatal("member did not resolve to the shared public endpoint", err)
		}
	}
}
