# Changelog

## 1.0.0 (Unreleased)

BREAKING CHANGES:

- **restructure**: Project restructured from Terraform provider (`terraform-provider-hrobot`) to pure Go client
  library (`hrobot-go`), following the `hcloud-go` pattern
- **restructure**: Module path changed to `github.com/kaltenecker-kg/hrobot-go`
- **restructure**: CLI tool and Terraform provider removed; this package is now a library only
- **restructure**: API client code moved from `pkg/hrobot/` to repository root

FEATURES:

- **subnet**: Add stub SubnetService for subnet management API
- **storagebox**: Add stub StorageBoxService for storage box API
- **boot**: Add GetLastLinux, GetWindows, ActivateWindows, DeactivateWindows methods
- **ordering**: Add ListMarketProducts, GetMarketProduct, ListTransactions, GetTransaction, ListAddonProducts methods

## 0.2.0 (2024-11-07)

BREAKING CHANGES:

- **firewall**: Renamed `source_ip` to `source_ips` (now requires list syntax)
- **firewall**: Renamed `destination_ip` to `destination_ips` (now requires list syntax)
- **firewall**: Changed `filter_ipv6` default from `false` to `true` for better security

FEATURES:

- **firewall**: Add automatic IP array expansion - rules with multiple source/destination IPs are now automatically
  expanded into individual firewall rules
- **firewall**: Add validation to ensure expanded rules don't exceed Hetzner's 10-rule limit per direction
- **firewall**: Automatically append `/32` CIDR notation to IPv4 addresses without it
- **server**: Fix import to properly populate `public_net` block, resolving output evaluation errors

IMPROVEMENTS:

- **docs**: Update all examples to demonstrate array syntax for firewall IPs
- **docs**: Clarify that IP attributes are lists, not single values

## 0.1.2

Rename `hrobot_failover` to `hrobot_failover_ip`.

## 0.1.1

Remove the endpoint configuration from the provider to make it more simple.

## 0.1.0

Initial release as Terraform provider with CLI tool.
