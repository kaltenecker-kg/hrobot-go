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
# @latest on purpose: a scanner should be current, Dependabot cannot bump a
# version literal here, and the module proxy plus sum.golang.org verify
# whatever version resolves. Not a go.mod `tool` dependency because
# golang.org/x/vuln would force the go directive to 1.26.0 (see go.mod).
# CI runs the same command.
vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: default all fmt lint vet test tidy-check verify vulncheck
