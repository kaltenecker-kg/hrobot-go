# Pinned deliberately for reproducibility; bump intentionally. The vulnerability
# database is still fetched fresh at run time regardless of this version.
GOVULNCHECK_VERSION := v1.6.0

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
vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

.PHONY: fmt lint vet test tidy-check verify vulncheck
