# Novelist Modernization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure Novelist into an idiomatic `cmd/novelist` + `internal/` layout, fix its code-level defects, and give it radiogogo's security-scanning, CI, and signed-release processes.

**Architecture:** `internal/storage` owns file I/O and returns errors; `internal/prompt` isolates the charm `huh` dependency; `cmd/novelist/main.go` is a thin `main → run(args, stdout, stderr) int` wiring layer that owns exit codes. Tooling (govulncheck, golangci-lint, CodeQL, Scorecard) and a hardened GoReleaser pipeline (cosign bundle signing, SBOM, provenance) wrap it.

**Tech Stack:** Go 1.26 (floor 1.26.2, `toolchain go1.26.5`), charm.land/huh/v2, adrg/xdg, testify, golangci-lint v2, GoReleaser v2, cosign 3.x, syft.

## Global Constraints

- Module path is `github.com/Morfo-si/novelist` — **do not change the casing**; the `cmd` package import path is `github.com/Morfo-si/novelist/cmd/novelist`.
- `go.mod`: floor `go 1.26.2`, plus `toolchain go1.26.5`. Never let a bare `go` directive be the only version source.
- Binary name is `novelist`; GoReleaser `project_name: novelist`; build main is `./cmd/novelist`.
- Library packages (`internal/*`) return errors and never call `os.Exit`, `log.Fatal`, or print. Only `cmd/novelist/main.go` calls `os.Exit`.
- Wrap errors with `%w`. Exit codes: `0` success, `1` runtime failure.
- Journal file is created `0600`.
- All GitHub Actions are pinned to a commit SHA with a trailing `# vN` comment and a minimal `permissions:` block.
- Conventional-commit prefixes (`feat:`, `fix:`, `build:`, `ci:`, `docs:`, `test:`, `chore:`) — GoReleaser's changelog groups on them.
- Owner/repo for release identity is `Morfo-si/novelist`.

---

### Task 1: `internal/storage` — file I/O core

**Files:**
- Create: `internal/storage/storage.go`
- Test: `internal/storage/storage_test.go`
- Test: `internal/storage/fuzz_test.go`

**Interfaces:**
- Produces:
  - `storage.Header` (`string`) = `"# Welcome to Novelist\n"`
  - `storage.EnsureFile(path string) error` — creates `path` with `Header` if absent; no-op if present; returns a wrapped error on a non-`ErrNotExist` stat failure or a create failure.
  - `storage.Append(path, story string) error` — appends `"\n## <timestamp>\n\n<story>\n"`; no-op when `story == ""`.

- [ ] **Step 1: Write the failing tests**

`internal/storage/storage_test.go`:

```go
package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureFileCreatesWithHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novel.md")

	require.NoError(t, EnsureFile(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, Header, string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestEnsureFileLeavesExistingUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novel.md")
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0o600))

	require.NoError(t, EnsureFile(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "existing", string(data))
}

func TestEnsureFileParentNotADirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR semantics differ on Windows")
	}
	parent := filepath.Join(t.TempDir(), "afile")
	require.NoError(t, os.WriteFile(parent, []byte("x"), 0o600))

	// Statting a path *under* a regular file yields ENOTDIR, which is not
	// ErrNotExist: EnsureFile must surface it, not treat it as "create me".
	err := EnsureFile(filepath.Join(parent, "child.md"))
	require.Error(t, err)
}

func TestAppendWritesTimestampedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novel.md")
	const story = "This is a test story."

	require.NoError(t, Append(path, story))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.True(t, strings.HasPrefix(content, "\n## "), "entry starts with a timestamp header")
	assert.True(t, strings.HasSuffix(content, "\n\n"+story+"\n"), "entry ends with the story")
}

func TestAppendEmptyStoryIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novel.md")

	require.NoError(t, Append(path, ""))

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "no file should be created for an empty story")
}

func TestAppendUnwritablePathReturnsError(t *testing.T) {
	// Parent directory does not exist, so O_CREATE cannot succeed.
	path := filepath.Join(t.TempDir(), "missing-dir", "novel.md")

	err := Append(path, "content")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/storage/ 2>&1 | head`
