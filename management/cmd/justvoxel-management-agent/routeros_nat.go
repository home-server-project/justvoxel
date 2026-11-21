package main

import (
	"strconv"
	"strings"
)

type routerOSPortRange struct{ first, last int }

type routerOSPortMatch struct {
	ranges []routerOSPortRange
	invert bool
	valid  bool
}

func parseRouterOSPorts(value string) routerOSPortMatch {
	value, invert := strings.CutPrefix(value, "!")
	if len(value) > 2048 || strings.Count(value, ",") >= 128 {
		return routerOSPortMatch{}
	}
	match := routerOSPortMatch{invert: invert, valid: true}
	if value == "" {
		match.valid = !invert
		return match
	}
	for _, part := range strings.Split(value, ",") {
		first, last, rangeSet := strings.Cut(strings.TrimSpace(part), "-")
		if !rangeSet {
			last = first
		}
		a, aErr := strconv.ParseUint(strings.TrimSpace(first), 10, 16)
		b, bErr := strconv.ParseUint(strings.TrimSpace(last), 10, 16)
		if aErr != nil || bErr != nil || a > b {
			return routerOSPortMatch{}
		}
		match.ranges = append(match.ranges, routerOSPortRange{int(a), int(b)})
	}
	return match
}

func (m routerOSPortMatch) matches(port int) bool {
	if len(m.ranges) == 0 {
		return true
	}
	for _, r := range m.ranges {
		if port >= r.first && port <= r.last {
			return !m.invert
		}
	}
	return m.invert
}

type routerOSNATRule struct {
	row         routerOSRow
	ports       routerOSPortMatch
	destination routerOSPortMatch
	conditional bool
}

func routerOSNATRules(rows []routerOSRow) ([]routerOSNATRule, bool) {
	rules := make([]routerOSNATRule, 0, len(rows))
	for _, row := range rows {
		if row["chain"] == "" || row["action"] == "" {
			return nil, false
		}
		if row["chain"] != "dstnat" || routerOSTrue(row["disabled"]) || routerOSTrue(row["invalid"]) || row["action"] == "passthrough" {
			continue
		}
		rule := routerOSNATRule{
			row: row, ports: parseRouterOSPorts(row["dst-port"]),
			destination: parseRouterOSPorts(row["to-ports"]),
		}
		for key, value := range row {
			if value == "" {
				continue
			}
			switch key {
			case ".id", "chain", "action", "dynamic", "protocol", "dst-port", "in-interface",
				"in-interface-list", "to-addresses", "to-ports", "comment", "log", "log-prefix", "bytes", "packets":
			case "disabled", "invalid":
				rule.conditional = rule.conditional || (value != "false" && value != "no")
			case "dst-address-type":
				rule.conditional = rule.conditional || value != "local"
			default:
				// Additional matchers (such as source address or connection marks)
				// may change which traffic reaches this rule or subsequent rules.
				rule.conditional = true
			}
		}
		rules = append(rules, rule)
	}
	return rules, true
}

type routerOSMatch uint8

const (
	routerOSNo routerOSMatch = iota
	routerOSYes
	routerOSMaybe
)

type routerOSIngress struct {
	name  string
	lists map[string]routerOSMatch
}

func (d routerOSDiscovery) ingress(selector routerOSSelector) ([]routerOSIngress, bool) {
	var selected []routerOSIngress
	// Bound list resolution and avoid publishing a partial picture for very large tables.
	if len(d.interfaces)*len(d.lists) > 100000 {
		return nil, false
	}
	for name := range d.interfaces {
		if selector.Type == "interface" && name != selector.Name {
			continue
		}
		iface := routerOSIngress{name: name, lists: make(map[string]routerOSMatch, len(d.lists))}
		for list := range d.lists {
			d.resolveList(&iface, list, 0)
		}
		if selector.Type == "list" {
			switch d.resolveList(&iface, selector.Name, 0) {
			case routerOSMaybe:
				return nil, false
			case routerOSNo:
				continue
			}
		}
		selected = append(selected, iface)
	}
	return selected, len(selected) > 0
}

