package urlencode

import "testing"

func TestEncodeFirewallRules_Deterministic(t *testing.T) {
	rules := map[string][]map[string]string{
		"input": {
			{"name": "allow-ssh", "action": "accept", "protocol": "tcp", "dst_port": "22"},
		},
	}

	// Per-rule field keys are sorted, so the output is fully specified.
	want := "rules[input][0][action]=accept&rules[input][0][dst_port]=22&" +
		"rules[input][0][name]=allow-ssh&rules[input][0][protocol]=tcp"

	first := EncodeFirewallRules(rules)
	if first != want {
		t.Errorf("EncodeFirewallRules() = %q, want %q", first, want)
	}

	// Repeated calls must produce identical output despite randomized map
	// iteration order.
	for i := 0; i < 50; i++ {
		if got := EncodeFirewallRules(rules); got != first {
			t.Fatalf("non-deterministic output on iteration %d: %q != %q", i, got, first)
		}
	}
}

func TestEncodeFirewallRules_SortsDirectionsAndPreservesRuleOrder(t *testing.T) {
	rules := map[string][]map[string]string{
		"output": {{"action": "accept"}},
		"input": {
			{"action": "accept"},
			{"action": "discard"},
		},
	}

	// input sorts before output; rule indices follow slice order.
	want := "rules[input][0][action]=accept&rules[input][1][action]=discard&" +
		"rules[output][0][action]=accept"
	if got := EncodeFirewallRules(rules); got != want {
		t.Errorf("EncodeFirewallRules() = %q, want %q", got, want)
	}
}

func TestEncodeFirewallRules_EscapesValues(t *testing.T) {
	rules := map[string][]map[string]string{
		"input": {{"src_ip": "0.0.0.0/0"}},
	}
	// The value's "/" is percent-encoded; the bracket key stays literal.
	want := "rules[input][0][src_ip]=0.0.0.0%2F0"
	if got := EncodeFirewallRules(rules); got != want {
		t.Errorf("EncodeFirewallRules() = %q, want %q", got, want)
	}
}

func TestEncodeFirewallRules_Empty(t *testing.T) {
	if got := EncodeFirewallRules(map[string][]map[string]string{}); got != "" {
		t.Errorf("EncodeFirewallRules(empty) = %q, want empty", got)
	}
}

func TestFirewallRuleEncoder_AddAndEncode(t *testing.T) {
	e := NewFirewallRuleEncoder()
	e.AddInputRule(map[string]string{"action": "accept"})
	e.AddOutputRule(map[string]string{"action": "discard"})

	want := "rules[input][0][action]=accept&rules[output][0][action]=discard"
	if got := e.Encode(); got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
}

func TestFirewallRuleEncoder_EncodeToString(t *testing.T) {
	e := NewFirewallRuleEncoder()
	e.AddInputRule(map[string]string{"action": "accept"})

	// Additional keys are sorted and precede the rules.
	want := "status=active&whitelist_hos=true&rules[input][0][action]=accept"
	got := e.EncodeToString(map[string]string{"whitelist_hos": "true", "status": "active"})
	if got != want {
		t.Errorf("EncodeToString() = %q, want %q", got, want)
	}
}

func TestFirewallRuleEncoder_EncodeToStringNoRules(t *testing.T) {
	e := NewFirewallRuleEncoder()
	want := "status=disabled"
	if got := e.EncodeToString(map[string]string{"status": "disabled"}); got != want {
		t.Errorf("EncodeToString() = %q, want %q", got, want)
	}
}

func TestRuleBuilder(t *testing.T) {
	rule := NewRuleBuilder().
		Name("allow-ssh").
		IPVersion("ipv4").
		Action("accept").
		Protocol("tcp").
		SourceIP("0.0.0.0/0").
		DestIP("1.2.3.4").
		SourcePort(1024).
		DestPort(uint16(22)).
		TCPFlags("syn").
		Build()

	want := map[string]string{
		"name":       "allow-ssh",
		"ip_version": "ipv4",
		"action":     "accept",
		"protocol":   "tcp",
		"src_ip":     "0.0.0.0/0",
		"dst_ip":     "1.2.3.4",
		"src_port":   "1024",
		"dst_port":   "22",
		"tcp_flags":  "syn",
	}
	if len(rule) != len(want) {
		t.Fatalf("rule has %d fields, want %d: %v", len(rule), len(want), rule)
	}
	for k, v := range want {
		if rule[k] != v {
			t.Errorf("rule[%q] = %q, want %q", k, rule[k], v)
		}
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"string", "80", "80"},
		{"int", 443, "443"},
		{"uint16", uint16(22), "22"},
		{"fallback", int64(65535), "65535"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toString(tt.in); got != tt.want {
				t.Errorf("toString(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
