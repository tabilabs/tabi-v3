# tabid Release Process

## Goal

Establish a controlled binary release process for `tabid` so downstream deployment repos, offline genesis packages, and ops workflows no longer depend on ad hoc binaries built on a random connected workstation.

The first phase of this plan covers only:

- `linux/amd64`
- `tabid`
- GitHub Release
- `SHA256SUMS`
- `release-manifest.json`
- GitHub artifact attestation

## Merge Recommendation

Merge this release workflow into `main` with **Squash and merge**.

That keeps the `main` branch history to a single workflow-introduction commit and makes rollback a single `git revert` instead of a multi-commit cleanup.

## Why This Exists

If the binary is produced only on a personal workstation, even with a recorded commit and sha256, the process still does not answer these questions cleanly:

1. Did this binary come from an approved release flow?
2. Was it built by controlled CI from a specific commit?
3. Does the downstream boss package contain the exact same binary as the official release?

The purpose of this release flow is to turn those questions into verifiable facts.

## Artifacts

The workflow is expected to publish these files:

```text
tabid-linux-amd64
tabid-linux-amd64.sha256
SHA256SUMS
release-manifest.json
```

Notes:

- `SHA256SUMS` verifies file integrity.
- `release-manifest.json` records source repo, source ref, resolved source commit, release tag, workflow run id, Go version, platform, and artifact sha256.
- GitHub attestation proves that the published artifacts came from the official workflow in this repository.

## Triggering

The current workflow is manually triggered:

```text
.github/workflows/release-tabid.yml
```

Inputs:

- `source_ref`
- `release_tag`

Use a tag or a manually approved commit first. Do not auto-publish a formal release for every `main` push while the release process is still stabilizing.

## Build Strategy

The workflow intentionally follows the existing Linux build recipe already used by the repository:

- Install base build dependencies
- Download `libwasmvm_muslc.a`
- Download `libwasmvm152_muslc.a`
- Download `libwasmvm155_muslc.a`
- Build with:
  - `LEDGER_ENABLED=false`
  - `BUILD_TAGS=muslc`
  - `LINK_STATICALLY=true`
  - `make build`

This keeps the release process aligned with the repository's existing Docker/Linux build path instead of introducing a disconnected publishing script.

The `tabilabs/tabi-wasmd` release tag used for `libwasmvm152` and `libwasmvm155` is controlled through the workflow environment variable `TABIWASMD_RELEASE_TAG` rather than being embedded directly in multiple URLs.

## Local Verification

If you already have a release artifact set locally, run:

```bash
./scripts/verify-tabid-release-artifacts.sh \
  --binary ./dist/tabid-linux-amd64 \
  --checksums ./dist/SHA256SUMS \
  --manifest ./dist/release-manifest.json \
  --expected-repo tabilabs/tabi-v3 \
  --expected-tag v3.0.0-tabid.1 \
  --expected-platform linux/amd64
```

## Next Steps

Recommended order:

1. Get the `linux/amd64` release flow working end to end.
2. Update downstream boss-package tooling to consume only release artifacts.
3. Add `darwin/arm64` later.
4. If better offline audit ergonomics are needed, add `SHA256SUMS.sig` or a cosign bundle.

## Rollback

If the workflow, artifacts, or published release turn out to be wrong, follow:

- `docs/release/ROLLBACK.md`

## Operational Runbooks

- `docs/release/PRE_MERGE_CHECKLIST.md`
- `docs/release/FIRST_RELEASE_RUNBOOK.md`
