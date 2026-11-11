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
