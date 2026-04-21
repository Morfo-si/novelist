# Novelist

Capture your thoughts and ideas for your next book or story from the command line.

![Alt Novelist screenshot](https://github.com/Morfo-si/novelist/assets/53362/64e81990-40f7-42b7-bc4e-2012d92ad599 "Novelist")

## Installation

### Binary

Grab the latest [binary](https://github.com/Morfo-si/novelist/releases) for your system.

### Go

Just install it with `go`:

```bash
go install github.com/Morfo-si/novelist@latest
```

### Build (requires Go 1.20+)

```bash
git clone https://github.com/Morfo-si/novelist.git
cd novelist
go build
```

Or, to build with the version string injected from the `VERSION` file:

```bash
make build
./novelist --version
```

## Releasing

Releases are fully automated through GitHub Actions and [GoReleaser](https://goreleaser.com). Pushing a `v*` tag triggers a workflow that cross-compiles binaries for Linux, macOS and Windows (amd64 + arm64), uploads them to a new GitHub Release, and generates a changelog from the commits since the previous tag.

### Cutting a release

1. **Bump the version.** Edit the [`VERSION`](./VERSION) file — it holds a single semver string (e.g. `0.0.7`). Do not prefix it with `v`; the `Makefile` and workflow add that for you.
2. **Commit and push to `main`.**

   ```bash
   git add VERSION
   git commit -m "chore: bump version to $(cat VERSION)"
   git push origin main
   ```

   Wait for the **PR Checks** workflow to go green.
3. **Tag and push.**

   ```bash
   make tag
   ```

   This runs `git tag -a v$(cat VERSION) -m "Release v$(cat VERSION)"` and pushes the tag to `origin`. The **Release** workflow fires on the tag push and publishes the release within a couple of minutes.

### Pre-releases

Any tag containing a pre-release identifier (e.g. `v0.1.0-beta.1`, `v1.0.0-rc.1`) is automatically marked as a pre-release on GitHub. To cut one, set `VERSION` to `0.1.0-beta.1` and run the same flow.

### Dry-running the release locally

Before tagging, you can exercise the entire GoReleaser pipeline locally without publishing anything:

```bash
make snapshot
```

This runs `goreleaser release --snapshot --clean --skip=publish`. Artifacts land in `./dist/` so you can inspect the archives, checksums and filenames.

### What gets published

Each release contains:

- Binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64` (tar.gz for Unix, zip for Windows).
- `checksums.txt` — SHA-256 sums for every archive.
- Auto-generated changelog (commits prefixed with `chore:`, `docs:`, `test:` and merge commits are filtered out).

### Commit message hints

The changelog is built from commit subjects. To keep releases readable, prefer [Conventional Commits](https://www.conventionalcommits.org/) style prefixes — `feat:`, `fix:`, `refactor:` show up in the changelog; `chore:`, `docs:`, `test:` are hidden.
