default: fmt lint vet test

lint:
	golangci-lint run

fmt:
	golangci-lint run --fix

vet:
	go vet ./...

test:
	go test -v -race -cover -timeout=180s ./...

# Fail if go.mod/go.sum are not tidy.
tidy-check:
	go mod tidy -diff

# Verify module dependencies against go.sum.
verify:
	go mod verify

# Scan for known vulnerabilities in dependencies and reachable code.
# govulncheck is pinned by commit SHA for reproducibility; Renovate bumps it.
vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@19b0bb6a272792b9afa8a6983c3e9b9a1816947f ./... # v1.6.0

.PHONY: fmt lint vet test tidy-check verify vulncheck
