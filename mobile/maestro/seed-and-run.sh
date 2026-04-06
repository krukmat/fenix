#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
MOBILE_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd -- "${MOBILE_DIR}/.." && pwd)"

APP_ID="com.fenixcrm.app"
APP_ACTIVITY="${APP_ID}/.MainActivity"
DEBUG_APK="${MOBILE_DIR}/android/app/build/outputs/apk/debug/app-debug.apk"
FLOW_FILE="${SCRIPT_DIR}/visual-audit.yaml"
OUTPUT_DIR="${MOBILE_DIR}/artifacts/screenshots"
ADB_BIN="${ANDROID_HOME:+${ANDROID_HOME}/platform-tools/}adb"
MAESTRO_BIN="${MAESTRO_BIN:-maestro}"

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

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"
}

SERIAL="${FENIX_SCREENSHOTS_DEVICE_SERIAL:-$("${ADB_BIN}" devices | awk '$2=="device"{print $1; exit}')}"

adb_cmd() {
  "${ADB_BIN}" -s "${SERIAL}" "$@"
}

adb_shell() {
  adb_cmd shell "$@"
}

wait_for_device() {
  "${ADB_BIN}" wait-for-device >/dev/null
}

wait_for_android_services() {
  local attempts=0
  while (( attempts < 90 )); do
    local services
    services="$(adb_shell service list 2>/dev/null || true)"
    if grep -q 'activity:' <<<"${services}" && grep -q 'package:' <<<"${services}"; then
      return 0
    fi
    sleep 2
    attempts=$((attempts + 1))
  done
  die "Android services are not ready on ${SERIAL}. Expected activity/package services."
}

wait_for_react_native_ready() {
  local attempts=0
  while (( attempts < 90 )); do
    if adb_cmd logcat -d -s ReactNativeJS 2>/dev/null | grep -Fq 'Running "main"'; then
      return 0
    fi
    sleep 2
    attempts=$((attempts + 1))
  done
  die "React Native app did not report readiness within 180s."
}

ensure_app_installed() {
  local package_path
  package_path="$(adb_shell pm path "${APP_ID}" 2>/dev/null | tr -d '\r' || true)"
  if [[ "${package_path}" == package:* ]]; then
    return 0
  fi
  [[ -f "${DEBUG_APK}" ]] || die "App ${APP_ID} is not installed and debug APK not found at ${DEBUG_APK}"
  log "Installing debug APK on ${SERIAL}..."
  "${ADB_BIN}" -s "${SERIAL}" install -r "${DEBUG_APK}" >/dev/null
}

unlock_device() {
  adb_shell input keyevent KEYCODE_WAKEUP >/dev/null 2>&1 || true
  adb_shell input keyevent 82 >/dev/null 2>&1 || true
}

seed_to_env_lines() {
  local seed_file="$1"
  node - "${seed_file}" <<'NODE'
const fs = require('fs');
const file = process.argv[2];
const seed = JSON.parse(fs.readFileSync(file, 'utf8'));
const pairs = {
  SEED_EMAIL: seed.credentials?.email ?? '',
  SEED_PASSWORD: seed.credentials?.password ?? '',
  SEED_ACCOUNT_ID: seed.account?.id ?? '',
  SEED_CONTACT_ID: seed.contact?.id ?? '',
  SEED_CONTACT_EMAIL: seed.contact?.email ?? '',
  SEED_DEAL_ID: seed.deal?.id ?? '',
  SEED_CASE_ID: seed.case?.id ?? '',
  SEED_CASE_SUBJECT: seed.case?.subject ?? '',
  SEED_WORKFLOW_ACTIVE_ID: seed.workflows?.activeId ?? '',
  SEED_AGENT_RUN_REJECTED_ID: seed.agentRuns?.rejectedId ?? '',
};
for (const [key, value] of Object.entries(pairs)) {
  process.stdout.write(`${key}=${String(value)}\n`);
}
NODE
}

