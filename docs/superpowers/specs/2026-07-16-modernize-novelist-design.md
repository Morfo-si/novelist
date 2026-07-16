# Modernizing Novelist

**Date:** 2026-07-16
**Status:** Approved

## Context

Novelist is a small command-line journaling tool: it prompts for a story with a
charm `huh` text form and appends it, timestamped, to a Markdown file under the
user's Documents directory. It is more mature than radiogogo was at the start of
its modernization — it already runs Go 1.26.2, has a real dependency tree
(`charm.land/huh/v2`, `github.com/adrg/xdg`, `github.com/stretchr/testify`), a
`Makefile`, a `.goreleaser.yaml`, Dependabot, and PR/release workflows.

The request is to give novelist the same structure and processes as radiogogo.
Two facts shape how that request should be applied rather than copied blindly:

- **Novelist's security shape is inverted from radiogogo's.** Radiogogo had zero
  third-party dependencies, so the standard library was its entire attack surface
  and `govulncheck` was the single highest-value scanner. Novelist has a large
  third-party tree (bubbletea and friends) and touches no network and no
  subprocess. Its real exposure is that dependency tree, so `govulncheck` plus
  Dependabot are the highest-value additions here — for the opposite reason.
- **Novelist's code-defect surface is much smaller.** There is no input to
  validate against injection. The defects that exist are unhandled I/O errors and
  a Windows-unsafe path join. They are worth fixing in the same pass so the new
  lint dashboard is green on day one; a scanner that reports known findings from
  the start trains everyone to ignore it.

## Goals

- Restructure to an idiomatic `cmd/novelist` + `internal/` layout.
- Fix the code-level defects (unhandled errors, path join, `log.Fatal` in
  library code) in the same pass.
- Add security scanning: govulncheck, golangci-lint, CodeQL, OpenSSF Scorecard.
- Harden CI: 3-OS matrix, SHA-pinned actions, minimal permissions.
- Harden releases: cosign signing, SBOM, build provenance, and the
  `GORELEASER_CURRENT_TAG` fix.
- Add a `toolchain` directive so builds use a patched toolchain.
- Expand Dependabot to github-actions with grouped updates.
- Move to radiogogo's git-tag release flow (`make release TAG=...`).

## Non-Goals

- New features. Novelist captures timestamped journal entries; that scope is
  unchanged.
- A configuration file for the file location or prompt.
- Testing the live interactive TUI session. Tests assert on form *construction*
  and on file I/O, never on an interactive terminal — the analog to radiogogo not
  testing actual audio playback.
- Changing the module path casing (`github.com/Morfo-si/novelist`). It is
  published and resolves correctly; leave it.

## Architecture

```
cmd/novelist/main.go     flag parsing (--version), path resolution, wiring, exit codes — no logic
internal/prompt/         the huh TUI form (isolates the charm dependency)
internal/storage/        file I/O: ensure-file-with-header, append timestamped entry
```

The split is driven by testability and dependency isolation, not tidiness. Today
all logic lives in `package main`, mixes the charm dependency with file I/O, and
calls `log.Fatal` — none of which can be unit-tested in isolation.

- `internal/storage` is the testable core. It imports nothing from charm and
  returns errors instead of exiting.
- `internal/prompt` is the only package that knows the TUI library exists, so the
  charm dependency is confined to one place and `main`/`storage` stay free of it.
- `cmd/novelist/main.go` follows radiogogo's `main → run(args, stdout, stderr) int`
  shape so exit codes are testable. `main` is the only component that calls
  `os.Exit`.

### Data flow

```
args → main → --version short-circuit (stamped via ldflags)
            → filepath.Join(xdg.UserDirs.Documents, "novelist.md")
            → prompt.New(&story).Run()          ← interactive; error handled
            → storage.EnsureFile(path)          ← creates with header if missing
            → storage.Append(path, story)       ← appends timestamped entry
```

### Package APIs

- `internal/prompt`
  - `New(story *string) *huh.Text` — builds the titled, char-limited text form
    (renamed from `GeneratePrompt`). The `CharLimit` constant lives here.
- `internal/storage`
  - `EnsureFile(path string) error` — creates the file with the
    `# Welcome to Novelist` header if it does not exist; a no-op if it does.
    Renamed from `FileExists`, which was a misnomer (it creates).
  - `Append(path, story string) error` — appends `\n## <timestamp>\n\n<story>\n`;
    a no-op when `story` is empty (preserves current behavior). The date format
    constant lives here.

## Code defects being fixed

1. **Discarded prompt error.** `main.go:56` ignores `prompt.Run()`'s error, so a
   TUI failure is silently followed by a save attempt. Fix: handle it; on error
   write to stderr and exit 1.
2. **Ignored write errors.** `EnsureFile` (`os.Create`, `WriteString`, `Close`)
   and `Append` (`file.Write`) discard errors (gosec G104). Fix: return them.
3. **`os.Stat` error conflated with "missing".** `main.go:68` treats any
   `os.Stat` error as "file absent", so a permission error would wrongly trigger a
   create. Fix: `errors.Is(err, os.ErrNotExist)`.
4. **Windows-unsafe path.** `fmt.Sprintf("%s/%s", ...)` hardcodes `/`. Fix:
   `filepath.Join`.
