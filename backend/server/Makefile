.PHONY: run fmt test vet build check

GO ?= go
GOFILES := $(shell find . -name '*.go' -not -path './admin/*')
CONFIG ?= configs/config.local.toml

# Single startup entry point: `make run` loads the TOML config, connects to
# primary/log PostgreSQL, applies explicit SQL migrations automatically,
# performs the idempotent bootstrap (super_admin role + initial administrator
# on a fresh database), and only then starts listening for HTTP.
run:
	$(GO) run ./cmd/server -config $(CONFIG)

fmt:
	gofmt -w $(GOFILES)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	$(GO) build ./...

check: fmt test vet build
