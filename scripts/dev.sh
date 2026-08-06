#!/usr/bin/env bash
# scripts/dev.sh -- build/vet/test/cov/scan/dist tasks for the `solace` Go CLI.
#
# Mirror of scripts/dev.ps1 (behaviourally identical). The USER runs this; CI
# (.github/workflows/ci.yml and tag.yml) calls task names only, so local == CI
# structurally. Works from any cwd. Accepts multiple tasks per invocation.
#
#   ./scripts/dev.sh build vet test
#   ./scripts/dev.sh all          # build vet test           (CI runs: all scan)
#   ./scripts/dev.sh full         # all + cov scan graphify  (pre-tag sweep)
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
LOG_DIR="${SCRIPT_DIR}/logs"
DIST_DIR="${REPO_ROOT}/dist"
COV_DIR="${REPO_ROOT}/coverage"
BIN_NAME="solace"

# Pinned tool versions (dependencies pinned and patched).
GOVULN_VERSION="${GOVULN_VERSION:-v1.1.4}"

# Local convenience cross-compile set (`dist`). CI does not use it: tag.yml
# matrixes over `build` with TARGET_OS/TARGET_ARCH from BUILD_TARGETS instead.
DIST_TARGETS=(linux/amd64 linux/arm64 darwin/arm64 windows/amd64)

# -race needs cgo + a C compiler; on by default, disable with SOLACE_RACE=0.
RACE_FLAG=(); COVERMODE="count"
if [[ "${SOLACE_RACE:-1}" != "0" ]]; then RACE_FLAG=(-race); COVERMODE="atomic"; fi

# Keep captured logs clean: no color, no progress spinners from tools.
export NO_COLOR=1

# --- pretty helpers (console only; logs are stripped of ANSI) -----------------
if [[ -t 1 ]]; then
  B=$'\033[1m'; GRN=$'\033[32m'; YLW=$'\033[33m'; RED=$'\033[31m'; CYN=$'\033[36m'; RST=$'\033[0m'
else
  B=""; GRN=""; YLW=""; RED=""; CYN=""; RST=""
fi
step() { printf '%s==>%s %s\n' "${CYN}${B}" "${RST}" "$*" >&2; }
ok()   { printf '%s[ ok ]%s %s\n' "${GRN}" "${RST}" "$*" >&2; }
warn() { printf '%s[warn]%s %s\n' "${YLW}" "${RST}" "$*" >&2; }
die()  { printf '%s[fail]%s %s\n' "${RED}${B}" "${RST}" "$*" >&2; exit 1; }

now() { date +%Y-%m-%dT%H:%M:%S%z; }

LOGFILE=""
log_init() {
  mkdir -p "${LOG_DIR}"
  LOGFILE="${LOG_DIR}/$1.log"
  printf '=== %s | %s ===\n' "$(now)" "$1" > "${LOGFILE}"
}
# cap runs a command, streaming combined stdout+stderr to the console and
# appending an ANSI-stripped copy to LOGFILE. Returns the command's exit code.
cap() {
  "$@" 2>&1 | tee >(sed -E $'s/\x1b\\[[0-9;?]*[a-zA-Z]//g' >> "${LOGFILE}")
  return "${PIPESTATUS[0]}"
}
# finish <task> <exit-code> <elapsed-seconds> -- contract footer, log + console.
finish() {
  local status="OK"
  (( $2 != 0 )) && status="FAILED (exit $2)"
  printf '%s | %s | %ss | %s\n' "$(now)" "$1" "$3" "${status}" | tee -a "${LOG_DIR}/$1.log"
}

# --- tasks --------------------------------------------------------------------
# Tasks return non-zero on failure; the dispatcher writes the footer and stops.

task_tidy() { cap go mod tidy; }
task_vet()  { cap go vet ./...; }
task_test() { cap go test "${RACE_FLAG[@]}" -count=1 ./...; }

# build_one <os> <arch> -- compile the CLI for one target into dist/. The
# target lands in the binary name because the release job merges every leg
# with merge-multiple: identical names would silently overwrite.
build_one() {
  local os=$1 arch=$2 out="${DIST_DIR}/${BIN_NAME}-$1-$2"
  [[ "${os}" == "windows" ]] && out="${out}.exe"
  mkdir -p "${DIST_DIR}"
  step "  ${os}/${arch}"
  CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" \
    cap go build -trimpath -ldflags "-s -w" -o "${out}" .
}

