#!/usr/bin/env bash
# Fenix task-ledger behavioral coverage validator.
# Adapted from DubBridge check-task-unit-coverage.sh; key differences:
#   - Test references use Go (_test.go::TestFuncName) or RN (.test.ts::desc > name) formats
#   - Gemma reviewer gate removed (Phase E, deferred)
#   - Reflection log gate removed (tied to Gemma quorum)
#   - Sentinel: "Behavioral coverage contract: unit-v1"
#
# Only validates task files that contain the sentinel line.
# All existing task files without the sentinel are silently skipped.
set -euo pipefail

shopt -s nullglob

repo_root=$(
  cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1
  pwd
)
cd "$repo_root"

SENTINEL="Behavioral coverage contract: unit-v1"

violations=""

add_violation() {
  if [[ -n "$violations" ]]; then
    violations="${violations}"$'\n'"$1"
  else
    violations="$1"
  fi
}

trim() {
  sed 's/^[[:space:]]*//; s/[[:space:]]*$//'
}

case_ids_for_prefix() {
  local section="$1"
  local prefix="$2"
  printf '%s\n' "$section" \
    | grep -E "^[[:space:]]*-[[:space:]]*(\*\*)?${prefix}-[0-9]+\b" \
    | grep -oE "${prefix}-[0-9]+" \
    | sort -u || true
}

is_completed_development_section() {
  local section="$1"
  printf '%s\n' "$section" | grep -Eq 'Status:.*\[[xX]\].*([Dd]one|DONE)' \
    && printf '%s\n' "$section" | grep -Eiq 'Type:.*development'
}

section_rri_value() {
  local section="$1"
  printf '%s\n' "$section" | sed -n 's/.*RRI:[[:space:]]*\([0-9][0-9]*\).*/\1/p' | head -n1
}

find_certification_row() {
  local section="$1"
  local case_id="$2"
  printf '%s\n' "$section" | grep -E "^\|[[:space:]]*(\*\*)?${case_id}(\*\*)?[[:space:]]*\|" || true
}

# Validate a single backtick-stripped test reference.
# Go format:   path/to/file_test.go::TestFuncName
# RN format:   path/to/file.test.ts::describe > test name
validate_test_ref() {
  local task_file="$1"
  local section_title="$2"
  local case_id="$3"
  local ref="$4"

  local ref_path ref_name
  ref_path="$(printf '%s\n' "$ref" | sed -n 's/^\(.*\)::\(.*\)$/\1/p')"
  ref_name="$(printf '%s\n' "$ref" | sed -n 's/^\(.*\)::\(.*\)$/\2/p')"

  if [[ -z "$ref_path" || -z "$ref_name" ]]; then
    add_violation "$task_file: $section_title: $case_id test evidence '$ref' must use path/to/file::name format"
    return
  fi

  if [[ ! -f "$ref_path" ]]; then
    add_violation "$task_file: $section_title: $case_id test evidence references missing file '$ref_path'"
    return
  fi

  if [[ "$ref_path" == *_test.go ]]; then
    # Go: look for func <TestFuncName>(
    if ! grep -Eq "^func[[:space:]]+${ref_name}[[:space:]]*\(" "$ref_path"; then
      add_violation "$task_file: $section_title: $case_id test evidence references missing Go test function '${ref_name}' in '$ref_path'"
    fi
  elif [[ "$ref_path" == *.test.ts || "$ref_path" == *.spec.ts || "$ref_path" == *.test.tsx ]]; then
    # RN/TS: look for it('name') or test('name') — match on the last segment after ' > '
    local test_name
    test_name="$(printf '%s\n' "$ref_name" | sed "s/.*>[[:space:]]*//")"
    if ! grep -Eq "(it|test)\(['\"]${test_name}['\"]" "$ref_path"; then
      add_violation "$task_file: $section_title: $case_id test evidence references missing RN test '${test_name}' in '$ref_path'"
    fi
  else
    add_violation "$task_file: $section_title: $case_id test evidence '$ref_path' must be a _test.go or .test.ts/.spec.ts file"
  fi
}

validate_case_certification() {
  local task_file="$1"
  local section_title="$2"
  local section="$3"
  local case_id="$4"

  local row
  row="$(find_certification_row "$section" "$case_id")"

  if [[ -z "$row" ]]; then
    add_violation "$task_file: $section_title: missing Unit coverage certification row for $case_id"
    return
  fi

  # Evidence is the second-to-last non-empty column; result is the last non-empty column.
  # This handles tables with any number of leading descriptor columns.
  local evidence result
  evidence="$(printf '%s\n' "$row" | awk -F'|' '{
    # collect non-empty fields
    n=0; for(i=2;i<=NF-1;i++){v=$i; gsub(/^[[:space:]]+|[[:space:]]+$/,"",v); if(v!="") a[n++]=v}
    if(n>=2) print a[n-2]; else if(n==1) print a[0]
  }' | trim)"
  result="$(printf '%s\n' "$row" | awk -F'|' '{
    n=0; for(i=2;i<=NF-1;i++){v=$i; gsub(/^[[:space:]]+|[[:space:]]+$/,"",v); if(v!="") a[n++]=v}
    if(n>=1) print a[n-1]
  }' | trim)"

  if [[ -z "$evidence" || "$evidence" =~ ^[Nn]/[Aa]$ ]]; then
    add_violation "$task_file: $section_title: $case_id has missing or N/A unit test evidence"
    return
  fi

  if [[ "$result" != "passed" ]]; then
    add_violation "$task_file: $section_title: $case_id result must be 'passed' (got '${result}')"
  fi

  local refs
  refs="$(printf '%s\n' "$evidence" \
    | grep -oE '`[^`]+::[^`]+`' \
    | tr -d '`' || true)"

  if [[ -z "$refs" ]]; then
    add_violation "$task_file: $section_title: $case_id unit test evidence must include at least one backticked path::name reference"
    return
  fi

  local ref
  while IFS= read -r ref; do
    [[ -n "$ref" ]] || continue
    validate_test_ref "$task_file" "$section_title" "$case_id" "$ref"
  done <<EOF
