// Package spectest validates HTTP fixtures used in this module's tests
// against the vendored OpenAPI contract (spec/robot.yaml). It wraps an
// inner http.Handler (typically a test's httptest.NewServer handler) and,
// for every request that passes through it, checks that:
//
//   - the request (path, method, parameters, and form body) conforms to
//     the operation the spec defines for that path/method;
//   - the response (status code and JSON body) conforms to the schema the
//     spec defines for that response.
//
// Mismatches fail the *testing.T immediately with a descriptive message,
// turning "fixtures must be doc-verbatim" from a convention into a
// machine-enforced gate.
//
// # Why kin-openapi
//
// The package plan called for evaluating github.com/getkin/kin-openapi
// against github.com/pb33f/libopenapi-validator. kin-openapi was chosen:
// it exposes the request/response validation primitives
// (openapi3filter.ValidateRequest / ValidateResponse) as composable
// functions rather than only as a Go-context/gorilla middleware, which is
// what lets this package validate directly against a *testing.T instead of
// writing an HTTP error response. Its form-urlencoded body validation is
// also good enough to check the Hetzner Robot API's standard POST bodies
// (single-level key/value pairs) out of the box.
//
// # Known exception: firewall bracket-keys
//
// The Robot API encodes firewall rule lists as repeated bracketed form
// keys, e.g. "rules[input][0][name]=allow-ssh". OpenAPI 3's
// application/x-www-form-urlencoded serialization rules
// (https://spec.openapis.org/oas/v3.0.3#encoding-object) have no way to
// express this nested-array-of-objects shape as flat form keys, so
// kin-openapi cannot validate it against a schema. Requests to
// /firewall/{server-id} are therefore excluded from schema-based form body
// validation; instead validateFirewallForm checks the key grammar by hand
// (see firewall.go).
//
// # Known exception: vSwitch server[] bracket-keys
//
// Similarly, the Robot API encodes the vSwitch add/remove-servers "server"
// array as repeated "server[]=value" form keys (see the doc's POST/DELETE
// /vswitch/{vswitch-id}/server examples), which OpenAPI 3's
// form-urlencoded serialization cannot express as a flat "server" array
// property either. Requests to /vswitch/{vswitch-id}/server are therefore
// also excluded from schema-based form body validation; instead
// validateVSwitchServerForm checks the key grammar by hand (see
// vswitch.go).
//
// # Known exception: traffic ip[]/subnet[] bracket-keys
//
// The Robot API encodes POST /traffic's optional multi-value "ip" and
// "subnet" parameters as repeated "ip[]=value"/"subnet[]=value" form keys
// (see the doc's "Query traffic data for multiple IPs" and "...for
// subnet" examples), the same bracket-key grammar OpenAPI 3's
// form-urlencoded serialization cannot express. Requests to /traffic are
// therefore also excluded from schema-based form body validation; instead
// validateTrafficForm checks the key grammar by hand (see traffic.go).
package spectest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// Reporter is the subset of *testing.T that Handler needs to report
// validation failures. It exists (rather than taking *testing.T directly)
// so this package's own tests can assert on a mismatch being reported
// without failing the test binary itself: see the fakeReporter used by
// TestHandler_CatchesSpecMismatch.
type Reporter interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// Spec is a loaded and validated OpenAPI document plus a router that
// resolves incoming requests to the operation they must conform to. Load it
// once (e.g. in TestMain or a package-level sync.OnceValue) and reuse it
// across tests.
type Spec struct {
	doc    *openapi3.T
	router routers.Router
}

var (
	loadOnce sync.Once
	loaded   *Spec
	loadErr  error
)

// Load parses and validates spec/robot.yaml relative to the given path and
// builds a router used to match test requests to operations. Call it once
// per test binary (it caches the result) and pass path
// filepath.Join("..", "spec", "robot.yaml") (or similar) from package
// directories one level below the module root.
func Load(path string) (*Spec, error) {
	loadOnce.Do(func() {
		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = false

		doc, err := loader.LoadFromFile(path)
		if err != nil {
			loadErr = fmt.Errorf("spectest: load %s: %w", path, err)
			return
		}
		if err := doc.Validate(context.Background()); err != nil {
			loadErr = fmt.Errorf("spectest: validate %s: %w", path, err)
			return
		}

		// Clear the documented server list ("https://robot-ws.your-server.de")
		// so the router matches requests by path alone. Tests run against
		// httptest.Server URLs on 127.0.0.1 with random ports, which would
		// otherwise never match the documented host.
		doc.Servers = nil

		router, err := gorillamux.NewRouter(doc)
		if err != nil {
			loadErr = fmt.Errorf("spectest: build router for %s: %w", path, err)
			return
		}

		loaded = &Spec{doc: doc, router: router}
	})
	return loaded, loadErr
}

