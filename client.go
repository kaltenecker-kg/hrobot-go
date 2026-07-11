package hrobot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Default client configuration values.
const (
	// DefaultBaseURL is the public Hetzner Robot endpoint.
	DefaultBaseURL = "https://robot-ws.your-server.de"
	// DefaultTimeout is the per-request HTTP timeout.
	DefaultTimeout = 30 * time.Second
	// UserAgent is the default User-Agent header value.
	UserAgent = "hrobot-go/1.0.0"
)

// Client is the main API client for Hetzner Robot.
type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
	userAgent  string
	logger     *slog.Logger

	rateLimitMu sync.RWMutex
	rateLimit   RateLimit

	// API Services
	Server     *ServerService
	Firewall   *FirewallService
	Reset      *ResetService
	Boot       *BootService
	IP         *IPService
	Key        *KeyService
	Auction    *AuctionService
	Ordering   *OrderingService
	VSwitch    *VSwitchService
	RDNS       *RDNSService
	Failover   *FailoverService
	Traffic    *TrafficService
	WOL        *WOLService
	Subnet     *SubnetService
	StorageBox *StorageBoxService
}

// RateLimit captures the rate-limit state reported by the most recent
// response. Zero values mean the server did not report that field.
type RateLimit struct {
	// Limit is the request quota for the current window.
	Limit int
	// Remaining is how many requests remain in the current window.
	Remaining int
	// Reset is when the current window resets.
	Reset time.Time
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(url, "/")
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithUserAgent sets a custom user agent.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// WithLogger attaches a structured logger. The client emits DEBUG-level
// events for each request, response, and retry, with attributes such as
// "method", "url", "status", "attempt", and "retry_after". Pass nil to
// silence (the default). POST requests are never retried on 5xx or
// transport errors because they may have side effects; 429/401 are safe to
// retry because the API did not execute the request.
//
// Authorization headers are never logged.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a new Hetzner Robot API client.
func NewClient(username, password string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL:   DefaultBaseURL,
		username:  username,
		password:  password,
		userAgent: UserAgent,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.logger == nil {
		c.logger = slog.New(slog.DiscardHandler)
	}

	c.Server = NewServerService(c)
	c.Firewall = NewFirewallService(c)
	c.Reset = NewResetService(c)
	c.Boot = NewBootService(c)
	c.IP = NewIPService(c)
	c.Key = NewKeyService(c)
	c.Auction = NewAuctionService(c)
	c.Ordering = NewOrderingService(c)
	c.VSwitch = NewVSwitchService(c)
	c.RDNS = NewRDNSService(c)
	c.Failover = NewFailoverService(c)
	c.Traffic = NewTrafficService(c)
	c.WOL = NewWOLService(c)
	c.Subnet = NewSubnetService(c)
	c.StorageBox = NewStorageBoxService(c)

	return c
}

// New creates a new Hetzner Robot API client (alias for NewClient).
func New(username, password string, opts ...ClientOption) *Client {
	return NewClient(username, password, opts...)
}

// LastRateLimit returns the rate-limit state observed on the most recent
// response. The zero value is returned when no rate-limit headers have been
// seen yet.
func (c *Client) LastRateLimit() RateLimit {
	c.rateLimitMu.RLock()
	defer c.rateLimitMu.RUnlock()
	return c.rateLimit
}

func (c *Client) updateRateLimit(h http.Header) {
	rl := RateLimit{}
	seen := false
	if v := h.Get("RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.Limit = n
			seen = true
		}
	}
	if v := h.Get("RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.Remaining = n
			seen = true
		}
	}
	if v := h.Get("RateLimit-Reset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			// Hetzner reports an absolute unix timestamp; some servers send
			// seconds-until-reset. Heuristic: small values are deltas.
			if n < 1_000_000_000 {
				rl.Reset = time.Now().Add(time.Duration(n) * time.Second)
			} else {
				rl.Reset = time.Unix(n, 0)
			}
			seen = true
		}
	}
	if !seen {
		return
	}
	c.rateLimitMu.Lock()
	c.rateLimit = rl
	c.rateLimitMu.Unlock()
}