resolve_signal_id() {
  local login_payload login_response token workspace_id signals_response
  login_payload="$(node -e "process.stdout.write(JSON.stringify({ email: process.env.SEED_EMAIL, password: process.env.SEED_PASSWORD }))")"
  login_response="$(curl -fsS -X POST -H 'Content-Type: application/json' --data "${login_payload}" 'http://localhost:3000/bff/auth/login' || true)"
  if [[ -z "${login_response}" ]]; then
    export SEED_SIGNAL_ID=""
    return 0
  fi

  token="$(printf '%s' "${login_response}" | node -e "let s='';process.stdin.on('data',d=>s+=d).on('end',()=>{const data=JSON.parse(s);process.stdout.write(data.token||'')})")"
  workspace_id="$(printf '%s' "${login_response}" | node -e "let s='';process.stdin.on('data',d=>s+=d).on('end',()=>{const data=JSON.parse(s);process.stdout.write(data.workspaceId||'')})")"
  if [[ -z "${token}" || -z "${workspace_id}" ]]; then
    export SEED_SIGNAL_ID=""
    return 0
  fi

  signals_response="$(curl -fsS -H "Authorization: Bearer ${token}" "http://localhost:3000/bff/api/v1/signals?workspace_id=${workspace_id}&status=active" || true)"
  if [[ -z "${signals_response}" ]]; then
    export SEED_SIGNAL_ID=""
    return 0
  fi

  export SEED_SIGNAL_ID
  SEED_SIGNAL_ID="$(
    printf '%s' "${signals_response}" | node -e "let s='';process.stdin.on('data',d=>s+=d).on('end',()=>{const data=JSON.parse(s);process.stdout.write(data?.data?.[0]?.id || '')})"
  )"
}

print_seed_summary() {
  log "Device: ${SERIAL}"
  log "SEED_EMAIL=${SEED_EMAIL}"
  log "SEED_PASSWORD=[redacted]"
  log "SEED_ACCOUNT_ID=${SEED_ACCOUNT_ID}"
  log "SEED_CONTACT_ID=${SEED_CONTACT_ID}"
  log "SEED_DEAL_ID=${SEED_DEAL_ID}"
  log "SEED_CASE_ID=${SEED_CASE_ID}"
  log "SEED_WORKFLOW_ACTIVE_ID=${SEED_WORKFLOW_ACTIVE_ID}"
  log "SEED_AGENT_RUN_REJECTED_ID=${SEED_AGENT_RUN_REJECTED_ID}"
  log "SEED_SIGNAL_ID=${SEED_SIGNAL_ID:-}"
}

main() {
  need_cmd "${ADB_BIN}"
  need_cmd curl
  need_cmd go
  need_cmd node
  need_cmd "${MAESTRO_BIN}"
  [[ -n "${SERIAL}" ]] || die 'No Android emulator/device connected.'
  [[ -f "${FLOW_FILE}" ]] || die "Missing Maestro flow: ${FLOW_FILE}"

  wait_for_device
  wait_for_android_services
  unlock_device
  ensure_app_installed

  local seed_file
  seed_file="$(mktemp)"
  trap 'rm -f "${seed_file}"' EXIT

  log 'Seeding deterministic mobile fixtures...'
  (
    cd "${REPO_ROOT}"
    go run ./scripts/e2e_seed_mobile_p2.go
  ) >"${seed_file}"

  while IFS='=' read -r key value; do
    export "${key}=${value}"
  done < <(seed_to_env_lines "${seed_file}")

  resolve_signal_id
  print_seed_summary

  log 'Preparing emulator networking...'
  adb_cmd reverse tcp:3000 tcp:3000 >/dev/null
  adb_cmd reverse tcp:8080 tcp:8080 >/dev/null

  log 'Resetting app state and pre-warming React Native...'
  adb_cmd logcat -c >/dev/null 2>&1 || true
  adb_shell pm clear "${APP_ID}" >/dev/null
  adb_shell am start -W -n "${APP_ACTIVITY}" >/dev/null || true
  wait_for_react_native_ready

  rm -rf "${OUTPUT_DIR}"
  mkdir -p "${OUTPUT_DIR}"

  log 'Running Maestro visual audit...'
  "${MAESTRO_BIN}" test \
    --device "${SERIAL}" \
    --test-output-dir "${OUTPUT_DIR}" \
    -e "SEED_EMAIL=${SEED_EMAIL}" \
    -e "SEED_PASSWORD=${SEED_PASSWORD}" \
    -e "SEED_ACCOUNT_ID=${SEED_ACCOUNT_ID}" \
    -e "SEED_CONTACT_ID=${SEED_CONTACT_ID}" \
    -e "SEED_CONTACT_EMAIL=${SEED_CONTACT_EMAIL}" \
    -e "SEED_DEAL_ID=${SEED_DEAL_ID}" \
    -e "SEED_CASE_ID=${SEED_CASE_ID}" \
    -e "SEED_CASE_SUBJECT=${SEED_CASE_SUBJECT}" \
    -e "SEED_WORKFLOW_ACTIVE_ID=${SEED_WORKFLOW_ACTIVE_ID}" \
    -e "SEED_AGENT_RUN_REJECTED_ID=${SEED_AGENT_RUN_REJECTED_ID}" \
    -e "SEED_SIGNAL_ID=${SEED_SIGNAL_ID:-}" \
    "${FLOW_FILE}"

  log "Screenshots available in ${OUTPUT_DIR}"
}

main "$@"
