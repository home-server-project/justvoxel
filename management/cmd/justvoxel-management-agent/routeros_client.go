package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	routerOSResponseLimit = 1 << 20
	routerOSCALimit       = 8 << 10
	routerOSRequestLimit  = 64 << 10
)

type routerOSError string

func (e routerOSError) Error() string { return string(e) }

const (
	errRouterOSUnavailable routerOSError = "unavailable"
	errRouterOSTLS         routerOSError = "tls_error"
	errRouterOSAuth        routerOSError = "auth_error"
	errRouterOSNetwork     routerOSError = "network_error"
	errRouterOSResponse    routerOSError = "response_error"
	errRouterOSInvalidCA   routerOSError = "invalid_ca"
)

type routerOSCAMetadata struct {
	CommonName  string    `json:"common_name,omitempty"`
	Fingerprint string    `json:"fingerprint"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func parseRouterOSCA(value string) (*x509.Certificate, error) {
	if len(value) > routerOSCALimit {
		return nil, errRouterOSInvalidCA
	}
	data := bytes.TrimSpace([]byte(value))
	if !bytes.HasPrefix(data, []byte("-----BEGIN CERTIFICATE-----")) ||
		bytes.Count(data, []byte("-----BEGIN ")) != 1 {
		return nil, errRouterOSInvalidCA
	}
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errRouterOSInvalidCA
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil || !cert.BasicConstraintsValid || !cert.IsCA {
		return nil, errRouterOSInvalidCA
	}
	// Key Usage is optional. When present, it must allow certificate signing.
	for _, ext := range cert.Extensions {
		if ext.Id.String() == "2.5.29.15" && cert.KeyUsage&x509.KeyUsageCertSign == 0 {
			return nil, errRouterOSInvalidCA
		}
	}
	return cert, nil
}

func routerOSCAInfo(cert *x509.Certificate, publicText func(string) string) *routerOSCAMetadata {
	if cert == nil {
		return nil
	}
	name := publicText(cert.Subject.CommonName)
	if strings.ContainsFunc(name, unicode.IsControl) {
		name = ""
	}
	sum := sha256.Sum256(cert.Raw)
	return &routerOSCAMetadata{
		CommonName: name, Fingerprint: hex.EncodeToString(sum[:]), ExpiresAt: cert.NotAfter.UTC(),
	}
}

type routerOSCredentials struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
	CAPEM    string `json:"ca_pem,omitempty"`
}

func (c *routerOSCredentials) normalize() bool {
	u, err := url.Parse(strings.TrimSpace(c.Endpoint))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" ||
		(u.Path != "" && u.Path != "/") || u.RawPath != "" || u.ForceQuery ||
		u.RawQuery != "" || strings.Contains(c.Endpoint, "#") || len(u.Host) > 260 {
		return false
	}
	host := strings.ToLower(u.Hostname())
	addr, ipErr := netip.ParseAddr(host)
	if ipErr != nil {
		if len(host) > 253 || strings.Contains(u.Host, "[") {
			return false
		}
		for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
			if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") ||
				strings.Trim(label, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
				return false
			}
		}
	} else if addr.Zone() != "" || (addr.Is6() != strings.HasPrefix(u.Host, "[")) {
		return false
	} else {
		host = addr.String()
	}
	if strings.HasSuffix(u.Host, ":") {
		return false
	}
	port := u.Port()
	if port != "" {
		number, err := strconv.ParseUint(port, 10, 16)
		if err != nil || number == 0 {
			return false
		}
		if number != 443 {
			host = net.JoinHostPort(host, strconv.FormatUint(number, 10))
		} else {
			port = ""
		}
	}
	if port == "" && addr.Is6() {
		host = "[" + host + "]"
	}
	c.Username = strings.TrimSpace(c.Username)
	if c.Username == "" || len(c.Username) > 128 || strings.Contains(c.Username, ":") ||
		c.Password == "" || len(c.Password) > 512 ||
		strings.ContainsFunc(c.Username, unicode.IsControl) || strings.ContainsFunc(c.Password, unicode.IsControl) {
		return false
	}
	c.Endpoint = "https://" + host
	return true
}

type routerOSResource uint8

const (
	routerOSIdentity routerOSResource = iota
	routerOSSystem
	routerOSInterfaces
	routerOSLists
	routerOSListMembers
	routerOSNAT
)

var routerOSPaths = [...]string{
	"/rest/system/identity",
	"/rest/system/resource",
	"/rest/interface",
	"/rest/interface/list",
	"/rest/interface/list/member",
	"/rest/ip/firewall/nat",
}

type routerOSRow map[string]string

type routerOSClient struct {
	transport *http.Transport
	http      *http.Client
}

func newRouterOSClient() routerOSClient {
	transport := &http.Transport{
		DialContext:            (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:    2 * time.Second,
		ResponseHeaderTimeout:  3 * time.Second,
		IdleConnTimeout:        10 * time.Second,
		MaxIdleConns:           8,
		MaxResponseHeaderBytes: 32 << 10,
		DisableCompression:     true,
	}
	return routerOSClient{transport: transport, http: &http.Client{
		Transport: transport,
		Timeout:   4 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

var liveRouterOS = newRouterOSClient()

// Use a fresh transport for each operation, including system-roots-only
// probes. A previously verified connection must not mask a router certificate
// change while testing replacement or removal.
func (c routerOSClient) withCA(cert *x509.Certificate) (routerOSClient, func(), error) {
	transport := c.transport.Clone()
	if cert != nil {
		roots := transport.TLSClientConfig.RootCAs
		if roots != nil {
			roots = roots.Clone()
		} else {
			var err error
			roots, err = x509.SystemCertPool()
			if err != nil {
				return routerOSClient{}, nil, errRouterOSTLS
			}
		}
		roots.AddCert(cert)
		transport.TLSClientConfig.RootCAs = roots
	}
	client := *c.http
	client.Transport = transport
	return routerOSClient{transport: transport, http: &client}, transport.CloseIdleConnections, nil
}

func (c routerOSClient) fetch(ctx context.Context, credentials routerOSCredentials, resource routerOSResource) ([]byte, error) {
	if int(resource) >= len(routerOSPaths) || !credentials.normalize() {
		return nil, errRouterOSUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, credentials.Endpoint+routerOSPaths[resource], nil)
	if err != nil {
		return nil, errRouterOSUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(credentials.Username, credentials.Password)
	resp, err := c.http.Do(req)
	if err != nil {
		var verification *tls.CertificateVerificationError
		var record tls.RecordHeaderError
		var alert tls.AlertError
		if errors.As(err, &verification) || errors.As(err, &record) || errors.As(err, &alert) ||
			errors.Is(err, http.ErrSchemeMismatch) {
			return nil, errRouterOSTLS
		}
		return nil, errRouterOSNetwork
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, errRouterOSAuth
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return nil, errRouterOSResponse
	default:
		return nil, errRouterOSUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, routerOSResponseLimit+1))
	if err != nil {
		return nil, errRouterOSNetwork
	}
	if len(data) > routerOSResponseLimit {
		return nil, errRouterOSResponse
	}
	return data, nil
}

func (c routerOSClient) readSingleton(ctx context.Context, credentials routerOSCredentials, resource routerOSResource) (routerOSRow, error) {
	if resource != routerOSIdentity && resource != routerOSSystem {
		return nil, errRouterOSUnavailable
	}
	data, err := c.fetch(ctx, credentials, resource)
	if err != nil {
		return nil, err
	}
	var row routerOSRow
	if json.Unmarshal(data, &row) != nil || len(row) == 0 {
		return nil, errRouterOSResponse
	}
	return row, nil
}

func (c routerOSClient) readList(ctx context.Context, credentials routerOSCredentials, resource routerOSResource) ([]routerOSRow, error) {
	if resource < routerOSInterfaces || resource > routerOSNAT {
		return nil, errRouterOSUnavailable
	}
	data, err := c.fetch(ctx, credentials, resource)
	if err != nil {
		return nil, err
	}
	var rows []routerOSRow
	if json.Unmarshal(data, &rows) != nil || rows == nil || len(rows) > 10000 {
		return nil, errRouterOSResponse
	}
	for _, row := range rows {
		if len(row) == 0 {
			return nil, errRouterOSResponse
		}
	}
	return rows, nil
}

type routerOSSelector struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (s routerOSSelector) valid() bool {
	return (s.Type == "interface" || s.Type == "list") && validRouterOSName(s.Name)
}

func validRouterOSName(name string) bool {
	return name != "" && len(name) <= 128 && !strings.HasPrefix(name, "!") && !strings.ContainsFunc(name, unicode.IsControl)
}

type routerOSDiscovery struct {
	name, version, board string
	interfaces           map[string]routerOSRow
	lists                map[string]routerOSRow
	members              map[string]map[string]bool
}

func (c routerOSClient) discover(ctx context.Context, credentials routerOSCredentials) (routerOSDiscovery, error) {
	var d routerOSDiscovery
	identity, err := c.readSingleton(ctx, credentials, routerOSIdentity)
	if err != nil {
		return d, err
	}
	system, err := c.readSingleton(ctx, credentials, routerOSSystem)
	if err != nil {
		return d, err
	}
	if identity["name"] == "" || system["version"] == "" {
		return d, errRouterOSResponse
	}
	d.name, d.version, d.board = identity["name"], system["version"], system["board-name"]
	if d.board == "" {
		d.board = system["platform"]
	}
	interfaces, err := c.readList(ctx, credentials, routerOSInterfaces)
	if err != nil {
		return d, err
	}
	lists, err := c.readList(ctx, credentials, routerOSLists)
	if err != nil {
		return d, err
	}
	members, err := c.readList(ctx, credentials, routerOSListMembers)
	if err != nil {
		return d, err
	}
	d.interfaces = make(map[string]routerOSRow, len(interfaces))
	d.lists = make(map[string]routerOSRow, len(lists))
	d.members = make(map[string]map[string]bool, len(lists))
	for _, row := range interfaces {
		if !validRouterOSName(row["name"]) || d.interfaces[row["name"]] != nil {
			return d, errRouterOSResponse
		}
		d.interfaces[row["name"]] = row
	}
	for _, row := range lists {
		if !validRouterOSName(row["name"]) || d.lists[row["name"]] != nil {
			return d, errRouterOSResponse
		}
		if len(row["include"]) > 2048 || len(row["exclude"]) > 2048 ||
			strings.Count(row["include"], ",") >= 32 || strings.Count(row["exclude"], ",") >= 32 {
			return d, errRouterOSResponse
		}
		d.lists[row["name"]] = row
	}
	for _, row := range members {
		name, iface := row["list"], row["interface"]
		if !validRouterOSName(name) || !validRouterOSName(iface) {
			return d, errRouterOSResponse
		}
		if routerOSTrue(row["disabled"]) {
			continue
		}
		if d.members[name] == nil {
			d.members[name] = map[string]bool{}
		}
		d.members[name][iface] = true
	}
	return d, nil
}

func (d routerOSDiscovery) has(s routerOSSelector) bool {
	if s.Type == "list" {
		return d.lists[s.Name] != nil
	}
	return s.Type == "interface" && d.interfaces[s.Name] != nil
}

func routerOSTrue(value string) bool {
	return value == "true" || value == "yes"
}
