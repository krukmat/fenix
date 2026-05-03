#!/usr/bin/env bash

# docs/plans/maestro-screenshot-auth-bypass-plan.md
#
# Two-phase Maestro screenshot runner:
#   Phase 1 — auth-surface.yaml        : cold launch + login-screen capture
#   Phase 2 — authenticated-audit.yaml : deep-link bootstrap + authenticated captures
#
# Auth is injected via an e2e-bootstrap deep link composed from the seeder's
# runtime session (seed.auth.{token,userId,workspaceId}). No login UI
# interaction occurs in the screenshot critical path.

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
MOBILE_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd -- "${MOBILE_DIR}/.." && pwd)"

APP_ID="com.fenixcrm.app"
APP_ACTIVITY="${APP_ID}/.MainActivity"
DEBUG_APK="${MOBILE_DIR}/android/app/build/outputs/apk/debug/app-debug.apk"
AUTH_SURFACE_FLOW="${SCRIPT_DIR}/auth-surface.yaml"
AUTHED_AUDIT_FLOW="${SCRIPT_DIR}/authenticated-audit.yaml"
OUTPUT_DIR="${MOBILE_DIR}/artifacts/screenshots"
TMP_BASE="${TMPDIR:-/tmp}"
REPORTS_DIR="${FENIX_MAESTRO_REPORTS_DIR:-${TMP_BASE%/}/fenixcrm-maestro-reports}"
ADB_BIN="${ANDROID_HOME:+${ANDROID_HOME}/platform-tools/}adb"
MAESTRO_BIN="${MAESTRO_BIN:-maestro}"
METRO_HOST="${FENIX_SCREENSHOTS_METRO_HOST:-127.0.0.1}"
METRO_PORT="${FENIX_SCREENSHOTS_METRO_PORT:-8081}"
METRO_URL="http://${METRO_HOST}:${METRO_PORT}/status"
METRO_LOG="${FENIX_SCREENSHOTS_METRO_LOG:-${TMP_BASE%/}/fenixcrm-metro.log}"
SEED_FILE=""
METRO_PID=""
ADB_TIMEOUT_SECS="${FENIX_SCREENSHOTS_ADB_TIMEOUT_SECS:-15}"

export PATH="${PATH}:${HOME}/.maestro/bin"
export MAESTRO_CLI_NO_ANALYTICS=1
export MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED=true

log() {
  printf '[screenshots] %s\n' "$*"
}

die() {
  printf '[screenshots] ERROR: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "${SEED_FILE}" ]]; then
    rm -f "${SEED_FILE}"
  fi
  if [[ -n "${METRO_PID}" ]]; then
    kill "${METRO_PID}" >/dev/null 2>&1 || true
    wait "${METRO_PID}" >/dev/null 2>&1 || true
  fi
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"
}

run_seed() {
  (
    cd "${REPO_ROOT}"
    go run ./scripts/e2e_seed_mobile_p2.go
  )
}

check_backend_health() {
  local health_url='http://localhost:8080/health'
  if curl -fsS --max-time 3 "${health_url}" >/dev/null; then
    return 0
  fi
  die "Go backend is not healthy at ${health_url}. Start it with: JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080"
}

check_bff_reachable() {
  local bff_url='http://localhost:3000'
  if curl -sS --max-time 3 "${bff_url}" >/dev/null; then
    return 0
  fi
  die "BFF is not reachable at ${bff_url}. Start the BFF on localhost:3000 before running screenshots."
}

is_metro_ready() {
  curl -fsS --max-time 3 "${METRO_URL}" 2>/dev/null | grep -q 'packager-status:running'
}

start_metro() {
  if is_metro_ready; then
    log "Using existing Metro server at ${METRO_URL}"
    return 0
  fi

  log "Starting Metro for debug APK at ${METRO_HOST}:${METRO_PORT}..."
  rm -f "${METRO_LOG}"
  (
    cd "${MOBILE_DIR}"
    npx expo start --port "${METRO_PORT}" --host localhost --non-interactive
  ) >"${METRO_LOG}" 2>&1 &
  METRO_PID=$!
}

