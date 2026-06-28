#!/usr/bin/env bash
# Fenix documentation consistency gate.
# Adapted from DubBridge check-doc-consistency.sh; key differences:
#   - ADRs live in docs/decisions/ (not docs/adr/)
#   - ADR status parsed from '## Status\n\n`<value>`' section (not '- **Status:**' bullet)
#   - Index file is docs/decisions/README.md
#   - Dangling-ref scan covers fenix Go/SQL paths (internal/, pkg/, cmd/)
set -euo pipefail

shopt -s nullglob

repo_root=$(
  cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1
  pwd
)
cd "$repo_root"

DECISIONS_DIR="docs/decisions"
INDEX_FILE="docs/decisions/README.md"

violations=""

add_violation() {
  if [[ -n "$violations" ]]; then
    violations="${violations}"$'\n'"$1"
  else
    violations="$1"
  fi
}

# Extract normalised status from the '## Status' prose section.
# Handles both `accepted` (backtick) and plain accepted.
file_status_token() {
  local file="$1"
  awk '/^## Status/{found=1; next} found && /[^[:space:]]/{gsub(/`/,""); print tolower($1); exit}' "$file"
}

adr_exists() {
  local adr_id="$1"
  local matches
  matches=("${DECISIONS_DIR}/${adr_id}"-*.md)
  [[ ${#matches[@]} -gt 0 ]]
}

# Parse index row for a given ADR-NNN id.
# Outputs: target_filename|status_token  (empty string if row not found)
index_row_data() {
  local adr_id="$1"
  local line
  line="$(grep -F "| [${adr_id}](" "$INDEX_FILE" || true)"
  if [[ -z "$line" ]]; then
    printf '\n'
    return
  fi
  local target
  local row_status
  target="$(printf '%s\n' "$line" | sed -n 's/^| \[[^]]*\](\([^)]*\)) | .* | .*$/\1/p')"
  row_status="$(printf '%s\n' "$line" | awk -F'|' '{print $4}' | sed 's/^ *//; s/ *$//')"
  printf '%s|%s\n' "$target" "$row_status"
}

check_status_parity_and_completeness() {
  local file
  for file in "${DECISIONS_DIR}"/ADR-*.md; do
    [[ -f "$file" ]] || continue

    local adr_name adr_id fm_status row_data row_target row_status

    adr_name="${file##*/}"
    adr_id="$(printf '%s\n' "$adr_name" | sed -n 's/^\(ADR-[0-9][0-9][0-9]\).*$/\1/p')"
    fm_status="$(file_status_token "$file")"

    if [[ -z "$fm_status" ]]; then
      add_violation "$file: missing or unparseable ## Status section"
      continue
    fi

    row_data="$(index_row_data "$adr_id")"
    if [[ -z "$row_data" ]]; then
      add_violation "$file: missing index row in ${INDEX_FILE}"
      continue
    fi

    row_target="${row_data%%|*}"
    row_status="${row_data#*|}"
    row_status="$(printf '%s\n' "$row_status" | sed 's/^ *//; s/ *$//')"

    if [[ "$fm_status" != "$row_status" ]]; then
      add_violation "$file: prose status '${fm_status}' does not match index status '${row_status}' for ${adr_id}"
    fi

    if [[ ! -f "${DECISIONS_DIR}/${row_target}" ]]; then
      add_violation "${INDEX_FILE}: index row for ${adr_id} points to missing file '${row_target}'"
    fi
  done

  # Reverse check: every row in the index must point to an existing file
  local line
  while IFS= read -r line; do
    [[ -n "$line" ]] || continue
    local adr_id target
    adr_id="$(printf '%s\n' "$line" | sed -n 's/^| \[\(ADR-[0-9][0-9][0-9]\)\].*$/\1/p')"
    target="$(printf '%s\n' "$line" | sed -n 's/^| \[[^]]*\](\([^)]*\)) | .*$/\1/p')"

    if [[ -z "$adr_id" || -z "$target" ]]; then
      add_violation "${INDEX_FILE}: could not parse ADR index row: ${line}"
      continue
    fi

    if [[ ! -f "${DECISIONS_DIR}/${target}" ]]; then
      add_violation "${INDEX_FILE}: index row for ${adr_id} points to missing file '${target}'"
      continue
    fi

    if [[ "$target" != "${adr_id}"-* ]]; then
      add_violation "${INDEX_FILE}: index row for ${adr_id} points to mismatched filename '${target}'"
    fi
  done < <(grep -E '^\| \[ADR-[0-9]{3}\]\([^)]+\) \|' "$INDEX_FILE" || true)
}

check_dangling_refs() {
  local docs_stream code_stream

  docs_stream="$(
    grep -R -nH 'ADR-[0-9]\{3\}' \
      "${DECISIONS_DIR}" docs/plans docs/tasks docs/policies 2>/dev/null || true
    grep -nH 'ADR-[0-9]\{3\}' docs/architecture.md README.md CLAUDE.md 2>/dev/null || true
  )"
  code_stream="$(
    grep -R -nH --include='*.go' 'ADR-[0-9]\{3\}' internal/ pkg/ cmd/ 2>/dev/null || true
    grep -R -nH --include='*.sql' 'ADR-[0-9]\{3\}' internal/ 2>/dev/null || true
  )"

  local old_ifs="$IFS"
  IFS=$'\n'
  local record
  for record in $docs_stream $code_stream; do
    [[ -n "$record" ]] || continue
    local file ref_file ref_line content tokens token
    ref_file="$(printf '%s\n' "$record" | sed -n 's/^\(.*\):[0-9][0-9]*:.*$/\1/p')"
    ref_line="$(printf '%s\n' "$record" | sed -n 's/^.*:\([0-9][0-9]*\):.*$/\1/p')"
    content="${record#"${ref_file}:${ref_line}:"}"
    tokens="$(printf '%s\n' "$content" | grep -oE 'ADR-[0-9]{3}' || true)"

    while IFS= read -r token; do
      [[ -n "$token" ]] || continue
      if ! adr_exists "$token"; then
        add_violation "dangling reference ${token} in ${ref_file}:${ref_line}"
      fi
    done <<EOF
$tokens
EOF
  done
  IFS="$old_ifs"
}

check_superseded_successors() {
  local lines
  lines="$(grep -nH 'Superseded by ADR-[0-9]\{3\}' "${DECISIONS_DIR}"/ADR-*.md || true)"
  local old_ifs="$IFS"
  IFS=$'\n'
  local record
  for record in $lines; do
    [[ -n "$record" ]] || continue
    local ref_file ref_line content tokens token
    ref_file="$(printf '%s\n' "$record" | sed -n 's/^\(.*\):[0-9][0-9]*:.*$/\1/p')"
    ref_line="$(printf '%s\n' "$record" | sed -n 's/^.*:\([0-9][0-9]*\):.*$/\1/p')"
    content="${record#"${ref_file}:${ref_line}:"}"
    tokens="$(printf '%s\n' "$content" | grep -oE 'ADR-[0-9]{3}' || true)"

    while IFS= read -r token; do
      [[ -n "$token" ]] || continue
      if ! adr_exists "$token"; then
        add_violation "${ref_file}:${ref_line}: superseded successor ${token} does not exist"
      fi
    done <<EOF
$tokens
EOF
  done
  IFS="$old_ifs"
}

check_status_parity_and_completeness
check_dangling_refs
check_superseded_successors

if [[ -n "$violations" ]]; then
  printf 'Documentation consistency check failed:\n'
  old_ifs="$IFS"
  IFS=$'\n'
  for violation in $violations; do
    [[ -n "$violation" ]] || continue
    printf ' - %s\n' "$violation"
  done
  IFS="$old_ifs"
  exit 1
fi

printf 'Documentation consistency check passed.\n'
