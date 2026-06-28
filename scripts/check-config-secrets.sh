#!/usr/bin/env bash
# Fenix config-secrets gate.
# Adapted from DubBridge check-config-secrets.sh; key differences:
#   - Scans .env.example files (fenix has no config/*.toml)
#   - Checks that actual .env files (not .example) are not git-tracked
#   - Placeholder values (empty, <…>, your-*, dev-*, changeme) are allowed in examples
set -euo pipefail

shopt -s nullglob

repo_root=$(
  cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1
  pwd
)
cd "$repo_root"

# When explicit files are passed, only scan those files (skip git-tracking check).
EXPLICIT_FILES=0
[[ $# -gt 0 ]] && EXPLICIT_FILES=1

has_violations=0

add_violation() {
  printf '%s\n' "$1" >&2
  has_violations=1
}

is_secret_key() {
  local key="$1"
  local normalized
  normalized="$(printf '%s' "$key" | tr '[:upper:]' '[:lower:]')"
  case "$normalized" in
    password|secret|token|key|\
    *_password|*_secret|*_token|*_key|\
    *password|*secret|*token)
      return 0 ;;
    *)
      return 1 ;;
  esac
}

# Placeholder values that are safe to commit in .env.example files.
is_placeholder_value() {
  local value="$1"
  # empty
  [[ -z "$value" ]] && return 0
  # angle-bracket template: <something>
  [[ "$value" == \<* ]] && return 0
  # dev- prefix (local dev defaults)
  [[ "$value" == dev-* ]] && return 0
  # your- prefix
  [[ "$value" == your-* ]] && return 0
  # changeme literal
  [[ "$value" == "changeme" ]] && return 0
  # placeholder or example literal
  [[ "$value" == "placeholder" || "$value" == "example" ]] && return 0
  return 1
}

# ── 1. Scan .env.example files for real secret values ────────────────────────

if [[ $EXPLICIT_FILES -eq 1 ]]; then
  example_files=("$@")
else
  example_files=()
  while IFS= read -r -d '' f; do
    example_files+=("$f")
  done < <(find . -name "*.env.example" -not -path "./.git/*" -print0 2>/dev/null)
  # Also catch .env.example at repo root (find may normalise path differently)
  [[ -f ".env.example" ]] && example_files+=("./.env.example")
fi

# Deduplicate via sort -u piped through read
unique_examples=()
while IFS= read -r f; do
  [[ -n "$f" ]] && unique_examples+=("$f")
done < <(printf '%s\n' "${example_files[@]}" | sort -u)

if [[ ${#unique_examples[@]} -eq 0 ]]; then
  printf 'No .env.example files found — skipping example scan.\n' >&2
else
  for file in "${unique_examples[@]}"; do
    [[ -f "$file" ]] || continue
    line_no=0
    while IFS= read -r line || [[ -n "$line" ]]; do
      line_no=$((line_no + 1))
      # Strip leading whitespace
      trimmed="${line#"${line%%[![:space:]]*}"}"
      # Skip blank lines, comments
      [[ -z "$trimmed" || "$trimmed" == \#* ]] && continue
      # Match KEY=VALUE
      if [[ "$trimmed" =~ ^([A-Za-z0-9_]+)[[:space:]]*=[[:space:]]*(.*) ]]; then
        key="${BASH_REMATCH[1]}"
        value="${BASH_REMATCH[2]}"
        # Strip inline comment from value
        value="${value%%#*}"
        # Strip surrounding quotes
        value="${value%\"}"
        value="${value#\"}"
        value="${value%\'}"
        value="${value#\'}"
        value="${value%% }"
        if is_secret_key "$key" && ! is_placeholder_value "$value"; then
          add_violation "Secret-looking key with non-placeholder value: ${file}:${line_no}: ${key}=${value}"
        fi
      fi
    done < "$file"
  done
fi

# ── 2. Check that real .env files are not git-tracked (full-repo mode only) ──

if [[ $EXPLICIT_FILES -eq 1 ]]; then
  # Explicit-file mode: skip git-tracking check
  tracked_env_files=""
else
  tracked_env_files="$(git ls-files | grep -E '(^|/)\.env$|(^|/)\.env\.[^e][^x]' || true)"
fi

# REPORT_ONLY_TRACKED: tracked .env violations are reported but do not cause exit 1.
# Remove this flag once bff/.env and mobile/.env are removed from git tracking.
REPORT_ONLY_TRACKED="${FENIX_SECRETS_REPORT_ONLY_TRACKED:-1}"

if [[ -n "$tracked_env_files" ]]; then
  while IFS= read -r tracked; do
    [[ -n "$tracked" ]] || continue
    # Skip .env.example and .env.e2e — those are intentionally committed
    [[ "$tracked" == *.env.example ]] && continue
    [[ "$tracked" == *.env.e2e ]] && continue
    if [[ "$REPORT_ONLY_TRACKED" == "1" ]]; then
      printf 'WARNING (report-only): real .env file is git-tracked: %s\n' "$tracked" >&2
    else
      add_violation "Real .env file is git-tracked (should be gitignored): ${tracked}"
    fi
  done <<EOF
$tracked_env_files
EOF
fi

# ── Result ────────────────────────────────────────────────────────────────────

if ((has_violations)); then
  printf '\nCommitted config files must not contain real secrets, and .env files must be gitignored.\n' >&2
  printf 'Move secrets to injected env vars and add real .env files to .gitignore.\n' >&2
  exit 1
fi

printf 'Config-secrets gate passed.\n'
