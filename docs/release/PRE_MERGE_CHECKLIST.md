# tabid Release Pre-Merge Checklist

Use this checklist before merging the `tabid` release workflow branch into `main`.

## Branch and Review

- The branch was created from the current `main`.
- The PR contains only release-workflow-related files.
- The PR is reviewed by at least one engineer who understands:
  - the Tabi build recipe
  - GitHub Actions release behavior
  - downstream artifact consumers

## Scope Control

- The PR only introduces the current release phase:
  - `linux/amd64`
  - `darwin/arm64`
  - `tabid`
  - runnable runtime bundles
  - GitHub Release
  - `SHA256SUMS`
  - per-platform release manifests
  - GitHub artifact attestation
- The PR does not silently introduce auto-release on every `main` push.
- The workflow remains `workflow_dispatch` only unless there is explicit approval to broaden the trigger.

## Build Logic

- The workflow build steps match the documented release strategy:
  - `LEDGER_ENABLED=false`
  - `make build`
  - collect the three required wasm runtime libraries from the pinned module cache
  - bundle the runtime libraries under `./lib`
  - patch Linux runpath to `$ORIGIN/lib`
  - patch macOS `LC_RPATH` to `@executable_path/lib`
  - package a tarball bundle per platform
- The workflow produces:
  - `tabid-linux-amd64-glibc.tar.gz`
  - `tabid-linux-amd64-glibc.tar.gz.sha256`
  - `tabid-darwin-arm64.tar.gz`
  - `tabid-darwin-arm64.tar.gz.sha256`
  - `SHA256SUMS`
  - `release-manifest-linux-amd64.json`
  - `release-manifest-darwin-arm64.json`

## Metadata and Verification

- Each release manifest uses a resolved `source_commit`, not an ambiguous commit field.
- The manifest schema and example file match.
- `scripts/verify-tabid-release-artifacts.sh` verifies:
  - artifact checksum
  - bundle contents for the selected platform tarball
  - manifest fields
  - expected repo/tag/platform
  - runtime resolution from the extracted bundle `lib/` directory when the local host matches the bundle platform

## Merge Strategy

- The intended merge method is **Squash and merge**.
- The team understands that the squash commit SHA on `main` is the rollback anchor.
- The rollback path in `docs/release/ROLLBACK.md` has been reviewed before merge.

## Release Readiness

- A test release tag format has been agreed, for example:
  - `v3.0.0-tabid.rc1`
  - `v3.0.0-tabid.test1`
- The first run will use a non-production tag.
- Downstream consumers will not point to `latest`.
- Downstream consumers will pin:
  - release tag
  - expected sha256
  - expected platform
- Release docs clearly state that:
  - the Linux artifact is a glibc bundle and requires a compatible host runtime
  - the macOS artifact is `darwin/arm64` only and should not be treated as universal for all Macs

## Final Go/No-Go

Do not merge until all items above are true.

After merge, continue with:

- `docs/release/FIRST_RELEASE_RUNBOOK.md`
