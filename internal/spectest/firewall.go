package spectest

import (
	"net/url"
	"regexp"
	"strings"
)

// firewallRuleKey matches the Hetzner Robot firewall rule form-key grammar:
// rules[<direction>][<index>][<field>], e.g. "rules[input][0][name]".
// See internal/urlencode for the encoder that produces these keys.
var firewallRuleKey = regexp.MustCompile(`^rules\[(input|output)\]\[\d+\]\[[a-z_]+\]$`)

// firewallAllowedFields is the set of per-rule fields the Robot API accepts,
// per the doc's POST /firewall/{server-id} Input section.
var firewallAllowedFields = map[string]bool{
	"name":       true,
	"ip_version": true,
	"dst_ip":     true,
	"src_ip":     true,
	"dst_port":   true,
	"src_port":   true,
	"protocol":   true,
	"tcp_flags":  true,
	"action":     true,
}

// validateFirewallForm checks a firewall POST body's key grammar by hand.
// OpenAPI 3's form-urlencoded serialization cannot express the nested
// rules[direction][index][field] shape (see the package doc comment for
// why), so this is a targeted check instead of a schema-driven one: every
// non-top-level key must match the bracket grammar, and its field name must
// be one the API documents.
//
// Top-level keys (whitelist_hos, status, and the like) are left unvalidated
// here; they are ordinary scalar form fields and pass through normal
// OpenAPI validation on other operations that share this shape.
func validateFirewallForm(t Reporter, method, path string, body []byte) {
	t.Helper()

	values, err := url.ParseQuery(string(body))
	if err != nil {
		t.Errorf("spectest: %s %s: firewall form body is not valid application/x-www-form-urlencoded: %v", method, path, err)
		return
	}

	for key := range values {
		if !firewallRuleKeyOrScalar(key) {
			t.Errorf("spectest: %s %s: form key %q does not match the documented rules[input|output][n][field] grammar", method, path, key)
		}
	}
}

// firewallRuleKeyOrScalar reports whether key is a valid firewall rule
// bracket-key or a non-bracketed scalar field (e.g. "whitelist_hos").
func firewallRuleKeyOrScalar(key string) bool {
	if !strings.ContainsRune(key, '[') {
		// Scalar top-level field; not part of the rules[...] grammar.
		return true
	}
	if !firewallRuleKey.MatchString(key) {
		return false
	}
	field := key[strings.LastIndexByte(key, '[')+1 : len(key)-1]
	return firewallAllowedFields[field]
}
