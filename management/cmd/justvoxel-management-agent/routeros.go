package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"os"
	"sort"
	"strings"
	"time"
)

const routerOSRoute = "/v1/admin/network/router/mikrotik"

var errRouterOSState = errors.New("MikroTik connector state unavailable")

func decodeRouterOSRequest(body io.Reader, out any) error {
	data, err := io.ReadAll(io.LimitReader(body, routerOSRequestLimit+1))
	if err != nil {
		return err
	}
	if len(data) > routerOSRequestLimit {
		return errors.New("RouterOS request too large")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return errors.New("invalid RouterOS request")
	}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return err
		}
		name, ok := key.(string)
		if !ok || seen[strings.ToLower(name)] {
			return errors.New("ambiguous RouterOS request")
		}
		seen[strings.ToLower(name)] = true
		var value json.RawMessage
		if err = d.Decode(&value); err != nil {
			return err
		}
	}
	if _, err = d.Token(); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("invalid RouterOS request")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(out)
}

type routerOSConfiguration struct {
	routerOSCredentials
	Target  string           `json:"target"`
	Ingress routerOSSelector `json:"ingress"`
}

func (c *routerOSConfiguration) valid() bool {
	return c.normalize() && routerTargetIPv4(c.Target) && c.Ingress.valid()
}

func routerTargetIPv4(value string) bool {
	ip, err := netip.ParseAddr(value)
	return err == nil && ip.Is4() && ip.IsGlobalUnicast()
}

func (s *webUIStore) readRouterOS(ctx context.Context) (*routerOSConfiguration, error) {
	if s == nil || s.db == nil {
		return nil, errRouterOSState
	}
	var c routerOSConfiguration
	var version int
	err := s.db.QueryRowContext(ctx, `SELECT schema_version, endpoint, username, password, target, ingress_type, ingress_name, ca_pem
		FROM routeros_state WHERE id=1`).Scan(&version, &c.Endpoint, &c.Username, &c.Password, &c.Target, &c.Ingress.Type, &c.Ingress.Name, &c.CAPEM)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	original := c
	if err != nil || version != 1 || !c.valid() || c != original {
		return nil, errRouterOSState
	}
	return &c, nil
}

func (s *webUIStore) readRouterOSCA(ctx context.Context) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, errRouterOSState
	}
	var pem string
	err := s.db.QueryRowContext(ctx, `SELECT ca_pem FROM routeros_state WHERE id=1`).Scan(&pem)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return pem, err == nil, err
}

func (s *webUIStore) writeRouterOS(ctx context.Context, c *routerOSConfiguration, actor session, action string) error {
	if s == nil || s.db == nil {
		return errRouterOSState
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return errRouterOSState
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA secure_delete = ON`); err != nil {
		return errRouterOSState
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return errRouterOSState
	}
	defer tx.Rollback()
	if c == nil {
		// Revocation must work even when the stored connector record is malformed.
		_, err = tx.ExecContext(ctx, `DELETE FROM routeros_state WHERE id=1`)
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO routeros_state
			(id, schema_version, endpoint, username, password, target, ingress_type, ingress_name, ca_pem)
			VALUES (1, 1, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET schema_version=1,
			endpoint=excluded.endpoint, username=excluded.username, password=excluded.password,
			target=excluded.target, ingress_type=excluded.ingress_type, ingress_name=excluded.ingress_name, ca_pem=excluded.ca_pem`,
			c.Endpoint, c.Username, c.Password, c.Target, c.Ingress.Type, c.Ingress.Name, c.CAPEM)
	}
	if err != nil || insertAuditEventTx(tx, s.now().UTC(), actor, action, "mikrotik", true, "") != nil || tx.Commit() != nil {
		return errRouterOSState
	}
	if c == nil {
		// A successful revocation removes prior frames containing the secret.
		var busy, frames, checkpointed int
		if err := conn.QueryRowContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&busy, &frames, &checkpointed); err != nil || busy != 0 {
			return errRouterOSState
		}
	}
	return nil
}

func (s *server) routerOSFactoryRevoke() error {
	s.routerOSMu.Lock()
	defer s.routerOSMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	return s.store.writeRouterOS(ctx, nil, session{Username: systemAdminUsername, Role: roleAdministrator}, "routeros_factory_revocation")
}