# CI sets TARGET_OS/TARGET_ARCH (from the BUILD_TARGETS repo variable); unset
# means build for the host, so local and CI output are the same shape.
task_build() {
  build_one "${TARGET_OS:-$(go env GOOS)}" "${TARGET_ARCH:-$(go env GOARCH)}"
}

task_dist() {
  local t
  for t in "${DIST_TARGETS[@]}"; do
    build_one "${t%/*}" "${t#*/}" || return 1
  done
  ok "binaries in ${DIST_DIR}"
}

task_cov() {
  mkdir -p "${COV_DIR}"
  local prof="${COV_DIR}/coverage.out" html="${COV_DIR}/coverage.html"
  # -count=1 forces a real run so a cached test result can't report a stale
  # coverage total and mask a drop below the floor (the previous total in
  # logs/cov.log; local only -- CI is a fresh checkout with no prior log).
  cap go test "${RACE_FLAG[@]}" -covermode="${COVERMODE}" -coverprofile="${prof}" -count=1 ./... || return 1
  cap go tool cover -html="${prof}" -o "${html}" || return 1
  local total; total="$(go tool cover -func="${prof}" | tail -n1)"
  printf '%s\n' "${total}" | tee -a "${LOGFILE}"
  ok "coverage -> ${html}  (${total##*$'\t'})"
}

# One task, every applicable check: govulncheck over source + deps. FATAL on
# findings, standalone or inside an aggregate; local and CI behave the same.
# (No image half: this project ships binaries only.)
task_scan() {
  cap go run "golang.org/x/vuln/cmd/govulncheck@${GOVULN_VERSION}" ./...
}

# Local only: the graph is a developer artifact, not a CI output.
task_graphify() {
  [[ -n "${CI:-}" ]] && { warn "graphify is local-only; skipping in CI"; return 0; }
  command -v graphify >/dev/null 2>&1 || { warn "graphify not on PATH; skipping"; return 0; }
  cap graphify update .
}

# --- dispatch -------------------------------------------------------------------
ALL="build vet test"
FULL="build vet test cov scan graphify"

usage() {
  cat >&2 <<EOF
${B}dev.sh${RST} -- build/test/scan tooling for the solace CLI

Usage: ${0##*/} <task> [task...]

Tasks:
  tidy     go mod tidy
  vet      go vet ./...
  build    compile -> dist/${BIN_NAME}-<os>-<arch>[.exe]; TARGET_OS/TARGET_ARCH
           pick the target, unset means host
  test     go test ${RACE_FLAG[*]:-} -count=1 ./...
  cov      coverage profile -> coverage/coverage.html + printed total
  scan     govulncheck (fatal on findings)
  dist     cross-compile ${DIST_TARGETS[*]}
  graphify refresh graphify-out/ (local only; skipped when CI is set)
  all      ${ALL}   (what CI runs, as: all scan)
  full     ${FULL}   (pre-tag sweep)

Env: SOLACE_RACE=0 disables -race; GOVULN_VERSION overrides the pinned govulncheck;
     TARGET_OS/TARGET_ARCH cross-compile a single \`build\`.
Logs: ${LOG_DIR}/<task>.log (each run closes with a timestamped footer)
EOF
}

main() {
  cd "${REPO_ROOT}"
  [[ $# -eq 0 ]] && { usage; exit 1; }
  case "$1" in -h|--help|help) usage; exit 0;; esac

  local tasks="" a t start code
  for a in "$@"; do
    case "$a" in
      all)      tasks+=" ${ALL}" ;;
      full)     tasks+=" ${FULL}" ;;
      binaries) tasks+=" dist" ;;
      *)        tasks+=" $a" ;;
    esac
  done

  for t in ${tasks}; do
    declare -F "task_${t}" >/dev/null || die "unknown task: ${t} (try: ${0##*/} help)"
    step "${t}"
    log_init "${t}"
    start=${SECONDS}
    code=0
    "task_${t}" || code=$?
    finish "${t}" "${code}" "$((SECONDS - start))"
    if (( code != 0 )); then
      warn "${t} failed; stopping"
      exit 1
    fi
    ok "${t}"
  done
  ok "done: $*"
}

main "$@"
