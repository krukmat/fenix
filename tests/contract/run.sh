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
                --exclude-path-regex '^/api/v1/copilot/.*' \
                --max-examples "${CONTRACT_MAX_EXAMPLES:-10}" \
                --suppress-health-check=all
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
                --exclude-path-regex '^/api/v1/copilot/.*' \
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
          '{\"api\":[\"admin\"],\"global\":[\"admin\",\"read_all\"],\"*\":[\"*\"]}',
          datetime('now'), datetime('now'));
  INSERT INTO user_role (id, user_id, role_id, created_at)
  VALUES ('$USER_ROLE_ID', '$USER_ID', '$ROLE_ID', datetime('now'));
"

# Stable FK targets for Schemathesis-generated request bodies. Some create
# endpoints require existing owner/account IDs, and random strings become
# database FK errors instead of useful contract cases.
sqlite3 "$DB_FILE" "
  INSERT INTO user_account (
    id, workspace_id, email, display_name, status, created_at, updated_at
  )
  VALUES (
    'contract-user', '$WORKSPACE_ID', 'contract-owner@test.com',
    'Contract Owner', 'active', datetime('now'), datetime('now')
  );

  INSERT OR IGNORE INTO user_role (id, user_id, role_id, created_at)
  VALUES ('contract-owner-role', 'contract-user', '$ROLE_ID', datetime('now'));

  INSERT INTO account (
    id, workspace_id, name, owner_id, created_at, updated_at
  )
  VALUES (
    'contract-account', '$WORKSPACE_ID', 'Contract Fixture Account',
    'contract-user', datetime('now'), datetime('now')
  );

  INSERT INTO contact (
    id, workspace_id, account_id, first_name, last_name, email, status, owner_id,
    created_at, updated_at
  )
  VALUES (
    'contract-contact', '$WORKSPACE_ID', 'contract-account', 'Contract',
    'Contact', 'contract-contact@test.com', 'active', 'contract-user',
    datetime('now'), datetime('now')
  );

  INSERT INTO pipeline (
    id, workspace_id, name, entity_type, created_at, updated_at
  )
  VALUES (
    'contract-pipeline', '$WORKSPACE_ID', 'Contract Fixture Pipeline',
    'deal', datetime('now'), datetime('now')
  );

  INSERT INTO pipeline_stage (
    id, pipeline_id, name, position, probability, created_at, updated_at
  )
  VALUES (
    'contract-stage', 'contract-pipeline', 'Contract Fixture Stage',
    1, 0.5, datetime('now'), datetime('now')
  );
"

