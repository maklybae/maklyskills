GO ?= go
TEST_MODE ?= memory

.PHONY: build test lint

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
