package hrobot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubnetService_Cancel_DisallowedByPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("Cancel must not perform an HTTP call; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))

	_, err := client.Subnet.Cancel(context.Background(), "2001:db8::", "2024-12-31")
	if !IsPolicyError(err) {
		t.Fatalf("expected policy error, got %v", err)
	}
	var e *Error
	if !errors.As(err, &e) || e.Status != 451 {
		t.Fatalf("expected status 451, got %v", err)
	}
}
