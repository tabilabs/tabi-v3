# tabid Release Process

## Goal

Establish a controlled binary release process for `tabid` so downstream deployment repos, offline genesis packages, and ops workflows no longer depend on ad hoc binaries built on a random connected workstation.

The first phase of this plan covers only:

- `linux/amd64`
- `tabid`
- glibc runtime bundle
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
tabid-linux-amd64-glibc.tar.gz
tabid-linux-amd64-glibc.tar.gz.sha256
SHA256SUMS
release-manifest.json
```

Notes:

- `tabid-linux-amd64-glibc.tar.gz` expands to a runnable bundle containing `tabid` and the required `.so` files under `lib/`.
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

The workflow intentionally publishes a runnable `linux/amd64` glibc bundle:

- Install base build dependencies
- Build `tabid` with:
  - `LEDGER_ENABLED=false`
  - `make build`
- Collect these pinned runtime libraries from the resolved Go module cache:
  - `libwasmvm.x86_64.so`
  - `libwasmvm152.x86_64.so`
  - `libwasmvm155.x86_64.so`
- Rewrite the binary runpath to `$ORIGIN/lib`
- Package the bundle as `tabid-linux-amd64-glibc.tar.gz`

Rationale:

- The current source tree requires three wasm runtime libraries at link/runtime time.
- The musl static release chain for `v152` and `v155` is not published by the current upstream dependency chain, so a static single-file release cannot yet be produced reproducibly in CI.
- The glibc bundle keeps the release flow automated and produces a self-contained artifact that downstream tooling can verify and execute.

Compatibility boundary:

- This artifact is not a fully static Linux binary. The target host still needs a compatible `glibc`.
- The current `linux/amd64` build has been observed to require `GLIBC_2.34` symbols from the host C runtime.
- Treat this release lane as `linux/amd64 + glibc >= 2.34` until a lower baseline is intentionally built and verified.
- Downstream deployment docs should tell operators to validate the target host before rollout instead of assuming all `linux/amd64` machines are compatible.

This keeps the release process aligned with the code that is actually pinned by `go.mod`, rather than depending on unpublished external release assets.

Bundle layout:

```text
tabid-linux-amd64-glibc/
  tabid
  lib/
    libwasmvm.x86_64.so
    libwasmvm152.x86_64.so
    libwasmvm155.x86_64.so
```

The binary is patched so it resolves those shared libraries from `./lib` after extraction.

## Local Verification

If you already have a release artifact set locally, run:

```bash
./scripts/verify-tabid-release-artifacts.sh \
  --bundle ./dist/tabid-linux-amd64-glibc.tar.gz \
  --checksums ./dist/SHA256SUMS \
  --manifest ./dist/release-manifest.json \
  --expected-repo tabilabs/tabi-v3 \
  --expected-tag v3.0.0-tabid.1 \
  --expected-platform linux/amd64
```

The verification script checks:

- artifact checksum
- bundle layout
- manifest fields
- a runtime smoke test on `linux/x86_64` hosts
- that the bundled wasm shared libraries are resolved from the extracted `./lib` directory, not from the local Go module cache or another host path

## Next Steps

Recommended order:

- Get the `linux/amd64` glibc bundle release flow working end to end.
- Update downstream boss-package tooling to consume the bundle release artifact.
- Add `darwin/arm64` later.
- If a proper musl static release chain is published upstream in the future, add a separate static release lane instead of replacing the glibc bundle blindly.
- If better offline audit ergonomics are needed, add `SHA256SUMS.sig` or a cosign bundle.

## Rollback

If the workflow, artifacts, or published release turn out to be wrong, follow:

- `docs/release/ROLLBACK.md`

## Operational Runbooks

- `docs/release/PRE_MERGE_CHECKLIST.md`
- `docs/release/FIRST_RELEASE_RUNBOOK.md`