$refs
EOF
}

validate_owner_verification() {
  local task_file="$1"
  local section_title="$2"
  local section="$3"

  if ! printf '%s\n' "$section" | grep -q 'Owner final verification'; then
    add_violation "$task_file: $section_title: missing 'Owner final verification' block"
    return
  fi

  if ! printf '%s\n' "$section" | grep -Eq '^[[:space:]]*-[[:space:]]*Owner:[[:space:]]*[^[:space:]].*'; then
    add_violation "$task_file: $section_title: Owner final verification missing non-empty Owner"
  fi
  if ! printf '%s\n' "$section" | grep -Eq '^[[:space:]]*-[[:space:]]*Date:[[:space:]]*[0-9]{4}-[0-9]{2}-[0-9]{2}[[:space:]]*$'; then
    add_violation "$task_file: $section_title: Owner final verification Date must use YYYY-MM-DD"
  fi
  if ! printf '%s\n' "$section" | grep -Eq '^[[:space:]]*-[[:space:]]*Statement:[[:space:]]*.*'; then
    add_violation "$task_file: $section_title: Owner final verification missing Statement"
  fi
  if ! printf '%s\n' "$section" | grep -Eq '^[[:space:]]*-[[:space:]]*Commands run:[[:space:]]*[^[:space:]].*'; then
    add_violation "$task_file: $section_title: Owner final verification missing Commands run"
  fi
}

validate_section() {
  local task_file="$1"
  local section_title="$2"
  local section="$3"

  if ! is_completed_development_section "$section"; then
    return
  fi

  if ! printf '%s\n' "$section" | grep -q 'Unit coverage certification'; then
    add_violation "$task_file: $section_title: missing 'Unit coverage certification' section"
  fi
  if ! printf '%s\n' "$section" | grep -q 'Happy paths considered'; then
    add_violation "$task_file: $section_title: missing 'Happy paths considered' section"
  fi
  if ! printf '%s\n' "$section" | grep -q 'Edge cases considered'; then
    add_violation "$task_file: $section_title: missing 'Edge cases considered' section"
  fi

  local hp_ids ec_ids
  hp_ids="$(case_ids_for_prefix "$section" "HP")"
  ec_ids="$(case_ids_for_prefix "$section" "EC")"

  if [[ -z "$hp_ids" ]]; then
    add_violation "$task_file: $section_title: Happy paths must define at least one stable HP-# case ID"
  fi
  if [[ -z "$ec_ids" ]]; then
    add_violation "$task_file: $section_title: Edge cases must define at least one stable EC-# case ID"
  fi

  local case_id
  while IFS= read -r case_id; do
    [[ -n "$case_id" ]] || continue
    validate_case_certification "$task_file" "$section_title" "$section" "$case_id"
  done <<EOF
$hp_ids
$ec_ids
EOF

  validate_owner_verification "$task_file" "$section_title" "$section"
}

validate_task_file() {
  local task_file="$1"

  if ! grep -qF "$SENTINEL" "$task_file"; then
    return
  fi

  local total_lines
  total_lines="$(wc -l < "$task_file" | tr -d ' ')"

  local headings=()
  local heading
  while IFS= read -r heading; do
    [[ -n "$heading" ]] || continue
    headings+=("$heading")
  done < <(grep -nE '^##[[:space:]]' "$task_file" | cut -d: -f1 || true)

  if [[ ${#headings[@]} -eq 0 ]]; then
    add_violation "$task_file: unit-v1 ledger has no ## sections"
    return
  fi

  local i
  for ((i = 0; i < ${#headings[@]}; i++)); do
    local start end section section_title
    start="${headings[$i]}"
    if (( i + 1 < ${#headings[@]} )); then
      end=$(( headings[i + 1] - 1 ))
    else
      end="$total_lines"
    fi

    section="$(sed -n "${start},${end}p" "$task_file")"
    section_title="$(printf '%s\n' "$section" | sed -n '1s/^##[[:space:]]*//p')"
    validate_section "$task_file" "$section_title" "$section"
  done
}

if [[ "$#" -gt 0 ]]; then
  task_files=("$@")
else
  task_files=(docs/tasks/*.md)
fi

for task_file in "${task_files[@]}"; do
  [[ -f "$task_file" ]] || continue
  validate_task_file "$task_file"
done

if [[ -n "$violations" ]]; then
  printf 'Task completion evidence check failed:\n'
  old_ifs="$IFS"
  IFS=$'\n'
  for violation in $violations; do
    [[ -n "$violation" ]] || continue
    printf ' - %s\n' "$violation"
  done
  IFS="$old_ifs"
  exit 1
fi

printf 'Task completion evidence check passed.\n'