func (d routerOSDiscovery) resolveList(iface *routerOSIngress, list string, depth int) routerOSMatch {
	switch list {
	case "all":
		return routerOSYes
	case "none":
		return routerOSNo
	case "dynamic", "static":
		switch d.interfaces[iface.name]["dynamic"] {
		case "true", "yes":
			if list == "dynamic" {
				return routerOSYes
			}
			return routerOSNo
		case "", "false", "no":
			if list == "static" {
				return routerOSYes
			}
			return routerOSNo
		default:
			return routerOSMaybe
		}
	}
	if result, known := iface.lists[list]; known {
		return result
	}
	row := d.lists[list]
	if row == nil || depth >= 32 {
		return routerOSMaybe
	}
	if d.members[list][iface.name] {
		iface.lists[list] = routerOSYes
		return routerOSYes
	}
	// A recursion placeholder detects include/exclude cycles. Explicit members
	// are applied last by RouterOS and were handled above.
	iface.lists[list] = routerOSMaybe
	included := d.resolveListSet(iface, row["include"], depth+1)
	excluded := d.resolveListSet(iface, row["exclude"], depth+1)
	result := routerOSMaybe
	switch {
	case included == routerOSNo || excluded == routerOSYes:
		result = routerOSNo
	case included == routerOSYes && excluded == routerOSNo:
		result = routerOSYes
	}
	iface.lists[list] = result
	return result
}

func (d routerOSDiscovery) resolveListSet(iface *routerOSIngress, lists string, depth int) routerOSMatch {
	result := routerOSNo
	for _, list := range strings.Split(lists, ",") {
		if list != "" {
			match := d.resolveList(iface, list, depth)
			if match == routerOSYes {
				return match
			}
			if match == routerOSMaybe {
				result = match
			}
		}
	}
	return result
}

func (iface *routerOSIngress) matches(value string, list bool, d routerOSDiscovery) routerOSMatch {
	if value == "" {
		return routerOSYes
	}
	name, invert := strings.CutPrefix(value, "!")
	match := routerOSNo
	if list {
		match = d.resolveList(iface, name, 0)
	} else if name == iface.name {
		match = routerOSYes
	} else if d.interfaces[name] == nil {
		match = routerOSMaybe
	}
	if invert && match != routerOSMaybe {
		return routerOSYes - match
	}
	return match
}

const (
	routerForwardOpen     = "open"
	routerForwardClosed   = "closed"
	routerForwardConflict = "conflict"
	routerForwardUnknown  = "unknown"
)

func routerOSPortState(rules []routerOSNATRule, d routerOSDiscovery, iface *routerOSIngress, target, protocol string, port int, budget *int) string {
	for _, rule := range rules {
		if *budget <= 0 {
			return routerForwardUnknown
		}
		*budget -= 1 + len(rule.ports.ranges)
		r := rule.row
		proto := strings.ToLower(r["protocol"])
		number := "6"
		if protocol == "udp" {
			number = "17"
		}
		if proto != "" && proto != protocol && proto != number && !strings.HasPrefix(proto, "!") {
			continue
		}
		if rule.ports.valid && !rule.ports.matches(port) {
			continue
		}
		direct := iface.matches(r["in-interface"], false, d)
		list := iface.matches(r["in-interface-list"], true, d)
		if direct == routerOSNo || list == routerOSNo {
			continue
		}
		if direct == routerOSMaybe || list == routerOSMaybe || !rule.ports.valid || rule.conditional || strings.HasPrefix(proto, "!") {
			return routerForwardUnknown
		}
		switch r["action"] {
		case "accept":
			return routerForwardClosed
		case "dst-nat":
			if r["to-addresses"] == "" {
				return routerForwardUnknown
			}
			if r["to-addresses"] != target {
				return routerForwardConflict
			}
			if r["to-ports"] == "" {
				return routerForwardOpen
			}
			if rule.destination.valid && !rule.destination.invert {
				if !rule.destination.matches(port) {
					return routerForwardConflict
				}
				if len(rule.destination.ranges) == 1 && rule.destination.ranges[0] == (routerOSPortRange{port, port}) {
					return routerForwardOpen
				}
			}
			return routerForwardUnknown
		case "redirect":
			return routerForwardConflict
		default:
			// A jump or an unmodeled action can affect the first applicable rule.
			return routerForwardUnknown
		}
	}
	return routerForwardClosed
}

func routerOSEndpointState(endpoint routerMinecraftEndpoint, rules []routerOSNATRule, d routerOSDiscovery, ingress []routerOSIngress, target string, budget *int) string {
	state := ""
	for _, protocol := range endpoint.Protocols {
		for port := endpoint.Port; port <= endpoint.LastPort; port++ {
			for i := range ingress {
				if *budget <= 0 {
					return routerForwardUnknown
				}
				*budget--
				result := routerOSPortState(rules, d, &ingress[i], target, protocol, port, budget)
				if result == routerForwardConflict {
					return result
				}
				if state == "" {
					state = result
				} else if state != result {
					state = routerForwardUnknown
				}
			}
		}
	}
	return state
}