// retryAfter returns the duration the server requested via Retry-After,
// or 0 if no parseable value is present.
func retryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// unwrapResponse strips Hetzner's `{"<resource>": ...}` envelope from a
// response body. The Robot API consistently wraps single-resource responses
// in an object with one key (e.g. `{"server": {...}}`) and list responses
// in an array of those wrappers (e.g. `[{"server": {...}}, ...]`). This
// helper auto-detects both shapes and yields the inner payload.
//
// Special cases:
//   - Bodies that already have an "id" top-level key are treated as
//     unwrapped resources and returned as-is.
//   - Arrays whose elements have differing or multiple keys, or scalar
//     bodies, are returned unchanged.
func unwrapResponse(data []byte) (json.RawMessage, error) {
	if len(data) == 0 {
		return data, nil
	}
	switch data[0] {
	case '[':
		var arr []map[string]json.RawMessage
		// Not an array of objects (e.g. array of scalars) → leave as-is.
		_ = json.Unmarshal(data, &arr)
		if len(arr) == 0 {
			return data, nil
		}
		var commonKey string
		for _, item := range arr {
			if len(item) != 1 {
				return data, nil
			}
			for k, v := range item {
				if len(v) == 0 || (v[0] != '{' && v[0] != '[') {
					// Single-key element with a scalar value isn't a wrapper.
					return data, nil
				}
				if commonKey == "" {
					commonKey = k
				} else if k != commonKey {
					return data, nil
				}
			}
		}
		out := make([]json.RawMessage, 0, len(arr))
		for _, item := range arr {
			out = append(out, item[commonKey])
		}
		return json.Marshal(out)
	case '{':
		var top map[string]json.RawMessage
		if err := json.Unmarshal(data, &top); err != nil {
			return data, err
		}
		if _, hasID := top["id"]; hasID {
			return data, nil
		}
		if len(top) != 1 {
			return data, nil
		}
		for _, v := range top {
			if len(v) > 0 && (v[0] == '{' || v[0] == '[') {
				return v, nil
			}
		}
	}
	return data, nil
}

// shouldRetry decides whether a response status warrants another attempt.
// 401 may flap on Hetzner, but invalid credentials should not loop forever,
// so it is retried at most once. 429 honors Retry-After. Both are safe to
// retry for any method because the API did not execute the request. 5xx
// retries with linear backoff, but only for idempotent methods (GET, DELETE,
// PUT): a 5xx response for a POST does not tell us whether the request was
// executed before failing, so retrying it risks duplicating side effects.
func shouldRetry(method string, statusCode, attempt int) bool {
	switch {
	case statusCode == http.StatusUnauthorized:
		return attempt == 0
	case statusCode == http.StatusTooManyRequests:
		return true
	case statusCode >= 500:
		return method != http.MethodPost
	}
	return false
}

// doRequest executes an HTTP request with authentication and automatic
// retry for transient errors. POST requests are never retried on 5xx
// responses or transport errors (including timeouts), since the server may
// have already executed the request before failing and POST is not
// idempotent; 429 and a single 401 retry remain safe for POST because the
// API did not execute the request in those cases.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := c.baseURL + path

	// Always read body bytes to support retries (body reader can only be read once)
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, NewNetworkError("failed to read request body", err)
		}
	}

	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, NewNetworkError("request cancelled", ctx.Err())
		default:
		}

		var bodyReader io.Reader
		if len(bodyBytes) > 0 {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return nil, NewNetworkError("failed to create request", err)
		}

		req.SetBasicAuth(c.username, c.password)
		req.Header.Set("User-Agent", c.userAgent)
		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		c.logger.LogAttrs(ctx, slog.LevelDebug, "hrobot request",
			slog.String("method", method),
			slog.String("url", reqURL),
			slog.Int("attempt", attempt+1),
			slog.Int("max_retries", maxRetries),
			slog.Int("body_bytes", len(bodyBytes)),
		)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = NewNetworkError("request failed", err)
			if method != http.MethodPost && attempt < maxRetries-1 {
				if err := sleepCtx(ctx, time.Duration(attempt+1)*500*time.Millisecond); err != nil {
					return nil, err
				}
				continue
			}
			return nil, lastErr
		}

		c.updateRateLimit(resp.Header)

		if shouldRetry(method, resp.StatusCode, attempt) && attempt < maxRetries-1 {
			delay := retryAfter(resp.Header)
			if delay == 0 {
				delay = time.Duration(attempt+1) * 500 * time.Millisecond
			}
			_ = resp.Body.Close()
			c.logger.LogAttrs(ctx, slog.LevelDebug, "hrobot retry",
				slog.Int("status", resp.StatusCode),
				slog.Duration("delay", delay),
				slog.Int("attempt", attempt+1),
				slog.Int("max_retries", maxRetries),
			)
			if err := sleepCtx(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

// sleepCtx sleeps for d, returning early if the context is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return NewNetworkError("request cancelled", ctx.Err())
	case <-t.C:
		return nil
	}
}

