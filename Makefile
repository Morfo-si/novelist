.PHONY: help all build fmt vet test test-cover tidy snapshot tag clean

# ----------------------------------------------------------------------------
# Configuration
# ----------------------------------------------------------------------------

GO      := go
BINARY  := novelist
CMD     := .
VERSION := $(shell cat VERSION)
GOFLAGS := -trimpath

# ----------------------------------------------------------------------------
# Targets
# ----------------------------------------------------------------------------

## help: show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -E 's/## /  /'

## all: tidy, fmt, vet, test and build
all: tidy fmt vet test build

## build: compile the binary with the current VERSION injected
build:
	$(GO) build $(GOFLAGS) -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) $(CMD)

## fmt: run gofmt on all Go files
fmt:
	$(GO) fmt ./...

## vet: run go vet on all packages
vet:
	$(GO) vet ./...

## test: run tests
test:
	$(GO) test -v ./...

## test-cover: run tests with coverage profile output to coverage.out
test-cover:
	$(GO) test -v -coverprofile=coverage.out ./...

## tidy: ensure go.mod and go.sum are tidy
tidy:
	$(GO) mod tidy

## snapshot: run a local GoReleaser build (no publish) to verify the release config
snapshot:
	goreleaser release --snapshot --clean --skip=publish

## tag: create and push a signed git tag for the current VERSION (triggers the release workflow)
##      Edit the VERSION file first, commit it, then run: make tag
tag:
	@echo "Tagging v$(VERSION)…"
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)
	@echo "Tag v$(VERSION) pushed — the release workflow will now build and publish the binaries."

## clean: remove built artifacts
clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist/
