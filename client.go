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

const (
	DefaultBaseURL = "https://robot-ws.your-server.de"
	DefaultTimeout = 30 * time.Second
	UserAgent      = "hrobot-go/1.0.0"
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
// silence (the default).
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

// responseWrapper unwraps Hetzner's `{"<resource>": ...}` envelopes for
// callers that pass a raw value (rather than their own wrapper struct) to
// client.Get/Post/Put. Each new endpoint either adds its key here or — the
// preferred newer pattern — defines a private wrapper struct in its own
// service file (see subnet.go and storagebox.go for examples).
//
// CAUTION: a JSON key listed here will be silently extracted from any
// response body, so do not introduce a wrapper key whose name collides with
// a real top-level field used by an unwrapped endpoint.
type responseWrapper struct {
	Server                  json.RawMessage `json:"server,omitempty"`
	Servers                 json.RawMessage `json:"servers,omitempty"`
	Firewall                json.RawMessage `json:"firewall,omitempty"`
	IP                      json.RawMessage `json:"ip,omitempty"`
	Reset                   json.RawMessage `json:"reset,omitempty"`
	Boot                    json.RawMessage `json:"boot,omitempty"`
	Rescue                  json.RawMessage `json:"rescue,omitempty"`
	Key                     json.RawMessage `json:"key,omitempty"`
	VSwitch                 json.RawMessage `json:"vswitch,omitempty"`
	RDNS                    json.RawMessage `json:"rdns,omitempty"`
	Failover                json.RawMessage `json:"failover,omitempty"`
	Traffic                 json.RawMessage `json:"traffic,omitempty"`
	ServerMarketProduct     json.RawMessage `json:"server_market_product,omitempty"`
	ServerMarketTransaction json.RawMessage `json:"server_market_transaction,omitempty"`
	ServerAddonTransaction  json.RawMessage `json:"server_addon_transaction,omitempty"`
	ServerAddonProduct      json.RawMessage `json:"server_addon_product,omitempty"`
	Transaction             json.RawMessage `json:"transaction,omitempty"`
}

// unwrapArrayResponse handles arrays where each item is wrapped in an object
// e.g. [{"server": {...}}, {"server": {...}}].
func unwrapArrayResponse(data []byte, wrapperKey string) (json.RawMessage, error) {
	var wrappers []map[string]json.RawMessage
	if err := json.Unmarshal(data, &wrappers); err != nil {
		return nil, err
	}

	result := make([]json.RawMessage, 0, len(wrappers))
	for _, wrapper := range wrappers {
		if item, ok := wrapper[wrapperKey]; ok {
			result = append(result, item)
		}
	}

	return json.Marshal(result)
}

// unwrapResponse extracts the actual data from Hetzner's wrapped response.
func unwrapResponse(data []byte) (json.RawMessage, error) {
	if len(data) > 0 && data[0] == '[' {
		// Try to unwrap array elements that might be wrapped in objects
		// e.g., [{"vswitch": {...}}] -> {...}
		var arrayOfWrappers []responseWrapper
		if err := json.Unmarshal(data, &arrayOfWrappers); err == nil && len(arrayOfWrappers) > 0 {
			wrapper := arrayOfWrappers[0]
			if len(wrapper.VSwitch) > 0 {
				return wrapper.VSwitch, nil
			}
			if len(wrapper.Server) > 0 {
				return wrapper.Server, nil
			}
			if len(wrapper.Firewall) > 0 {
				return wrapper.Firewall, nil
			}
			if len(wrapper.Key) > 0 {
				return wrapper.Key, nil
			}
			if len(wrapper.Failover) > 0 {
				return wrapper.Failover, nil
			}
		}
		return data, nil
	}

	// If the JSON has an "id" key at the top level, treat as already unwrapped.
	var topLevelKeys map[string]json.RawMessage
	if err := json.Unmarshal(data, &topLevelKeys); err == nil {
		if _, hasID := topLevelKeys["id"]; hasID {
			return data, nil
		}
	}

	var wrapper responseWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return data, err
	}

	if len(wrapper.Server) > 0 {
		return wrapper.Server, nil
	}
	if len(wrapper.Servers) > 0 {
		return wrapper.Servers, nil
	}
	if len(wrapper.Firewall) > 0 {
		return wrapper.Firewall, nil
	}
	if len(wrapper.IP) > 0 {
		return wrapper.IP, nil
	}
	if len(wrapper.Reset) > 0 {
		return wrapper.Reset, nil
	}
	if len(wrapper.Boot) > 0 {
		return wrapper.Boot, nil
	}
	if len(wrapper.Rescue) > 0 {
		return wrapper.Rescue, nil
	}
	if len(wrapper.Key) > 0 {
		return wrapper.Key, nil
	}
	if len(wrapper.VSwitch) > 0 {
		return wrapper.VSwitch, nil
	}
	if len(wrapper.RDNS) > 0 {
		return wrapper.RDNS, nil
	}
	if len(wrapper.Failover) > 0 {
		return wrapper.Failover, nil
	}
	if len(wrapper.Traffic) > 0 {
		return wrapper.Traffic, nil
	}
	if len(wrapper.ServerMarketProduct) > 0 {
		return wrapper.ServerMarketProduct, nil
	}
	if len(wrapper.ServerMarketTransaction) > 0 {
		return wrapper.ServerMarketTransaction, nil
	}
	if len(wrapper.ServerAddonTransaction) > 0 {
		return wrapper.ServerAddonTransaction, nil
	}
	if len(wrapper.ServerAddonProduct) > 0 {
		return wrapper.ServerAddonProduct, nil
	}
	if len(wrapper.Transaction) > 0 {
		return wrapper.Transaction, nil
	}

	return data, nil
}

