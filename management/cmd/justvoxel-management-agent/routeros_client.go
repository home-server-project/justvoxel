package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
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

const routerOSResponseLimit = 1 << 20

var errRouterOSUnavailable = errors.New("router unavailable")

type routerOSCredentials struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
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

func (c routerOSClient) read(ctx context.Context, credentials routerOSCredentials, resource routerOSResource) ([]routerOSRow, error) {
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
		return nil, errRouterOSUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errRouterOSUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, routerOSResponseLimit+1))
	if err != nil || len(data) > routerOSResponseLimit {
		return nil, errRouterOSUnavailable
	}
	var rows []routerOSRow
	if json.Unmarshal(data, &rows) != nil || rows == nil || len(rows) > 10000 {
		return nil, errRouterOSUnavailable
	}
	for _, row := range rows {
		if len(row) == 0 {
			return nil, errRouterOSUnavailable
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
	identity, err := c.read(ctx, credentials, routerOSIdentity)
	if err != nil || len(identity) != 1 || identity[0]["name"] == "" {
		return d, errRouterOSUnavailable
	}
	system, err := c.read(ctx, credentials, routerOSSystem)
	if err != nil || len(system) != 1 || system[0]["version"] == "" {
		return d, errRouterOSUnavailable
	}
	d.name, d.version, d.board = identity[0]["name"], system[0]["version"], system[0]["board-name"]
	if d.board == "" {
		d.board = system[0]["platform"]
	}
	interfaces, err := c.read(ctx, credentials, routerOSInterfaces)
	if err != nil {
		return d, err
	}
	lists, err := c.read(ctx, credentials, routerOSLists)
	if err != nil {
		return d, err
	}
	members, err := c.read(ctx, credentials, routerOSListMembers)
	if err != nil {
		return d, err
	}
	d.interfaces = make(map[string]routerOSRow, len(interfaces))
	d.lists = make(map[string]routerOSRow, len(lists))
	d.members = make(map[string]map[string]bool, len(lists))
	for _, row := range interfaces {
		if !validRouterOSName(row["name"]) || d.interfaces[row["name"]] != nil {
			return d, errRouterOSUnavailable
		}
		d.interfaces[row["name"]] = row
	}
	for _, row := range lists {
		if !validRouterOSName(row["name"]) || d.lists[row["name"]] != nil {
			return d, errRouterOSUnavailable
		}
		if len(row["include"]) > 2048 || len(row["exclude"]) > 2048 ||
			strings.Count(row["include"], ",") >= 32 || strings.Count(row["exclude"], ",") >= 32 {
			return d, errRouterOSUnavailable
		}
		d.lists[row["name"]] = row
	}
	for _, row := range members {
		name, iface := row["list"], row["interface"]
		if !validRouterOSName(name) || !validRouterOSName(iface) {
			return d, errRouterOSUnavailable
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
