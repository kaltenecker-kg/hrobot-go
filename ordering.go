package hrobot

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"
)

// OrderingService provides access to server and addon ordering functions in the Hetzner Robot API.
type OrderingService struct {
	client *Client
}

// NewOrderingService creates a new OrderingService.
func NewOrderingService(client *Client) *OrderingService {
	return &OrderingService{client: client}
}

// Product represents a standard product server available for order.
type Product struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     []string       `json:"description"`
	Traffic         string         `json:"traffic"`
	Distributions   []string       `json:"dist"`
	Languages       []string       `json:"lang"`
	Locations       []string       `json:"location"`
	Prices          []ProductPrice `json:"prices"`
	OrderableAddons []ProductAddon `json:"orderable_addons"`
}

// ProductPrice represents location-specific pricing for a product.
type ProductPrice struct {
	Location   string           `json:"location"`
	Price      ProductPriceInfo `json:"price"`
	PriceSetup ProductPriceInfo `json:"price_setup"`
}

// ProductPriceInfo represents price details.
type ProductPriceInfo struct {
	Net         StringFloat `json:"net"`
	Gross       StringFloat `json:"gross"`
	HourlyNet   StringFloat `json:"hourly_net"`
	HourlyGross StringFloat `json:"hourly_gross"`
}

// ProductAddon represents an addon that can be purchased with a product server.
type ProductAddon struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Min    uint32              `json:"min"`
	Max    uint32              `json:"max"`
	Prices []ProductAddonPrice `json:"price"`
}

// ProductAddonPrice represents the price for an addon in a specific location.
type ProductAddonPrice struct {
	Location        string  `json:"location"`
	Price           float64 `json:"price"`
	PriceSetup      float64 `json:"price_setup"`
	PriceMonthly    float64 `json:"price_monthly"`
	PriceMonthlyVAT float64 `json:"price_monthly_vat"`
	PriceSetupVAT   float64 `json:"price_setup_vat"`
}

// AuthorizationMethod specifies how to authorize access to a newly provisioned server.
type AuthorizationMethod struct {
	// SSH key fingerprints (use this OR password, not both)
	Keys []string
	// Root password (use this OR keys, not both)
	Password string
}

// MarketProductOrder represents an order for a server from the auction market.
type MarketProductOrder struct {
	// Auction server ID
	ProductID uint32
	// Authorization method (SSH keys or password)
	Auth AuthorizationMethod
	// Distribution to install (optional)
	Distribution string
	// Language for the distribution (optional)
	Language string
	// Server name (optional)
	ServerName string
	// Comment for the order (optional, requires manual provisioning)
	Comment string
	// Addon IDs to purchase alongside the server
	Addons []string
	// Set to true to actually place the order (false for test mode)
	Test bool
}

// ProductOrder represents an order for a standard product server.
type ProductOrder struct {
	// Product ID (e.g., "AX41-NVMe", "EX40")
	ProductID string
	// Authorization method (SSH keys or password)
	Auth AuthorizationMethod
	// Distribution to install (optional)
	Distribution string
	// Language for the distribution (optional)
	Language string
	// Datacenter location (e.g., "FSN1", "HEL1", "NBG1")
	Location string
	// Server name (optional)
	ServerName string
	// Comment for the order (optional, requires manual provisioning)
	Comment string
	// Addon IDs to purchase alongside the server
	Addons []string
	// Set to true to actually place the order (false for test mode)
	Test bool
}

// AddonOrder represents an order for an addon (e.g., additional IPs or subnets).
type AddonOrder struct {
	// Addon product ID (e.g., "additional_ipv4", "subnet_ipv4", "subnet_ipv6")
	ProductID string
	// Server number to attach the addon to
	ServerNumber int
	// RIPE reason (required for IP/subnet addons)
	Reason string
	// Gateway/routing target for subnets (optional, defaults to server's primary IP)
	Gateway string
	// Set to true to actually place the order (false for test mode)
	Test bool
}

// Transaction represents a purchase transaction.
type Transaction struct {
	ID            string     `json:"id"`
	Date          BerlinTime `json:"date"`
	Status        string     `json:"status"`
	ServerNumber  *int       `json:"server_number"`
	ServerIP      *string    `json:"server_ip"`
	AuthorizedKey []BootKey  `json:"authorized_key"`
	HostKey       []BootKey  `json:"host_key"`
	Comment       *string    `json:"comment"`
}

// MarketTransaction represents a marketplace server purchase transaction.
type MarketTransaction struct {
	Transaction
	Product PurchasedMarketProduct `json:"product"`
	Addons  []string               `json:"addons"`
}

