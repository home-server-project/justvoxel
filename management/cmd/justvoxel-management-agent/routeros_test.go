package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
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
			fmt.Fprint(w, "{"+strings.Repeat(" ", routerOSResponseLimit)+"}")
		case 3:
			fmt.Fprint(w, `{"name":"lab","number":1}`)
		default:
			fmt.Fprint(w, `{"name":"lab","number":"1","flag":"true"}`)
		}
	}))
	defer router.Close()
	c := testRouterConfiguration(router.URL).routerOSCredentials
	production := newRouterOSClient()
	if production.transport.Proxy != nil || production.transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("RouterOS transport uses proxy or bypasses certificates")
	}
	if _, err := production.readSingleton(ctx, c, routerOSIdentity); err != errRouterOSTLS {
		t.Fatal("accepted an untrusted certificate")
	}
	client := trustTestRouter(t, router)
	for _, endpoint := range []string{"http://router.example", "https://name:secret@router.example", "https://router.example?", "https://router.example#", "https://router.example/rest/system/identity"} {
		invalid := c
		invalid.Endpoint = endpoint
		if _, err := client.readSingleton(ctx, invalid, routerOSIdentity); err != errRouterOSUnavailable {
			t.Errorf("accepted invalid endpoint %q", endpoint)
		}
	}
	if _, err := client.readList(ctx, c, routerOSResource(100)); err != errRouterOSUnavailable {
		t.Fatal("accepted an unlisted REST resource")
	}
	row, err := client.readSingleton(ctx, c, routerOSIdentity)
	if err != nil || row["name"] != "lab" || row["number"] != "1" || !routerOSTrue(row["flag"]) {
		t.Fatal("failed to read RouterOS string values")
	}
	for index, want := range []routerOSError{errRouterOSUnavailable, errRouterOSResponse, errRouterOSResponse} {
		mode.Store(int32(index + 1))
		if _, err := client.readSingleton(ctx, c, routerOSIdentity); err != want {
			t.Error("accepted a redirect, oversized body or invalid response")
		}
	}
	if redirected.Load() {
		t.Fatal("followed a redirect")
	}
}

func TestRouterOSResourceShapes(t *testing.T) {
	records := [...]string{
		`{"name":"router"}`,
		`{"version":"7.23.8","board-name":"test-board"}`,
		`{"name":"ether1"}`,
		`{"name":"uplinks"}`,
		`{"list":"uplinks","interface":"ether1"}`,
		`{"chain":"dstnat","action":"dst-nat"}`,
	}
	for resource, path := range routerOSPaths {
		t.Run(path, func(t *testing.T) {
			var array atomic.Bool
			router := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != path {
					t.Errorf("unexpected resource %q", r.URL.Path)
				}
				if array.Load() {
					fmt.Fprint(w, "["+records[resource]+"]")
				} else {
					fmt.Fprint(w, records[resource])
				}
			}))
			defer router.Close()
			client := trustTestRouter(t, router)
			credentials := testRouterConfiguration(router.URL).routerOSCredentials
			for _, isArray := range []bool{false, true} {
				array.Store(isArray)
				var err error
				singleton := resource <= int(routerOSSystem)
				if singleton {
					_, err = client.readSingleton(context.Background(), credentials, routerOSResource(resource))
				} else {
					_, err = client.readList(context.Background(), credentials, routerOSResource(resource))
				}
				var want error
				if singleton == isArray {
					want = errRouterOSResponse
				}
				if err != want {
					t.Errorf("array=%v: got %v, want %v", isArray, err, want)
				}
			}
		})
	}
}

func testRouterCertificates(t *testing.T, name string, usage x509.KeyUsage) (string, tls.Certificate, string) {
	t.Helper()
	key := func() *ecdsa.PrivateKey {
		t.Helper()
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	pair := func(template, parent *x509.Certificate, subject, signer *ecdsa.PrivateKey) []byte {
		t.Helper()
		data, err := x509.CreateCertificate(rand.Reader, template, parent, &subject.PublicKey, signer)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	now := time.Now()
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour),
		BasicConstraintsValid: true, IsCA: true, KeyUsage: usage,
	}
	caKey := key()
	caDER := pair(ca, ca, caKey, caKey)
	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "router"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(12 * time.Hour),
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	leafKey := key()
	leafDER := pair(leaf, ca, leafKey, caKey)
	caPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))
	privateDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateDER}))
	return caPEM, tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}, privatePEM
}

