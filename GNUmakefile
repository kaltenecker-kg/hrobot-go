# Print the available targets; run `make all` for the full pipeline.
default:
	@echo "make all         fmt + lint + vet + test"
	@echo "make fmt         golangci-lint run --fix"
	@echo "make lint        golangci-lint run"
	@echo "make vet         go vet ./..."
	@echo "make test        go test -race -cover -timeout=180s ./..."
	@echo "make tidy-check  fail if go.mod/go.sum are not tidy"
	@echo "make verify      verify module dependencies against go.sum"
	@echo "make vulncheck   scan dependencies for known vulnerabilities"

# Serial by design: fmt and lint both invoke golangci-lint, which refuses to
# run concurrently with itself, so `make -j all` would fail.
.NOTPARALLEL:

all: fmt lint vet test

lint:
	golangci-lint run

fmt:
	golangci-lint run --fix

vet:
	go vet ./...

test:
	go test -race -cover -timeout=180s ./...

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

.PHONY: default all fmt lint vet test tidy-check verify vulncheck