Expected: FAIL — `undefined: EnsureFile`, `undefined: Append`, `undefined: Header` (package does not compile yet).

- [ ] **Step 3: Write the implementation**

`internal/storage/storage.go`:

```go
// Package storage persists Novelist journal entries to a Markdown file.
package storage

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	// Header is written to a freshly created journal file.
	Header = "# Welcome to Novelist\n"
	// dateFormat is the Unix date layout used to stamp each entry.
	dateFormat = "Mon Jan _2 15:04:05 MST 2006"
	// filePerm keeps the journal private: it is a personal diary, so it must
	// not be world-readable.
	filePerm = 0o600
)

// EnsureFile creates path with the welcome Header if it does not already
// exist. It is a no-op when the file is present. A stat error that is not
// "file does not exist" (for example a permission problem) is returned rather
// than silently treated as absence.
func EnsureFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	//nolint:gosec // G304: path is Novelist's own journal file under the user's
	// Documents directory, not attacker-controlled input.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.WriteString(Header); err != nil {
		return fmt.Errorf("write header to %s: %w", path, err)
	}
	return nil
}

// Append writes a timestamped entry containing story to path. It is a no-op
// when story is empty, preserving the original "save nothing for empty input"
// behavior.
func Append(path, story string) error {
	if story == "" {
		return nil
	}

	//nolint:gosec // G304: path is Novelist's own journal file, not
	// attacker-controlled input.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	entry := fmt.Sprintf("\n## %s\n\n%s\n", time.Now().Format(dateFormat), story)
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("append to %s: %w", path, err)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/storage/ -v 2>&1 | tail -20`
Expected: PASS for all six tests.

- [ ] **Step 5: Write the fuzz test**

`internal/storage/fuzz_test.go`:

```go
package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzAppend asserts the round-trip invariant: after appending a non-empty
// story the file ends with that story, and arbitrary input never panics.
// Lower-value than a security fuzz target (Novelist has no injection surface),
// but the round-trip property is real.
func FuzzAppend(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("multi\nline\n## looks like a header")

	f.Fuzz(func(t *testing.T, story string) {
		path := filepath.Join(t.TempDir(), "novel.md")
		if err := EnsureFile(path); err != nil {
			t.Fatalf("EnsureFile: %v", err)
		}
		if err := Append(path, story); err != nil {
			t.Fatalf("Append: %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if story != "" && !strings.HasSuffix(string(data), story+"\n") {
			t.Fatalf("file does not end with the appended story")
		}
	})
}
```

- [ ] **Step 6: Run the fuzz target briefly**

Run: `go test ./internal/storage/ -run '^$' -fuzz FuzzAppend -fuzztime 10s 2>&1 | tail -10`
Expected: `PASS` — no new corpus failures within 10s.

- [ ] **Step 7: Confirm gosec's G304 is actually silenced (bite-check)**

Run (after golangci-lint is installed; if not yet installed, defer this check to Task 4 Step 5): `golangci-lint run ./internal/storage/ 2>&1 | head`
Expected: no output (clean). If G304 still fires, the `//nolint:gosec` directive must sit on the same line as the `os.OpenFile` call — widen or re-place it, do not delete the check.

- [ ] **Step 8: Commit**

```bash
git add internal/storage/
git commit -m "feat: add internal/storage package for journal file I/O

EnsureFile and Append return errors instead of calling log.Fatal, create
the journal 0600, use errors.Is for the missing-file check, and are covered
by unit tests plus a round-trip fuzz target."
```

---

### Task 2: `internal/prompt` — the huh form

**Files:**
- Create: `internal/prompt/prompt.go`
- Test: `internal/prompt/prompt_test.go`

**Interfaces:**
- Consumes: `charm.land/huh/v2`
- Produces:
  - `prompt.CharLimit` (`int`) = `10 * 1024`
  - `prompt.New(story *string) *huh.Text` — the titled, char-limited text form bound to `story`.

