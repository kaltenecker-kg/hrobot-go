package hrobot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
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

	t.Run("WithApplication name and version", func(t *testing.T) {
		client := NewClient("user", "pass", WithApplication("my-app", "2.3.4"))
		want := "my-app/2.3.4 " + UserAgent
		if client.userAgent != want {
			t.Errorf("userAgent = %s, want %s", client.userAgent, want)
		}
	})

	t.Run("WithApplication name only", func(t *testing.T) {
		client := NewClient("user", "pass", WithApplication("my-app", ""))
		want := "my-app " + UserAgent
		if client.userAgent != want {
			t.Errorf("userAgent = %s, want %s", client.userAgent, want)
		}
	})

	t.Run("WithApplication empty name is ignored", func(t *testing.T) {
		client := NewClient("user", "pass", WithApplication("", "2.3.4"))
		if client.userAgent != UserAgent {
			t.Errorf("userAgent = %s, want default %s", client.userAgent, UserAgent)
		}
	})

	t.Run("WithMaxRetryAfter overrides default", func(t *testing.T) {
		client := NewClient("user", "pass", WithMaxRetryAfter(2*time.Minute))
		if client.maxRetryAfter != 2*time.Minute {
			t.Errorf("maxRetryAfter = %v, want %v", client.maxRetryAfter, 2*time.Minute)
		}
	})

	t.Run("WithMaxRetryAfter ignores non-positive", func(t *testing.T) {
		client := NewClient("user", "pass", WithMaxRetryAfter(0))
		if client.maxRetryAfter != DefaultMaxRetryAfter {
			t.Errorf("maxRetryAfter = %v, want default %v", client.maxRetryAfter, DefaultMaxRetryAfter)
		}
	})

	t.Run("WithEndpoint aliases WithBaseURL", func(t *testing.T) {
		client := NewClient("user", "pass", WithEndpoint("https://custom.example.com/"))
		expected := "https://custom.example.com"
		if client.baseURL != expected {
			t.Errorf("baseURL = %s, want %s", client.baseURL, expected)
		}
	})

	t.Run("UserAgent derives from Version", func(t *testing.T) {
		want := "hrobot-go/" + Version
		if UserAgent != want {
			t.Errorf("UserAgent = %s, want %s", UserAgent, want)
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

func TestCredentialValidation(t *testing.T) {
	cases := []struct {
		name, user, pass string
	}{
		{"empty username", "", "pass"},
		{"empty password", "user", ""},
		{"both empty", "", ""},
		{"colon in username", "us:er", "pass"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				hits.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient(tc.user, tc.pass, WithBaseURL(server.URL))
			err := client.Get(context.Background(), "/server", nil)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !IsUnauthorizedError(err) {
				t.Errorf("IsUnauthorizedError = false, want true (err: %v)", err)
			}
			// The request must be rejected locally, before any HTTP call.
			if n := hits.Load(); n != 0 {
				t.Errorf("server received %d requests, want 0", n)
			}
		})
	}

	t.Run("valid credentials pass validation", func(t *testing.T) {
		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
		if err := client.Get(context.Background(), "/server", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n := hits.Load(); n != 1 {
			t.Errorf("server received %d requests, want 1", n)
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

func TestDoRequest_PostNotRetriedOn5xx(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewClient("user", "pass", WithBaseURL(srv.URL))

	resp, err := client.doRequest(context.Background(), http.MethodPost, "/test", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("requests = %d, want 1 (POST must not be retried on 5xx)", got)
	}
}

func TestDoRequest_GetRetriedOn5xxThenSucceeds(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient("user", "pass", WithBaseURL(srv.URL))

	resp, err := client.doRequest(context.Background(), http.MethodGet, "/test", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := atomic.LoadInt32(&requests); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
}

func TestDoRequest_PostRetriedOn429ThenSucceeds(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient("user", "pass", WithBaseURL(srv.URL))

	resp, err := client.doRequest(context.Background(), http.MethodPost, "/test", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("requests = %d, want 2 (POST must retry on 429)", got)
	}
}

func TestRetryAfter(t *testing.T) {
	const maxAfter = DefaultMaxRetryAfter
	future := time.Now().Add(365 * 24 * time.Hour).UTC().Format(http.TimeFormat)
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"absent", "", 0},
		{"zero seconds", "0", 0},
		{"small seconds", "5", 5 * time.Second},
		{"exactly cap", "30", maxAfter},
		{"above cap clamped", "315360000", maxAfter}, // ~10 years
		{"overflow value clamped", "99999999999999999", maxAfter},
		{"out-of-range value clamped", "99999999999999999999999999", maxAfter},
		{"negative ignored", "-5", 0},
		{"garbage ignored", "soon", 0},
		{"far-future http date clamped", future, maxAfter},
		{"past http date ignored", "Mon, 02 Jan 2006 15:04:05 GMT", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.value != "" {
				h.Set("Retry-After", tt.value)
			}
			if got := retryAfter(h, maxAfter); got != tt.want {
				t.Errorf("retryAfter(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestDoRequest_RetryAfterClamped proves a hostile Retry-After cannot pin the
// caller: the retry sleep is bounded by the configured cap even though the
// context has no deadline and the header requests a ~10-year delay. It also
// exercises WithMaxRetryAfter.
func TestDoRequest_RetryAfterClamped(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "315360000") // ~10 years
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Shorten the cap via the public option so we don't wait the full default.
	client := NewClient("user", "pass", WithBaseURL(srv.URL), WithMaxRetryAfter(50*time.Millisecond))

	done := make(chan struct{})
	start := time.Now()
	go func() {
		resp, err := client.doRequest(context.Background(), http.MethodGet, "/test", nil)
		if resp != nil {
			resp.Body.Close()
		}
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("retry sleep not clamped: took %v", elapsed)
		}
		if got := atomic.LoadInt32(&requests); got != 2 {
			t.Errorf("requests = %d, want 2 (retried after clamped delay)", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("doRequest still blocked after 5s: Retry-After not clamped")
	}
}

func TestDoRequest_UnauthorizedRetriedOnceThenFails(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewClient("user", "pass", WithBaseURL(srv.URL))

	resp, err := client.doRequest(context.Background(), http.MethodPost, "/test", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("requests = %d, want 2 (401 retried exactly once)", got)
	}
}

func TestUpdateRateLimit(t *testing.T) {
	t.Run("parses limit and remaining", func(t *testing.T) {
		c := NewClient("user", "pass")
		h := http.Header{}
		h.Set("RateLimit-Limit", "200")
		h.Set("RateLimit-Remaining", "197")
		c.updateRateLimit(h)

		rl := c.LastRateLimit()
		if rl.Limit != 200 {
			t.Errorf("Limit = %d, want 200", rl.Limit)
		}
		if rl.Remaining != 197 {
			t.Errorf("Remaining = %d, want 197", rl.Remaining)
		}
	})

	t.Run("reset as absolute unix timestamp", func(t *testing.T) {
		c := NewClient("user", "pass")
		abs := time.Now().Add(time.Hour).Unix() // well above the 1e9 heuristic threshold
		h := http.Header{}
		h.Set("RateLimit-Reset", strconv.FormatInt(abs, 10))
		c.updateRateLimit(h)

		if got := c.LastRateLimit().Reset.Unix(); got != abs {
			t.Errorf("Reset = %d, want %d (absolute unix timestamp)", got, abs)
		}
	})

	t.Run("reset as seconds-until-reset delta", func(t *testing.T) {
		c := NewClient("user", "pass")
		before := time.Now()
		h := http.Header{}
		h.Set("RateLimit-Reset", "60") // small value: treated as a delta
		c.updateRateLimit(h)

		reset := c.LastRateLimit().Reset
		// Expect roughly now+60s; allow slack for test execution time.
		if reset.Before(before.Add(59*time.Second)) || reset.After(before.Add(70*time.Second)) {
			t.Errorf("Reset = %v, want ~%v", reset, before.Add(60*time.Second))
		}
	})

	t.Run("no headers preserves prior state", func(t *testing.T) {
		c := NewClient("user", "pass")
		seed := http.Header{}
		seed.Set("RateLimit-Limit", "100")
		seed.Set("RateLimit-Remaining", "42")
		c.updateRateLimit(seed)
		want := c.LastRateLimit()

		c.updateRateLimit(http.Header{}) // nothing seen: must not clobber state
		if got := c.LastRateLimit(); got != want {
			t.Errorf("LastRateLimit = %+v, want unchanged %+v", got, want)
		}
	})

	t.Run("garbage values preserve prior state", func(t *testing.T) {
		c := NewClient("user", "pass")
		seed := http.Header{}
		seed.Set("RateLimit-Limit", "100")
		seed.Set("RateLimit-Remaining", "42")
		c.updateRateLimit(seed)
		want := c.LastRateLimit()

		garbage := http.Header{}
		garbage.Set("RateLimit-Limit", "not-a-number")
		garbage.Set("RateLimit-Reset", "soon")
		c.updateRateLimit(garbage) // nothing parses: must not clobber state
		if got := c.LastRateLimit(); got != want {
			t.Errorf("LastRateLimit = %+v, want unchanged %+v", got, want)
		}
	})
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		statusCode int
		attempt    int
		want       bool
	}{
		{"401 first attempt any method retries", http.MethodPost, http.StatusUnauthorized, 0, true},
		{"401 second attempt does not retry", http.MethodPost, http.StatusUnauthorized, 1, false},
		{"429 GET retries", http.MethodGet, http.StatusTooManyRequests, 0, true},
		{"429 POST retries", http.MethodPost, http.StatusTooManyRequests, 0, true},
		{"5xx GET retries", http.MethodGet, http.StatusInternalServerError, 0, true},
		{"5xx DELETE retries", http.MethodDelete, http.StatusInternalServerError, 0, true},
		{"5xx PUT retries", http.MethodPut, http.StatusInternalServerError, 0, true},
		{"5xx POST does not retry", http.MethodPost, http.StatusInternalServerError, 0, false},
		{"2xx does not retry", http.MethodGet, http.StatusOK, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetry(tt.method, tt.statusCode, tt.attempt)
			if got != tt.want {
				t.Errorf("shouldRetry(%s, %d, %d) = %v, want %v", tt.method, tt.statusCode, tt.attempt, got, tt.want)
			}
		})
	}
}