wait_for_metro() {
  local attempts=0
  while (( attempts < 90 )); do
    if is_metro_ready; then
      log "Metro is ready at ${METRO_URL}"
      return 0
    fi
    if [[ -n "${METRO_PID}" ]] && ! kill -0 "${METRO_PID}" >/dev/null 2>&1; then
      tail -n 40 "${METRO_LOG}" >&2 || true
      die "Metro exited before becoming ready. See ${METRO_LOG}"
    fi
    sleep 2
    attempts=$((attempts + 1))
  done

  if [[ -f "${METRO_LOG}" ]]; then
    tail -n 40 "${METRO_LOG}" >&2 || true
  fi
  die "Metro did not become ready at ${METRO_URL}"
}

SERIAL="${FENIX_SCREENSHOTS_DEVICE_SERIAL:-$("${ADB_BIN}" devices | awk '$2=="device"{print $1; exit}')}"

adb_cmd() {
  "${ADB_BIN}" -s "${SERIAL}" "$@"
}

run_with_timeout() {
  python3 - "$@" <<'PY'
import subprocess
import sys

timeout = float(sys.argv[1])
command = sys.argv[2:]

try:
    completed = subprocess.run(command, timeout=timeout, check=False)
except subprocess.TimeoutExpired:
    sys.stderr.write(f"[screenshots] ERROR: command timed out after {timeout:.0f}s: {' '.join(command)}\n")
    sys.exit(124)

sys.exit(completed.returncode)
PY
}

adb_cmd_timed() {
  run_with_timeout "${ADB_TIMEOUT_SECS}" "${ADB_BIN}" -s "${SERIAL}" "$@"
}

adb_shell() {
  adb_cmd_timed shell "$@"
}

wait_for_device() {
  "${ADB_BIN}" wait-for-device >/dev/null
}

wait_for_android_services() {
  local attempts=0
  while (( attempts < 90 )); do
    local boot_completed package_path
    boot_completed="$(adb_shell getprop sys.boot_completed 2>/dev/null | tr -d '\r' || true)"
    package_path="$(adb_shell pm path android 2>/dev/null | tr -d '\r' || true)"
    if [[ "${boot_completed}" == "1" && "${package_path}" == package:* ]]; then
      return 0
    fi
    sleep 2
    attempts=$((attempts + 1))
  done
  die "Android services are not ready on ${SERIAL}. Expected sys.boot_completed=1 and a working package manager shell."
}

ensure_app_installed() {
  local package_path
  package_path="$(adb_shell pm path "${APP_ID}" 2>/dev/null | tr -d '\r' || true)"
  if [[ -f "${DEBUG_APK}" ]]; then
    log "Installing debug APK on ${SERIAL}..."
    "${ADB_BIN}" -s "${SERIAL}" install -r "${DEBUG_APK}" >/dev/null
    return 0
  fi
  if [[ "${package_path}" == package:* ]]; then
    log "Using already installed ${APP_ID}; debug APK not found at ${DEBUG_APK}"
    return 0
  fi
  die "App ${APP_ID} is not installed and debug APK not found at ${DEBUG_APK}"
}

unlock_device() {
  adb_shell input keyevent KEYCODE_WAKEUP >/dev/null 2>&1 || true
  adb_shell input keyevent 82 >/dev/null 2>&1 || true
}

launch_app_via_adb() {
  adb_shell monkey -p "${APP_ID}" -c android.intent.category.LAUNCHER 1 >/dev/null 2>&1 \
    || adb_shell am start -W -n "${APP_ACTIVITY}" >/dev/null 2>&1 \
    || die "Unable to start ${APP_ID} via adb."
}

# url_encode: percent-encodes a single value using Node's encodeURIComponent.
# Used for JWTs and ids that contain `.`, `+`, `/`, `=` which would otherwise
# break the Android Intent URI parser when embedded in a deep link.
url_encode() {
  node -e 'process.stdout.write(encodeURIComponent(process.argv[1] || ""))' -- "$1"
}

