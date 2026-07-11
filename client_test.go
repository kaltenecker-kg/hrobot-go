package hrobot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnwrapResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "array response",
			input:    `[{"id":1},{"id":2}]`,
			expected: `[{"id":1},{"id":2}]`,
		},
		{
			name:     "wrapped in server key",
			input:    `{"server":{"id":123,"name":"test"}}`,
			expected: `{"id":123,"name":"test"}`,
		},
		{
			// Single-key wrappers are auto-unwrapped regardless of name.
			// Real traffic responses have multiple top-level fields (`type`,
			// `from`, `to`, `data`), so the heuristic leaves them alone.
			name:     "single-key wrapper auto-unwraps",
			input:    `{"data":{"id":123}}`,
			expected: `{"id":123}`,
		},
		{
			name:     "multi-key object passes through",
			input:    `{"type":"day","data":{"in":1}}`,
			expected: `{"type":"day","data":{"in":1}}`,
		},
		{
			name:     "wrapped in firewall key",
			input:    `{"firewall":{"status":"active"}}`,
			expected: `{"status":"active"}`,
		},
		{
			name:     "no wrapper",
			input:    `{"id":123}`,
			expected: `{"id":123}`,
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unwrapResponse([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare as normalized JSON to ignore whitespace differences
			var resultJSON, expectedJSON any
			if err := json.Unmarshal(result, &resultJSON); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &expectedJSON); err != nil {
				t.Fatalf("failed to unmarshal expected: %v", err)
			}

			resultBytes, _ := json.Marshal(resultJSON)
			expectedBytes, _ := json.Marshal(expectedJSON)

			if string(resultBytes) != string(expectedBytes) {
				t.Errorf("unwrapResponse() = %s, want %s", string(resultBytes), string(expectedBytes))
			}
		})
	}
}

func TestUnwrapResponse_Array(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wrapped servers",
			input:    `[{"server":{"id":1,"name":"s1"}},{"server":{"id":2,"name":"s2"}}]`,
			expected: `[{"id":1,"name":"s1"},{"id":2,"name":"s2"}]`,
		},
		{
			name:     "wrapped ips",
			input:    `[{"ip":{"address":"1.2.3.4"}},{"ip":{"address":"5.6.7.8"}}]`,
			expected: `[{"address":"1.2.3.4"},{"address":"5.6.7.8"}]`,
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: `[]`,
		},
		{
			name:     "single item",
			input:    `[{"server":{"id":1}}]`,
			expected: `[{"id":1}]`,
		},
		{
			name:     "mixed keys returns input unchanged",
			input:    `[{"server":{}},{"firewall":{}}]`,
			expected: `[{"server":{}},{"firewall":{}}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unwrapResponse([]byte(tt.input))
			if err != nil {
				t.Fatalf("unwrapResponse() error = %v", err)
			}
			var got, want any
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &want); err != nil {
				t.Fatalf("failed to unmarshal expected: %v", err)
			}
			gotBytes, _ := json.Marshal(got)
			wantBytes, _ := json.Marshal(want)
			if string(gotBytes) != string(wantBytes) {
				t.Errorf("unwrapResponse() = %s, want %s", gotBytes, wantBytes)
			}
		})
	}
}

func TestClientOptions(t *testing.T) {
	t.Run("WithBaseURL", func(t *testing.T) {
		client := NewClient("user", "pass", WithBaseURL("https://custom.example.com/"))
		expected := "https://custom.example.com"
		if client.baseURL != expected {
			t.Errorf("baseURL = %s, want %s", client.baseURL, expected)
		}
	})

	t.Run("WithUserAgent", func(t *testing.T) {
		customUA := "custom-agent/1.0"
		client := NewClient("user", "pass", WithUserAgent(customUA))
		if client.userAgent != customUA {
			t.Errorf("userAgent = %s, want %s", client.userAgent, customUA)
		}
	})

	t.Run("default values", func(t *testing.T) {
		client := NewClient("user", "pass")
		if client.baseURL != DefaultBaseURL {
			t.Errorf("baseURL = %s, want %s", client.baseURL, DefaultBaseURL)
		}
		if client.userAgent != UserAgent {
			t.Errorf("userAgent = %s, want %s", client.userAgent, UserAgent)
		}
		if client.username != "user" {
			t.Errorf("username = %s, want user", client.username)
		}
		if client.password != "pass" {
			t.Errorf("password = %s, want pass", client.password)
		}
	})

	t.Run("services initialized", func(t *testing.T) {
		client := NewClient("user", "pass")
		if client.Server == nil {
			t.Error("Server service not initialized")
		}
		if client.Firewall == nil {
			t.Error("Firewall service not initialized")
		}
		if client.IP == nil {
			t.Error("IP service not initialized")
		}
		if client.Boot == nil {
			t.Error("Boot service not initialized")
		}
		if client.Reset == nil {
			t.Error("Reset service not initialized")
		}
	})

	t.Run("New alias works", func(t *testing.T) {
		client := New("user", "pass")
		if client == nil {
			t.Fatal("New() returned nil")
			return
		}
		if client.username != "user" {
			t.Errorf("username = %s, want user", client.username)
		}
	})
}

func TestClient_DeleteRaw(t *testing.T) {
	t.Run("sends DELETE with raw body and form content type", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE request, got '%s'", r.Method)
			}

			if r.URL.Path != "/vswitch/12345/server" {
				t.Errorf("expected path '/vswitch/12345/server', got '%s'", r.URL.Path)
			}

			contentType := r.Header.Get("Content-Type")
			if contentType != "application/x-www-form-urlencoded" {
				t.Errorf("expected Content-Type 'application/x-www-form-urlencoded', got '%s'", contentType)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			if string(body) != "server[]=1.2.3.4&server[]=5.6.7.8" {
				t.Errorf("expected body 'server[]=1.2.3.4&server[]=5.6.7.8', got '%s'", string(body))
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
		ctx := context.Background()

		err := client.DeleteRaw(ctx, "/vswitch/12345/server", "server[]=1.2.3.4&server[]=5.6.7.8", nil)
		if err != nil {
			t.Fatalf("DeleteRaw returned error: %v", err)
		}
	})

	t.Run("empty body sends no body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE request, got '%s'", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			if len(body) != 0 {
				t.Errorf("expected empty body, got '%s'", string(body))
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
		ctx := context.Background()

		err := client.DeleteRaw(ctx, "/vswitch/12345/server", "", nil)
		if err != nil {
			t.Fatalf("DeleteRaw returned error: %v", err)
		}
	})
}
