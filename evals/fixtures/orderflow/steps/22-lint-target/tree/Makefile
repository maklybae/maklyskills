GO ?= go

.PHONY: build test lint

build:
	$(GO) build ./...

test:
	$(GO) test ./... -race -count=1

lint:
	$(GO) vet ./...
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt reports:"; echo "$$unformatted"; exit 1; \
	fi
