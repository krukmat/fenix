#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> BFF pre-push QA"
echo "Root: $ROOT_DIR"

cd "$ROOT_DIR/bff"

echo "==> Gate: typecheck"
npm run build -- --noEmit 2>/dev/null || npx tsc --noEmit

echo "==> Gate: lint (production-grade ESLint)"
npm run lint

echo "==> Gate: test:coverage"
npm run test:coverage

echo "==> BFF pre-push QA passed"
