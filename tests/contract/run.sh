#!/bin/bash
# Task Gateway: API contract test runner using Schemathesis.
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PORT=8081
DB_FILE=$(mktemp /tmp/fenix_contract_XXXXXX.db)
CONTRACT_MODE=${CONTRACT_MODE:-smoke}

run_schemathesis() {
    case "$CONTRACT_MODE" in
        strict)
            # Strict contract mode: fail on API/OpenAPI drift.
            ./.venv/bin/schemathesis run "$PROJECT_ROOT/docs/openapi.yaml" \
                --url "http://localhost:$PORT" \
                --header "Authorization: Bearer $TOKEN" \
                --header "Content-Type: application/json" \
                --seed 1 \
                --checks not_a_server_error \
                --checks status_code_conformance \
                --checks response_schema_conformance \
                --checks content_type_conformance \
                --phases examples,fuzzing \
                --max-examples "${CONTRACT_MAX_EXAMPLES:-10}"
            ;;
        smoke)
            # Fast smoke mode: quick signal without full contract hardening.
            ./.venv/bin/schemathesis run "$PROJECT_ROOT/docs/openapi.yaml" \
                --url "http://localhost:$PORT" \
                --header "Authorization: Bearer $TOKEN" \
                --header "Content-Type: application/json" \
                --seed 1 \
                --checks not_a_server_error \
                --phases examples \
                --max-examples 1
            ;;
        *)
            echo "ERROR: CONTRACT_MODE invalid: '$CONTRACT_MODE' (allowed: smoke|strict)"
            exit 1
            ;;
    esac
}

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

REGISTER_RESP=$(curl -sf -X POST "http://localhost:$PORT/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"email":"contract@test.com","password":"ContractTest1234!","displayName":"Contract Tester","workspaceName":"ContractWS"}')

TOKEN=$(echo "$REGISTER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
USER_ID=$(echo "$REGISTER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['userId'])")
WORKSPACE_ID=$(echo "$REGISTER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['workspaceId'])")

# Grant admin role so /admin/*, /workflows/*, and /signals/* return 2xx instead of 403.
# Role is resolved per-request from DB (not embedded in JWT), so this takes effect immediately.
ROLE_ID=$(python3 -c "import uuid; print(str(uuid.uuid4()))")
USER_ROLE_ID=$(python3 -c "import uuid; print(str(uuid.uuid4()))")
sqlite3 "$DB_FILE" "
  INSERT INTO role (id, workspace_id, name, permissions, created_at, updated_at)
  VALUES ('$ROLE_ID', '$WORKSPACE_ID', 'contract-admin',
          '{\"api\":[\"admin\"],\"global\":[\"read_all\"]}',
          datetime('now'), datetime('now'));
  INSERT INTO user_role (id, user_id, role_id, created_at)
  VALUES ('$USER_ROLE_ID', '$USER_ID', '$ROLE_ID', datetime('now'));
"

echo "Running contract tests in mode: $CONTRACT_MODE"
run_schemathesis