func TestRouterOSCAKeepsEndpointVerification(t *testing.T) {
	caPEM, leaf, _ := testRouterCertificates(t, "router-ca", x509.KeyUsageCertSign)
	router := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("sent an HTTP request despite a mismatched certificate SAN")
		w.WriteHeader(http.StatusForbidden)
	}))
	router.TLS = &tls.Config{Certificates: []tls.Certificate{leaf}}
	router.StartTLS()
	defer router.Close()

	ca, err := parseRouterOSCA(caPEM)
	if err != nil {
		t.Fatal(err)
	}
	base := newRouterOSClient()
	base.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, network, router.Listener.Addr().String())
	}
	client, closeIdle, err := base.withCA(ca)
	if err != nil {
		t.Fatal(err)
	}
	defer closeIdle()
	credentials := testRouterConfiguration("https://router.example").routerOSCredentials
	if _, err := client.readSingleton(context.Background(), credentials, routerOSIdentity); err != errRouterOSTLS {
		t.Fatal("expected TLS diagnostic for an endpoint missing from the certificate SAN", err)
	}
}

func TestRouterOSCAValidation(t *testing.T) {
	ca, leaf, key := testRouterCertificates(t, "private-ca", x509.KeyUsageCertSign|x509.KeyUsageCRLSign)
	wrongUsage, _, _ := testRouterCertificates(t, "wrong-usage", x509.KeyUsageCRLSign)
	if _, err := parseRouterOSCA(strings.ReplaceAll(ca, "\n", "\r\n")); err != nil {
		t.Fatal("rejected a public CA exported as PEM", err)
	}
	leafPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leaf.Certificate[0]}))
	for name, input := range map[string]string{
		"malformed": "-----BEGIN CERTIFICATE-----\nwhat?\n-----END CERTIFICATE-----",
		"key":       key, "key then cert": key + ca, "key after cert": ca + key,
		"non-CA": leafPEM, "key usage": wrongUsage, "multiple blocks": ca + ca,
		"over limit": ca + strings.Repeat(" ", routerOSCALimit),
		"garbage":    ca + "trailing data",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseRouterOSCA(input); err != errRouterOSInvalidCA {
				t.Fatal("accepted invalid CA input")
			}
		})
	}
}

