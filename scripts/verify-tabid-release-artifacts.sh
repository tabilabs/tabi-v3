#!/usr/bin/env bash

set -euo pipefail
set +x

BINARY=""
CHECKSUMS=""
MANIFEST=""
EXPECTED_REPO=""
EXPECTED_TAG=""
EXPECTED_PLATFORM=""

log() { printf '[INFO] %s\n' "$*"; }
ok() { printf '[OK] %s\n' "$*"; }
die() { printf '[ERROR] %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Usage:
  ./scripts/verify-tabid-release-artifacts.sh --binary <path> --checksums <path> [options]

Options:
  --binary <path>             Path to the tabid binary
  --checksums <path>          Path to SHA256SUMS
  --manifest <path>           Path to release-manifest.json
  --expected-repo <value>     Expected source_repo value
  --expected-tag <value>      Expected release_tag value
  --expected-platform <value> Expected platform value, for example linux/amd64
  --help                      Show help
EOF
}

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{print $1}'
    return 0
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${file}" | awk '{print $1}'
    return 0
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "${file}" | awk '{print $2}'
    return 0
  fi
  die "missing sha256 tool; need sha256sum, shasum, or openssl"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --binary) BINARY="$2"; shift 2 ;;
      --checksums) CHECKSUMS="$2"; shift 2 ;;
      --manifest) MANIFEST="$2"; shift 2 ;;
      --expected-repo) EXPECTED_REPO="$2"; shift 2 ;;
      --expected-tag) EXPECTED_TAG="$2"; shift 2 ;;
      --expected-platform) EXPECTED_PLATFORM="$2"; shift 2 ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        die "unknown argument: $1"
        ;;
    esac
  done
}

verify_inputs() {
  [[ -n "${BINARY}" ]] || die "missing --binary"
  [[ -n "${CHECKSUMS}" ]] || die "missing --checksums"
  [[ -f "${BINARY}" ]] || die "binary does not exist: ${BINARY}"
  [[ -f "${CHECKSUMS}" ]] || die "SHA256SUMS does not exist: ${CHECKSUMS}"
  if [[ -n "${MANIFEST}" ]]; then
    [[ -f "${MANIFEST}" ]] || die "manifest does not exist: ${MANIFEST}"
  fi
  command -v python3 >/dev/null 2>&1 || die "missing required command: python3"
}

verify_checksums() {
  local actual_sha expected_sha binary_name

  binary_name="$(basename "${BINARY}")"
  actual_sha="$(sha256_file "${BINARY}")"
  expected_sha="$(python3 - "${CHECKSUMS}" "${binary_name}" <<'PY'
import sys
from pathlib import Path

checksums = Path(sys.argv[1]).read_text().splitlines()
needle = sys.argv[2]

for line in checksums:
    line = line.strip()
    if not line:
        continue
    parts = line.split()
    if len(parts) < 2:
        continue
    name = parts[-1]
    if name.startswith("./"):
        name = name[2:]
    if name == needle:
        print(parts[0])
        raise SystemExit(0)

raise SystemExit(1)
PY
)"

  [[ -n "${expected_sha}" ]] || die "did not find ${binary_name} in SHA256SUMS"
  [[ "${actual_sha}" == "${expected_sha}" ]] || die "binary sha256 mismatch: actual=${actual_sha}, expected=${expected_sha}"
  ok "SHA256SUMS check passed: ${binary_name} sha256=${actual_sha}"
}

verify_manifest() {
  [[ -n "${MANIFEST}" ]] || return 0

  local actual_sha binary_name
  actual_sha="$(sha256_file "${BINARY}")"
  binary_name="$(basename "${BINARY}")"

  python3 - "${MANIFEST}" "${binary_name}" "${actual_sha}" "${EXPECTED_REPO}" "${EXPECTED_TAG}" "${EXPECTED_PLATFORM}" <<'PY'
import json
import sys
from pathlib import Path

manifest_path, binary_name, actual_sha, expected_repo, expected_tag, expected_platform = sys.argv[1:]
data = json.loads(Path(manifest_path).read_text())

required = [
    "source_repo",
    "source_commit",
    "release_tag",
    "platform",
    "artifact_name",
    "artifact_sha256",
]
for key in required:
    if not data.get(key):
        raise SystemExit(f"manifest is missing required field: {key}")

if data["artifact_name"] != binary_name:
    raise SystemExit(f"manifest artifact_name mismatch: {data['artifact_name']} != {binary_name}")
if data["artifact_sha256"] != actual_sha:
    raise SystemExit(f"manifest artifact_sha256 mismatch: {data['artifact_sha256']} != {actual_sha}")
if expected_repo and data["source_repo"] != expected_repo:
    raise SystemExit(f"manifest source_repo mismatch: {data['source_repo']} != {expected_repo}")
if expected_tag and data["release_tag"] != expected_tag:
    raise SystemExit(f"manifest release_tag mismatch: {data['release_tag']} != {expected_tag}")
if expected_platform and data["platform"] != expected_platform:
    raise SystemExit(f"manifest platform mismatch: {data['platform']} != {expected_platform}")

print(json.dumps({
    "source_repo": data["source_repo"],
    "source_commit": data["source_commit"],
    "release_tag": data["release_tag"],
    "platform": data["platform"],
}, ensure_ascii=False))
PY
}

main() {
  local summary=""
  parse_args "$@"
  verify_inputs
  verify_checksums

  if [[ -n "${MANIFEST}" ]]; then
    summary="$(verify_manifest)"
    ok "release manifest check passed: ${summary}"
  else
    log "no manifest provided; only SHA256SUMS was verified"
  fi

  ok "tabid release artifact verification completed successfully"
}

main "$@"