type routerMinecraftEndpoint struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name,omitempty"`
	Network   bool     `json:"network,omitempty"`
	Edition   string   `json:"edition"`
	Protocols []string `json:"protocols"`
	Port      int      `json:"port"`
	LastPort  int      `json:"last_port"`
	State     string   `json:"state"`
}

type routerOSStatus struct {
	Configured     bool                      `json:"configured"`
	CanConfigure   bool                      `json:"can_configure"`
	Status         string                    `json:"status"`
	Diagnostic     string                    `json:"diagnostic,omitempty"`
	TrustedCA      *routerOSCAMetadata       `json:"trusted_ca,omitempty"`
	Identity       string                    `json:"identity,omitempty"`
	Version        string                    `json:"version,omitempty"`
	Board          string                    `json:"board,omitempty"`
	Endpoint       string                    `json:"endpoint,omitempty"`
	Username       string                    `json:"username,omitempty"`
	Target         string                    `json:"target,omitempty"`
	Ingress        *routerOSSelector         `json:"ingress,omitempty"`
	IngressChoices []routerOSSelector        `json:"ingress_choices,omitempty"`
	TargetChoices  []string                  `json:"target_choices,omitempty"`
	Endpoints      []routerMinecraftEndpoint `json:"endpoints,omitempty"`
	InventoryError bool                      `json:"inventory_error,omitempty"`
	ServerName     string                    `json:"server_name,omitempty"`
	NetworkMember  bool                      `json:"network_member,omitempty"`
}

func (c routerOSCredentials) publicTextFilter() func(string) string {
	// A remote service can reflect credentials or the Basic Auth token in any field.
	token := base64.RawStdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
	return func(value string) string {
		if len(value) > 300 || strings.Contains(value, c.Password) || strings.Contains(value, token) {
			return ""
		}
		return value
	}
}

func (out *routerOSStatus) discovery(d routerOSDiscovery, publicText func(string) string) {
	out.Identity = publicText(d.name)
	out.Version = publicText(d.version)
	out.Board = publicText(d.board)
}

func routerTargetAddresses(ctx context.Context) ([]string, error) {
	client, err := openNetworkClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	snapshot, err := client.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	unique := map[string]bool{}
	var addresses []string
	for _, device := range snapshot.Devices {
		for _, address := range device.IPv4.Addresses {
			if routerTargetIPv4(address.Address) && !unique[address.Address] {
				addresses = append(addresses, address.Address)
				unique[address.Address] = true
			}
		}
	}
	sort.Strings(addresses)
	return addresses, nil
}

func routerMinecraftPorts(record instanceRuntimeRecord, network minecraftNetworkConfig, name string, publicNetwork bool) []routerMinecraftEndpoint {
	connection := effectiveMinecraftConnection(record, network)
	base := routerMinecraftEndpoint{ID: record.Runtime.ID, Name: name, Network: publicNetwork, State: routerForwardUnknown}
	publication := record.Runtime
	if connection.networkMember {
		// The shared entrance publishes Java and cross-play as Velocity's runtime does.
		publication = minecraftInstanceRuntimeConfig{JavaPort: connection.javaPort, BedrockPort: connection.bedrockPort, BedrockEnabled: true}
	}
	var endpoints []routerMinecraftEndpoint
	add := func(edition string, port, last int) {
		if port <= 0 || last < port || last > 65535 {
			return
		}
		endpoint := base
		endpoint.Edition, endpoint.Port, endpoint.LastPort = edition, port, last
		for _, protocol := range []string{"tcp", "udp"} {
			if instanceRuleForRuntime(instanceFirewallRule{Protocol: protocol, Port: port}, publication) &&
				instanceRuleForRuntime(instanceFirewallRule{Protocol: protocol, Port: last}, publication) {
				endpoint.Protocols = append(endpoint.Protocols, protocol)
			}
		}
		if len(endpoint.Protocols) != 0 {
			endpoints = append(endpoints, endpoint)
		}
	}
	if connection.javaPort != 0 {
		add("java", connection.javaPort, connection.javaPort)
	}
	if connection.bedrockEnabled {
		add("bedrock", connection.bedrockPort, connection.bedrockPort)
	}
	if connection.bedrockTCP != 0 {
		add("bedrock", connection.bedrockTCP, connection.bedrockTCP)
		add("bedrock", connection.udpStart, connection.udpEnd)
	}
	return endpoints
}

