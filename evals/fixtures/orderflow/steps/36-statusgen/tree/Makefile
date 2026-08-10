GO ?= go
TEST_MODE ?= memory

.PHONY: build test lint generate

build:
	$(GO) build ./...

test:
	ORDERFLOW_TEST_MODE=$(TEST_MODE) $(GO) test ./... -race -count=1

lint:
	$(GO) vet ./...
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt reports:"; echo "$$unformatted"; exit 1; \
	fi

generate:
	$(GO) run ./tools/statusgen \
		-src services/orders/internal/core/status.go \
		-out services/orders/internal/generated/status_strings.go