- [ ] **Step 1: Write the failing test**

`internal/prompt/prompt_test.go`:

```go
package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContainsTitle(t *testing.T) {
	var story string
	p := New(&story)
	assert.Contains(t, p.View(), "Tell me a story", "prompt renders the title")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/prompt/ 2>&1 | head`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Write the implementation**

`internal/prompt/prompt.go`:

```go
// Package prompt builds the interactive text form Novelist shows the user.
package prompt

import "charm.land/huh/v2"

// CharLimit caps the story length at 10 KB.
const CharLimit = 10 * 1024

// New returns a text form bound to story.
func New(story *string) *huh.Text {
	return huh.NewText().
		Title("Tell me a story.").
		Value(story).
		Placeholder("What's on your mind?").
		CharLimit(CharLimit)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/prompt/ -v 2>&1 | tail -10`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/prompt/
git commit -m "feat: add internal/prompt package isolating the huh form"
```

---

### Task 3: `cmd/novelist/main.go` — thin wiring, and remove the old layout

**Files:**
- Create: `cmd/novelist/main.go`
- Test: `cmd/novelist/main_test.go`
- Delete: `main.go`, `main_test.go`, `VERSION`

**Interfaces:**
- Consumes: `prompt.New`, `storage.EnsureFile`, `storage.Append`, `github.com/adrg/xdg`
- Produces: `run(args []string, stdout, stderr io.Writer) int` (package-private; the testable seam)

- [ ] **Step 1: Write the failing test**

`cmd/novelist/main_test.go`:

```go
package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunVersionFlag(t *testing.T) {
	for _, arg := range []string{"--version", "-v", "version"} {
		var out bytes.Buffer
		code := run([]string{arg}, &out, io.Discard)
		assert.Equal(t, 0, code, "arg %q exits 0", arg)
		assert.Contains(t, out.String(), "novelist ", "arg %q prints the version", arg)
	}
}
```

(The interactive save path calls `huh`'s `Run()`, which needs a TTY, so it is
not unit-tested — the analog to radiogogo not testing audio playback. The
`--version` seam is what we assert.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./cmd/novelist/ 2>&1 | head`
Expected: FAIL — `undefined: run` (and package currently does not exist).

- [ ] **Step 3: Write the implementation**

`cmd/novelist/main.go`:

```go
// Command novelist captures timestamped journal entries from the terminal.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"

	"github.com/Morfo-si/novelist/internal/prompt"
	"github.com/Morfo-si/novelist/internal/storage"
)

// version is injected at build time via -ldflags "-X main.version=...".
// It falls back to "dev" for local builds without the ldflag set.
var version = "dev"

// novelFile is the journal filename created under the user's Documents dir.
const novelFile = "novelist.md"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses args, drives the prompt, and persists the entry. It returns a
// process exit code and is the seam that keeps main testable.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "--version", "-v", "version":
			fmt.Fprintf(stdout, "novelist %s\n", version)
			return 0
		}
	}

	path := filepath.Join(xdg.UserDirs.Documents, novelFile)

	var story string
	if err := prompt.New(&story).Run(); err != nil {
		fmt.Fprintf(stderr, "novelist: %v\n", err)
		return 1
	}

	if err := storage.EnsureFile(path); err != nil {
		fmt.Fprintf(stderr, "novelist: %v\n", err)
		return 1
	}
	if err := storage.Append(path, story); err != nil {
		fmt.Fprintf(stderr, "novelist: %v\n", err)
		return 1
	}

	if story != "" {
		fmt.Fprintf(stdout, "Okay. Your thoughts have been saved to %s\n", path)
	}
	return 0
}
```

- [ ] **Step 4: Remove the old flat layout**

```bash
git rm main.go main_test.go VERSION
```

- [ ] **Step 5: Verify the whole module builds and tests pass**

Run: `go build ./... && go test ./... 2>&1 | tail -20`
Expected: build succeeds; PASS for `internal/storage`, `internal/prompt`, `cmd/novelist`.

