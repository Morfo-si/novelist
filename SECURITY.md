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
