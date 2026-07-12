package hrobot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// auctionListJSON is a doc-verbatim (abridged to one product) GET
// /order/server_market/product response body.
const auctionListJSON = `[
	{
		"product": {
			"id": 282323,
			"name": "SB109",
			"description": [
				"Intel Core i7 980x",
				"6x RAM 4096 MB DDR3",
				"2x SSD 120 GB SATA",
				"NIC 1000Mbit PCI - Intel Pro1000GT PWLA8391GT",
				"RAID Controller 4-Port SATA PCI-E - Adaptec 5405"
			],
			"traffic": "20 TB",
			"dist": [
				"Rescue system"
			],
			"arch": [
				64
			],
			"lang": [
				"en"
			],
			"cpu": "Intel Core i7 980x",
			"cpu_benchmark": 8944,
			"memory_size": 24,
			"hdd_size": 120,
			"hdd_text": "ESAS HWR",
			"hdd_count": 2,
			"datacenter": "FSN1-DC4",
			"network_speed": "200 Mbit/s",
			"price": "91.6000",
			"price_hourly": "0.1468",
			"price_setup": "0.0000",
			"price_vat": "91.6000",
			"price_hourly_vat": "0.1468",
			"price_setup_vat": "0.0000",
			"fixed_price": false,
			"next_reduce": -10800,
			"next_reduce_date": "2018-05-01 12:22:00",
			"orderable_addons": [
				{
					"id": "primary_ipv4",
					"name": "Primary IPv4",
					"min": 0,
					"max": 1,
					"prices": [
						{
							"location": "FSN1",
							"price": { "net": "1.7000", "gross": "1.7000", "hourly_net": "0.0027", "hourly_gross": "0.0027" },
							"price_setup": { "net": "0.0000", "gross": "0.0000" }
						},
						{
							"location": "NBG1",
							"price": { "net": "1.7000", "gross": "1.7000", "hourly_net": "0.0027", "hourly_gross": "0.0027" },
							"price_setup": { "net": "0.0000", "gross": "0.0000" }
						}
					]
				}
			]
		}
	}
]`

// auctionGetJSON is a doc-verbatim GET /order/server_market/product/{product-id}
// response body: the product is wrapped in a "product" envelope.
const auctionGetJSON = `{
	"product": {
		"id": 282323,
		"name": "SB109",
		"description": [
			"Intel Core i7 980x",
			"6x RAM 4096 MB DDR3",
			"2x SSD 120 GB SATA",
			"NIC 1000Mbit PCI - Intel Pro1000GT PWLA8391GT",
			"RAID Controller 4-Port SATA PCI-E - Adaptec 5405"
		],
		"traffic": "20 TB",
		"dist": [
			"Rescue system"
		],
		"arch": [
			64
		],
		"lang": [
			"en"
		],
		"cpu": "Intel Core i7 980x",
		"cpu_benchmark": 8944,
		"memory_size": 24,
		"hdd_size": 120,
		"hdd_text": "ENT.HDD ECC INIC",
		"hdd_count": 2,
		"datacenter": "FSN1-DC4",
		"network_speed": "100 Mbit/s",
		"price": "91.6000",
		"price_hourly": "0.1468",
		"price_setup": "0.0000",
		"price_vat": "91.6000",
		"price_hourly_vat": "0.1468",
		"price_setup_vat": "0.0000",
		"fixed_price": false,
		"next_reduce": -10800,
		"next_reduce_date": "2018-05-01 12:22:00",
		"orderable_addons": [
			{
				"id": "primary_ipv4",
				"name": "Primary IPv4",
				"min": 0,
				"max": 1,
				"prices": [
					{
						"location": "FSN1",
						"price": { "net": "1.7000", "gross": "1.7000", "hourly_net": "0.0027", "hourly_gross": "0.0027" },
						"price_setup": { "net": "0.0000", "gross": "0.0000" }
					},
					{
						"location": "NBG1",
						"price": { "net": "1.7000", "gross": "1.7000", "hourly_net": "0.0027", "hourly_gross": "0.0027" },
						"price_setup": { "net": "0.0000", "gross": "0.0000" }
					}
				]
			}
		]
	}
}`

func TestAuctionService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/server_market/product" {
			t.Errorf("expected path '/order/server_market/product', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(auctionListJSON))
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

	if auctions[0].ID != 282323 {
		t.Errorf("expected id 282323, got %d", auctions[0].ID)
	}

	if auctions[0].Name != "SB109" {
		t.Errorf("expected name 'SB109', got '%s'", auctions[0].Name)
	}

	if float64(auctions[0].Price) != 91.6 {
		t.Errorf("expected price 91.6, got %v", auctions[0].Price)
	}

	if auctions[0].CPU != "Intel Core i7 980x" {
		t.Errorf("expected cpu 'Intel Core i7 980x', got '%s'", auctions[0].CPU)
	}
}

func TestAuctionService_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Specifically verify that id=282323 produces the path "/order/server_market/product/282323"
		// and not a single-rune ("*") path corruption.
		if r.URL.Path != "/order/server_market/product/282323" {
			t.Errorf("expected path '/order/server_market/product/282323', got '%s'", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET request, got '%s'", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(auctionGetJSON))
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	auction, err := client.Auction.Get(ctx, 282323)
	if err != nil {
		t.Fatalf("Auction.Get returned error: %v", err)
	}

	if auction.ID != 282323 {
		t.Errorf("expected id 282323, got %d", auction.ID)
	}

	if auction.Name != "SB109" {
		t.Errorf("expected name 'SB109', got '%s'", auction.Name)
	}

	if auction.CPU != "Intel Core i7 980x" {
		t.Errorf("expected cpu 'Intel Core i7 980x', got '%s'", auction.CPU)
	}
}

// auctionOrderableAddonsJSON is a doc-verbatim orderable_addons[].prices
// fragment for GET /order/server_market/product(/{id}).
const auctionOrderableAddonsJSON = `[
	{
		"id": "primary_ipv4",
		"name": "Primary IPv4",
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
			"product": {
				"id": 42,
				"name": "Auction Server #42",
				"price": "39.00",
				"cpu": "Intel Xeon E3-1245",
				"arch": [64],
				"network_speed": "100 Mbit/s",
				"orderable_addons": ` + auctionOrderableAddonsJSON + `
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-user", "test-pass", WithBaseURL(server.URL))
	ctx := context.Background()

	auction, err := client.Auction.Get(ctx, 42)
	if err != nil {
		t.Fatalf("Auction.Get returned error: %v", err)
	}

	if len(auction.Arch) != 1 || auction.Arch[0] != 64 {
		t.Errorf("expected arch [64], got %v", auction.Arch)
	}

	if auction.NetworkSpeed != "100 Mbit/s" {
		t.Errorf("expected network_speed '100 Mbit/s', got '%s'", auction.NetworkSpeed)
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