- [ ] **Step 6: Verify `go vet` is clean**

Run: `go vet ./...`
Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add cmd/novelist/ main.go main_test.go VERSION
git commit -m "feat: move main into cmd/novelist with a testable run() seam

Replaces the flat package main. run() handles --version, resolves the
journal path with filepath.Join, and maps errors to exit codes; only main
calls os.Exit. Removes the VERSION file (the git tag now drives the version)."
```

---

### Task 4: `go.mod` toolchain, `.golangci.yml`, and the Makefile

**Files:**
- Modify: `go.mod` (add `toolchain` after the `go` directive)
- Create: `.golangci.yml`
- Modify: `Makefile` (full rewrite)

**Interfaces:**
- Produces: `make check` (= `vet lint test-race vuln`), `make build`, `make release TAG=...`, `make release-snapshot`.

- [ ] **Step 1: Add the toolchain directive**

Edit `go.mod` so the top reads exactly:

```
module github.com/Morfo-si/novelist

go 1.26.2

toolchain go1.26.5
```

- [ ] **Step 2: Create `.golangci.yml`**

```yaml
version: "2"

linters:
  default: standard
  enable:
    - bodyclose
    - errorlint
    - gosec
    - misspell
    - revive
    - unconvert
  exclusions:
    presets:
      # The long-standing errcheck carve-out for writes to stdout/stderr and
      # for Flush/Close — there is nothing useful to do about those errors in a
      # CLI, and checking each one buries the real handling.
      - std-error-handling
    rules:
      # Test fixtures deliberately ignore write errors and construct odd paths.
      - path: _test\.go
        linters:
          - errcheck
          - gosec

formatters:
  enable:
    - gofmt
    - goimports
```

- [ ] **Step 3: Rewrite the Makefile**

```makefile
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
```

- [ ] **Step 4: Install golangci-lint if missing**

Run: `command -v golangci-lint || brew install golangci-lint`
Expected: a path, or a successful install. Confirm it is v2: `golangci-lint version 2>&1 | head -1`.

- [ ] **Step 5: Run `make check` and confirm it is green**

Run: `make check 2>&1 | tail -30`
Expected: `go vet` silent, `golangci-lint` reports **no issues** (this is where Task 1's `//nolint:gosec` directives are validated end-to-end), race tests PASS, `govulncheck` reports `No vulnerabilities found`.
If govulncheck reports a stdlib finding, the `toolchain` line is wrong or missing — fix it, do not suppress the finding.

- [ ] **Step 6: Confirm `make build` stamps a version**

Run: `make build && ./dist/novelist --version`
Expected: `novelist <git-describe-or-dev>` (non-`dev` if any tag is reachable).

- [ ] **Step 7: Commit**

```bash
git add go.mod .golangci.yml Makefile
git commit -m "build: add toolchain directive, golangci-lint v2, and a full Makefile

Pins toolchain go1.26.5 so builds get the patched stdlib; adds a check
aggregate (vet lint test-race vuln) and a git-tag release flow."
```

---

### Task 5: CI workflow (`ci.yml`, replacing `pr.yml`)

**Files:**
- Create: `.github/workflows/ci.yml`
- Delete: `.github/workflows/pr.yml`

- [ ] **Step 1: Create `.github/workflows/ci.yml`**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  test:
    name: Test (${{ matrix.os }})
    runs-on: ${{ matrix.os }}
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    defaults:
      run:
        # The windows runner defaults to PowerShell, which splits Go flags like
        # -coverprofile=coverage.out into a bogus package. bash keeps all three
        # platforms running identical commands.
        shell: bash
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7

      - uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6
        with:
          go-version-file: go.mod
          check-latest: true

      - name: Verify go.mod is tidy
        run: |
          go mod tidy
          git diff --exit-code -- go.mod go.sum

      - name: Test with race detector
        run: go test -race -count=1 -cover ./...

      - name: Build
        run: go build ./...

  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7

      - uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6
        with:
          go-version-file: go.mod
          check-latest: true

      - uses: golangci/golangci-lint-action@ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a # v9
        with:
          version: latest

  vuln:
    name: govulncheck
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7

      - uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6
        with:
          go-version-file: go.mod
          check-latest: true

      - name: Scan
        run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

