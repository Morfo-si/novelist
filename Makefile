# novelist — developer and release tasks.
# Run `make` or `make help` for the target list.

GO      ?= go
BINARY  := novelist
CMD     := ./cmd/novelist
PKGS    := ./...
DIST    := dist
COVER   := coverage.out

# VERSION describes the working tree; releases override it via the git tag.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Intentionally unpinned: govulncheck fetches its vulnerability database at run
# time, so a scanner should not silently miss newly added checks.
GOVULNCHECK_VERSION ?= latest

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nnovelist\n\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@echo

##@ Development

.PHONY: build
build: ## Build the binary into ./dist
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY) $(CMD)

.PHONY: install
install: ## Install the binary into $(GOPATH)/bin
	$(GO) install -trimpath -ldflags '$(LDFLAGS)' $(CMD)

.PHONY: run
run: ## Run it (make run ARGS='--version')
	$(GO) run $(CMD) $(ARGS)

.PHONY: fmt
fmt: ## Format the source
	$(GO) fmt $(PKGS)

.PHONY: vet
vet: ## Run go vet
	$(GO) vet $(PKGS)

.PHONY: tidy
tidy: ## Tidy and verify go.mod
	$(GO) mod tidy
	$(GO) mod verify

##@ Testing

.PHONY: test
test: ## Run the tests
	$(GO) test -count=1 $(PKGS)

.PHONY: test-race
test-race: ## Run the tests with the race detector
	$(GO) test -race -count=1 $(PKGS)

.PHONY: cover
cover: ## Run the tests and open an HTML coverage report
	$(GO) test -covermode=atomic -coverprofile=$(COVER) $(PKGS)
	$(GO) tool cover -func=$(COVER) | tail -n 1
	$(GO) tool cover -html=$(COVER)

##@ Security

.PHONY: lint
lint: ## Run golangci-lint (includes gosec)
	golangci-lint run

.PHONY: vuln
vuln: ## Scan for known vulnerabilities, including in the Go standard library
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) $(PKGS)

.PHONY: check
check: vet lint test-race vuln ## Run everything CI runs

##@ Release

.PHONY: version
version: ## Print the version this build would stamp
	@echo $(VERSION)

.PHONY: release-check
release-check: ## Validate .goreleaser.yaml
	goreleaser check

.PHONY: release-snapshot
release-snapshot: ## Build a full release locally, without tagging or publishing
	goreleaser release --snapshot --clean --skip=publish,sign

.PHONY: release
release: ## Tag and push a release (make release TAG=v0.0.8)
	@test -n "$(TAG)" || { echo "usage: make release TAG=v0.0.8"; exit 1; }
	@test -z "$$(git status --porcelain)" || { echo "working tree is dirty; commit first"; exit 1; }
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)
	@echo "Pushed $(TAG); GitHub Actions will build and publish the release."

##@ Housekeeping

.PHONY: clean
clean: ## Remove build output
	rm -rf $(DIST) $(COVER)