seed_to_env_lines() {
  # Maps seed JSON into KEY=value lines for the shell.
  # Screenshot auth bypass: exposes seed.auth.* as SEED_AUTH_TOKEN /
  # SEED_USER_ID / SEED_WORKSPACE_ID so the runner can compose the
  # e2e-bootstrap deep link. SEED_PASSWORD is NOT exported — login UI is
  # removed from the screenshot critical path.
  local seed_file="$1"
  node - "${seed_file}" <<'NODE'
const fs = require('fs');
const file = process.argv[2];
const seed = JSON.parse(fs.readFileSync(file, 'utf8'));
const pairs = {
  SEED_EMAIL:              seed.credentials?.email ?? '',
  SEED_ACCOUNT_ID:         seed.account?.id ?? '',
  SEED_CONTACT_ID:         seed.contact?.id ?? '',
  SEED_CONTACT_EMAIL:      seed.contact?.email ?? '',
  SEED_LEAD_ID:            seed.lead?.id ?? '',
  SEED_DEAL_ID:            seed.deal?.id ?? '',
  SEED_PIPELINE_ID:        seed.pipeline?.id ?? '',
  SEED_STAGE_ID:           seed.stage?.id ?? '',
  SEED_STALE_DEAL_ID:      seed.staleDeal?.id ?? seed.deal?.id ?? '',
  SEED_CASE_ID:            seed.case?.id ?? '',
  SEED_CASE_SUBJECT:       seed.case?.subject ?? '',
  SEED_RESOLVED_CASE_ID:   seed.resolvedCase?.id ?? '',
  SEED_RESOLVED_CASE_SUBJECT: seed.resolvedCase?.subject ?? '',
  SEED_RUN_COMPLETED_ID:   seed.agentRuns?.completedId ?? '',
  SEED_RUN_HANDOFF_ID:     seed.agentRuns?.handoffId ?? '',
  SEED_RUN_DENIED_ID:      seed.agentRuns?.deniedByPolicyId ?? '',
  SEED_APPROVAL_ID:        seed.inbox?.approvalId ?? '',
  SEED_SIGNAL_ID:          seed.inbox?.signalId ?? '',
  SEED_WORKFLOW_ID:        seed.workflow?.id ?? '',
  SEED_AUTH_TOKEN:         seed.auth?.token ?? '',
  SEED_USER_ID:            seed.auth?.userId ?? '',
  SEED_WORKSPACE_ID:       seed.auth?.workspaceId ?? '',
};
for (const [key, value] of Object.entries(pairs)) {
  process.stdout.write(`${key}=${String(value)}\n`);
}
NODE
}

compose_bootstrap_url() {
  # Hard-coded landing route. /inbox bypasses the /home → /inbox redirect hop.
  local landing_route='/inbox'
  local enc_token enc_user enc_workspace enc_redirect
  enc_token="$(url_encode "${SEED_AUTH_TOKEN}")"
  enc_user="$(url_encode "${SEED_USER_ID}")"
  enc_workspace="$(url_encode "${SEED_WORKSPACE_ID}")"
  enc_redirect="$(url_encode "${landing_route}")"
  printf 'fenixcrm:///e2e-bootstrap?token=%s&userId=%s&workspaceId=%s&redirect=%s' \
    "${enc_token}" "${enc_user}" "${enc_workspace}" "${enc_redirect}"
}

print_seed_summary() {
  # Secrets are NEVER printed. Token and password are redacted by design.
  log "Device: ${SERIAL}"
  log "SEED_EMAIL=${SEED_EMAIL}"
  log "SEED_ACCOUNT_ID=${SEED_ACCOUNT_ID}"
  log "SEED_CONTACT_ID=${SEED_CONTACT_ID}"
  log "SEED_DEAL_ID=${SEED_DEAL_ID}"
  log "SEED_PIPELINE_ID=${SEED_PIPELINE_ID}"
  log "SEED_STAGE_ID=${SEED_STAGE_ID}"
  log "SEED_STALE_DEAL_ID=${SEED_STALE_DEAL_ID}"
  log "SEED_LEAD_ID=${SEED_LEAD_ID}"
  log "SEED_CASE_ID=${SEED_CASE_ID}"
  log "SEED_RESOLVED_CASE_ID=${SEED_RESOLVED_CASE_ID}"
  log "SEED_RUN_COMPLETED_ID=${SEED_RUN_COMPLETED_ID}"
  log "SEED_RUN_HANDOFF_ID=${SEED_RUN_HANDOFF_ID}"
  log "SEED_RUN_DENIED_ID=${SEED_RUN_DENIED_ID}"
  log "SEED_APPROVAL_ID=${SEED_APPROVAL_ID}"
  log "SEED_SIGNAL_ID=${SEED_SIGNAL_ID}"
  log "SEED_WORKFLOW_ID=${SEED_WORKFLOW_ID}"
  log "SEED_USER_ID=${SEED_USER_ID}"
  log "SEED_WORKSPACE_ID=${SEED_WORKSPACE_ID}"
  log "SEED_AUTH_TOKEN=[redacted len=${#SEED_AUTH_TOKEN}]"
}