- [ ] **Step 2: Delete the old workflow**

```bash
git rm .github/workflows/pr.yml
```

- [ ] **Step 3: Validate the YAML parses**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('ok')"`
Expected: `ok`.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml .github/workflows/pr.yml
git commit -m "ci: replace pr.yml with a 3-OS matrix plus lint and vuln jobs

SHA-pinned actions, bash shell on all platforms, concurrency cancellation."
```

---

### Task 6: CodeQL, Scorecard, and SECURITY.md

**Files:**
- Create: `.github/workflows/codeql.yml`
- Create: `.github/workflows/scorecard.yml`
- Create: `SECURITY.md`

- [ ] **Step 1: Create `.github/workflows/codeql.yml`**

```yaml
name: CodeQL

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    # Mondays at 07:00 UTC — catches newly published queries between changes.
    - cron: '0 7 * * 1'

permissions:
  contents: read

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  analyze:
    name: Analyze Go
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
      actions: read
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7

      - uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6
        with:
          go-version-file: go.mod

      - name: Initialize CodeQL
        uses: github/codeql-action/init@99df26d4f13ea111d4ec1a7dddef6063f76b97e9 # v4
        with:
          languages: go
          queries: security-and-quality

      - name: Autobuild
        uses: github/codeql-action/autobuild@99df26d4f13ea111d4ec1a7dddef6063f76b97e9 # v4

      - name: Analyze
        uses: github/codeql-action/analyze@99df26d4f13ea111d4ec1a7dddef6063f76b97e9 # v4
        with:
          category: "/language:go"
```

- [ ] **Step 2: Create `.github/workflows/scorecard.yml`**

```yaml
name: Scorecard

on:
  branch_protection_rule:
  push:
    branches: [main]
  schedule:
    - cron: '0 8 * * 1'
  workflow_dispatch:

permissions: read-all

jobs:
  analysis:
    name: Scorecard analysis
    runs-on: ubuntu-latest
    permissions:
      security-events: write
      id-token: write
      contents: read
      actions: read
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7
        with:
          persist-credentials: false

      - name: Run analysis
        uses: ossf/scorecard-action@4eaacf0543bb3f2c246792bd56e8cdeffafb205a # v2.4.3
        with:
          results_file: results.sarif
          results_format: sarif
          publish_results: true

      - name: Upload artifact
        uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7
        with:
          name: scorecard-results
          path: results.sarif
          retention-days: 5

      - name: Upload to code scanning
        uses: github/codeql-action/upload-sarif@99df26d4f13ea111d4ec1a7dddef6063f76b97e9 # v4
        with:
          sarif_file: results.sarif
```

- [ ] **Step 3: Create `SECURITY.md`**

```markdown
# Security Policy

Novelist is a small hobby CLI maintained by one person in their spare time.
This policy is honest about what that means: reports are handled best-effort,
with no SLA.

## Scope

Novelist shows a text prompt in the terminal and appends what you type,
timestamped, to a Markdown file (`novelist.md`) under your Documents directory.
It makes no network connections and starts no subprocesses. The realistic
security surface is therefore two things:

- **Its dependencies.** The interactive prompt is built on the charm libraries,
  which are updated via Dependabot and scanned by `govulncheck` in CI.
- **Local file handling.** The journal is created `0600` (owner-only) and the
  file path is fixed to your Documents directory; there is no user-supplied path.

If you find a way to make Novelist write outside that file, act on input in an
unsafe way, or a vulnerability in a bundled dependency that affects it, that is
a real finding.

## Supported Versions

Only the latest release is supported. If you are running something older,
please upgrade before reporting; the issue may already be fixed.

## Reporting a Vulnerability

Please use GitHub's private vulnerability reporting instead of a public issue:
go to the repository's **Security** tab and click **Report a vulnerability**.
This keeps details out of public view until there is a fix.

