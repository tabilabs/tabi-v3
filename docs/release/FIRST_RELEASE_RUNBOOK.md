# tabid First Release Runbook

This runbook is for the first real execution of the `tabid` release workflow after it is merged into `main`.

## Goal

Publish the first controlled `tabid` binary release and verify that:

- the workflow completes successfully
- the release artifacts are internally consistent
- both supported release lanes are published
- the release can be consumed by downstream tooling

This first execution should use a non-production release tag.

## Recommended First Tag

Use a non-production tag such as:

- `v3.0.0-tabid.rc1`
- `v3.0.0-tabid.test1`

Do not use a production tag for the first ever run.

## Preconditions

Before starting:

- The workflow branch has already been merged into `main`.
- The merge was done with **Squash and merge**.
- The squash commit SHA on `main` has been recorded.
- `docs/release/ROLLBACK.md` has been reviewed.

## Step 1: Sync Local State

```bash
git checkout main
git pull --ff-only origin main
git log --oneline --decorate -n 5
```

Record the merge commit SHA or squash commit SHA that introduced the release workflow.

## Step 2: Trigger the Workflow

In GitHub Actions:

- Open the `release-tabid` workflow
- Click `Run workflow`
- Set:
  - `source_ref`: a reviewed commit or `main`
  - `release_tag`: a non-production tag

Suggested first run:

- `source_ref=main`
- `release_tag=v3.0.0-tabid.test1`

## Step 3: Watch the Workflow

The workflow should complete these major stages successfully:

1. checkout
2. Go setup
3. Linux dependency install
4. Linux bundle build and smoke test
5. macOS bundle build and smoke test
6. workflow artifact upload
7. build attestation
8. consolidated checksum assembly
9. GitHub Release publish

If any stage fails:

- do not retry blindly
- inspect the exact failing step
- use `docs/release/ROLLBACK.md` if the merge itself must be reverted

## Step 4: Verify Published Release

After workflow success, verify the GitHub Release contains:

- `tabid-linux-amd64-glibc.tar.gz`
- `tabid-linux-amd64-glibc.tar.gz.sha256`
- `tabid-darwin-arm64.tar.gz`
- `tabid-darwin-arm64.tar.gz.sha256`
- `SHA256SUMS`
- `release-manifest-linux-amd64.json`
- `release-manifest-darwin-arm64.json`

Confirm the release tag matches the intended test tag.

## Step 5: Verify Artifacts Locally

Download the artifacts locally and run:

```bash
./scripts/verify-tabid-release-artifacts.sh \
  --bundle ./tabid-linux-amd64-glibc.tar.gz \
  --checksums ./SHA256SUMS \
  --manifest ./release-manifest-linux-amd64.json \
  --expected-repo tabilabs/tabi-v3 \
  --expected-tag <release_tag> \
  --expected-platform linux/amd64
```

```bash
./scripts/verify-tabid-release-artifacts.sh \
  --bundle ./tabid-darwin-arm64.tar.gz \
  --checksums ./SHA256SUMS \
  --manifest ./release-manifest-darwin-arm64.json \
  --expected-repo tabilabs/tabi-v3 \
  --expected-tag <release_tag> \
  --expected-platform darwin/arm64
```

Expected outcome:

- checksum verification passes for both bundles
- bundle content verification passes for both bundles
- manifest verification passes for both bundles
- runtime smoke test passes on a matching host platform
- bundled wasm libraries resolve from the extracted `lib/` directory instead of another workstation path

Compatibility notes:

- The Linux release is a `linux/amd64` glibc bundle, not a fully static binary.
- Before handing it to operators, verify the target host provides `glibc >= 2.34` or run the verification script directly on the target class of machine.
- The macOS release is a `darwin/arm64` bundle only. Do not hand it to Intel Mac operators unless a separate `darwin/amd64` release lane exists.

## Step 6: Verify Attestation

Verify the GitHub artifact attestation using the organization’s normal verification process.

The exact verification command may vary depending on whether the team uses:

- GitHub UI verification
- GitHub CLI
- Sigstore tooling

The key requirement is that the release artifacts must be provably tied back to the workflow run that produced them.

## Step 7: Downstream Smoke Test

Before calling the release usable, run one downstream smoke test:

- take the published artifacts
- consume them from a deployment or packaging workflow
- confirm the downstream workflow accepts the pinned tag and sha256

This does not need to be a full production deployment, but it must prove that the release format is consumable.

## Success Criteria

The first release is considered successful only if all of the following are true:

- workflow completed successfully
- GitHub Release contains the expected artifacts
- local artifact verification passed
- attestation verification passed
- at least one downstream smoke test passed

## If Something Goes Wrong

If any of the following happens:

- artifacts are missing
- manifest fields are wrong
- checksums do not match
- attestation is missing or invalid
- downstream smoke test fails

then:

1. stop using that release tag
2. do not publish a production tag
3. use `docs/release/ROLLBACK.md` to decide whether to:
   - delete the bad release tag
   - revert the merge on `main`
   - fix forward on a new branch

Do not reuse a broken tag with different artifacts.
