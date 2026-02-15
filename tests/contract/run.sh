#!/bin/bash
# Task Gateway: API contract test runner using Schemathesis.
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PORT=8081
DB_FILE=$(mktemp /tmp/fenix_contract_XXXXXX.db)

cleanup() {
    kill "$SERVER_PID" 2>/dev/null || true
    rm -f "$DB_FILE"
}
trap cleanup EXIT

cd "$PROJECT_ROOT"
make build

JWT_SECRET="test-secret-32-chars-minimum!!!" DATABASE_URL="$DB_FILE" "$PROJECT_ROOT/fenix" serve --port "$PORT" &
SERVER_PID=$!

for i in $(seq 1 30); do
    if curl -sf "http://localhost:$PORT/health" >/dev/null 2>&1; then
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "ERROR: Server did not start"
        exit 1
    fi
    sleep 1
done

TOKEN=$(curl -sf -X POST "http://localhost:$PORT/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"email":"contract@test.com","password":"ContractTest1234!","displayName":"Contract Tester","workspaceName":"ContractWS"}' \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

./.venv/bin/schemathesis run "$PROJECT_ROOT/docs/openapi.yaml" \
    --url "http://localhost:$PORT" \
    --header "Authorization: Bearer $TOKEN" \
    --header "Content-Type: application/json" \
    --checks all \
    --seed 1 \
    --phases examples,coverage,fuzzing,stateful
