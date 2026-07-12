package spectest

import (
	"net/url"
	"strings"
)

// validateVSwitchServerForm checks a vSwitch add/remove-servers POST/DELETE
// body's key grammar by hand. The Robot API encodes the "server" array as
// repeated "server[]=value" form keys (see the doc's POST/DELETE
// /vswitch/{vswitch-id}/server examples), which OpenAPI 3's
// form-urlencoded serialization cannot express as a flat "server" array
// property (see the package doc comment for the analogous firewall
// bracket-key exception), so this is a targeted check instead of a
// schema-driven one: every key must be the literal "server[]" and there
// must be at least one value.
func validateVSwitchServerForm(t Reporter, method, path string, body []byte) {
	t.Helper()

	values, err := url.ParseQuery(string(body))
	if err != nil {
		t.Errorf("spectest: %s %s: vswitch server form body is not valid application/x-www-form-urlencoded: %v", method, path, err)
		return
	}

	for key := range values {
		if key != "server[]" {
			t.Errorf("spectest: %s %s: form key %q does not match the documented server[] grammar", method, path, key)
		}
	}

	if len(values["server[]"]) == 0 {
		t.Errorf("spectest: %s %s: expected at least one server[] value in form body", method, path)
	}
}

// isVSwitchServerPath reports whether path is the vSwitch add/remove-servers
// endpoint, whose POST/DELETE body uses server[] bracket-key grammar that
// OpenAPI form-urlencoded serialization cannot express.
func isVSwitchServerPath(path string) bool {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	return len(segments) == 3 && segments[0] == "vswitch" && segments[2] == "server"
}
