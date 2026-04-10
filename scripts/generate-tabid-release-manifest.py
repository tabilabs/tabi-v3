#!/usr/bin/env python3

import argparse
import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def go_version() -> str:
    import subprocess
    return subprocess.check_output(["go", "version"], text=True).strip()


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate tabid release-manifest.json")
    parser.add_argument("--artifact", required=True, help="Artifact path, for example ./dist/tabid-linux-amd64")
    parser.add_argument("--release-tag", required=True, help="release tag")
    parser.add_argument("--source-ref", required=True, help="Source ref used for the build")
    parser.add_argument("--source-commit", required=True, help="Resolved source commit used for the build")
    parser.add_argument("--workflow", required=True, help="Workflow name")
    parser.add_argument("--workflow-run-id", required=True, help="Workflow run id")
    parser.add_argument("--workflow-run-attempt", required=True, help="Workflow run attempt")
    parser.add_argument("--source-repo", required=True, help="github.repository")
    parser.add_argument("--platform", required=True, help="Target platform, for example linux/amd64")
    parser.add_argument("--checksum-file", default="SHA256SUMS", help="Checksum file name")
    parser.add_argument("--out", required=True, help="Output path")
    args = parser.parse_args()

    artifact = Path(args.artifact)
    if not artifact.is_file():
        raise SystemExit(f"artifact does not exist: {artifact}")

    payload = {
        "source_repo": args.source_repo,
        "source_ref": args.source_ref,
        "source_commit": args.source_commit,
        "release_tag": args.release_tag,
        "workflow": args.workflow,
        "workflow_run_id": args.workflow_run_id,
        "workflow_run_attempt": args.workflow_run_attempt,
        "manifest_generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "go_version": go_version(),
        "platform": args.platform,
        "artifact_name": artifact.name,
        "artifact_sha256": sha256_file(artifact),
        "checksum_file": args.checksum_file,
    }

    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(payload, indent=2) + "\n")


if __name__ == "__main__":
    main()
