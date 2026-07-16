# Novelist

[![CI](https://github.com/Morfo-si/novelist/actions/workflows/ci.yml/badge.svg)](https://github.com/Morfo-si/novelist/actions/workflows/ci.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/Morfo-si/novelist/badge)](https://scorecard.dev/viewer/?uri=github.com/Morfo-si/novelist)

Capture your thoughts and ideas for your next book or story from the command line.

![Alt Novelist screenshot](https://github.com/Morfo-si/novelist/assets/53362/64e81990-40f7-42b7-bc4e-2012d92ad599 "Novelist")

## Installation

### Binary

Grab the latest [binary](https://github.com/Morfo-si/novelist/releases) for your system.

### Go

Just install it with `go`:

```bash
go install github.com/Morfo-si/novelist/cmd/novelist@latest
```

(The install path is `.../cmd/novelist` as of v0.0.8; older `.../novelist@latest` no longer resolves.)

### Build

```bash
git clone https://github.com/Morfo-si/novelist.git
cd novelist
make build          # stamps the version from the current git tag
./dist/novelist --version
```

## Releasing

Releases are fully automated through GitHub Actions and [GoReleaser](https://goreleaser.com). Pushing a `v*` tag triggers a workflow that cross-compiles binaries for Linux, macOS and Windows (amd64 + arm64), uploads them to a new GitHub Release, and generates a changelog from the commits since the previous tag.

### Cutting a release

There is no `VERSION` file to bump — the version is derived entirely from the git tag you push.

1. **Make sure `main` is green and your working tree is clean.**
2. **Tag and push.**

   ```bash
   make release TAG=v0.0.8
   ```

   This tags the current commit (`git tag -a v0.0.8 -m "Release v0.0.8"`) and pushes the tag to `origin`. The **Release** workflow fires on the tag push and publishes the release within a couple of minutes.

### Pre-releases

Any tag containing a pre-release identifier (e.g. `v0.1.0-beta.1`, `v1.0.0-rc.1`) is automatically marked as a pre-release on GitHub. To cut one, run `make release TAG=v0.1.0-beta.1`.

### Dry-running the release locally

Before tagging, you can exercise the entire GoReleaser pipeline locally without publishing anything:

```bash
make release-snapshot
```

This runs `goreleaser release --snapshot --clean --skip=publish,sign`. Artifacts land in `./dist/` so you can inspect the archives, checksums and filenames.

### What gets published

Each release contains:

- Binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64` (tar.gz for Unix, zip for Windows).
- `checksums.txt` — SHA-256 sums for every archive.
- Auto-generated changelog (commits prefixed with `chore:`, `docs:`, `test:` and merge commits are filtered out).

### Commit message hints

The changelog is built from commit subjects. To keep releases readable, prefer [Conventional Commits](https://www.conventionalcommits.org/) style prefixes — `feat:`, `fix:`, `refactor:` show up in the changelog; `chore:`, `docs:`, `test:` are hidden.
