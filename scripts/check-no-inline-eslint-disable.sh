#!/usr/bin/env bash
# Gate script: Fail if any inline eslint-disable comments detected in mobile/ or bff/

set -euo pipefail

PATTERN='eslint-disable'
DIRS="mobile bff"

echo "🔍 Checking for inline eslint-disable comments..."

if grep -r --include="*.ts" --include="*.tsx" --include="*.js" --include="*.jsx" \
     -l "$PATTERN" $DIRS 2>/dev/null | grep -q .; then
  echo "❌ ERROR: inline eslint-disable detected:"
  grep -r --include="*.ts" --include="*.tsx" --include="*.js" --include="*.jsx" \
       -n "$PATTERN" $DIRS || true
  exit 1
fi

echo "✅ OK: no inline eslint-disable found."
exit 0
