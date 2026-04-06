#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Go pre-push QA"
echo "Root: $ROOT_DIR"

cd "$ROOT_DIR"

echo "==> Gate: fmt-check"
make fmt-check

echo "==> Gate: complexity"
make complexity

echo "==> Gate: lint"
make lint

echo "==> Gate: test"
make test

echo "==> Gate: coverage-gate"
make coverage-gate

echo "==> Gate: coverage-tdd"
make coverage-tdd

echo "==> Gate: deadcode"
deadcode -test ./... 2>&1 \
  | grep -v "mcp_adapter\|MCPGateway\|BuildServer\|MCPResourceProvider\|MCPResourceDescriptor\|MCPResourcePayload" \
  | grep -v "_test\.go:\|ruleguard" \
  | tee /tmp/deadcode-report.txt || true
LINES=$(grep -c "." /tmp/deadcode-report.txt 2>/dev/null || echo 0)
echo "Dead code findings (after MCP allowlist): $LINES"
if [ "$LINES" -gt 0 ]; then
  echo "FAILED: $LINES unexpected dead code finding(s)"
  exit 1
fi
echo "PASSED: deadcode gate"

if [ -f .venv/bin/doorstop ]; then
  echo "==> Gate: traceability (doorstop + bdd-trace)"
  make doorstop-check
  make bdd-trace-check
else
  echo "==> Gate: traceability — SKIPPED (no .venv/bin/doorstop found)"
fi

if echo "${CHANGED_FILES:-}" | grep -qE '(go\.mod|go\.sum)'; then
  echo "==> Gate: govulncheck (dependency changes detected)"
  govulncheck ./...
else
  echo "==> Gate: govulncheck — SKIPPED (no go.mod/go.sum changes)"
fi

echo "==> Gate: race-stability (count=1 local, count=3 in CI)"
RACE_STABILITY_COUNT=1 make race-stability

echo "==> Gate: pattern-refactor-gate"
make pattern-refactor-gate

echo "==> Go pre-push QA passed"