func (out *routerOSStatus) inventory(id string) error {
	n, err := networkRead(instanceRuntimes)
	if err != nil || networkPending(instanceRuntimes) != nil {
		return errInstanceRecovery
	}
	registered, err := minecraftInstances.listInstances("")
	if err != nil {
		return err
	}
	names := make(map[string]string, len(registered))
	for _, instance := range registered {
		names[instance.ID] = instance.DisplayName
	}
	publicNetwork := func() {
		out.Endpoints = append(out.Endpoints, routerMinecraftPorts(instanceRuntimeRecord{NetworkMember: true}, n.Config, "", true)...)
	}
	if id != "" {
		out.ServerName = names[id]
		record, err := instanceRuntimes.readRecord(id)
		if errors.Is(err, os.ErrNotExist) && out.ServerName != "" {
			return nil
		}
		if err != nil {
			return err
		}
		out.NetworkMember = effectiveMinecraftConnection(record, n.Config).networkMember
		if out.NetworkMember {
			publicNetwork()
		} else {
			out.Endpoints = routerMinecraftPorts(record, n.Config, names[id], false)
		}
		return nil
	}
	records, err := instanceRuntimes.records()
	if err != nil {
		return err
	}
	if n.Config.Enabled {
		publicNetwork()
	}
	for _, record := range records {
		if !effectiveMinecraftConnection(record, n.Config).networkMember {
			out.Endpoints = append(out.Endpoints, routerMinecraftPorts(record, n.Config, names[record.Runtime.ID], false)...)
		}
	}
	return nil
}

func containsRouterTarget(addresses []string, target string) bool {
	i := sort.SearchStrings(addresses, target)
	return i < len(addresses) && addresses[i] == target
}

func registerRouterOSRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET "+routerOSRoute, s.routerOSStatus)
	mux.HandleFunc("GET "+routerOSRoute+"/instances/{id}", s.routerOSStatus)
	mux.HandleFunc("POST "+routerOSRoute+"/test", s.routerOSTest)
	mux.HandleFunc("PUT "+routerOSRoute, s.routerOSSave)
	mux.HandleFunc("DELETE "+routerOSRoute, s.routerOSForget)
	mux.HandleFunc("PUT "+routerOSRoute+"/ca", s.routerOSChangeCA)
	mux.HandleFunc("DELETE "+routerOSRoute+"/ca", s.routerOSChangeCA)
}

func (s *server) routerOSStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if r.URL.ForceQuery || r.URL.RawQuery != "" || (id != "" && !instanceIDPattern.MatchString(id)) {
		writeError(w, 400, "invalid_request")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	out := routerOSStatus{Status: "not_configured", CanConfigure: id == "" && actor.Username == systemAdminUsername && actor.ConnectionID == ""}
	inventoryErr := out.inventory(id)
	if id != "" && errors.Is(inventoryErr, errInstanceNotFound) {
		writeError(w, 404, "inventory_error")
		return
	}
	out.InventoryError = inventoryErr != nil
	c, err := s.store.readRouterOS(ctx)
	switch {
	case err != nil:
		out.Configured, out.Status = true, "configuration_error"
	case c != nil:
		out.inspect(ctx, *c, id == "")
	}
	writeJSON(w, 200, out)
}

