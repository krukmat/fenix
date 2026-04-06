#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Mobile pre-push QA"
echo "Root: $ROOT_DIR"

cd "$ROOT_DIR"

echo "==> Gate: no inline eslint-disable"
bash scripts/check-no-inline-eslint-disable.sh

cd "$ROOT_DIR/mobile"

echo "==> Gate: typecheck"
npm run typecheck

echo "==> Gate: lint"
npm run lint

echo "==> Gate: architecture"
npm run quality:arch

echo "==> Gate: coverage"
npm run test:coverage

echo "==> Mobile pre-push QA passed"
