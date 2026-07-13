package hrobot

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorCreation(t *testing.T) {
	t.Run("NewAPIError", func(t *testing.T) {
		err := NewAPIError(ErrServerNotFound, "server 123 not found")
		if err == nil {
			t.Fatal("expected error, got nil")
			return
		}

		if err.Kind != ErrKindAPI {
			t.Errorf("Kind = %s, want %s", err.Kind, ErrKindAPI)
		}
		if err.Code != ErrServerNotFound {
			t.Errorf("Code = %s, want %s", err.Code, ErrServerNotFound)
		}
		if err.Message != "server 123 not found" {
			t.Errorf("Message = %s, want 'server 123 not found'", err.Message)
		}
	})

	t.Run("NewNetworkError", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := NewNetworkError("failed to connect", cause)
		if err == nil {
			t.Fatal("expected error, got nil")
			return
		}

		if err.Kind != ErrKindNetwork {
			t.Errorf("Kind = %s, want %s", err.Kind, ErrKindNetwork)
		}
		if err.Message != "failed to connect" {
			t.Errorf("Message = %s, want 'failed to connect'", err.Message)
		}
		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}
	})

	t.Run("NewParseError", func(t *testing.T) {
		cause := errors.New("invalid JSON")
		err := NewParseError("failed to parse response", cause)
		if err == nil {
			t.Fatal("expected error, got nil")
			return
		}

		if err.Kind != ErrKindParse {
			t.Errorf("Kind = %s, want %s", err.Kind, ErrKindParse)
		}
		if err.Message != "failed to parse response" {
			t.Errorf("Message = %s, want 'failed to parse response'", err.Message)
		}
		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}
	})

	t.Run("NewAuthError", func(t *testing.T) {
		err := NewAuthError("invalid credentials")
		if err == nil {
			t.Fatal("expected error, got nil")
			return
		}

		if err.Kind != ErrKindAuth {
			t.Errorf("Kind = %s, want %s", err.Kind, ErrKindAuth)
		}
		if err.Message != "invalid credentials" {
			t.Errorf("Message = %s, want 'invalid credentials'", err.Message)
		}
	})
}

func TestErrorString(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "API error with code",
			err: &Error{
				Kind:    ErrKindAPI,
				Code:    ErrServerNotFound,
				Message: "server not found",
			},
			want: "API[SERVER_NOT_FOUND]: server not found",
		},
		{
			name: "Network error with cause",
			err: &Error{
				Kind:    ErrKindNetwork,
				Message: "connection failed",
				Cause:   errors.New("timeout"),
			},
			want: "Network: connection failed: timeout",
		},
		{
			name: "Parse error",
			err: &Error{
				Kind:    ErrKindParse,
				Message: "invalid JSON",
			},
			want: "Parse: invalid JSON",
		},
		{
			name: "Auth error",
			err: &Error{
				Kind:    ErrKindAuth,
				Message: "unauthorized",
			},
			want: "Auth: unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("original error")
	err := NewNetworkError("wrapper", cause)

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestIsAPIError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code ErrorCode
		want bool
	}{
		{
			name: "matching API error",
			err:  NewAPIError(ErrServerNotFound, "not found"),
			code: ErrServerNotFound,
			want: true,
		},
		{
			name: "non-matching API error",
			err:  NewAPIError(ErrServerNotFound, "not found"),
			code: ErrIPNotFound,
			want: false,
		},
		{
			name: "Network error",
			err:  NewNetworkError("failed", nil),
			code: ErrServerNotFound,
			want: false,
		},
		{
			name: "Standard error",
			err:  errors.New("some error"),
			code: ErrServerNotFound,
			want: false,
		},
		{
			name: "Nil error",
			err:  nil,
			code: ErrServerNotFound,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAPIError(tt.err, tt.code)
			if got != tt.want {
				t.Errorf("IsAPIError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Rate limit error",
			err:  NewAPIError(ErrRateLimitExceeded, "too many requests"),
			want: true,
		},
		{
			name: "Other API error",
			err:  NewAPIError(ErrServerNotFound, "not found"),
			want: false,
		},
		{
			name: "Network error",
			err:  NewNetworkError("failed", nil),
			want: false,
		},
		{
			name: "Nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitError(tt.err)
			if got != tt.want {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Server not found",
			err:  NewAPIError(ErrServerNotFound, "not found"),
			want: true,
		},
		{
			name: "IP not found",
			err:  NewAPIError(ErrIPNotFound, "ip not found"),
			want: true,
		},
		{
			name: "Other API error",
			err:  NewAPIError(ErrInvalidInput, "invalid"),
			want: false,
		},
		{
			name: "Network error",
			err:  NewNetworkError("failed", nil),
			want: false,
		},
		{
			name: "Nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFoundError(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that all error codes are defined
	codes := []ErrorCode{
		ErrUnknown,
		ErrInvalidInput,
		ErrServerNotFound,
		ErrIPNotFound,
		ErrIPLocked,
		ErrFirewallInProcess,
		ErrFirewallAlreadyActive,
		ErrFirewallConfigInvalid,
		ErrRescueAlreadyActive,
		ErrVNCNotAvailable,
		ErrResetNotAvailable,
		ErrReverseDNSInvalid,
		ErrReverseDNSNotFound,
		ErrRateLimitExceeded,
	}

	for _, code := range codes {
		if string(code) == "" {
			t.Errorf("Error code %v is empty", code)
		}
	}
}

func TestErrorKinds(t *testing.T) {
	// Test that all error kinds are defined
	kinds := []ErrorKind{
		ErrKindAPI,
		ErrKindNetwork,
		ErrKindParse,
		ErrKindAuth,
	}

	for _, kind := range kinds {
		if string(kind) == "" {
			t.Errorf("Error kind %v is empty", kind)
		}
	}
}

func TestErrorFromResponse(t *testing.T) {
	t.Run("well-formed error body", func(t *testing.T) {
		body := []byte(`{"error":{"status":404,"code":"SERVER_NOT_FOUND","message":"server not found"}}`)
		err := errorFromResponse(404, body)

		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if e.Code != ErrServerNotFound {
			t.Errorf("Code = %q, want %q", e.Code, ErrServerNotFound)
		}
		if e.Status != 404 {
			t.Errorf("Status = %d, want 404", e.Status)
		}
		if e.Message != "server not found" {
			t.Errorf("Message = %q, want %q", e.Message, "server not found")
		}
	})

	t.Run("body status zero falls back to http status", func(t *testing.T) {
		body := []byte(`{"error":{"code":"INVALID_INPUT","message":"bad"}}`)
		err := errorFromResponse(400, body)

		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if e.Status != 400 {
			t.Errorf("Status = %d, want 400 (fallback to http status)", e.Status)
		}
	})

	t.Run("unparseable body falls back to raw", func(t *testing.T) {
		body := []byte(`<html>502 Bad Gateway</html>`)
		err := errorFromResponse(502, body)

		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if e.Kind != ErrKindAPI {
			t.Errorf("Kind = %q, want %q", e.Kind, ErrKindAPI)
		}
		if e.Code != ErrUnknown {
			t.Errorf("Code = %q, want %q", e.Code, ErrUnknown)
		}
		if e.Status != 502 {
			t.Errorf("Status = %d, want 502", e.Status)
		}
		if !strings.Contains(e.Message, "HTTP 502") || !strings.Contains(e.Message, "Bad Gateway") {
			t.Errorf("Message = %q, want it to include the raw body and status", e.Message)
		}
	})
}
