// Package urlencode encodes Hetzner Robot firewall payloads. The Robot
// API expects literal `[` and `]` in keys like `rules[input][0][name]`,
// so this package builds a query string by hand instead of going through
// url.Values (which would percent-encode the brackets).
package urlencode

import (
	"fmt"
	"net/url"
	"strconv"
)

// EncodeFirewallRules encodes firewall rules into Hetzner's hierarchical format
// Example: rules[input][0][name]=rule1&rules[input][0][action]=accept.
// Note: Returns a string instead of url.Values because Hetzner's API expects
// brackets in keys to NOT be URL-encoded.
func EncodeFirewallRules(rules map[string][]map[string]string) string {
	var parts []string

	for direction, ruleList := range rules {
		for i, rule := range ruleList {
			for key, value := range rule {
				// Build hierarchical key: rules[direction][index][field]
				// Encode only the value, not the key (brackets must stay literal)
				hierKey := fmt.Sprintf("rules[%s][%d][%s]", direction, i, key)
				encodedValue := url.QueryEscape(value)
				parts = append(parts, fmt.Sprintf("%s=%s", hierKey, encodedValue))
			}
		}
	}

	return joinParts(parts, "&")
}

// joinParts joins string parts with a separator.
func joinParts(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
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

// EncodeToString returns the complete encoded form string with additional values.
func (e *FirewallRuleEncoder) EncodeToString(additional map[string]string) string {
	var parts []string

	// Add additional values first
	for key, value := range additional {
		encodedValue := url.QueryEscape(value)
		parts = append(parts, fmt.Sprintf("%s=%s", key, encodedValue))
	}

	// Add rules
	rulesStr := e.Encode()
	if rulesStr != "" {
		parts = append(parts, rulesStr)
	}

	return joinParts(parts, "&")
}

// RuleBuilder helps build individual firewall rules.
type RuleBuilder struct {
	data map[string]string
}

// NewRuleBuilder creates a new rule builder.
func NewRuleBuilder() *RuleBuilder {
	return &RuleBuilder{
		data: make(map[string]string),
	}
}

// Name sets the rule name.
func (r *RuleBuilder) Name(name string) *RuleBuilder {
	r.data["name"] = name
	return r
}

// IPVersion sets the IP version.
func (r *RuleBuilder) IPVersion(version string) *RuleBuilder {
	r.data["ip_version"] = version
	return r
}

// Action sets the action (accept/discard).
func (r *RuleBuilder) Action(action string) *RuleBuilder {
	r.data["action"] = action
	return r
}

// Protocol sets the protocol.
func (r *RuleBuilder) Protocol(protocol string) *RuleBuilder {
	r.data["protocol"] = protocol
	return r
}

// SourceIP sets the source IP.
func (r *RuleBuilder) SourceIP(ip string) *RuleBuilder {
	r.data["src_ip"] = ip
	return r
}

// DestIP sets the destination IP.
func (r *RuleBuilder) DestIP(ip string) *RuleBuilder {
	r.data["dst_ip"] = ip
	return r
}

// SourcePort sets the source port.
func (r *RuleBuilder) SourcePort(port any) *RuleBuilder {
	r.data["src_port"] = toString(port)
	return r
}

// DestPort sets the destination port.
func (r *RuleBuilder) DestPort(port any) *RuleBuilder {
	r.data["dst_port"] = toString(port)
	return r
}

// TCPFlags sets TCP flags.
func (r *RuleBuilder) TCPFlags(flags string) *RuleBuilder {
	r.data["tcp_flags"] = flags
	return r
}

// Build returns the rule data.
func (r *RuleBuilder) Build() map[string]string {
	return r.data
}

// toString converts various types to string.
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	default:
		return fmt.Sprintf("%v", val)
	}
}