// Handler wraps inner with request/response validation against spec. Every
// request handled by the returned http.Handler is checked against the
// OpenAPI operation matching its path and method; every response written by
// inner is checked against that operation's response schema. Mismatches
// call t.Errorf with a descriptive message identifying the request that
// failed and why.
func Handler(t Reporter, spec *Spec, inner http.Handler) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, pathParams, err := spec.router.FindRoute(r)
		if err != nil {
			t.Errorf("spectest: %s %s does not match any documented operation in spec/robot.yaml: %v", r.Method, r.URL.Path, err)
			inner.ServeHTTP(w, r)
			return
		}

		// Buffer the body: openapi3filter consumes r.Body, but the inner
		// handler (and, for POST, r.ParseForm) needs to read it too.
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("spectest: read request body for %s %s: %v", r.Method, r.URL.Path, err)
			}
			_ = r.Body.Close()
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// The spec declares global HTTP Basic Auth security requirements.
		// Fixture servers authenticate via the test client's transport, not
		// via a security scheme this package evaluates, so authentication
		// itself is out of scope here: treat every request as authenticated.
		validationOptions := &openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		}

		isFirewallBody := isFirewallRulesPath(r.URL.Path) && r.Method == http.MethodPost
		isVSwitchServerBody := isVSwitchServerPath(r.URL.Path) && (r.Method == http.MethodPost || r.Method == http.MethodDelete)
		isTrafficBody := isTrafficPath(r.URL.Path) && r.Method == http.MethodPost
		switch {
		case isFirewallBody:
			validateFirewallForm(t, r.Method, r.URL.Path, bodyBytes)
		case isVSwitchServerBody:
			validateVSwitchServerForm(t, r.Method, r.URL.Path, bodyBytes)
		case isTrafficBody:
			validateTrafficForm(t, r.Method, r.URL.Path, bodyBytes)
		default:
			reqInput := &openapi3filter.RequestValidationInput{
				Request:    r,
				PathParams: pathParams,
				Route:      route,
				Options:    validationOptions,
			}
			if err := openapi3filter.ValidateRequest(r.Context(), reqInput); err != nil {
				t.Errorf("spectest: request %s %s does not conform to spec/robot.yaml: %v", r.Method, r.URL.Path, err)
			}
		}

		// Restore the body for the inner handler.
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		rec := httptest.NewRecorder()
		// Fixture handlers rarely set Content-Type explicitly (they just
		// json.NewEncoder(w).Encode(...)); httptest.ResponseRecorder then
		// defaults it via content-sniffing on the first Write, typically
		// landing on text/plain. Every Robot API response is JSON per the
		// doc, so pre-set it here (inner can still override it before its
		// first Write) instead of forcing every fixture to repeat that
		// boilerplate.
		rec.Header().Set("Content-Type", "application/json")
		inner.ServeHTTP(rec, r)

		// Copy the recorded response to the real ResponseWriter so the test
		// server behaves exactly as it would without spectest wrapping it.
		for k, vs := range rec.Header() {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(rec.Code)
		_, _ = w.Write(rec.Body.Bytes())

		respInput := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: &openapi3filter.RequestValidationInput{
				Request:    r,
				PathParams: pathParams,
				Route:      route,
				Options:    validationOptions,
			},
			Status:  rec.Code,
			Header:  rec.Header(),
			Body:    io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
			Options: validationOptions,
		}
		if err := openapi3filter.ValidateResponse(r.Context(), respInput); err != nil {
			t.Errorf("spectest: response %d from %s %s does not conform to spec/robot.yaml: %v\nbody: %s",
				rec.Code, r.Method, r.URL.Path, err, rec.Body.String())
		}
	})
}

// isFirewallRulesPath reports whether path is a firewall endpoint whose POST
// body may carry the rules[direction][index][field] bracket-key grammar that
// OpenAPI form-urlencoded serialization cannot express: either
// /firewall/{server-id}, /firewall/template, or /firewall/template/{id}.
func isFirewallRulesPath(path string) bool {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if segments[0] != "firewall" {
		return false
	}
	switch len(segments) {
	case 2:
		// /firewall/{server-id} or /firewall/template
		return true
	case 3:
		// /firewall/template/{template-id}
		return segments[1] == "template"
	default:
		return false
	}
}
