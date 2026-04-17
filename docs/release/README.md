# tabid Release Process

## Goal

Establish a controlled binary release process for `tabid` so downstream deployment repos, offline genesis packages, and ops workflows no longer depend on ad hoc binaries built on a random connected workstation.

The current release flow covers:

- `linux/amd64`
- `darwin/arm64`
- `tabid`
- runnable runtime bundles
- GitHub Release
- `SHA256SUMS`
- per-platform `release-manifest-<platform>.json`
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
tabid-darwin-arm64.tar.gz
tabid-darwin-arm64.tar.gz.sha256
SHA256SUMS
release-manifest-linux-amd64.json
release-manifest-darwin-arm64.json
```

Notes:

- `tabid-linux-amd64-glibc.tar.gz` expands to a runnable Linux bundle containing `tabid` and the required `.so` files under `lib/`.
- `tabid-darwin-arm64.tar.gz` expands to a runnable macOS bundle containing `tabid` and the required `.dylib` files under `lib/`.
- `SHA256SUMS` verifies the tarball artifacts for every release lane.
- Each release manifest records source repo, source ref, resolved source commit, release tag, workflow run id, Go version, platform, and artifact sha256 for one release bundle.
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

The workflow intentionally publishes runnable bundles rather than a bare binary.

Shared build rules:

- `LEDGER_ENABLED=false`
- `make build`
- collect the required wasm runtime libraries from the resolved Go module cache
- bundle `tabid` together with its runtime libraries under `./lib`
- produce a per-artifact `.sha256` sidecar and a per-platform release manifest

### linux/amd64 lane

The Linux lane publishes `tabid-linux-amd64-glibc.tar.gz`:

- install base build dependencies
- collect these pinned runtime libraries:
  - `libwasmvm.x86_64.so`
  - `libwasmvm152.x86_64.so`
  - `libwasmvm155.x86_64.so`
- rewrite the binary runpath to `$ORIGIN/lib`
- smoke test the extracted bundle with `readelf`, `ldd`, and `tabid version`

Compatibility boundary:

- This artifact is not a fully static Linux binary. The target host still needs a compatible `glibc`.
- The current `linux/amd64` build has been observed to require `GLIBC_2.34` symbols from the host C runtime.
- Treat this release lane as `linux/amd64 + glibc >= 2.34` until a lower baseline is intentionally built and verified.
- Downstream deployment docs should tell operators to validate the target host before rollout instead of assuming all `linux/amd64` machines are compatible.

Bundle layout:

```text
tabid-linux-amd64-glibc/
  tabid
  lib/
    libwasmvm.x86_64.so
    libwasmvm152.x86_64.so
    libwasmvm155.x86_64.so
```

### darwin/arm64 lane

The macOS lane publishes `tabid-darwin-arm64.tar.gz`:

- build on a macOS runner
- collect these pinned runtime libraries:
  - `libwasmvm.dylib`
  - `libwasmvm152.dylib`
  - `libwasmvm155.dylib`
- remove any absolute `LC_RPATH` entries from the built binary
- add `@executable_path/lib`
- ad-hoc sign the bundled dylibs and `tabid`
- smoke test the extracted bundle with `otool`, `codesign`, and `tabid version`

Compatibility boundary:

- This artifact targets Apple Silicon machines only.
- Treat this release lane as `darwin/arm64` until a separate `darwin/amd64` lane is intentionally built and verified.
- The bundle is expected to run after extraction without depending on libraries from the original Go module cache or another workstation path.

Bundle layout:

```text
tabid-darwin-arm64/
  tabid
  lib/
    libwasmvm.dylib
    libwasmvm152.dylib
    libwasmvm155.dylib
```

This keeps the release process aligned with the code that is actually pinned by `go.mod`, rather than depending on unpublished external release assets.

## Local Verification

If you already have a release artifact set locally, verify the Linux bundle with:

```bash
./scripts/verify-tabid-release-artifacts.sh \
  --bundle ./tabid-linux-amd64-glibc.tar.gz \
  --checksums ./SHA256SUMS \
  --manifest ./release-manifest-linux-amd64.json \
  --expected-repo tabilabs/tabi-v3 \
  --expected-tag v3.0.0-tabid.1 \
  --expected-platform linux/amd64
```

Verify the macOS bundle with:

```bash
./scripts/verify-tabid-release-artifacts.sh \
  --bundle ./tabid-darwin-arm64.tar.gz \
  --checksums ./SHA256SUMS \
  --manifest ./release-manifest-darwin-arm64.json \
  --expected-repo tabilabs/tabi-v3 \
  --expected-tag v3.0.0-tabid.1 \
  --expected-platform darwin/arm64
```

The verification script checks:

- artifact checksum
- bundle layout for the selected platform
- manifest fields
- a runtime smoke test when the local host matches the bundle platform
- that the bundled wasm runtime libraries are resolved from the extracted `./lib` directory instead of another host path

## Next Steps

Recommended order:

- Get the `linux/amd64` and `darwin/arm64` bundle release flow working end to end.
- Update downstream boss-package tooling to consume a pinned GitHub Release artifact instead of a workstation build.
- If `darwin/amd64` is needed later, add it as a separate release lane with its own verification instead of assuming the arm64 bundle is universal.
- If a proper musl static release chain is published upstream in the future, add a separate static release lane instead of replacing the glibc bundle blindly.
- If better offline audit ergonomics are needed, add `SHA256SUMS.sig` or a cosign bundle.

## Rollback

If the workflow, artifacts, or published release turn out to be wrong, follow:

- `docs/release/ROLLBACK.md`

## Operational Runbooks

- `docs/release/PRE_MERGE_CHECKLIST.md`
- `docs/release/FIRST_RELEASE_RUNBOOK.md`
