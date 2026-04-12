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

- The PR only introduces the first release phase:
  - `linux/amd64`
  - `tabid`
  - glibc runtime bundle
  - GitHub Release
  - `SHA256SUMS`
  - `release-manifest.json`
  - GitHub artifact attestation
- The PR does not silently introduce auto-release on every `main` push.
- The workflow remains `workflow_dispatch` only unless there is explicit approval to broaden the trigger.

## Build Logic

- The workflow build steps match the documented release strategy:
  - `LEDGER_ENABLED=false`
  - `make build`
  - collect the three required glibc `.so` files from the pinned module cache
  - patch the binary runpath to `$ORIGIN/lib`
  - package a tarball bundle
- The workflow produces:
  - `tabid-linux-amd64-glibc.tar.gz`
  - `tabid-linux-amd64-glibc.tar.gz.sha256`
  - `SHA256SUMS`
  - `release-manifest.json`

## Metadata and Verification

- `release-manifest.json` uses a resolved `source_commit`, not an ambiguous commit field.
- The manifest schema and example file match.
- `scripts/verify-tabid-release-artifacts.sh` verifies:
  - artifact checksum
  - bundle contents for the glibc tarball
  - manifest fields
  - expected repo/tag/platform
  - linux/x86_64 runtime resolution from the extracted bundle `lib/` directory

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
- Release docs clearly state that the current artifact is a glibc bundle and requires a compatible host runtime instead of claiming universal `linux/amd64` compatibility.

## Final Go/No-Go

Do not merge until all items above are true.

After merge, continue with:

- `docs/release/FIRST_RELEASE_RUNBOOK.md`
