# tabid Release Rollback Runbook

## Goal

This document defines how to roll back the `tabid` release workflow safely if the merged workflow, generated artifacts, or published GitHub Release turn out to be incorrect.

The rollback goal is:

1. Stop producing new incorrect release artifacts.
2. Remove or invalidate incorrect release artifacts that were already published.
3. Return downstream consumers to the last known-good release.

## Recommended Merge Strategy

Use **Squash and merge** when merging the release workflow branch into `main`.

Reason:

- `main` will receive a single commit instead of multiple workflow-setup commits.
- Rollback becomes a single `git revert` on `main`.
- Audit history stays cleaner.

If the workflow branch is merged with a merge commit instead of squash, rollback is still possible, but the revert command is slightly less clean.

## Rollback Scenarios

### Scenario 1: PR not merged yet

Action:

- Do not merge.
- Close the PR or push fixes to the branch.

No repository rollback is needed.

### Scenario 2: Merged to `main`, but no release was published

Symptoms:

- The workflow file exists on `main`.
- No GitHub Release tag was created.
- No downstream consumer has used the new artifacts yet.

Action:

1. Revert the merge on `main`.
2. Push the revert commit.

If the PR was squash-merged:

```bash
git checkout main
git pull --ff-only origin main
git revert <squash_commit_sha>
git push origin main
```

If the PR was merged as a merge commit:

```bash
git checkout main
git pull --ff-only origin main
git revert -m 1 <merge_commit_sha>
git push origin main
```

### Scenario 3: Merged to `main`, and a bad GitHub Release was published

Symptoms:

- A release tag exists.
- Uploaded artifacts are wrong, incomplete, or unverifiable.
- The workflow itself should be disabled or reverted.

Action order:

1. Revert the workflow commit on `main`.
2. Delete the incorrect GitHub Release.
3. Delete the incorrect Git tag if it should not be reused.
4. Regenerate artifacts from a fixed branch or fall back to the previous known-good release.

Repository rollback:

```bash
git checkout main
git pull --ff-only origin main
git revert <squash_commit_sha>
git push origin main
```

Release cleanup:

```bash
gh release delete <release_tag> --yes
git push origin :refs/tags/<release_tag>
```

If the local tag also exists:

```bash
git tag -d <release_tag>
```

### Scenario 4: A bad release was already consumed downstream

Symptoms:

- A deployment repo, packaging repo, or boss package already used the bad release.
- Even if the workflow is reverted, downstream consumers still point at the bad tag.

Action order:

1. Complete Scenario 3.
2. Update downstream consumers to the previous known-good release tag.
3. Rebuild or repackage any downstream artifacts that referenced the bad release.
4. Re-run downstream verification before reuse.

For downstream consumers, the key rule is:

- do not point to `latest`
- pin a specific known-good release tag

## Operational Checklist

When rolling back, verify all of the following:

- The problematic workflow commit is reverted on `main`.
- The problematic release tag no longer exists, or is clearly marked invalid and unused.
- The last known-good release tag is still available.
- Downstream repos and packaging logic are pinned back to the known-good release.
- Local documentation and incident notes record:
  - what failed
  - which tag was affected
  - which commit was reverted
  - which known-good tag was restored

## Recommended Known-Good Pinning Policy

To make rollback practical, downstream repos should always store:

- release tag
- artifact sha256
- expected platform

That way rollback becomes:

1. switch the release tag back
2. switch the expected sha256 back
3. regenerate downstream package

without needing to guess which artifact was previously trusted.

## After the Rollback

After rollback is complete:

1. open a fix branch
2. reproduce the issue
3. patch the workflow or scripts
4. validate again in a non-production tag
5. only then publish a new release tag

Do not reuse the broken tag for a different artifact payload.