5. **`log.Fatal` in library code.** Untestable and calls `os.Exit` from a
   non-main package. Fix: return errors; only `main` exits.
6. **World-readable journal.** The file is created 0644. It is a personal diary in
   `~/Documents`. Fix: create it 0600.

## CLI surface

Unchanged for users:

```
novelist                 open the prompt, append the entry
novelist --version|-v    print the stamped version
```

Breaking change: the `go install` path moves with the layout, from
`github.com/Morfo-si/novelist@latest` to
`github.com/Morfo-si/novelist/cmd/novelist@latest`. The README is updated in the
same change and notes that the old root path no longer resolves.

## Error handling

Errors wrap with `%w` and surface at `main`, the only component that calls
`os.Exit`. Library packages return errors and never exit or print. Exit codes:
`0` success, `1` runtime failure.

## Testing

Table-driven, testify (already the project's assertion library), weighted toward
`storage`.

- **storage**: ensure-creates-with-header, ensure-leaves-existing-file-untouched,
  append round-trips the timestamped entry, empty-story is a no-op, append to an
  unwritable path returns an error.
- **prompt**: `New(&s).View()` contains the title (ports the existing
  `TestGeneratePrompt`).
- **fuzz**: `FuzzAppend` asserts the round-trip invariant (after appending, the
  file ends with the story) and that arbitrary input never panics. This is
  honestly lower-value than radiogogo's security fuzz — novelist has no injection
  surface — but the round-trip property is real and it keeps the practice in
  place.

CI runs tests on ubuntu, macos, and windows. No runner needs a TTY because the
tests exercise construction and I/O, not an interactive session.

## Toolchain

- `go.mod` keeps the floor `go 1.26.2` and adds `toolchain go1.26.5` (the latest
  1.26 patch at time of writing). This is the exact trap radiogogo hit: a `go`
  directive alone let CI build with an unpatched toolchain that `govulncheck`
  then flagged. The `go` directive stays the language floor consumers must meet;
  `toolchain` names the version builds actually use. `setup-go` honours
  `toolchain` over `go`.
- The `toolchain` line is manually maintained. Dependabot's `gomod` updates bump
  module requirements, not `toolchain` directives, so a future stdlib CVE fixed in
  a later patch will redden `govulncheck` until someone bumps the line by hand.
  Accepted as ongoing maintenance.

## Tooling

**Makefile** (align to radiogogo). Development: `build test test-race cover lint
fmt vet vuln tidy clean run install`. Aggregate: `check: vet lint test-race vuln`
— `fmt` is deliberately excluded because it mutates the tree. Release:
`release-snapshot`, `release-check`, `release` (takes `TAG=`), `version`. The
`VERSION` file and its `tag` target are removed; `make build` stamps the version
from `git describe --tags --always --dirty`, falling back to `dev`.

**Workflows** (`.github/workflows/`):

| File | Runs |
|---|---|
| `ci.yml` (renamed from `pr.yml`) | test matrix (ubuntu/macos/windows, bash shell), golangci-lint, govulncheck |
| `codeql.yml` | semantic analysis, security-and-quality, PR + weekly |
| `scorecard.yml` | repo hygiene, badge |
| `release.yml` | goreleaser, cosign, SBOM, attestation |

All actions are pinned to a commit SHA with minimal `permissions:` blocks. The
`ci.yml` test job sets `defaults.run.shell: bash` so the Windows runner does not
tokenize `-coverprofile=coverage.out` into a bogus package, and adds a
`concurrency` block.

**Dependabot** (`.github/dependabot.yml`): `gomod` and `github-actions`, with
grouped minor/patch updates to limit PR volume.

**SECURITY.md** added.

**Release** is tag-triggered. `release.yml` gets `permissions: contents: write,
id-token: write, attestations: write`; a test + govulncheck gate before release;
`GORELEASER_CURRENT_TAG: ${{ github.ref_name }}`; and a build-provenance
attestation over `dist/*.tar.gz, dist/*.zip`. `.goreleaser.yaml` adds
`mod_timestamp` for reproducibility, a `before.hooks: go mod verify` step, a syft
SBOM, cosign keyless bundle-format signing, and a footer documenting
`cosign verify-blob --bundle`.

## Decisions and risks

- **`go install` path changes** (`.../novelist@latest` →
  `.../novelist/cmd/novelist@latest`). A breaking change for existing installers,
  accepted for the idiomatic layout. README updated in the same change.
- **File permissions tighten 0644 → 0600.** Visible only via `ls -l`. Correct
  default for a personal journal.
- **`internal/prompt` is a thin package** (one small constructor). Kept separate
  so the charm dependency lives in exactly one place, mirroring radiogogo's
  dependency-isolation rationale.
- **`toolchain go1.26.5` is manually maintained** — see Toolchain above.
- **The `VERSION` file is deleted.** Both its consumers (ldflags via `make build`,
  and the `make tag` target) are replaced by the git-tag flow.
- **CodeQL and Scorecard are lower marginal value** on a small local-file app than
  on a networked one, but are included to match radiogogo's full process set and
  because Scorecard's pinning requirement is satisfied anyway.
