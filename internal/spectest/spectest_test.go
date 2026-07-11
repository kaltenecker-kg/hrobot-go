package spectest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeReporter is a minimal Reporter that records failures instead of
// failing the enclosing test binary, so this package can assert that
// Handler *does* report a spec mismatch without that mismatch failing
// TestHandler_CatchesSpecMismatch itself.
type fakeReporter struct {
	errors []string
}

func (f *fakeReporter) Helper() {}

func (f *fakeReporter) Errorf(format string, args ...any) {
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}

func (f *fakeReporter) Fatalf(format string, args ...any) {
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}

func loadTestSpec(t *testing.T) *Spec {
	t.Helper()
	spec, err := Load("../../spec/robot.yaml")
	if err != nil {
		t.Fatalf("failed to load ../../spec/robot.yaml: %v", err)
	}
	return spec
}

// TestHandler_CatchesSpecMismatch proves the gate works: a fixture that
// deliberately disagrees with the spec (the numeric "traffic": 5368709120
// instead of the documented human-readable string like "5 TB" or
// "unlimited") must be reported as a mismatch by Handler.
func TestHandler_CatchesSpecMismatch(t *testing.T) {
	spec := loadTestSpec(t)

	reporter := &fakeReporter{}
	badHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := map[string]any{
			"server": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"server_name":   "server1",
				"product":       "EX41",
				"dc":            "FSN1-DC5",
				"traffic":       5368709120, // wrong: spec requires a string like "5 TB"
				"status":        "ready",
				"cancelled":     false,
				"paid_until":    "2024-12-31",
				"ip":            []string{"123.123.123.123"},
				"subnet":        []map[string]any{},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	server := httptest.NewServer(Handler(reporter, spec, badHandler))
	defer server.Close()

	resp, err := http.Get(server.URL + "/server/321")
	if err != nil {
		t.Fatalf("GET /server/321: %v", err)
	}
	defer resp.Body.Close()

	if len(reporter.errors) == 0 {
		t.Fatalf("expected Handler to report a spec mismatch for a numeric traffic field, but it reported none")
	}

	found := false
	for _, e := range reporter.errors {
		if strings.Contains(e, "does not conform to spec/robot.yaml") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a 'does not conform to spec/robot.yaml' error, got: %v", reporter.errors)
	}
}

// TestHandler_AcceptsValidFixture is the negative control for
// TestHandler_CatchesSpecMismatch: a doc-verbatim fixture must not be
// reported as a mismatch.
func TestHandler_AcceptsValidFixture(t *testing.T) {
	spec := loadTestSpec(t)

	reporter := &fakeReporter{}
	goodHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := map[string]any{
			"server": map[string]any{
				"server_ip":     "123.123.123.123",
				"server_number": 321,
				"server_name":   "server1",
				"product":       "EX41",
				"dc":            "FSN1-DC5",
				"traffic":       "5 TB",
				"status":        "ready",
				"cancelled":     false,
				"paid_until":    "2024-12-31",
				"ip":            []string{"123.123.123.123"},
				"subnet":        []map[string]any{},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	server := httptest.NewServer(Handler(reporter, spec, goodHandler))
	defer server.Close()

	resp, err := http.Get(server.URL + "/server/321")
	if err != nil {
		t.Fatalf("GET /server/321: %v", err)
	}
	defer resp.Body.Close()

	if len(reporter.errors) != 0 {
		t.Fatalf("expected no spec mismatches for a doc-verbatim fixture, got: %v", reporter.errors)
	}
}
