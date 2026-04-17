#!/usr/bin/env bash

set -euo pipefail
set +x

export LC_ALL="${LC_ALL:-C}"
export LANG="${LANG:-C}"

BINARY=""
BUNDLE=""
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
  ./scripts/verify-tabid-release-artifacts.sh (--binary <path> | --bundle <path>) --checksums <path> [options]

Options:
  --binary <path>             Path to the tabid binary
  --bundle <path>             Path to a release bundle tar.gz
  --checksums <path>          Path to SHA256SUMS
  --manifest <path>           Path to a release manifest json
  --expected-repo <value>     Expected source_repo value
  --expected-tag <value>      Expected release_tag value
  --expected-platform <value> Expected platform value, for example linux/amd64 or darwin/arm64
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

bundle_platform_from_name() {
  local artifact_name="$1"
  case "${artifact_name}" in
    tabid-linux-amd64-glibc.tar.gz) printf 'linux/amd64\n' ;;
    tabid-darwin-arm64.tar.gz) printf 'darwin/arm64\n' ;;
    *) printf '\n' ;;
  esac
}

resolve_bundle_platform() {
  local artifact_name="$1"
  local inferred
  inferred="$(bundle_platform_from_name "${artifact_name}")"

  if [[ -n "${EXPECTED_PLATFORM}" && -n "${inferred}" && "${EXPECTED_PLATFORM}" != "${inferred}" ]]; then
    die "bundle platform mismatch: inferred=${inferred}, expected=${EXPECTED_PLATFORM}"
  fi
  if [[ -n "${EXPECTED_PLATFORM}" ]]; then
    printf '%s\n' "${EXPECTED_PLATFORM}"
    return 0
  fi
  [[ -n "${inferred}" ]] || die "could not infer bundle platform from artifact name: ${artifact_name}; pass --expected-platform"
  printf '%s\n' "${inferred}"
}

bundle_libs_for_platform() {
  local platform="$1"
  case "${platform}" in
    linux/amd64)
      cat <<'EOF'
libwasmvm.x86_64.so
libwasmvm152.x86_64.so
libwasmvm155.x86_64.so
EOF
      ;;
    darwin/arm64)
      cat <<'EOF'
libwasmvm.dylib
libwasmvm152.dylib
libwasmvm155.dylib
EOF
      ;;
    *)
      die "unsupported platform: ${platform}"
      ;;
  esac
}

host_matches_platform() {
  local platform="$1"
  case "${platform}" in
    linux/amd64)
      [[ "$(uname -s)" == "Linux" && "$(uname -m)" == "x86_64" ]]
      ;;
    darwin/arm64)
      [[ "$(uname -s)" == "Darwin" && "$(uname -m)" == "arm64" ]]
      ;;
    *)
      return 1
      ;;
  esac
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --binary) BINARY="$2"; shift 2 ;;
      --bundle) BUNDLE="$2"; shift 2 ;;
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
  if [[ -n "${BINARY}" && -n "${BUNDLE}" ]]; then
    die "use either --binary or --bundle, not both"
  fi
  if [[ -z "${BINARY}" && -z "${BUNDLE}" ]]; then
    die "missing --binary or --bundle"
  fi
  [[ -n "${CHECKSUMS}" ]] || die "missing --checksums"

  if [[ -n "${BINARY}" ]]; then
    [[ -f "${BINARY}" ]] || die "binary does not exist: ${BINARY}"
  fi
  if [[ -n "${BUNDLE}" ]]; then
    [[ -f "${BUNDLE}" ]] || die "bundle does not exist: ${BUNDLE}"
    command -v tar >/dev/null 2>&1 || die "missing required command: tar"
  fi

  [[ -f "${CHECKSUMS}" ]] || die "SHA256SUMS does not exist: ${CHECKSUMS}"
  if [[ -n "${MANIFEST}" ]]; then
    [[ -f "${MANIFEST}" ]] || die "manifest does not exist: ${MANIFEST}"
  fi
  command -v python3 >/dev/null 2>&1 || die "missing required command: python3"

  if [[ "$(uname -s)" == "Linux" && "$(uname -m)" == "x86_64" ]]; then
    command -v ldd >/dev/null 2>&1 || die "missing required command on linux/x86_64: ldd"
    command -v readelf >/dev/null 2>&1 || die "missing required command on linux/x86_64: readelf"
  fi
  if [[ "$(uname -s)" == "Darwin" && "$(uname -m)" == "arm64" ]]; then
    command -v otool >/dev/null 2>&1 || die "missing required command on darwin/arm64: otool"
    command -v codesign >/dev/null 2>&1 || die "missing required command on darwin/arm64: codesign"
  fi
}

verify_checksums() {
  local actual_sha expected_sha artifact_path artifact_name

  artifact_path="${BINARY:-${BUNDLE}}"
  artifact_name="$(basename "${artifact_path}")"
  actual_sha="$(sha256_file "${artifact_path}")"
  expected_sha="$(python3 - "${CHECKSUMS}" "${artifact_name}" <<'PY'
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
    if name == needle or Path(name).name == needle:
        print(parts[0])
        raise SystemExit(0)

raise SystemExit(1)
PY
)"

  [[ -n "${expected_sha}" ]] || die "did not find ${artifact_name} in SHA256SUMS"
  [[ "${actual_sha}" == "${expected_sha}" ]] || die "artifact sha256 mismatch: actual=${actual_sha}, expected=${expected_sha}"
  ok "SHA256SUMS check passed: ${artifact_name} sha256=${actual_sha}"
}

