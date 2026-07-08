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

echo "==> Gate: wrapcheck-gate"
make wrapcheck-gate

echo "==> Gate: test"
make test

echo "==> Gate: coverage-gate"
make coverage-gate

echo "==> Gate: coverage-tdd"
make coverage-tdd

echo "==> Gate: deadcode"
# Resolve the deadcode binary explicitly. It is a `go install`ed tool that lives
# in $(go env GOPATH)/bin, which is not always on PATH when this script runs from
# the pre-push hook. Without this, a bare `deadcode` call fails with
# "command not found", and because stderr is folded into the pipe below, that
# shell error survives the grep filters and is miscounted as a real dead-code
# finding — a false failure. Fail loudly with an actionable message instead.
DEADCODE_BIN="$(command -v deadcode || true)"
if [ -z "$DEADCODE_BIN" ]; then
  DEADCODE_BIN="$(go env GOPATH)/bin/deadcode"
fi
if [ ! -x "$DEADCODE_BIN" ]; then
  echo "FAILED: deadcode tool not found. Install it with:"
  echo "  go install golang.org/x/tools/cmd/deadcode@latest"
  exit 1
fi
"$DEADCODE_BIN" -test ./... 2>&1 \
  | grep -v "mcp_adapter\|MCPGateway\|BuildServer\|MCPResourceProvider\|MCPResourceDescriptor\|MCPResourcePayload" \
  | grep -v "_test\.go:\|ruleguard" \
  | grep -v "bff/node_modules/" \
  | grep -v "web/node_modules/" \
  | tee /tmp/deadcode-report.txt || true
if [ -f /tmp/deadcode-report.txt ]; then
  LINES="$(wc -l < /tmp/deadcode-report.txt | tr -d '[:space:]')"
else
  LINES=0
fi
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
  # Resolve explicitly: it's a `go install`ed tool under $(go env GOPATH)/bin,
  # which is not always on PATH when this script runs from the pre-push hook.
  GOVULNCHECK_BIN="$(command -v govulncheck || true)"
  if [ -z "$GOVULNCHECK_BIN" ]; then
    GOVULNCHECK_BIN="$(go env GOPATH)/bin/govulncheck"
  fi
  if [ ! -x "$GOVULNCHECK_BIN" ]; then
    echo "FAILED: govulncheck tool not found. Install it with:"
    echo "  go install golang.org/x/vuln/cmd/govulncheck@latest"
    exit 1
  fi
  "$GOVULNCHECK_BIN" ./...
else
  echo "==> Gate: govulncheck — SKIPPED (no go.mod/go.sum changes)"
fi

echo "==> Gate: race-stability (count=1 local, count=3 in CI)"
RACE_STABILITY_COUNT=1 make race-stability

echo "==> Gate: pattern-refactor-gate"
make pattern-refactor-gate

echo "==> Go pre-push QA passed"
