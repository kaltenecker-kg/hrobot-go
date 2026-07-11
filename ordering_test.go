package hrobot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

// marketTransactionJSON is a doc-verbatim GET /order/server_market/transaction
// payload: the product id is a JSON number and "arch" is a string.
const marketTransactionJSON = `{
	"server_market_transaction": {
		"id": "B20150121-345678",
		"date": "2015-01-21 15:57:31",
		"status": "ready",
		"server_number": 2417234,
		"server_ip": "123.123.123.123",
		"authorized_key": [
			{
				"key": {
					"name": "key1",
					"fingerprint": "15:28:...",
					"type": "ED25519",
					"size": 256
				}
			}
		],
		"host_key": [
			{
				"key": {
					"name": "host1",
					"fingerprint": "aa:bb:...",
					"type": "ED25519",
					"size": 256
				}
			}
		],
		"comment": null,
		"product": {
			"id": 283693,
			"name": "SB110",
			"traffic": "20 TB",
			"dist": "Rescue system",
			"arch": "64",
			"lang": "en"
		},
		"addons": []
	}
}`

// serverTransactionJSON is a doc-verbatim GET /order/server/transaction
// payload: the product id is a JSON string (e.g. "EX40").
const serverTransactionJSON = `{
	"transaction": {
		"id": "B20150121-345678",
		"date": "2015-01-21 15:57:31",
		"status": "ready",
		"server_number": 2417234,
		"server_ip": "123.123.123.123",
		"authorized_key": [
			{
				"key": {
					"name": "key1",
					"fingerprint": "15:28:...",
					"type": "ED25519",
					"size": 256
				}
			}
		],
		"host_key": [
			{
				"key": {
					"name": "host1",
					"fingerprint": "aa:bb:...",
					"type": "ED25519",
					"size": 256
				}
			}
		],
		"comment": null,
		"product": {
			"id": "EX40",
			"name": "EX40",
			"traffic": "unlimited",
			"dist": "Rescue system",
			"arch": "64",
			"lang": "en"
		},
		"addons": []
	}
}`

func newTestOrderingClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient("test-user", "test-pass", WithBaseURL(server.URL))
}

func assertTransactionCommon(t *testing.T, tx *MarketTransaction) {
	t.Helper()
	if tx.ID != "B20150121-345678" {
		t.Errorf("ID = %q, want %q", tx.ID, "B20150121-345678")
	}
	if tx.Status != "ready" {
		t.Errorf("Status = %q, want %q", tx.Status, "ready")
	}
	if len(tx.AuthorizedKey) != 1 || tx.AuthorizedKey[0].Key.Fingerprint != "15:28:..." {
		t.Errorf("AuthorizedKey = %+v, want a single key with fingerprint 15:28:...", tx.AuthorizedKey)
	}
	if len(tx.HostKey) != 1 || tx.HostKey[0].Key.Fingerprint != "aa:bb:..." {
		t.Errorf("HostKey = %+v, want a single key with fingerprint aa:bb:...", tx.HostKey)
	}
	if tx.Product.Arch != "64" {
		t.Errorf("Product.Arch = %q, want %q", tx.Product.Arch, "64")
	}
}

func TestOrderingService_GetMarketTransaction_DecodesDocVerbatimPayload(t *testing.T) {
	client := newTestOrderingClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/server_market/transaction/B20150121-345678" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(marketTransactionJSON))
	})

	tx, err := client.Ordering.GetMarketTransaction(context.Background(), "B20150121-345678")
	if err != nil {
		t.Fatalf("GetMarketTransaction() error = %v", err)
	}

	assertTransactionCommon(t, tx)
	if tx.Product.ID != FlexibleID("283693") {
		t.Errorf("Product.ID = %q, want %q", tx.Product.ID, "283693")
	}
}

func TestOrderingService_ListMarketTransactions_DecodesDocVerbatimPayload(t *testing.T) {
	client := newTestOrderingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[" + strings.TrimSpace(marketTransactionJSON) + "]"))
	})

	txs, err := client.Ordering.ListMarketTransactions(context.Background())
	if err != nil {
		t.Fatalf("ListMarketTransactions() error = %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("len(txs) = %d, want 1", len(txs))
	}

	assertTransactionCommon(t, &txs[0])
	if txs[0].Product.ID != FlexibleID("283693") {
		t.Errorf("Product.ID = %q, want %q", txs[0].Product.ID, "283693")
	}
}

func TestOrderingService_GetTransaction_DecodesDocVerbatimPayload(t *testing.T) {
	client := newTestOrderingClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/server/transaction/B20150121-345678" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(serverTransactionJSON))
	})

	tx, err := client.Ordering.GetTransaction(context.Background(), "B20150121-345678")
	if err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}

	assertTransactionCommon(t, tx)
	if tx.Product.ID != FlexibleID("EX40") {
		t.Errorf("Product.ID = %q, want %q", tx.Product.ID, "EX40")
	}
}

func TestOrderingService_ListTransactions_DecodesDocVerbatimPayload(t *testing.T) {
	client := newTestOrderingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[" + strings.TrimSpace(serverTransactionJSON) + "]"))
	})

	txs, err := client.Ordering.ListTransactions(context.Background())
	if err != nil {
		t.Fatalf("ListTransactions() error = %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("len(txs) = %d, want 1", len(txs))
	}

	assertTransactionCommon(t, &txs[0])
	if txs[0].Product.ID != FlexibleID("EX40") {
		t.Errorf("Product.ID = %q, want %q", txs[0].Product.ID, "EX40")
	}
}

func TestOrderingService_WaitForMarketTransactionCompletion_ZeroIntervalDoesNotPanic(t *testing.T) {
	client := newTestOrderingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(marketTransactionJSON))
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		tx, err := client.Ordering.WaitForMarketTransactionCompletion(context.Background(), "B20150121-345678", 0)
		if err != nil {
			t.Errorf("WaitForMarketTransactionCompletion() error = %v", err)
			return
		}
		if tx.Status != "ready" {
			t.Errorf("Status = %q, want %q", tx.Status, "ready")
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForMarketTransactionCompletion did not return in time")
	}
}

func TestOrderingService_WaitForMarketTransactionCompletion_ErrorIncludesTransactionID(t *testing.T) {
	const errorTransactionJSON = `{
		"server_market_transaction": {
			"id": "B20150121-999999",
			"date": "2015-01-21 15:57:31",
			"status": "error",
			"server_number": null,
			"server_ip": null,
			"authorized_key": [],
			"host_key": [],
			"comment": null,
			"product": {
				"id": 283693,
				"name": "SB110",
				"traffic": "20 TB",
				"dist": "Rescue system",
				"arch": "64",
				"lang": "en"
			},
			"addons": []
		}
	}`

	client := newTestOrderingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(errorTransactionJSON))
	})

	_, err := client.Ordering.WaitForMarketTransactionCompletion(context.Background(), "B20150121-999999", time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "B20150121-999999") {
		t.Errorf("error %q does not contain transaction ID", err.Error())
	}
}