# Stable AI-layer FK targets for Schemathesis-generated request bodies and
# hard-coded specialized agent runtimes.
sqlite3 "$DB_FILE" "
  INSERT OR IGNORE INTO agent_definition (
    id, workspace_id, name, description, agent_type, objective, allowed_tools,
    limits, trigger_config, status, created_at, updated_at
  )
  VALUES
    (
      'contract-agent', '$WORKSPACE_ID', 'Contract Fixture Agent',
      'Generic contract-test agent', 'custom',
      '{\"role\":\"contract_fixture\"}', '[]',
      '{\"max_cost_day\":10,\"max_runs_day\":100}', '{\"type\":\"manual\"}',
      'active', datetime('now'), datetime('now')
    ),
    (
      'support-agent', '$WORKSPACE_ID', 'Contract Support Agent',
      'Support contract-test agent', 'support',
      '{\"role\":\"support\"}', '[\"update_case\",\"send_reply\",\"search_knowledge\"]',
      '{\"max_cost_day\":10,\"max_runs_day\":100}', '{\"type\":\"manual\"}',
      'active', datetime('now'), datetime('now')
    ),
    (
      'kb-agent', '$WORKSPACE_ID', 'Contract KB Agent',
      'KB contract-test agent', 'kb',
      '{\"role\":\"knowledge_curator\"}', '[\"search_knowledge\",\"create_knowledge_item\"]',
      '{\"max_cost_day\":10,\"max_runs_day\":100}', '{\"type\":\"manual\"}',
      'active', datetime('now'), datetime('now')
    ),
    (
      'prospecting-agent', '$WORKSPACE_ID', 'Contract Prospecting Agent',
      'Prospecting contract-test agent', 'prospecting',
      '{\"role\":\"sales_dev\"}', '[\"search_knowledge\",\"create_task\"]',
      '{\"max_cost_day\":10,\"max_runs_day\":100}', '{\"type\":\"manual\"}',
      'active', datetime('now'), datetime('now')
    ),
    (
      'insights-agent', '$WORKSPACE_ID', 'Contract Insights Agent',
      'Insights contract-test agent', 'insights',
      '{\"role\":\"business_analyst\"}', '[\"search_knowledge\",\"query_metrics\"]',
      '{\"max_cost_day\":10,\"max_runs_day\":100}', '{\"type\":\"manual\"}',
      'active', datetime('now'), datetime('now')
    );

  INSERT OR IGNORE INTO prompt_version (
    id, workspace_id, agent_definition_id, version_number, system_prompt,
    user_prompt_template, config, status, created_by, created_at
  )
  VALUES
    (
      'contract-prompt-v1', '$WORKSPACE_ID', 'contract-agent', 1,
      'You are a deterministic contract-test assistant.',
      'Respond to contract-test input.', '{}', 'active', '$USER_ID',
      datetime('now')
    ),
    (
      'contract-prompt-v2', '$WORKSPACE_ID', 'contract-agent', 2,
      'You are a candidate deterministic contract-test assistant.',
      'Respond to contract-test input.', '{}', 'archived', '$USER_ID',
      datetime('now')
    );

  INSERT OR IGNORE INTO approval_request (
    id, workspace_id, requested_by, approver_id, action, resource_type,
    resource_id, payload, reason, status, expires_at, created_at, updated_at
  )
  VALUES (
    'contract-approval', '$WORKSPACE_ID', '$USER_ID', '$USER_ID',
    'approve_contract_fixture', 'agent_definition', 'contract-agent',
    '{\"fixture\":true}', 'Contract test approval fixture', 'pending',
    datetime('now', '+1 day'), datetime('now'), datetime('now')
  );

  INSERT OR IGNORE INTO eval_suite (
    id, workspace_id, name, domain, test_cases, thresholds, created_at, updated_at
  )
  VALUES (
    'contract-eval-suite', '$WORKSPACE_ID', 'Contract Eval Suite', 'general',
    '[{\"input\":\"contract\",\"expected_keywords\":[\"contract\"],\"should_abstain\":false}]',
    '{\"groundedness\":0.5,\"exactitude\":0.5,\"abstention\":0.5,\"policy\":0.5}',
    datetime('now'), datetime('now')
  );

  INSERT OR IGNORE INTO lead (
    id, workspace_id, contact_id, account_id, source, status, owner_id, score,
    metadata, created_at, updated_at
  )
  VALUES (
    'contract-lead', '$WORKSPACE_ID', 'contract-contact', 'contract-account',
    'contract_test', 'new', 'contract-user', 50,
    '{\"fixture\":true}', datetime('now'), datetime('now')
  );

  INSERT OR IGNORE INTO case_ticket (
    id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id,
    subject, description, priority, status, channel, metadata, created_at,
    updated_at
  )
  VALUES (
    'contract-case', '$WORKSPACE_ID', 'contract-account', 'contract-contact',
    'contract-pipeline', 'contract-stage', 'contract-user',
    'Contract fixture case', 'Resolved fixture case for contract tests',
    'medium', 'resolved', 'contract_test', '{\"fixture\":true}',
    datetime('now'), datetime('now')
  );
"

echo "Running contract tests in mode: $CONTRACT_MODE"
run_schemathesis