func TestRouterOSMigrationPersistenceAndRevocation(t *testing.T) {
	store, path := openTestWebUIStoreAtVersion(t, 12)
	if _, err := store.db.Exec(`INSERT INTO routeros_state VALUES
		(1, 1, 'https://router.example', 'justvoxel', ?, '192.0.2.9', 'list', 'uplinks')`, testRouterPassword); err != nil {
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
	if got, err := store.readRouterOS(ctx); err != nil || got == nil || *got != c {
		t.Fatal("migration changed existing connector data", err)
	}
	c.CAPEM, _, _ = testRouterCertificates(t, "stored-ca", x509.KeyUsageCertSign)
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
		if bytes.Contains(data, []byte(testRouterPassword)) || bytes.Contains(data, []byte("-----BEGIN CERTIFICATE-----")) {
			t.Fatal("revoked credential or CA remains in SQLite data")
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
			c.CAPEM, _, _ = testRouterCertificates(t, "reset-ca", x509.KeyUsageCertSign)
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
	var statusOverride atomic.Int32
	var wrongShape atomic.Bool
	authToken := base64.StdEncoding.EncodeToString([]byte("justvoxel:" + testRouterPassword))
	ca, leaf, _ := testRouterCertificates(t, "router-ca-original", x509.KeyUsageCertSign)
	replacementCA, replacementLeaf, _ := testRouterCertificates(t, "router-ca-replacement", x509.KeyUsageCertSign)
	var activeLeaf atomic.Pointer[tls.Certificate]
	activeLeaf.Store(&leaf)
	router := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		username, password, ok := r.BasicAuth()
		if r.Method != http.MethodGet || r.URL.RawQuery != "" || !ok || username != "justvoxel" || password != testRouterPassword {
			t.Error("unexpected RouterOS request")
		}
		if status := statusOverride.Load(); status != 0 {
			w.WriteHeader(int(status))
			fmt.Fprint(w, "sensitive-remote-error "+testRouterPassword)
			return
		}
		switch r.URL.Path {
		case "/rest/system/identity":
			if wrongShape.Load() {
				fmt.Fprint(w, `[{"name":"sensitive-remote-error"}]`)
				return
			}
			fmt.Fprint(w, `{"name":"router-lab","ignored":"`+testRouterPassword+`"}`)
		case "/rest/system/resource":
			fmt.Fprint(w, `{"version":"7.23.8","board-name":"test-board"}`)
		case "/rest/interface":
			fmt.Fprint(w, `[{"name":"ethernet"},{"name":"vlan"},{"name":"`+testRouterPassword+`"},{"name":"Basic `+authToken+`"}]`)
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
	router.TLS = &tls.Config{GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
		return &tls.Config{Certificates: []tls.Certificate{*activeLeaf.Load()}}, nil
	}}
	router.StartTLS()
	defer router.Close()
	oldClient, oldNetwork := liveRouterOS, openNetworkClient
	t.Cleanup(func() { liveRouterOS, openNetworkClient = oldClient, oldNetwork })
	liveRouterOS = newRouterOSClient()
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
		if rr.Code != want || strings.Contains(rr.Body.String(), testRouterPassword) || strings.Contains(rr.Body.String(), "private-router-rule") ||
			strings.Contains(rr.Body.String(), "sensitive-remote-error") || strings.Contains(rr.Body.String(), authToken) ||
			strings.Contains(rr.Body.String(), "BEGIN CERTIFICATE") || strings.Contains(rr.Body.String(), `"ca_pem"`) ||
			strings.Contains(rr.Body.String(), `"password"`) {
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
	call(http.MethodPost, "/forwarding", "secondary", nil, 403)
	call(http.MethodPost, "/forwarding", "primary", map[string]string{"scope": "jv-family1", "action": "enable", "port": "22"}, 400)
	call(http.MethodPost, "/forwarding", "primary", map[string]string{"scope": "jv-family1", "action": "post"}, 400)
	call(http.MethodPut, "/ca", "secondary", nil, 403)
	call(http.MethodDelete, "/ca", "secondary", nil, 403)
	call(http.MethodGet, "?port=22", "secondary", nil, 400)
	call(http.MethodGet, "/instances/invalid", "secondary", nil, 400)
	call(http.MethodGet, "", "secondary", nil, 200)
	expectCode := func(rr *httptest.ResponseRecorder, code routerOSError) {
		t.Helper()
		if strings.TrimSpace(rr.Body.String()) != fmt.Sprintf(`{"error":%q}`, code) {
			t.Fatalf("expected safe error %q, got %s", code, rr.Body.String())
		}
	}
	expectCode(call(http.MethodPost, "/test", "primary", c.routerOSCredentials, 502), errRouterOSTLS)
	c.CAPEM = "invalid"
	expectCode(call(http.MethodPost, "/test", "primary", c.routerOSCredentials, 400), errRouterOSInvalidCA)
	c.CAPEM = ca
	status := call(http.MethodPost, "/test", "primary", c.routerOSCredentials, 200)
	for _, want := range []string{"router-lab", "7.23.8", "test-board", "uplinks",
		"trusted_ca", "router-ca-original", "fingerprint", "expires_at"} {
		if !strings.Contains(status.Body.String(), want) {
			t.Errorf("test response omitted %s", want)
		}
	}
	if strings.Contains(status.Body.String(), "192.0.2.9") || strings.Contains(status.Body.String(), "target_choices") {
		t.Fatal("test exposed an address from an unclassified device")
	}
	if strings.Contains(status.Body.String(), "ethernet") || strings.Contains(status.Body.String(), "vlan") {
		t.Fatal("test exposed raw RouterOS interface inventory")
	}
	call(http.MethodPost, "/ca", "secondary", map[string]string{"ca_pem": ca}, 403)
	call(http.MethodPost, "/ca", "primary", map[string]string{"ca_pem": "invalid"}, 400)
	preview := call(http.MethodPost, "/ca", "primary", map[string]string{"ca_pem": ca}, 200)
	if !strings.Contains(preview.Body.String(), "router-ca-original") {
		t.Fatal("missing CA preview")
	}
	if stored, err := s.store.readRouterOS(context.Background()); stored != nil || err != nil {
		t.Fatal("testing persisted candidate credentials or a CA")
	}
	withoutCA := c.routerOSCredentials
	withoutCA.CAPEM = ""
	expectCode(call(http.MethodPost, "/test", "primary", withoutCA, 502), errRouterOSTLS)
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
	for _, remoteStatus := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		statusOverride.Store(int32(remoteStatus))
		expectCode(call(http.MethodPost, "/test", "primary", c.routerOSCredentials, 502), errRouterOSAuth)
	}
	statusOverride.Store(http.StatusServiceUnavailable)
	expectCode(call(http.MethodPost, "/test", "primary", c.routerOSCredentials, 502), errRouterOSUnavailable)
	statusOverride.Store(0)
	wrongShape.Store(true)
	expectCode(call(http.MethodPost, "/test", "primary", c.routerOSCredentials, 502), errRouterOSResponse)
	wrongShape.Store(false)

	assertStoredCA := func(want string) {
		t.Helper()
		stored, err := s.store.readRouterOS(context.Background())
		expected := c
		expected.CAPEM = want
		if err != nil || stored == nil || *stored != expected {
			t.Fatal("CA change did not preserve the expected connector record", err)
		}
	}
	replacement := map[string]string{"ca_pem": replacementCA}
	expectCode(call(http.MethodPut, "/ca", "primary", replacement, 502), errRouterOSTLS)
	assertStoredCA(ca)
	expectCode(call(http.MethodDelete, "/ca", "primary", nil, 502), errRouterOSTLS)
	assertStoredCA(ca)

	activeLeaf.Store(&replacementLeaf)
	call(http.MethodPut, "/ca", "primary", replacement, 200)
	assertStoredCA(replacementCA)
	status = call(http.MethodGet, "", "primary", nil, 200)
	if !strings.Contains(status.Body.String(), "router-ca-replacement") || strings.Contains(status.Body.String(), "router-ca-original") {
		t.Fatal("status did not expose only replacement CA metadata")
	}
	cert, err := parseRouterOSCA(replacementCA)
	if err != nil {
		t.Fatal(err)
	}
	// Use an explicit test root pool in place of the appliance system roots.
	liveRouterOS.transport.TLSClientConfig.RootCAs = x509.NewCertPool()
	liveRouterOS.transport.TLSClientConfig.RootCAs.AddCert(cert)
	call(http.MethodDelete, "/ca", "primary", nil, 200)
	assertStoredCA("")
	if status := call(http.MethodGet, "", "primary", nil, 200); strings.Contains(status.Body.String(), "trusted_ca") {
		t.Fatal("removed CA remains in status")
	}
	call(http.MethodPut, "/ca", "primary", replacement, 200)
	assertStoredCA(replacementCA)
	router.Close()
	expectCode(call(http.MethodPost, "/test", "primary", c.routerOSCredentials, 502), errRouterOSNetwork)
	out := routerOSStatus{Endpoints: []routerMinecraftEndpoint{{Port: 25570, LastPort: 25570, Protocols: []string{"tcp"}, State: routerForwardUnknown}}}
	out.inspect(context.Background(), c, true)
	if out.Status != "unavailable" || out.Diagnostic != string(errRouterOSNetwork) || out.Endpoints[0].State != routerForwardUnknown {
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
		if err := selected.inventory(config.ID); err != nil || !selected.NetworkMember || selected.ForwardingScope != routerNetworkOwner ||
			selected.ServerName == "" || len(selected.Endpoints) != 2 || selected.Endpoints[0].Port != n.Config.JavaPort {
			t.Fatal("member did not resolve to the shared public endpoint", err)
		}
	}
}

func routerTestDevice(address string, kind networking.DeviceKind) networking.Device {
	return networking.Device{Kind: kind, Managed: true, State: 100, ActiveConnection: &networking.ConnectionProfile{},
		IPv4: networking.IPConfig{Addresses: []networking.IPAddress{{Address: address, Prefix: 24}}}}
}

func routerTestSnapshot() networking.Snapshot {
	return networking.Snapshot{Devices: []networking.Device{
		routerTestDevice("10.88.0.1", networking.DeviceKindOther),
		routerTestDevice("100.102.16.12", networking.DeviceKindOther),
		routerTestDevice("100.122.64.28", networking.DeviceKindOther),
		routerTestDevice("192.168.0.28", networking.DeviceKindEthernet),
		routerTestDevice("192.168.125.72", networking.DeviceKindEthernet),
	}}
}

func TestRouterMutationTargetPreflight(t *testing.T) {
	const safeTarget = "192.168.0.28"
	var writes atomic.Int32
	var saved atomic.Pointer[routerOSRow]
	router := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes.Add(1)
			var row routerOSRow
			if r.Method != http.MethodPut || r.URL.Path != routerNATPath ||
				json.NewDecoder(r.Body).Decode(&row) != nil || row["to-addresses"] != safeTarget {
				t.Error("unexpected forwarding write", r.Method, r.URL.Path, row)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			row[".id"] = "*1"
			saved.Store(&row)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(row)
			return
		}
		switch r.URL.Path {
		case "/rest/system/identity":
			fmt.Fprint(w, `{"name":"router"}`)
		case "/rest/system/resource":
			fmt.Fprint(w, `{"version":"7"}`)
		case "/rest/interface":
			fmt.Fprint(w, `[{"name":"ether1"}]`)
		case "/rest/interface/list":
			fmt.Fprint(w, `[{"name":"uplinks"}]`)
		case "/rest/interface/list/member":
			fmt.Fprint(w, `[{"list":"uplinks","interface":"ether1"}]`)
		case routerNATPath:
			rows := []routerOSRow{}
			if row := saved.Load(); row != nil {
				rows = append(rows, *row)
			}
			json.NewEncoder(w).Encode(rows)
		default:
			t.Error("unexpected read", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer router.Close()
	oldClient, oldNetwork := liveRouterOS, openNetworkClient
	t.Cleanup(func() { liveRouterOS, openNetworkClient = oldClient, oldNetwork })
	liveRouterOS = trustTestRouter(t, router)
	liveRouterOS.transport.TLSClientConfig.ServerName = "127.0.0.1"
	liveRouterOS.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, router.Listener.Addr().String())
	}
	openNetworkClient = func(context.Context) (networkClient, error) {
		return &fakeNetworkClient{snapshot: routerTestSnapshot()}, nil
	}
	s := surfaceTestServer(t, roleAdministrator)
	f := newInstanceActivationFixture(t)
	oldRuntime, oldNames := instanceRuntimes, minecraftInstances
	t.Cleanup(func() { instanceRuntimes, minecraftInstances = oldRuntime, oldNames })
	f.backend.networkRoot = filepath.Join(t.TempDir(), "network")
	instanceRuntimes, minecraftInstances = f.backend, f.backend.identities()

	for _, target := range []string{"10.88.0.1", "100.102.16.12", "100.122.64.28", "192.168.125.72", safeTarget} {
		t.Run(target, func(t *testing.T) {
			config := testRouterConfiguration("https://192.168.0.1")
			config.Target = target
			ctx := context.Background()
			if err := s.store.writeRouterOS(ctx, &config, administratorSession(time.Now()), "routeros_save"); err != nil {
				t.Fatal(err)
			}
			saved.Store(nil)
			writes.Store(0)
			out, code := s.mutateForwarding(ctx, f.configs[0].ID, "enable")
			if target == safeTarget {
				if code != "" || writes.Load() != 1 || out.ForwardingState != "open" {
					t.Fatalf("safe target: code %q, writes %d, state %q", code, writes.Load(), out.ForwardingState)
				}
			} else if code != "target_unavailable" || writes.Load() != 0 {
				t.Fatalf("unsafe target: code %q, writes %d", code, writes.Load())
			}
		})
	}
}

func TestRouterDestinationRecommendation(t *testing.T) {
	snapshot := routerTestSnapshot()
	check := func(endpoint, recommendation string, choices ...string) {
		t.Helper()
		var out routerOSStatus
		out.forwardingTargets(endpoint, snapshot, func(s string) string { return s })
		if out.RecommendedTarget != recommendation || !slices.Equal(out.TargetChoices, choices) {
			t.Errorf("%s: recommendation %q, choices %v; want %q, %v", endpoint, out.RecommendedTarget, out.TargetChoices, recommendation, choices)
		}
	}
	const endpoint = "https://192.168.0.1"
	check(endpoint, "192.168.0.28")
	unmanaged := routerTestDevice("192.168.0.3", networking.DeviceKindEthernet)
	unmanaged.Managed = false
	inactive := routerTestDevice("192.168.0.4", networking.DeviceKindEthernet)
	inactive.State = 30
	disconnected := routerTestDevice("192.168.0.5", networking.DeviceKindWiFi)
	disconnected.ActiveConnection = nil
	wifi := routerTestDevice("192.168.0.29", networking.DeviceKindWiFi)
	snapshot.Devices = append(snapshot.Devices, unmanaged, inactive, disconnected, wifi, wifi,
		routerTestDevice("169.254.1.2", networking.DeviceKindEthernet), routerTestDevice("2001:db8::1", networking.DeviceKindWiFi))
	for _, host := range []string{"https://router.example", "https://172.16.0.1", "https://10.88.0.2"} {
		check(host, "", "192.168.0.28", "192.168.0.29", "192.168.125.72")
	}
	snapshot.Devices = append(snapshot.Devices, routerTestDevice("192.168.0.1", networking.DeviceKindEthernet))
	check(endpoint, "", "192.168.0.28", "192.168.0.29")
}

func TestRouterIngressRecommendation(t *testing.T) {
	d := routerOSDiscovery{lists: map[string]routerOSRow{}, interfaces: map[string]routerOSRow{"pppoe-wan": {}, "bridge": {}}}
	for _, name := range []string{"LAN", "WaN", "all", "dynamic", "none", "static", "uplinks"} {
		d.lists[name] = routerOSRow{}
	}
	endpoints := []routerMinecraftEndpoint{{Protocols: []string{"tcp"}, Port: 25565, LastPort: 25565}}
	snapshot := routerTestSnapshot()
	test := func(rows []routerOSRow, want string) {
		t.Helper()
		var out routerOSStatus
		publicText := func(s string) string { return s }
		targets := out.forwardingTargets("https://192.168.0.1", snapshot, publicText)
		out.forwardingIngress(d, rows, endpoints, targets, publicText)
		if want == "" && out.RecommendedIngress != nil || want != "" && (out.RecommendedIngress == nil || *out.RecommendedIngress != (routerOSSelector{Type: "list", Name: want})) {
			t.Errorf("ingress = %v, want %q", out.RecommendedIngress, want)
		}
		var choices []routerOSSelector
		if want == "" {
			choices = []routerOSSelector{{Type: "list", Name: "uplinks"}}
		}
		if !slices.Equal(out.IngressChoices, choices) {
			t.Errorf("ingress choices = %v, want %v", out.IngressChoices, choices)
		}
	}
	test(nil, "WaN")
	rule := routerOSRow{"chain": "dstnat", "action": "dst-nat", "protocol": "tcp", "dst-port": "25565", "to-addresses": "192.168.0.28", "in-interface-list": "uplinks"}
	test([]routerOSRow{rule}, "uplinks")
	rule["disabled"] = "true"
	test([]routerOSRow{rule}, "WaN")
	delete(d.lists, "WaN")
	test(nil, "")
	delete(rule, "disabled")
	conflict := routerOSRow{"chain": "dstnat", "action": "dst-nat", "protocol": "tcp", "dst-port": "25565", "to-addresses": "192.168.0.28", "in-interface": "pppoe-wan"}
	test([]routerOSRow{rule, conflict}, "")
	snapshot.Devices = append(snapshot.Devices, routerTestDevice("192.168.0.29", networking.DeviceKindWiFi))
	rule["to-addresses"] = "192.168.0.29"
	test([]routerOSRow{rule}, "uplinks")
	for _, address := range []string{"10.88.0.1", "100.102.16.12", "100.122.64.28", "192.168.125.72"} {
		rule["to-addresses"] = address
		test([]routerOSRow{rule}, "")
	}
}