There is no guaranteed response time. Reports will be looked at and acknowledged
as soon as possible, but this is one maintainer, not a staffed security team.

## Verifying Releases

Release checksums are signed keylessly with
[cosign](https://docs.sigstore.dev/). See the "Verifying this release" section
of each release's notes for the exact `cosign verify-blob` command and expected
identity.
```

- [ ] **Step 4: Validate both workflows parse**

Run: `for f in codeql scorecard; do python3 -c "import yaml; yaml.safe_load(open('.github/workflows/$f.yml')); print('$f ok')"; done`
Expected: `codeql ok` and `scorecard ok`.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/codeql.yml .github/workflows/scorecard.yml SECURITY.md
git commit -m "ci: add CodeQL and OpenSSF Scorecard workflows plus SECURITY.md"
```

---

### Task 7: Expand Dependabot

**Files:**
- Modify: `.github/dependabot.yml` (full rewrite)

- [ ] **Step 1: Rewrite `.github/dependabot.yml`**

```yaml
version: 2

updates:
  # Keeps the third-party Go module tree (charm, xdg, testify, …) current.
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
      day: monday
      time: "07:00"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "build"
      include: scope
    groups:
      go-dependencies:
        patterns: ["*"]
        update-types: [minor, patch]

  # Keeps the pinned action SHAs in .github/workflows current; pinned SHAs
  # never update by themselves.
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
      day: monday
      time: "07:00"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "ci"
      include: scope
    groups:
      actions:
        patterns: ["*"]
        update-types: [minor, patch]
```

- [ ] **Step 2: Validate it parses**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/dependabot.yml')); print('ok')"`
Expected: `ok`.

- [ ] **Step 3: Commit**

```bash
git add .github/dependabot.yml
git commit -m "ci: group gomod updates and add github-actions to Dependabot"
```

---

### Task 8: Harden `.goreleaser.yaml` (SBOM, cosign, provenance-ready)

**Files:**
- Modify: `.goreleaser.yaml` (full rewrite)

- [ ] **Step 1: Rewrite `.goreleaser.yaml`**

```yaml
# .goreleaser.yaml — https://goreleaser.com
version: 2

project_name: novelist

before:
  hooks:
    # Read-only by design: GoReleaser validates git state before these run, so
    # a hook that mutated go.mod would go undetected and the release would not
    # match the tagged commit.
    - go mod verify

builds:
  - id: novelist
    main: ./cmd/novelist
    binary: novelist
    env:
      - CGO_ENABLED=0
    flags:
      - -trimpath
    ldflags:
      - -s -w -X main.version={{ .Version }}
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    mod_timestamp: "{{ .CommitTimestamp }}"

archives:
  - id: default
    ids:
      - novelist
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        formats:
          - zip
    files:
      - README.md
      - LICENSE

checksum:
  name_template: checksums.txt
  algorithm: sha256

sboms:
  - id: archives
    artifacts: archive

signs:
  # cosign 3.x (installed by sigstore/cosign-installer@v4.1.2) dropped separate
  # signature/certificate outputs in favour of a single bundle: keep this in
  # bundle form as long as cosign-installer tracks the 3.x line.
  - id: checksums
    cmd: cosign
    signature: "${artifact}.bundle"
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: checksum
    output: true

snapshot:
  version_template: "{{ incpatch .Version }}-snapshot"

changelog:
  sort: asc
  use: github
  groups:
    - title: Features
      regexp: '^.*?feat(\(.+\))??!?:.+$'
      order: 0
    - title: Bug fixes
      regexp: '^.*?fix(\(.+\))??!?:.+$'
      order: 1
    - title: Build and CI
      regexp: '^.*?(build|ci)(\(.+\))??!?:.+$'
      order: 2
    - title: Other
      order: 999
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
      - "Merge pull request"

release:
  github:
    owner: Morfo-si
    name: novelist
  draft: false
  prerelease: auto
  footer: |
    ## Verifying this release

    Checksums are signed with [cosign](https://docs.sigstore.dev/) keylessly —
    there is no private key. Download `checksums.txt` and
    `checksums.txt.bundle`, then:

    ```sh
    cosign verify-blob \
      --bundle checksums.txt.bundle \
      --certificate-identity-regexp 'https://github\.com/Morfo-si/novelist/\.github/workflows/.+' \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com \
      checksums.txt
    ```

    Then check your archive against it:

    ```sh
    sha256sum --check --ignore-missing checksums.txt
    ```

    ## Installation

    Download the archive for your platform, extract it, and put the `novelist`
    binary on your `$PATH`.
```

- [ ] **Step 2: Validate the config**

Run: `command -v goreleaser || brew install goreleaser; goreleaser check`
Expected: `check` passes (may print info-level notices; no errors).

- [ ] **Step 3: Build a snapshot locally (signing skipped)**

Run: `make release-snapshot 2>&1 | tail -20 && ls dist/*.tar.gz dist/*.zip`
Expected: archives named `novelist_<version>-snapshot_<os>_<arch>.{tar.gz,zip}` exist for linux/darwin/windows × amd64/arm64.

- [ ] **Step 4: Confirm a snapshot binary reports its version**

Run: `tar -xzf "$(ls dist/novelist_*_$(go env GOOS)_$(go env GOARCH).tar.gz | head -1)" -C dist novelist && ./dist/novelist --version`
Expected: `novelist <version>-snapshot`.

- [ ] **Step 5: Commit**

```bash
git add .goreleaser.yaml
git commit -m "build: add SBOM, cosign bundle signing, and reproducible builds

Moves the build to ./cmd/novelist, adds mod_timestamp, a syft SBOM, keyless
cosign bundle signing of the checksums, and a documented verify-blob footer."
```

---

### Task 9: Harden `release.yml`

**Files:**
- Modify: `.github/workflows/release.yml` (full rewrite)

- [ ] **Step 1: Rewrite `.github/workflows/release.yml`**

```yaml
name: Release

on:
  push:
    tags: ["v*"]

permissions:
  contents: read

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write      # create the release and upload assets
      id-token: write      # keyless cosign signing and attestation
      attestations: write  # build provenance
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7
        with:
          fetch-depth: 0   # GoReleaser needs full history for the changelog

      - uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6
        with:
          go-version-file: go.mod
          check-latest: true

      - name: Test before releasing
        run: go test -race -count=1 ./...

      - name: Scan before releasing
        run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...

      - uses: sigstore/cosign-installer@6f9f17788090df1f26f669e9d70d6ae9567deba6 # v4.1.2
      - uses: anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 # v0

      - uses: goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94 # v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          # Without this GoReleaser derives the version from git describe, which
          # is ambiguous when several tags point at one commit -- the ordinary
          # case of promoting a release candidate to final. It then builds and
          # publishes under whichever tag it picked, not the one that triggered
          # this run.
          GORELEASER_CURRENT_TAG: ${{ github.ref_name }}

      - name: Attest build provenance
        uses: actions/attest-build-provenance@0f67c3f4856b2e3261c31976d6725780e5e4c373 # v4
        with:
          subject-path: dist/*.tar.gz, dist/*.zip
```

- [ ] **Step 2: Validate it parses**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml')); print('ok')"`
Expected: `ok`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: harden release with signing, provenance, and tag pinning

Adds id-token/attestations permissions, a test+govulncheck gate, cosign and
syft installers, build-provenance attestation, and GORELEASER_CURRENT_TAG so
promoting an rc to final publishes under the triggering tag."
```

---

### Task 10: README — install path, build instructions, badges

**Files:**
- Modify: `README.md`

**Context:** The `go install` path changed to `github.com/Morfo-si/novelist/cmd/novelist@latest`. The `VERSION` file is gone, so any "edit VERSION" build note must be replaced with the git-tag flow. Add CI and Scorecard badges near the top.

- [ ] **Step 1: Read the current README to find the exact blocks to change**

Run: `grep -n "go install\|VERSION\|go build\|badge\|img.shields\|## Installation\|## Build" README.md`
Expected: line numbers for the install snippet, the build section, and any VERSION reference.

- [ ] **Step 2: Update the `go install` path**

Replace the install command:

```bash
go install github.com/Morfo-si/novelist/cmd/novelist@latest
```

Add a one-line note beneath it: *"(The install path is `.../cmd/novelist` as of v0.0.8; older `.../novelist@latest` no longer resolves.)"*

- [ ] **Step 3: Update the build-from-source instructions**

Replace any "edit the VERSION file" wording with:

````markdown
### Build

```bash
git clone https://github.com/Morfo-si/novelist.git
cd novelist
make build          # stamps the version from the current git tag
./dist/novelist --version
```

To cut a release, tag it: `make release TAG=v0.0.8`.
````

- [ ] **Step 4: Add badges under the title**

Insert directly beneath the `# Novelist` heading (replace `main` workflow file names only if they differ):

```markdown
[![CI](https://github.com/Morfo-si/novelist/actions/workflows/ci.yml/badge.svg)](https://github.com/Morfo-si/novelist/actions/workflows/ci.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/Morfo-si/novelist/badge)](https://scorecard.dev/viewer/?uri=github.com/Morfo-si/novelist)
```

- [ ] **Step 5: Verify no stale references remain**

Run: `grep -n "VERSION file\|novelist@latest[^/]" README.md || echo "clean"`
Expected: `clean` (no bare `novelist@latest` install path, no "VERSION file" mention).

- [ ] **Step 6: Commit**

```bash
git add README.md
git commit -m "docs: document cmd/novelist install path, git-tag builds, and badges"
```

---

### Task 11: Final integration check

- [ ] **Step 1: Full local gate**

Run: `make check 2>&1 | tail -30`
Expected: vet silent, lint clean, race tests PASS, `No vulnerabilities found`.

- [ ] **Step 2: Confirm the module is tidy**

Run: `go mod tidy && git diff --exit-code -- go.mod go.sum && echo tidy`
Expected: `tidy` (no diff).

- [ ] **Step 3: Confirm goreleaser config still validates**

Run: `goreleaser check`
Expected: passes.

- [ ] **Step 4: Push the branch and open a PR (do not tag a release)**

```bash
git push -u origin HEAD
gh pr create --fill
```

Expected: CI, CodeQL, and Scorecard workflows run on the PR. Releasing is a
separate, explicitly-authorized step (`make release TAG=...`) — do not tag as
part of this plan.

---

## Self-Review

**Spec coverage:**
- Restructure to `cmd/novelist` + `internal/` → Tasks 1–3. ✓
- Fix code defects (errors, `errors.Is`, `filepath.Join`, no `log.Fatal`, 0600) → Tasks 1, 3. ✓
- govulncheck, golangci-lint, CodeQL, Scorecard → Tasks 4, 5, 6. ✓
- `toolchain go1.26.5` → Task 4. ✓
- Dependabot gomod + github-actions grouped → Task 7. ✓
- git-tag release flow, VERSION removed → Tasks 3, 4. ✓
- cosign bundle signing, SBOM, mod_timestamp, footer → Task 8. ✓
- release.yml SHA-pin, permissions, gates, GORELEASER_CURRENT_TAG, provenance → Task 9. ✓
- README install-path change + badges → Task 10. ✓
- SECURITY.md → Task 6. ✓
- Fuzz round-trip → Task 1. ✓
- 3-OS CI with bash shell → Task 5. ✓

**Placeholder scan:** No TBDs; every code and config step carries full content. README edits (Task 10) are described against exact `grep`-located blocks because the current README wording is not fully known in advance — Step 1 locates them, Steps 2–4 give the exact replacement text.

**Type consistency:** `EnsureFile(path string) error`, `Append(path, story string) error`, `Header`, `New(story *string) *huh.Text`, `run([]string, io.Writer, io.Writer) int` are named identically in their producing task and every consuming task.