// errorFromResponse builds an *Error from a non-2xx response body.
func errorFromResponse(statusCode int, body []byte) error {
	var apiErr APIErrorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return newAPIErrorWithStatus(ErrUnknown, fmt.Sprintf("HTTP %d: %s", statusCode, body), statusCode)
	}
	status := apiErr.Error.Status
	if status == 0 {
		status = statusCode
	}
	return newAPIErrorWithStatus(apiErr.Error.Code, apiErr.Error.Message, status)
}

// handleResponse processes the HTTP response and handles errors.
func (c *Client) handleResponse(ctx context.Context, resp *http.Response, v any) error {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewNetworkError("failed to read response body", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelDebug, "hrobot response",
		slog.Int("status", resp.StatusCode),
		slog.Int("body_bytes", len(body)),
	)

	if resp.StatusCode >= 400 {
		return errorFromResponse(resp.StatusCode, body)
	}

	if resp.StatusCode == http.StatusNoContent || len(body) == 0 {
		return nil
	}

	unwrapped, err := unwrapResponse(body)
	if err != nil {
		return NewParseError("failed to unwrap response", err)
	}

	if v != nil {
		if err := json.Unmarshal(unwrapped, v); err != nil {
			return NewParseError("failed to unmarshal response", err)
		}
	}

	return nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, v any) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, v)
}

// Post performs a POST request with form data.
func (c *Client) Post(ctx context.Context, path string, data url.Values, v any) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, v)
}

// PostRaw performs a POST request with pre-encoded form data string.
// This is useful when the API expects literal brackets in form keys (not URL-encoded).
func (c *Client) PostRaw(ctx context.Context, path string, data string, v any) error {
	var body io.Reader
	if data != "" {
		body = strings.NewReader(data)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, v)
}

// Put performs a PUT request with form data.
func (c *Client) Put(ctx context.Context, path string, data url.Values, v any) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, v)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, nil)
}

// DeleteWithBody performs a DELETE request with form data.
// This is used for APIs that require a DELETE request with a body, like vSwitch cancellation.
func (c *Client) DeleteWithBody(ctx context.Context, path string, data url.Values, v any) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	resp, err := c.doRequest(ctx, http.MethodDelete, path, body)
	if err != nil {
		return err
	}
	return c.handleResponse(ctx, resp, v)
}

// GetWrappedList performs a GET for an array response whose elements are
// wrapped, e.g. `[{"server": {...}}, ...]`.
//
// Deprecated: callers can now pass a plain slice to Get; the wrapper key is
// auto-detected. This method is retained for source compatibility and
// ignores wrapperKey.
func (c *Client) GetWrappedList(ctx context.Context, path string, _ string, v any) error {
	return c.Get(ctx, path, v)
}

// PostJSON performs a POST request with JSON body.
func (c *Client) PostJSON(ctx context.Context, path string, body any, v any) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return NewParseError("failed to marshal request body", err)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return NewNetworkError("failed to create request", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return NewNetworkError("request failed", err)
	}

	c.updateRateLimit(resp.Header)
	return c.handleResponse(ctx, resp, v)
}