func (out *routerOSStatus) inspect(ctx context.Context, c routerOSConfiguration, details bool) {
	out.Configured, out.Status = true, "unavailable"
	publicText := c.publicTextFilter()
	if details {
		out.Endpoint = publicText(c.Endpoint)
		out.Username = publicText(c.Username)
		out.Target = publicText(c.Target)
		if publicText(c.Ingress.Name) != "" {
			out.Ingress = &c.Ingress
		}
	}

	client, cert, closeIdle, err := routerOSConnection(c.CAPEM)
	if details {
		out.TrustedCA = routerOSCAInfo(cert, publicText)
	}
	if err != nil {
		if details {
			out.Diagnostic = string(routerOSErrorCode(err))
		}
		return
	}
	defer closeIdle()
	d, err := client.discover(ctx, c.routerOSCredentials)
	if details {
		out.discovery(d, publicText)
	}
	if err != nil {
		if details {
			out.Diagnostic = string(routerOSErrorCode(err))
		}
		return
	}
	addresses, err := routerTargetAddresses(ctx)
	if err != nil {
		return
	}
	if !containsRouterTarget(addresses, c.Target) || !d.has(c.Ingress) {
		out.Status = "configuration_error"
		return
	}
	rows, err := client.readList(ctx, c.routerOSCredentials, routerOSNAT)
	if err != nil {
		if details {
			out.Diagnostic = string(routerOSErrorCode(err))
		}
		return
	}
	out.Status = "connected"
	rules, valid := routerOSNATRules(rows)
	ingress, known := d.ingress(c.Ingress)
	if !valid || !known {
		return
	}
	budget := 3000000
	for i := range out.Endpoints {
		out.Endpoints[i].State = routerOSEndpointState(out.Endpoints[i], rules, d, ingress, c.Target, &budget)
	}
}

func (s *server) routerOSTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if _, ok := s.requirePrimaryAdministrator(w, r); !ok {
		return
	}
	var c routerOSCredentials
	if r.URL.ForceQuery || r.URL.RawQuery != "" || decodeRouterOSRequest(r.Body, &c) != nil || !c.normalize() {
		writeError(w, 400, "invalid_credentials")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if c.CAPEM == "" {
		storedCA, _, err := s.store.readRouterOSCA(ctx)
		if err != nil {
			writeError(w, 503, "configuration_error")
			return
		}
		c.CAPEM = storedCA
	}
	client, cert, closeIdle, err := routerOSConnection(c.CAPEM)
	if err != nil {
		writeRouterOSConnectionError(w, err)
		return
	}
	defer closeIdle()
	d, err := client.discover(ctx, c)
	if err != nil {
		writeRouterOSConnectionError(w, err)
		return
	}
	addresses, err := routerTargetAddresses(ctx)
	if err != nil {
		writeError(w, 503, "target_unavailable")
		return
	}
	publicText := c.publicTextFilter()
	out := routerOSStatus{Status: "connected", TrustedCA: routerOSCAInfo(cert, publicText)}
	out.discovery(d, publicText)
	for name := range d.lists {
		if publicText(name) != "" {
			out.IngressChoices = append(out.IngressChoices, routerOSSelector{Type: "list", Name: name})
		}
	}
	for name := range d.interfaces {
		if publicText(name) != "" {
			out.IngressChoices = append(out.IngressChoices, routerOSSelector{Type: "interface", Name: name})
		}
	}
	sort.Slice(out.IngressChoices, func(i, j int) bool {
		a, b := out.IngressChoices[i], out.IngressChoices[j]
		if a.Type != b.Type {
			return a.Type == "list"
		}
		return a.Name < b.Name
	})
	for _, address := range addresses {
		if publicText(address) != "" {
			out.TargetChoices = append(out.TargetChoices, address)
		}
	}
	writeJSON(w, 200, out)
}

