// Package urlencode encodes Hetzner Robot firewall payloads. The Robot
// API expects literal `[` and `]` in keys like `rules[input][0][name]`,
// so this package builds a query string by hand instead of going through
// url.Values (which would percent-encode the brackets).
package urlencode

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// EncodeFirewallRules encodes firewall rules into Hetzner's hierarchical
// format, e.g. rules[input][0][name]=rule1&rules[input][0][action]=accept.
// It returns a string instead of url.Values because the API expects the
// brackets in those keys to stay literal rather than percent-encoded.
//
// Directions and per-rule field keys are emitted in sorted order so the output
// is deterministic for a given input (Go map iteration order is randomized).
// Rule ordering within a direction follows the slice as given.
func EncodeFirewallRules(rules map[string][]map[string]string) string {
	var parts []string

	for _, direction := range sortedKeys(rules) {
		for i, rule := range rules[direction] {
			for _, key := range sortedKeys(rule) {
				// Build hierarchical key: rules[direction][index][field]
				// Encode only the value, not the key (brackets must stay literal)
				hierKey := fmt.Sprintf("rules[%s][%d][%s]", direction, i, key)
				encodedValue := url.QueryEscape(rule[key])
				parts = append(parts, fmt.Sprintf("%s=%s", hierKey, encodedValue))
			}
		}
	}

	return strings.Join(parts, "&")
}

// sortedKeys returns the keys of m in ascending order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FirewallRuleEncoder helps build firewall rules.
type FirewallRuleEncoder struct {
	rules map[string][]map[string]string
}

// NewFirewallRuleEncoder creates a new encoder.
func NewFirewallRuleEncoder() *FirewallRuleEncoder {
	return &FirewallRuleEncoder{
		rules: make(map[string][]map[string]string),
	}
}

// AddInputRule adds an input rule.
func (e *FirewallRuleEncoder) AddInputRule(rule map[string]string) {
	if e.rules["input"] == nil {
		e.rules["input"] = []map[string]string{}
	}
	e.rules["input"] = append(e.rules["input"], rule)
}

// AddOutputRule adds an output rule.
func (e *FirewallRuleEncoder) AddOutputRule(rule map[string]string) {
	if e.rules["output"] == nil {
		e.rules["output"] = []map[string]string{}
	}
	e.rules["output"] = append(e.rules["output"], rule)
}

// Encode returns the encoded form string.
func (e *FirewallRuleEncoder) Encode() string {
	return EncodeFirewallRules(e.rules)
}

// EncodeToString returns the complete encoded form string with additional
// values. The additional keys are emitted in sorted order for deterministic
// output, followed by the encoded rules.
func (e *FirewallRuleEncoder) EncodeToString(additional map[string]string) string {
	var parts []string

	// Add additional values first, in a stable order.
	for _, key := range sortedKeys(additional) {
		encodedValue := url.QueryEscape(additional[key])
		parts = append(parts, fmt.Sprintf("%s=%s", key, encodedValue))
	}

	// Add rules
	rulesStr := e.Encode()
	if rulesStr != "" {
		parts = append(parts, rulesStr)
	}

	return strings.Join(parts, "&")
}
