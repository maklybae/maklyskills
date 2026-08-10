GO ?= go
TEST_MODE ?= memory

.PHONY: build test lint generate run-orders run-inventory

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

run-orders:
	$(GO) run ./services/orders/cmd/orders -inventory http://127.0.0.1:8081

run-inventory:
	$(GO) run ./services/inventory/cmd/inventory -snapshot services/inventory/testdata/stock.json