run_maestro_flow() {
  local flow="$1"
  "${MAESTRO_BIN}" test \
    --device "${SERIAL}" \
    --test-output-dir "${REPORTS_DIR}" \
    -e "SEED_EMAIL=${SEED_EMAIL}" \
    -e "SEED_ACCOUNT_ID=${SEED_ACCOUNT_ID}" \
    -e "SEED_CONTACT_ID=${SEED_CONTACT_ID}" \
    -e "SEED_CONTACT_EMAIL=${SEED_CONTACT_EMAIL}" \
    -e "SEED_LEAD_ID=${SEED_LEAD_ID}" \
    -e "SEED_DEAL_ID=${SEED_DEAL_ID}" \
    -e "SEED_PIPELINE_ID=${SEED_PIPELINE_ID}" \
    -e "SEED_STAGE_ID=${SEED_STAGE_ID}" \
    -e "SEED_STALE_DEAL_ID=${SEED_STALE_DEAL_ID}" \
    -e "SEED_CASE_ID=${SEED_CASE_ID}" \
    -e "SEED_CASE_SUBJECT=${SEED_CASE_SUBJECT}" \
    -e "SEED_RESOLVED_CASE_ID=${SEED_RESOLVED_CASE_ID}" \
    -e "SEED_RESOLVED_CASE_SUBJECT=${SEED_RESOLVED_CASE_SUBJECT}" \
    -e "SEED_RUN_COMPLETED_ID=${SEED_RUN_COMPLETED_ID}" \
    -e "SEED_RUN_HANDOFF_ID=${SEED_RUN_HANDOFF_ID}" \
    -e "SEED_RUN_DENIED_ID=${SEED_RUN_DENIED_ID}" \
    -e "SEED_APPROVAL_ID=${SEED_APPROVAL_ID}" \
    -e "SEED_SIGNAL_ID=${SEED_SIGNAL_ID}" \
    -e "SEED_WORKFLOW_ID=${SEED_WORKFLOW_ID}" \
    -e "SEED_BOOTSTRAP_URL=${SEED_BOOTSTRAP_URL}" \
    "${flow}"
}

copy_reports_screenshots() {
  # Maestro writes PNGs under the test-output-dir. Collect any PNGs from the
  # reports tree into the stable output dir. Successful runs should contain the
  # full visual audit set; failed runs keep completed screenshots plus Maestro's
  # failure screenshot for debugging.
  mkdir -p "${OUTPUT_DIR}"
  find "${REPORTS_DIR}" -type f -name '*.png' -print0 2>/dev/null \
    | while IFS= read -r -d '' file; do
        cp -f "${file}" "${OUTPUT_DIR}/"
      done
}

sanitize_reports() {
  local redacted_url='fenixcrm:///e2e-bootstrap?token=[redacted]&userId=[redacted]&workspaceId=[redacted]&redirect=%2Finbox'
  REPORTS_DIR_ENV="${REPORTS_DIR}" \
  SEED_BOOTSTRAP_URL_ENV="${SEED_BOOTSTRAP_URL}" \
  REDACTED_URL_ENV="${redacted_url}" \
  node <<'NODE'
const fs = require('fs');
const path = require('path');

const root = process.env.REPORTS_DIR_ENV;
const bootstrapUrl = process.env.SEED_BOOTSTRAP_URL_ENV;
const redactedUrl = process.env.REDACTED_URL_ENV;

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(full);
      continue;
    }
    const content = fs.readFileSync(full, 'utf8');
    let next = content.split(bootstrapUrl).join(redactedUrl);
    next = next.replace(/token=eyJ[a-zA-Z0-9._-]*/g, 'token=[redacted]');
    if (next !== content) {
      fs.writeFileSync(full, next, 'utf8');
    }
  }
}