func (s *server) routerOSSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := s.requirePrimaryAdministrator(w, r)
	if !ok {
		return
	}
	var c routerOSConfiguration
	if r.URL.ForceQuery || r.URL.RawQuery != "" || decodeRouterOSRequest(r.Body, &c) != nil || !c.valid() {
		writeError(w, 400, "invalid_configuration")
		return
	}
	s.routerOSMu.Lock()
	defer s.routerOSMu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	addresses, err := routerTargetAddresses(ctx)
	if err != nil || !containsRouterTarget(addresses, c.Target) {
		writeError(w, 400, "target_unavailable")
		return
	}
	storedCA, configured, err := s.store.readRouterOSCA(ctx)
	if err != nil {
		writeError(w, 503, "configuration_error")
		return
	}
	if configured {
		if c.CAPEM != "" {
			writeError(w, 409, "ca_change_required")
			return
		}
		c.CAPEM = storedCA
	}
	client, _, closeIdle, err := routerOSConnection(c.CAPEM)
	if err != nil {
		writeRouterOSConnectionError(w, err)
		return
	}
	defer closeIdle()
	d, err := client.discover(ctx, c.routerOSCredentials)
	if err != nil {
		writeRouterOSConnectionError(w, err)
		return
	}
	if !d.has(c.Ingress) {
		writeError(w, 400, "ingress_unavailable")
		return
	}
	if _, err := client.readList(ctx, c.routerOSCredentials, routerOSNAT); err != nil {
		writeRouterOSConnectionError(w, err)
		return
	}
	addresses, err = routerTargetAddresses(ctx)
	if err != nil || !containsRouterTarget(addresses, c.Target) {
		writeError(w, 400, "target_unavailable")
		return
	}
	if !s.allowRouterOSSave(w) {
		return
	}
	if err := s.store.writeRouterOS(ctx, &c, actor, "routeros_save"); err != nil {
		writeError(w, 503, "configuration_error")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *server) routerOSForget(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := s.requirePrimaryAdministrator(w, r)
	if !ok {
		return
	}
	if r.URL.ForceQuery || r.URL.RawQuery != "" || r.ContentLength != 0 {
		writeError(w, 400, "invalid_request")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	s.routerOSMu.Lock()
	defer s.routerOSMu.Unlock()
	if s.store.writeRouterOS(ctx, nil, actor, "routeros_forget") != nil {
		writeError(w, 503, "configuration_error")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func routerOSConnection(caPEM string) (routerOSClient, *x509.Certificate, func(), error) {
	var cert *x509.Certificate
	if caPEM != "" {
		var err error
		cert, err = parseRouterOSCA(caPEM)
		if err != nil {
			return routerOSClient{}, nil, nil, err
		}
	}
	client, closeIdle, err := liveRouterOS.withCA(cert)
	return client, cert, closeIdle, err
}

func routerOSErrorCode(err error) routerOSError {
	var category routerOSError
	if errors.As(err, &category) {
		return category
	}
	return errRouterOSUnavailable
}

func writeRouterOSConnectionError(w http.ResponseWriter, err error) {
	code := routerOSErrorCode(err)
	status := http.StatusBadGateway
	if code == errRouterOSInvalidCA {
		status = http.StatusBadRequest
	}
	writeError(w, status, string(code))
}

func (s *server) allowRouterOSSave(w http.ResponseWriter) bool {
	if s.operations != nil {
		current, err := s.operations.currentFactoryReset()
		if err != nil || (current != nil && operationIsCurrent(current.State)) {
			writeError(w, 409, "factory_reset_pending")
			return false
		}
	}
	return true
}

func (s *server) routerOSChangeCA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := s.requirePrimaryAdministrator(w, r)
	if !ok {
		return
	}
	if r.URL.ForceQuery || r.URL.RawQuery != "" {
		writeError(w, 400, "invalid_request")
		return
	}
	var input struct {
		CAPEM string `json:"ca_pem"`
	}
	action := "routeros_ca_remove"
	if r.Method == http.MethodPut {
		if decodeRouterOSRequest(r.Body, &input) != nil || input.CAPEM == "" {
			writeRouterOSConnectionError(w, errRouterOSInvalidCA)
			return
		}
		action = "routeros_ca_replace"
	} else if r.ContentLength != 0 {
		writeError(w, 400, "invalid_request")
		return
	}

	s.routerOSMu.Lock()
	defer s.routerOSMu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	c, err := s.store.readRouterOS(ctx)
	if err != nil {
		writeError(w, 503, "configuration_error")
		return
	}
	if c == nil {
		writeError(w, 409, "ca_change_required")
		return
	}
	if !s.allowRouterOSSave(w) {
		return
	}
	c.CAPEM = input.CAPEM
	client, _, closeIdle, err := routerOSConnection(c.CAPEM)
	if err != nil {
		writeRouterOSConnectionError(w, err)
		return
	}
	defer closeIdle()
	if _, err := client.discover(ctx, c.routerOSCredentials); err != nil {
		writeRouterOSConnectionError(w, err)
		return
	}
	if _, err := client.readList(ctx, c.routerOSCredentials, routerOSNAT); err != nil {
		writeRouterOSConnectionError(w, err)
		return
	}
	if !s.allowRouterOSSave(w) {
		return
	}
	if err := s.store.writeRouterOS(ctx, c, actor, action); err != nil {
		writeError(w, 503, "configuration_error")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
