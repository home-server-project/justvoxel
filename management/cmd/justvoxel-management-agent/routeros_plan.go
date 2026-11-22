package main

import (
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const routerNetworkOwner = "minecraft-network"

var routerRoles = [...]string{"java", "bedrock-tcp", "bedrock-udp"}

// RouterOS REST returns internal record numbers as an asterisk and hexadecimal digits.
var routerRecordID = regexp.MustCompile(`^\*[0-9A-Fa-f]+$`)

type routerNATChange struct {
	role   string
	method string
	id     string
	fields routerOSRow
}

type routerNATPlan struct {
	state, action string
	endpoints     []routerMinecraftEndpoint
	changes       []routerNATChange
	owned         int
	existing      bool
}

func routerScopeValid(scope string) bool {
	return scope == routerNetworkOwner || instanceIDPattern.MatchString(scope)
}

func routerOwnerComment(scope, role string) string {
	return "JustVoxel | " + scope + " | " + role
}

func routerDesiredRules(scope string, endpoints []routerMinecraftEndpoint, config routerOSConfiguration) (map[string]routerOSRow, []routerMinecraftEndpoint, bool) {
	desired := map[string]routerOSRow{}
	var split []routerMinecraftEndpoint
	if len(endpoints) == 0 || !routerScopeValid(scope) || !routerTargetIPv4(config.Target) || !config.Ingress.valid() {
		return nil, nil, false
	}
	for _, ep := range endpoints {
		if ep.Network != (scope == routerNetworkOwner) || (!ep.Network && ep.ID != scope) ||
			ep.Port < 1 || ep.LastPort < ep.Port || ep.LastPort > 65535 || len(ep.Protocols) == 0 {
			return nil, nil, false
		}
		ports := strconv.Itoa(ep.Port)
		if ep.LastPort != ep.Port {
			ports += "-" + strconv.Itoa(ep.LastPort)
		}
		for _, proto := range ep.Protocols {
			role := ep.Edition
			switch {
			case role == "java" && proto == "tcp":
			case role == "bedrock" && (proto == "tcp" || proto == "udp"):
				role += "-" + proto
			default:
				return nil, nil, false
			}
			if desired[role] != nil {
				return nil, nil, false
			}
			row := routerOSRow{"chain": "dstnat", "action": "dst-nat", "protocol": proto, "dst-port": ports,
				"to-addresses": config.Target, "to-ports": ports, "comment": routerOwnerComment(scope, role)}
			field := "in-interface"
			if config.Ingress.Type == "list" {
				field += "-list"
			}
			row[field] = config.Ingress.Name
			desired[role] = row
			entry := ep
			entry.Protocols = []string{proto}
			split = append(split, entry)
		}
	}
	return desired, split, true
}

// Evaluate foreign rules separately so a matching first rule cannot hide a later conflict.
// A partial match cannot satisfy the role or safely justify an overlapping new rule.
func routerForeignState(ep routerMinecraftEndpoint, foreign []routerOSNATRule, d routerOSDiscovery, ingress []routerOSIngress, target string, budget *int) string {
	state := routerForwardClosed
	for _, rule := range foreign {
		// An explicit accept terminates dstnat evaluation before an appended rule.
		if rule.row["action"] == "accept" {
			rule.row = maps.Clone(rule.row)
			rule.row["action"] = "redirect"
		}
		if ep.LastPort > ep.Port && rule.row["action"] == "dst-nat" &&
			rule.row["dst-port"] == strconv.Itoa(ep.Port)+"-"+strconv.Itoa(ep.LastPort) &&
			rule.row["to-ports"] == rule.row["dst-port"] {
			rule.row = maps.Clone(rule.row)
			delete(rule.row, "to-ports")
		}
		switch routerOSEndpointState(ep, []routerOSNATRule{rule}, d, ingress, target, budget) {
		case routerForwardOpen:
			state = routerForwardOpen
		case routerForwardClosed:
		case routerForwardConflict:
			return routerForwardConflict
		default:
			return routerForwardUnknown
		}
	}
	return state
}

// Only clear supported drift. Unknown active match fields require manual inspection.
func routerOwnedRepair(row, desired routerOSRow) (routerOSRow, bool) {
	patch := routerOSRow{}
	for key, value := range desired {
		if row[key] != value {
			patch[key] = value
		}
	}
	for key, value := range row {
		if value == "" || desired[key] != "" {
			continue
		}
		switch key {
		case ".id", "bytes", "packets", "log", "log-prefix":
		case "in-interface", "in-interface-list":
			patch[key] = ""
		case "disabled":
			if routerOSTrue(value) {
				patch[key] = "false"
			} else if value != "no" && value != "false" {
				return nil, false
			}
		case "dynamic", "invalid":
			if value != "no" && value != "false" {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	return patch, true
}

func planRouterNAT(scope string, endpoints []routerMinecraftEndpoint, config routerOSConfiguration, rows []routerOSRow, d routerOSDiscovery, disable bool) routerNATPlan {
	block := func(state string) routerNATPlan { return routerNATPlan{state: state} }
	desired, split, valid := routerDesiredRules(scope, endpoints, config)
	ingress, known := d.ingress(config.Ingress)
	if !valid || !known {
		return block(routerForwardUnknown)
	}
	owned := map[string][]routerOSRow{}
	var foreign []routerOSRow
	ids := map[string]bool{}
	for _, row := range rows {
		id := row[".id"]
		if !routerRecordID.MatchString(id) || ids[id] {
			return block(routerForwardUnknown)
		}
		ids[id] = true
		role := ""
		for _, candidate := range routerRoles {
			if row["comment"] == routerOwnerComment(scope, candidate) {
				role = candidate
				break
			}
		}
		if role == "" {
			foreign = append(foreign, row)
			continue
		}
		if row["chain"] != "dstnat" || row["action"] == "" || !routerStaticRecord(row) {
			return block(routerForwardUnknown)
		}
		if desired[role] != nil {
			if _, safe := routerOwnedRepair(row, desired[role]); !safe {
				return block(routerForwardUnknown)
			}
		}
		owned[role] = append(owned[role], row)
	}
	rules, valid := routerOSNATRules(foreign)
	if !valid {
		return block(routerForwardUnknown)
	}
	p := routerNATPlan{state: routerForwardOpen, endpoints: split}
	budget := 3000000
	existing := map[string]bool{}
	for i, ep := range split {
		role := ep.Edition
		if role == "bedrock" {
			role += "-" + ep.Protocols[0]
		}
		state := routerForeignState(ep, rules, d, ingress, config.Target, &budget)
		if state == routerForwardUnknown || state == routerForwardConflict {
			return block(state)
		}
		existing[role] = state == routerForwardOpen
		p.existing = p.existing || existing[role]
		if existing[role] {
			p.endpoints[i].Source = "existing"
		} else if len(owned[role]) != 0 {
			patch, ok := routerOwnedRepair(owned[role][0], desired[role])
			if !ok {
				return block(routerForwardUnknown)
			}
			if len(patch) == 0 {
				state, p.endpoints[i].Source = routerForwardOpen, "managed"
			} else {
				state = "needs_repair"
			}
		}
		p.endpoints[i].State = state
	}
	for _, role := range routerRoles {
		group := owned[role]
		want := desired[role]
		p.owned += len(group)
		slices.SortFunc(group, func(a, b routerOSRow) int { return strings.Compare(a[".id"], b[".id"]) })
		if !disable && want != nil && (!existing[role] || len(group) != 0) {
			if len(group) == 0 {
				p.changes = append(p.changes, routerNATChange{role: role, method: http.MethodPut, fields: want})
			} else {
				keep := 0
				for i, row := range group {
					patch, ok := routerOwnedRepair(row, want)
					if ok && len(patch) == 0 {
						keep = i
						break
					}
				}
				row := group[keep]
				patch, ok := routerOwnedRepair(row, want)
				if !ok {
					return block(routerForwardUnknown)
				}
				if len(patch) > 0 && existing[role] {
					// An existing mapping already serves this role. Remove redundant drift.
					p.changes = append(p.changes, routerNATChange{role: role, method: http.MethodDelete, id: row[".id"]})
				} else if len(patch) > 0 {
					p.changes = append(p.changes, routerNATChange{role, http.MethodPatch, row[".id"], patch})
				}
				group = slices.Delete(group, keep, keep+1)
			}
		}
		for _, row := range group {
			p.changes = append(p.changes, routerNATChange{role: role, method: http.MethodDelete, id: row[".id"]})
		}
	}
	switch {
	case len(p.changes) > 0 && p.owned > 0:
		p.action, p.state = "repair", "needs_repair"
	case len(p.changes) > 0:
		p.action, p.state = "enable", routerForwardClosed
	case p.owned > 0:
		p.action = "disable"
	case !p.existing:
		p.state = routerForwardClosed
	}
	return p
}