// AddonTransaction represents an addon purchase transaction.
type AddonTransaction struct {
	Transaction
	Product PurchasedAddon `json:"product"`
}

// PurchasedMarketProduct represents a server purchased from the market.
type PurchasedMarketProduct struct {
	ID           FlexibleID `json:"id"` // Numeric for market transactions (e.g. 283693), string like "EX40" for server transactions
	Name         string     `json:"name"`
	Description  []string   `json:"description"`
	Traffic      string     `json:"traffic"`
	Dist         string     `json:"dist"`
	Arch         string     `json:"arch"` // Deprecated upstream; API sends it as a string (e.g. "64")
	Lang         string     `json:"lang"`
	Location     *string    `json:"location"`
	Datacenter   *string    `json:"datacenter"`
	CPU          string     `json:"cpu"`
	CPUBenchmark uint32     `json:"cpu_benchmark"`
	MemorySize   float64    `json:"memory_size"`
	HDDSize      float64    `json:"hdd_size"`
	HDDText      string     `json:"hdd_text"`
	HDDCount     uint8      `json:"hdd_count"`
	NetworkSpeed *string    `json:"network_speed"`
}

// PurchasedAddon represents an addon that was purchased.
type PurchasedAddon struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PlaceMarketOrder places an order for a server from the auction market.
//
// POST /order/server_market/transaction
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-order-server-market-transaction
//
// Disallowed by client policy: this operation is implemented but never
// invoked. Place market orders via the Hetzner Robot UI.
func (o *OrderingService) PlaceMarketOrder(_ context.Context, _ MarketProductOrder) (*MarketTransaction, error) {
	return nil, NewPolicyError("OrderingService.PlaceMarketOrder")
}

// PlaceProductOrder places an order for a standard product server.
//
// POST /order/server/transaction
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-order-server-transaction
//
// Disallowed by client policy: this operation is implemented but never
// invoked. Place product orders via the Hetzner Robot UI.
func (o *OrderingService) PlaceProductOrder(_ context.Context, _ ProductOrder) (*MarketTransaction, error) {
	return nil, NewPolicyError("OrderingService.PlaceProductOrder")
}

// PlaceAddonOrder places an order for an addon (e.g., additional IP addresses or subnets).
//
// POST /order/server_addon/transaction
//
// See: https://robot.hetzner.com/doc/webservice/en.html#post-order-server-addon-transaction
//
// Disallowed by client policy: this operation is implemented but never
// invoked. Place addon orders via the Hetzner Robot UI.
func (o *OrderingService) PlaceAddonOrder(_ context.Context, _ AddonOrder) (*AddonTransaction, error) {
	return nil, NewPolicyError("OrderingService.PlaceAddonOrder")
}

