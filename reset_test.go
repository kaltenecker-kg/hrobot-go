package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaltenecker-kg/hrobot-go/v2/internal/spectest"
)

func TestResetService_Get(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset/321" {
			t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		body := `{
			"reset": {
				"server_ip": "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"server_number": 321,
				"type": [
					"sw",
					"hw",
					"man"
				],
				"operating_status": "not supported"
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	reset, err := client.Reset.Get(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Reset.Get returned error: %v", err)
	}

	if reset.ServerNumber != 321 {
		t.Errorf("expected server number 321, got %d", reset.ServerNumber)
	}

	if reset.ServerIPv6Net != "2a01:4f8:111:4221::" {
		t.Errorf("expected server_ipv6_net '2a01:4f8:111:4221::', got '%s'", reset.ServerIPv6Net)
	}

	if reset.OperatingStatus != "not supported" {
		t.Errorf("expected operating_status 'not supported', got '%s'", reset.OperatingStatus)
	}

	if len(reset.Type) != 3 {
		t.Errorf("expected 3 reset types, got %d", len(reset.Type))
	}

	expectedTypes := []ResetType{ResetTypeSoftware, ResetTypeHardware, ResetTypeManual}
	for i, expectedType := range expectedTypes {
		if i >= len(reset.Type) || reset.Type[i] != expectedType {
			t.Errorf("expected reset type %s at index %d, got %s", expectedType, i, reset.Type[i])
		}
	}
}

func TestResetService_Execute(t *testing.T) {
	tests := []struct {
		name      string
		resetType ResetType
	}{
		{
			name:      "software reset",
			resetType: ResetTypeSoftware,
		},
		{
			name:      "hardware reset",
			resetType: ResetTypeHardware,
		},
		{
			name:      "power reset",
			resetType: ResetTypePower,
		},
		{
			name:      "power long reset",
			resetType: ResetTypePowerLong,
		},
		{
			name:      "manual reset",
			resetType: ResetTypeManual,
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/reset/321" {
					t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("expected POST request, got '%s'", r.Method)
				}

				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				if r.FormValue("type") != string(tt.resetType) {
					t.Errorf("expected type '%s', got '%s'", tt.resetType, r.FormValue("type"))
				}

				// Doc-verbatim example from POST /reset/{server-number}: the
				// response's "type" is a single string (the executed reset
				// option), unlike GET /reset which returns an array of
				// available options.
				body := `{
					"reset": {
						"server_ip": "123.123.123.123",
						"server_ipv6_net": "2a01:4f8:111:4221::",
						"type": "` + string(tt.resetType) + `"
					}
				}`
				if _, err := w.Write([]byte(body)); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			reset, err := client.Reset.Execute(ctx, ServerID(321), tt.resetType)
			if err != nil {
				t.Fatalf("Reset.Execute returned error: %v", err)
			}

			if reset.ServerIP.String() != "123.123.123.123" {
				t.Errorf("expected server_ip '123.123.123.123', got '%s'", reset.ServerIP.String())
			}

			if len(reset.Type) != 1 || reset.Type[0] != tt.resetType {
				t.Errorf("expected type [%s], got %v", tt.resetType, reset.Type)
			}
		})
	}
}

func TestResetService_ExecuteSoftware(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset/321" {
			t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("type") != "sw" {
			t.Errorf("expected type 'sw', got '%s'", r.FormValue("type"))
		}

		body := `{
			"reset": {
				"server_ip": "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"type": "sw"
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	reset, err := client.Reset.ExecuteSoftware(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Reset.ExecuteSoftware returned error: %v", err)
	}

	if len(reset.Type) != 1 || reset.Type[0] != ResetTypeSoftware {
		t.Errorf("expected type [sw], got %v", reset.Type)
	}
}

func TestResetService_ExecuteHardware(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset/321" {
			t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("type") != "hw" {
			t.Errorf("expected type 'hw', got '%s'", r.FormValue("type"))
		}

		body := `{
			"reset": {
				"server_ip": "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"type": "hw"
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	reset, err := client.Reset.ExecuteHardware(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Reset.ExecuteHardware returned error: %v", err)
	}

	if len(reset.Type) != 1 || reset.Type[0] != ResetTypeHardware {
		t.Errorf("expected type [hw], got %v", reset.Type)
	}
}

func TestResetService_ExecutePower(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset/321" {
			t.Errorf("expected path '/reset/321', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST request, got '%s'", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if r.FormValue("type") != "power" {
			t.Errorf("expected type 'power', got '%s'", r.FormValue("type"))
		}

		body := `{
			"reset": {
				"server_ip": "123.123.123.123",
				"server_ipv6_net": "2a01:4f8:111:4221::",
				"type": "power"
			}
		}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	reset, err := client.Reset.ExecutePower(ctx, ServerID(321))
	if err != nil {
		t.Fatalf("Reset.ExecutePower returned error: %v", err)
	}

	if len(reset.Type) != 1 || reset.Type[0] != ResetTypePower {
		t.Errorf("expected type [power], got %v", reset.Type)
	}
}

func TestResetService_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		method     string
	}{
		{
			name:       "Get not found",
			statusCode: http.StatusNotFound,
			method:     "get",
		},
		{
			name:       "Execute unauthorized",
			statusCode: http.StatusUnauthorized,
			method:     "execute",
		},
		{
			name:       "ExecuteSoftware error",
			statusCode: http.StatusInternalServerError,
			method:     "executesoftware",
		},
	}

	spec := loadSpec(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"status":  tt.statusCode,
						"code":    "ERROR",
						"message": "test error",
					},
				})
			})))
			defer server.Close()

			client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
			ctx := context.Background()

			var err error
			switch tt.method {
			case "get":
				_, err = client.Reset.Get(ctx, ServerID(321))
			case "execute":
				_, err = client.Reset.Execute(ctx, ServerID(321), ResetTypeSoftware)
			case "executesoftware":
				_, err = client.Reset.ExecuteSoftware(ctx, ServerID(321))
			}

			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestResetService_List(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset" {
			t.Errorf("expected path '/reset', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got '%s'", r.Method)
		}
		body := `[
			{
				"reset": {
					"server_ip": "123.123.123.123",
					"server_ipv6_net": "2a01:4f8:111:4221::",
					"server_number": 321,
					"type": [
						"sw",
						"hw",
						"man"
					]
				}
			},
			{
				"reset": {
					"server_ip": "111.111.111.111",
					"server_ipv6_net": "2a01:4f8:111:4221::",
					"server_number": 111,
					"type": [
						"power",
						"power_long",
						"hw",
						"man"
					]
				}
			}
		]`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	resets, err := client.Reset.List(context.Background())
	if err != nil {
		t.Fatalf("Reset.List returned error: %v", err)
	}
	if len(resets) != 2 {
		t.Fatalf("expected 2 resets, got %d", len(resets))
	}
	if resets[0].ServerNumber != 321 {
		t.Errorf("expected server_number 321 on first, got %d", resets[0].ServerNumber)
	}
	if len(resets[0].Type) != 3 {
		t.Errorf("expected 3 reset types on first, got %d", len(resets[0].Type))
	}
	expectedFirst := []ResetType{ResetTypeSoftware, ResetTypeHardware, ResetTypeManual}
	for i, want := range expectedFirst {
		if resets[0].Type[i] != want {
			t.Errorf("expected reset type %s at index %d, got %s", want, i, resets[0].Type[i])
		}
	}
	if resets[1].ServerNumber != 111 {
		t.Errorf("expected server_number 111 on second, got %d", resets[1].ServerNumber)
	}
	if len(resets[1].Type) != 4 {
		t.Errorf("expected 4 reset types on second, got %d", len(resets[1].Type))
	}
	expectedSecond := []ResetType{ResetTypePower, ResetTypePowerLong, ResetTypeHardware, ResetTypeManual}
	for i, want := range expectedSecond {
		if resets[1].Type[i] != want {
			t.Errorf("expected reset type %s at index %d, got %s", want, i, resets[1].Type[i])
		}
	}
}

// TestResetService_List_Empty verifies an empty array response decodes to an empty slice, not an error.
func TestResetService_List_Empty(t *testing.T) {
	spec := loadSpec(t)
	server := httptest.NewServer(spectest.Handler(t, spec, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reset" {
			t.Errorf("expected path '/reset', got '%s'", r.URL.Path)
		}
		_, _ = w.Write([]byte("[]"))
	})))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	got, err := client.Reset.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if got == nil {
		t.Error("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}
