package hrobot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrderingService_PolicyShortCircuit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("ordering Place* methods must not perform an HTTP call; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	cases := []struct {
		name string
		call func() error
	}{
		{"PlaceMarketOrder", func() error {
			_, err := client.Ordering.PlaceMarketOrder(ctx, MarketProductOrder{ProductID: 1})
			return err
		}},
		{"PlaceProductOrder", func() error {
			_, err := client.Ordering.PlaceProductOrder(ctx, ProductOrder{ProductID: "1"})
			return err
		}},
		{"PlaceAddonOrder", func() error {
			_, err := client.Ordering.PlaceAddonOrder(ctx, AddonOrder{ProductID: "1", ServerNumber: 321})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if !IsPolicyError(err) {
				t.Fatalf("expected policy error, got %v", err)
			}
			var e *Error
			if !errors.As(err, &e) || e.Status != 451 {
				t.Fatalf("expected status 451, got %v", err)
			}
		})
	}
}
