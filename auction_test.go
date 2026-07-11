package hrobot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuctionService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/server_market/product" {
			t.Errorf("expected path '/order/server_market/product', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := []map[string]any{
			{
				"product": map[string]any{
					"id":    42,
					"name":  "Auction Server #42",
					"price": "39.00",
					"cpu":   "Intel Xeon E3-1245",
				},
			},
			{
				"product": map[string]any{
					"id":    99,
					"name":  "Auction Server #99",
					"price": "49.00",
					"cpu":   "AMD Ryzen 5",
				},
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	auctions, err := client.Auction.List(ctx)
	if err != nil {
		t.Fatalf("Auction.List returned error: %v", err)
	}

	if len(auctions) != 2 {
		t.Fatalf("expected 2 auctions, got %d", len(auctions))
	}

	if auctions[0].ID != 42 {
		t.Errorf("expected id 42, got %d", auctions[0].ID)
	}

	if auctions[0].Name != "Auction Server #42" {
		t.Errorf("expected name 'Auction Server #42', got '%s'", auctions[0].Name)
	}

	if float64(auctions[0].Price) != 39.00 {
		t.Errorf("expected price 39.00, got %v", auctions[0].Price)
	}

	if auctions[1].CPU != "AMD Ryzen 5" {
		t.Errorf("expected cpu 'AMD Ryzen 5', got '%s'", auctions[1].CPU)
	}
}

func TestAuctionService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Specifically verify that id=42 produces the path "/order/server_market/product/42"
		// and not a single-rune ("*") path corruption.
		if r.URL.Path != "/order/server_market/product/42" {
			t.Errorf("expected path '/order/server_market/product/42', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		response := map[string]any{
			"id":    42,
			"name":  "Auction Server #42",
			"price": "39.00",
			"cpu":   "Intel Xeon E3-1245",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	auction, err := client.Auction.Get(ctx, 42)
	if err != nil {
		t.Fatalf("Auction.Get returned error: %v", err)
	}

	if auction.ID != 42 {
		t.Errorf("expected id 42, got %d", auction.ID)
	}

	if auction.Name != "Auction Server #42" {
		t.Errorf("expected name 'Auction Server #42', got '%s'", auction.Name)
	}

	if auction.CPU != "Intel Xeon E3-1245" {
		t.Errorf("expected cpu 'Intel Xeon E3-1245', got '%s'", auction.CPU)
	}
}

// auctionOrderableAddonsJSON is a doc-verbatim orderable_addons[].prices
// fragment for GET /order/server_market/product(/{id}).
const auctionOrderableAddonsJSON = `[
	{
		"id": "primary_ipv4",
		"name": "Primary IPv4",
		"location": null,
		"min": 0,
		"max": 1,
		"prices": [
			{
				"location": "FSN1",
				"price": { "net": "1.7000", "gross": "1.7000", "hourly_net": "0.0027", "hourly_gross": "0.0027" },
				"price_setup": { "net": "0.0000", "gross": "0.0000" }
			}
		]
	}
]`

func TestAuctionService_Get_DecodesOrderableAddonPrices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/server_market/product/42" {
			t.Errorf("expected path '/order/server_market/product/42', got '%s'", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 42,
			"name": "Auction Server #42",
			"price": "39.00",
			"cpu": "Intel Xeon E3-1245",
			"orderable_addons": ` + auctionOrderableAddonsJSON + `
		}`))
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	auction, err := client.Auction.Get(ctx, 42)
	if err != nil {
		t.Fatalf("Auction.Get returned error: %v", err)
	}

	if len(auction.OrderableAddons) != 1 {
		t.Fatalf("expected 1 orderable addon, got %d", len(auction.OrderableAddons))
	}

	addon := auction.OrderableAddons[0]
	if len(addon.Prices) != 1 {
		t.Fatalf("expected 1 price entry, got %d", len(addon.Prices))
	}

	price := addon.Prices[0]
	if price.Location != "FSN1" {
		t.Errorf("expected location 'FSN1', got '%s'", price.Location)
	}
	if price.Price.Net.Float64() != 1.7 {
		t.Errorf("expected Price.Net 1.7, got %v", price.Price.Net.Float64())
	}
	if price.Price.Gross.Float64() != 1.7 {
		t.Errorf("expected Price.Gross 1.7, got %v", price.Price.Gross.Float64())
	}
	if price.Price.HourlyNet.Float64() != 0.0027 {
		t.Errorf("expected Price.HourlyNet 0.0027, got %v", price.Price.HourlyNet.Float64())
	}
	if price.PriceSetup.Net.Float64() != 0 {
		t.Errorf("expected PriceSetup.Net 0, got %v", price.PriceSetup.Net.Float64())
	}
}

func TestAuctionService_List_DecodesOrderableAddonPrices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"product": {
					"id": 42,
					"name": "Auction Server #42",
					"price": "39.00",
					"cpu": "Intel Xeon E3-1245",
					"orderable_addons": ` + auctionOrderableAddonsJSON + `
				}
			}
		]`))
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	auctions, err := client.Auction.List(ctx)
	if err != nil {
		t.Fatalf("Auction.List returned error: %v", err)
	}
	if len(auctions) != 1 {
		t.Fatalf("expected 1 auction, got %d", len(auctions))
	}

	if len(auctions[0].OrderableAddons) != 1 || len(auctions[0].OrderableAddons[0].Prices) != 1 {
		t.Fatalf("expected 1 orderable addon with 1 price, got %+v", auctions[0].OrderableAddons)
	}
	if auctions[0].OrderableAddons[0].Prices[0].Price.Net.Float64() != 1.7 {
		t.Errorf("expected Price.Net 1.7, got %v", auctions[0].OrderableAddons[0].Prices[0].Price.Net.Float64())
	}
}