// ListMarketTransactions lists marketplace transaction history from the last 30 days.
//
// GET /order/server_market/transaction
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-order-server-market-transaction
func (o *OrderingService) ListMarketTransactions(ctx context.Context) ([]MarketTransaction, error) {
	path := "/order/server_market/transaction"
	var result []MarketTransaction
	if err := o.client.GetWrappedList(ctx, path, "server_market_transaction", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMarketTransaction retrieves a specific marketplace transaction by ID.
//
// GET /order/server_market/transaction/{id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-order-server-market-transaction-id
func (o *OrderingService) GetMarketTransaction(ctx context.Context, transactionID string) (*MarketTransaction, error) {
	path := fmt.Sprintf("/order/server_market/transaction/%s", url.PathEscape(transactionID))
	var result MarketTransaction
	if err := o.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListAddonTransactions lists addon transaction history from the last 30 days.
//
// GET /order/server_addon/transaction
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-order-server-addon-transaction
func (o *OrderingService) ListAddonTransactions(ctx context.Context) ([]AddonTransaction, error) {
	path := "/order/server_addon/transaction"
	var result []AddonTransaction
	if err := o.client.GetWrappedList(ctx, path, "server_addon_transaction", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAddonTransaction retrieves a specific addon transaction by ID.
//
// GET /order/server_addon/transaction/{id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-order-server-addon-transaction-id
func (o *OrderingService) GetAddonTransaction(ctx context.Context, transactionID string) (*AddonTransaction, error) {
	path := fmt.Sprintf("/order/server_addon/transaction/%s", url.PathEscape(transactionID))
	var result AddonTransaction
	if err := o.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// WaitForMarketTransactionCompletion polls the transaction status until it's completed or an error occurs.
//
// checkInterval controls the polling frequency; if it is zero or negative, it
// defaults to 30 seconds.
func (o *OrderingService) WaitForMarketTransactionCompletion(ctx context.Context, transactionID string, checkInterval time.Duration) (*MarketTransaction, error) {
	if checkInterval <= 0 {
		checkInterval = 30 * time.Second
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Initial check before first tick
	tx, err := o.GetMarketTransaction(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	o.client.logger.LogAttrs(ctx, slog.LevelDebug, "transaction status",
		slog.String("id", transactionID), slog.String("status", tx.Status))

	switch tx.Status {
	case "ready":
		return tx, nil
	case "cancelled", "error":
		return tx, fmt.Errorf("transaction %s: %s", transactionID, tx.Status)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			tx, err := o.GetMarketTransaction(ctx, transactionID)
			if err != nil {
				return nil, err
			}

			o.client.logger.LogAttrs(ctx, slog.LevelDebug, "transaction status",
				slog.String("id", transactionID), slog.String("status", tx.Status))

			switch tx.Status {
			case "ready":
				return tx, nil
			case "cancelled", "error":
				return tx, fmt.Errorf("transaction %s: %s", transactionID, tx.Status)
			}
			// Otherwise keep waiting
		}
	}
}

// MarketProduct represents a server available on the auction market.
type MarketProduct struct {
	ID            uint32   `json:"id"`
	Name          string   `json:"name"`
	Description   []string `json:"description"`
	Traffic       string   `json:"traffic"`
	Dist          []string `json:"dist"`
	Arch          []int    `json:"arch"`
	Lang          []string `json:"lang"`
	CPU           string   `json:"cpu"`
	CPUBenchmark  uint32   `json:"cpu_benchmark"`
	CPUCount      int      `json:"cpu_count"`
	MemorySize    float64  `json:"memory_size"`
	HDDSize       float64  `json:"hdd_size"`
	HDDText       string   `json:"hdd_text"`
	HDDCount      uint8    `json:"hdd_count"`
	Datacenter    string   `json:"datacenter"`
	NetworkSpeed  string   `json:"network_speed"`
	Price         string   `json:"price"`
	PriceSetup    string   `json:"price_setup"`
	PriceVAT      string   `json:"price_vat"`
	PriceSetupVAT string   `json:"price_setup_vat"`
}

// AddonProduct represents an addon product available for order.
type AddonProduct struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Min    uint32              `json:"min"`
	Max    uint32              `json:"max"`
	Prices []ProductAddonPrice `json:"prices"`
}

// ListMarketProducts retrieves all servers available on the auction market.
//
// GET /order/server_market/product.
func (o *OrderingService) ListMarketProducts(ctx context.Context) ([]MarketProduct, error) {
	path := "/order/server_market/product"
	var result []MarketProduct
	if err := o.client.GetWrappedList(ctx, path, "server_market_product", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMarketProduct retrieves a specific auction market server by ID.
//
// GET /order/server_market/product/{id}.
func (o *OrderingService) GetMarketProduct(ctx context.Context, productID uint32) (*MarketProduct, error) {
	path := fmt.Sprintf("/order/server_market/product/%d", productID)
	var result MarketProduct
	if err := o.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTransactions lists standard server transaction history from the last 30 days.
//
// GET /order/server/transaction.
func (o *OrderingService) ListTransactions(ctx context.Context) ([]MarketTransaction, error) {
	path := "/order/server/transaction"
	var result []MarketTransaction
	if err := o.client.GetWrappedList(ctx, path, "transaction", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetTransaction retrieves a specific standard server transaction by ID.
//
// GET /order/server/transaction/{id}.
func (o *OrderingService) GetTransaction(ctx context.Context, transactionID string) (*MarketTransaction, error) {
	path := fmt.Sprintf("/order/server/transaction/%s", url.PathEscape(transactionID))
	var result MarketTransaction
	if err := o.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListAddonProducts retrieves all addon products available for a server.
//
// GET /order/server_addon/product.
func (o *OrderingService) ListAddonProducts(ctx context.Context, serverNumber int) ([]AddonProduct, error) {
	path := fmt.Sprintf("/order/server_addon/product?server_number=%d", serverNumber)
	var result []AddonProduct
	if err := o.client.GetWrappedList(ctx, path, "server_addon_product", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListProducts retrieves all standard product servers available for order.
//
// GET /order/server/product
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-order-server-product
func (o *OrderingService) ListProducts(ctx context.Context) ([]Product, error) {
	path := "/order/server/product"
	var result []Product
	if err := o.client.GetWrappedList(ctx, path, "product", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetProduct retrieves a specific product server by ID.
//
// GET /order/server/product/{id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-order-server-product-id
func (o *OrderingService) GetProduct(ctx context.Context, productID string) (*Product, error) {
	path := fmt.Sprintf("/order/server/product/%s", productID)
	var result Product
	if err := o.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
