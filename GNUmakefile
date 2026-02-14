default: fmt lint test

lint:
	golangci-lint run

fmt:
	golangci-lint run --fix

test:
	go test -v -cover -timeout=120s ./...

.PHONY: fmt lint test