// shouldRetry decides whether a response status warrants another attempt.
// 401 may flap on Hetzner, but invalid credentials should not loop forever,
// so it is retried at most once. 429 honors Retry-After. 5xx retries with
// linear backoff.
func shouldRetry(statusCode, attempt int) bool {
	switch {
	case statusCode == http.StatusUnauthorized:
		return attempt == 0
	case statusCode == http.StatusTooManyRequests:
		return true
	case statusCode >= 500:
		return true
	}
	return false
}

// doRequest executes an HTTP request with authentication and automatic retry for transient errors.
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
			if attempt < maxRetries-1 {
				if err := sleepCtx(ctx, time.Duration(attempt+1)*500*time.Millisecond); err != nil {
					return nil, err
				}
				continue
			}
			return nil, lastErr
		}

		c.updateRateLimit(resp.Header)

		if shouldRetry(resp.StatusCode, attempt) && attempt < maxRetries-1 {
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
func (c *Client) handleResponse(resp *http.Response, v any) error {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewNetworkError("failed to read response body", err)
	}

	c.logger.LogAttrs(context.Background(), slog.LevelDebug, "hrobot response",
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
	return c.handleResponse(resp, v)
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
	return c.handleResponse(resp, v)
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
	return c.handleResponse(resp, v)
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
	return c.handleResponse(resp, v)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
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
	return c.handleResponse(resp, v)
}

// GetWrappedList performs a GET request for array responses where each item is wrapped
// e.g. [{"server": {...}}, {"server": {...}}].
func (c *Client) GetWrappedList(ctx context.Context, path string, wrapperKey string, v any) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewNetworkError("failed to read response body", err)
	}

	if resp.StatusCode >= 400 {
		return errorFromResponse(resp.StatusCode, body)
	}

	unwrapped, err := unwrapArrayResponse(body, wrapperKey)
	if err != nil {
		return NewParseError("failed to unwrap array response", err)
	}

	if v != nil {
		if err := json.Unmarshal(unwrapped, v); err != nil {
			return NewParseError("failed to unmarshal response", err)
		}
	}

	return nil
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
	return c.handleResponse(resp, v)
}