walk(root);
NODE
}

finish() {
  local code=$?
  trap - EXIT

  if [[ -d "${REPORTS_DIR}" ]]; then
    if [[ -n "${SEED_BOOTSTRAP_URL:-}" ]]; then
      sanitize_reports || true
    fi
    copy_reports_screenshots || true
  fi

  cleanup

  if [[ -d "${OUTPUT_DIR}" ]]; then
    log "Screenshots available in ${OUTPUT_DIR}"
  fi
  if [[ -d "${REPORTS_DIR}" ]]; then
    log "Temporary Maestro reports available in ${REPORTS_DIR}"
  fi

  exit "${code}"
}

main() {
  need_cmd "${ADB_BIN}"
  need_cmd curl
  need_cmd go
  need_cmd node
  need_cmd "${MAESTRO_BIN}"
  [[ -n "${SERIAL}" ]] || die 'No Android emulator/device connected.'
  [[ -f "${AUTH_SURFACE_FLOW}" ]] || die "Missing Maestro flow: ${AUTH_SURFACE_FLOW}"
  [[ -f "${AUTHED_AUDIT_FLOW}" ]] || die "Missing Maestro flow: ${AUTHED_AUDIT_FLOW}"
  trap finish EXIT

  wait_for_device
  wait_for_android_services
  unlock_device
  ensure_app_installed
  check_backend_health
  check_bff_reachable
  start_metro
  wait_for_metro

  SEED_FILE="$(mktemp)"

  log 'Seeding deterministic mobile fixtures...'
  run_seed >"${SEED_FILE}"

  while IFS='=' read -r key value; do
    export "${key}=${value}"
  done < <(seed_to_env_lines "${SEED_FILE}")

  [[ -n "${SEED_AUTH_TOKEN}" ]] || die 'Seeder did not return auth.token — cannot bootstrap authenticated phase.'
  [[ -n "${SEED_USER_ID}" ]]    || die 'Seeder did not return auth.userId.'
  [[ -n "${SEED_WORKSPACE_ID}" ]] || die 'Seeder did not return auth.workspaceId.'
  [[ -n "${SEED_WORKFLOW_ID}" ]] || die 'Seeder did not return workflow.id — cannot capture workflow graph.'

  SEED_BOOTSTRAP_URL="$(compose_bootstrap_url)"
  export SEED_BOOTSTRAP_URL

  print_seed_summary

  log 'Preparing emulator networking...'
  adb_cmd reverse tcp:3000 tcp:3000 >/dev/null
  adb_cmd reverse tcp:8080 tcp:8080 >/dev/null
  adb_cmd reverse tcp:8081 tcp:8081 >/dev/null

  log 'Enabling BFF screenshot fixture mode (bypasses LLM for sales-brief)...'
  curl -s -X POST http://localhost:3000/bff/api/v1/copilot/internal/screenshot-mode \
    -H 'Content-Type: application/json' \
    -d '{"enabled":true}' >/dev/null \
    || log 'WARNING: could not enable screenshot mode — sales-brief will call LLM'

  log 'Resetting app state...'
  adb_cmd logcat -c >/dev/null 2>&1 || true
  adb_shell pm clear "${APP_ID}" >/dev/null
  launch_app_via_adb

  rm -rf "${OUTPUT_DIR}" "${REPORTS_DIR}"
  mkdir -p "${OUTPUT_DIR}" "${REPORTS_DIR}"

  log 'Phase 1/2: capturing auth surface...'
  run_maestro_flow "${AUTH_SURFACE_FLOW}"

  log 'Phase 2/2: capturing authenticated audit via e2e-bootstrap deep link...'
  run_maestro_flow "${AUTHED_AUDIT_FLOW}"

  log 'Disabling BFF screenshot fixture mode...'
  curl -s -X POST http://localhost:3000/bff/api/v1/copilot/internal/screenshot-mode \
    -H 'Content-Type: application/json' \
    -d '{"enabled":false}' >/dev/null || true
}

main "$@"