verify_linux_runtime() {
  local bundle_root="$1"
  local tmpdir="$2"
  local runpath ldd_output actual_path expected_path home_dir lib_name

  runpath="$(readelf -d "${bundle_root}/tabid" | awk '/(RPATH|RUNPATH)/ { print $NF }')"
  [[ -n "${runpath}" ]] || die "bundle tabid does not contain an RPATH/RUNPATH entry"
  [[ "${runpath}" == *'$ORIGIN/lib'* ]] || die "bundle tabid RPATH/RUNPATH does not include \$ORIGIN/lib: ${runpath}"

  ldd_output="$(ldd "${bundle_root}/tabid")"
  while IFS= read -r lib_name; do
    expected_path="${bundle_root}/lib/${lib_name}"
    actual_path="$(printf '%s\n' "${ldd_output}" | awk -v lib="${lib_name}" '$1 == lib { print $3 }')"
    [[ -n "${actual_path}" ]] || die "ldd did not resolve ${lib_name}"
    [[ "${actual_path}" == "${expected_path}" ]] || die "ldd resolved ${lib_name} outside the bundle: ${actual_path}"
  done < <(bundle_libs_for_platform "linux/amd64")

  home_dir="${tmpdir}/home"
  mkdir -p "${home_dir}"
  if env -i PATH="${PATH}" HOME="${home_dir}" LD_LIBRARY_PATH= "${bundle_root}/tabid" version >/dev/null 2>&1; then
    ok "bundle runtime smoke test passed with bundle-local shared libraries"
  else
    die "bundle runtime smoke test failed"
  fi
}

verify_darwin_runtime() {
  local bundle_root="$1"
  local tmpdir="$2"
  local rpaths home_dir

  rpaths="$(otool -l "${bundle_root}/tabid" | awk '/cmd LC_RPATH/{getline; getline; if ($1 == "path") print $2}')"
  [[ -n "${rpaths}" ]] || die "bundle tabid does not contain an LC_RPATH entry"
  printf '%s\n' "${rpaths}" | grep -Fx '@executable_path/lib' >/dev/null || die "bundle tabid LC_RPATH does not include @executable_path/lib"
  if printf '%s\n' "${rpaths}" | grep -Eq '^/'; then
    die "bundle tabid still contains an absolute LC_RPATH entry: ${rpaths}"
  fi

  codesign --verify --deep --strict "${bundle_root}/tabid" >/dev/null 2>&1 || die "codesign verification failed for bundle tabid"
  home_dir="${tmpdir}/home"
  mkdir -p "${home_dir}"
  if env -i PATH="${PATH}" HOME="${home_dir}" "${bundle_root}/tabid" version >/dev/null 2>&1; then
    ok "bundle runtime smoke test passed with bundle-local dylibs"
  else
    die "bundle runtime smoke test failed"
  fi
}

verify_bundle_contents() {
  [[ -n "${BUNDLE}" ]] || return 0

  local tmpdir bundle_name bundle_root platform lib_name
  tmpdir="$(mktemp -d)"
  trap "rm -rf -- '${tmpdir}'" RETURN
  bundle_name="$(basename "${BUNDLE}")"
  bundle_root="${tmpdir}/${bundle_name%.tar.gz}"
  platform="$(resolve_bundle_platform "${bundle_name}")"

  tar -xzf "${BUNDLE}" -C "${tmpdir}"
  [[ -d "${bundle_root}" ]] || die "bundle root directory missing after extraction: ${bundle_root}"
  [[ -x "${bundle_root}/tabid" ]] || die "bundle is missing executable tabid"
  while IFS= read -r lib_name; do
    [[ -f "${bundle_root}/lib/${lib_name}" ]] || die "bundle is missing lib/${lib_name}"
  done < <(bundle_libs_for_platform "${platform}")
  ok "bundle content check passed: ${bundle_name} (${platform})"

  if host_matches_platform "${platform}"; then
    case "${platform}" in
      linux/amd64) verify_linux_runtime "${bundle_root}" "${tmpdir}" ;;
      darwin/arm64) verify_darwin_runtime "${bundle_root}" "${tmpdir}" ;;
      *) die "unsupported runtime smoke test platform: ${platform}" ;;
    esac
  else
    log "skipping runtime smoke test because host does not match bundle platform ${platform}"
  fi
}

verify_manifest() {
  [[ -n "${MANIFEST}" ]] || return 0

  local actual_sha artifact_name artifact_path
  artifact_path="${BINARY:-${BUNDLE}}"
  actual_sha="$(sha256_file "${artifact_path}")"
  artifact_name="$(basename "${artifact_path}")"

  python3 - "${MANIFEST}" "${artifact_name}" "${actual_sha}" "${EXPECTED_REPO}" "${EXPECTED_TAG}" "${EXPECTED_PLATFORM}" <<'PY'
import json
import sys
from pathlib import Path

manifest_path, artifact_name, actual_sha, expected_repo, expected_tag, expected_platform = sys.argv[1:]
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

if data["artifact_name"] != artifact_name:
    raise SystemExit(f"manifest artifact_name mismatch: {data['artifact_name']} != {artifact_name}")
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
  verify_bundle_contents

  if [[ -n "${MANIFEST}" ]]; then
    summary="$(verify_manifest)"
    ok "release manifest check passed: ${summary}"
  else
    log "no manifest provided; only SHA256SUMS was verified"
  fi

  ok "tabid release artifact verification completed successfully"
}

main "$@"
