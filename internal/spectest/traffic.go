package spectest

import (
	"net/url"
	"strings"
)

// trafficAllowedScalarFields is the set of non-bracketed form keys the
// Robot API accepts for POST /traffic, per the doc's Input section (plus
// the doc's single-IP example, which uses a bare "ip" key rather than
// "ip[]").
var trafficAllowedScalarFields = map[string]bool{
	"ip":            true,
	"from":          true,
	"to":            true,
	"type":          true,
	"single_values": true,
}

// validateTrafficForm checks a POST /traffic body's key grammar by hand.
// The Robot API encodes "ip[]" and "subnet[]" as repeated bracket-key form
// fields (see the doc's "Query traffic data for multiple IPs" and "...for
// subnet" examples), which OpenAPI 3's form-urlencoded serialization
// cannot express as flat array properties (see the package doc comment for
// the analogous firewall/vSwitch bracket-key exceptions), so this is a
// targeted check instead of a schema-driven one: every key must either be
// one of the documented scalar fields or the literal "ip[]"/"subnet[]".
func validateTrafficForm(t Reporter, method, path string, body []byte) {
	t.Helper()

	values, err := url.ParseQuery(string(body))
	if err != nil {
		t.Errorf("spectest: %s %s: traffic form body is not valid application/x-www-form-urlencoded: %v", method, path, err)
		return
	}

	for key := range values {
		if trafficAllowedScalarFields[key] || key == "ip[]" || key == "subnet[]" {
			continue
		}
		t.Errorf("spectest: %s %s: form key %q does not match the documented traffic field grammar", method, path, key)
	}
}

// isTrafficPath reports whether path is the traffic query endpoint, whose
// POST body mixes ordinary scalar fields with ip[]/subnet[] bracket-key
// grammar that OpenAPI form-urlencoded serialization cannot express.
func isTrafficPath(path string) bool {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	return len(segments) == 1 && segments[0] == "traffic"
}
