package main

import (
	"maps"
	"reflect"
	"testing"
)

func TestRouterNATScopePlan(t *testing.T) {
	const scope = "jv-family1"
	c := testRouterConfiguration("https://router.example")
	d := routerOSDiscovery{
		interfaces: map[string]routerOSRow{"ether1": {"name": "ether1"}},
		lists:      map[string]routerOSRow{"uplinks": {"name": "uplinks"}},
		members:    map[string]map[string]bool{"uplinks": {"ether1": true}},
	}
	java := routerMinecraftEndpoint{ID: scope, Name: "Family", Edition: "java", Protocols: []string{"tcp"}, Port: 25565, LastPort: 25565}
	bedrock := routerMinecraftEndpoint{ID: scope, Edition: "bedrock", Protocols: []string{"udp"}, Port: 19132, LastPort: 19132}
	want := routerOSRow{"chain": "dstnat", "action": "dst-nat", "protocol": "tcp", "dst-port": "25565", "to-addresses": "192.0.2.9", "to-ports": "25565", "comment": "JustVoxel | jv-family1 | java", "in-interface-list": "uplinks"}
	with := func(row routerOSRow, pairs ...string) routerOSRow {
		copy := maps.Clone(row)
		for i := 0; i < len(pairs); i += 2 {
			copy[pairs[i]] = pairs[i+1]
		}
		return copy
	}
	owned := with(want, ".id", "*4")
	foreign := with(owned, "comment", "JustVoxel | jv-other12 | java", ".id", "*A")
	drift := with(owned, "to-ports", "25566")
	obsolete := with(owned, "comment", "JustVoxel | jv-family1 | bedrock-tcp", ".id", "*B")
	plan := func(rows ...routerOSRow) routerNATPlan {
		return planRouterNAT(scope, []routerMinecraftEndpoint{java}, c, rows, d, false)
	}
	check := func(p routerNATPlan, state string, changes ...routerNATChange) {
		t.Helper()
		if state != p.state || !reflect.DeepEqual(changes, p.changes) {
			t.Fatalf("got %s %#v; want %s %#v", p.state, p.changes, state, changes)
		}
	}
	check(plan(), "closed", routerNATChange{role: "java", method: "PUT", fields: want})
	check(plan(owned), "open")
	java.Name = "Renamed"
	check(plan(owned), "open")
	check(plan(drift), "needs_repair", routerNATChange{"java", "PATCH", "*4", routerOSRow{"to-ports": "25565"}})
	check(plan(obsolete, owned), "needs_repair", routerNATChange{role: "bedrock-tcp", method: "DELETE", id: "*B"})
	check(plan(with(owned, ".id", "*8"), owned), "needs_repair", routerNATChange{role: "java", method: "DELETE", id: "*8"})
	check(plan(with(drift, ".id", "*8"), drift), "needs_repair", routerNATChange{"java", "PATCH", "*4", routerOSRow{"to-ports": "25565"}}, routerNATChange{role: "java", method: "DELETE", id: "*8"})
	check(plan(foreign), "open")
	if p := plan(foreign); !p.existing || p.action != "" || p.endpoints[0].Source != "existing" {
		t.Fatal(p)
	}
	conflict := with(foreign, ".id", "*C", "to-addresses", "192.0.2.8")
	check(plan(owned, foreign, conflict), "conflict")
	check(plan(owned, with(conflict, "src-address-list", "special")), "unknown")
	check(plan(with(owned, ".id", "*4/../../system")), "unknown")
	check(planRouterNAT(scope, []routerMinecraftEndpoint{java, java}, c, nil, d, false), "unknown")
	before := maps.Clone(foreign)
	mixed := planRouterNAT(scope, []routerMinecraftEndpoint{java, bedrock}, c, []routerOSRow{foreign}, d, false)
	if len(mixed.changes) != 1 || mixed.changes[0].method != "PUT" || mixed.changes[0].fields["comment"] != "JustVoxel | jv-family1 | bedrock-udp" || !maps.Equal(foreign, before) {
		t.Fatal(mixed)
	}
	disable := planRouterNAT(scope, []routerMinecraftEndpoint{java}, c, []routerOSRow{owned, foreign}, d, true)
	check(disable, "needs_repair", routerNATChange{role: "java", method: "DELETE", id: "*4"})
	check(plan(foreign), "open")
	check(planRouterNAT(scope, []routerMinecraftEndpoint{java}, c, []routerOSRow{owned, conflict}, d, true), "conflict")
	java.Network = true
	check(plan(), "unknown")
	shared := planRouterNAT(routerNetworkOwner, []routerMinecraftEndpoint{java}, c, nil, d, false)
	if len(shared.changes) != 1 || shared.changes[0].fields["comment"] != "JustVoxel | minecraft-network | java" {
		t.Fatal(shared)
	}
}

func TestRouterRecordID(t *testing.T) {
	for _, id := range []string{"*0", "*1", "*A", "*a", "*01", "*123456789", "*abcdef012345"} {
		if !routerRecordID.MatchString(id) {
			t.Errorf("rejected valid record ID %q", id)
		}
	}
	for _, id := range []string{"", "*", "1", "*G", "*A?x", "*1/2", "*../1", "*1%2F2", " *1", "*1 ", "*1\n", "*1\x00"} {
		if routerRecordID.MatchString(id) {
			t.Errorf("accepted invalid record ID %q", id)
		}
	}
}
