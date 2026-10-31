#!/usr/bin/env python3
# ARCHIVED example-specific test for review_backlog draft. Not the active generic S2S envelope.
# Generic contract verification lives in scripts/int_b1_envelope_selftest.py.
"""B1 contract schema unit checks only. This NEVER certifies a live integration.

Usage from repository root:
    python3 scripts/int_b1_contract_selftest.py

Requires jsonschema with Draft 2020-12 support. Does not contact networks,
read credentials, start services, or fabricate E2E acceptance evidence.
"""
import copy
import json
import sys
from pathlib import Path

try:
    from jsonschema import Draft202012Validator, FormatChecker
except ImportError:
    sys.exit("B1 CONTRACT_SELFTEST=BLOCKED: install jsonschema in a disposable local dev environment")

ROOT = Path(__file__).resolve().parents[1] / "docs" / "contracts"


def load_schema(name):
    data = json.loads((ROOT / name).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(data)
    return Draft202012Validator(data, format_checker=FormatChecker())


def main():
    request = load_schema("dubbridge-fenix-v1-request.schema.json")
    response = load_schema("dubbridge-fenix-v1-run.schema.json")
    valid_request = {
        "contract_version": "1",
        "operation_id": "analysis-operation-0001",
        "operation": "dubbridge.review_backlog.analyze",
        "trace_id": "trace-0001",
        "source": {
            "product": "dubbridge",
            "org_id": "550e8400-e29b-41d4-a716-446655440000",
            "project_id": "550e8400-e29b-41d4-a716-446655440001",
        },
        "input": {"review_counts": {"pending": 2, "approved": 1, "rejected": 0}},
    }
    valid_response = {
        "contract_version": "1",
        "operation_id": "analysis-operation-0001",
        "run_id": "550e8400-e29b-41d4-a716-446655440002",
        "trace_id": "trace-0001",
        "status": "accepted",
        "result": None,
    }
    request.validate(valid_request)
    response.validate(valid_response)

    negatives = []
    def must_reject(validator, sample, label):
        if validator.is_valid(sample):
            raise AssertionError("contract erroneously accepted: " + label)
        negatives.append(label)

    bad = copy.deepcopy(valid_request)
    bad["input"]["review_counts"]["pending"] = -1
    must_reject(request, bad, "negative_count")

    bad = copy.deepcopy(valid_request)
    bad["source"]["project_id"] = "not-a-uuid"
    must_reject(request, bad, "invalid_project_uuid")

    bad = copy.deepcopy(valid_request)
    bad["source"]["reviewer_email"] = "secret@example.invalid"
    must_reject(request, bad, "source_pii")

    bad = copy.deepcopy(valid_request)
    bad["input"]["reviews"] = [{"id": "unallowed"}]
    must_reject(request, bad, "raw_review_payload")

    bad = copy.deepcopy(valid_request)
    bad["workflow_id"] = "user-selected-workflow"
    must_reject(request, bad, "client_workflow_selection")

    bad = copy.deepcopy(valid_request)
    bad["operation_id"] = "bad newline\nInjected: true"
    must_reject(request, bad, "unsafe_operation_id")

    bad = copy.deepcopy(valid_response)
    bad["reasoningTrace"] = [{"secret": "do not expose"}]
    must_reject(response, bad, "reasoning_trace_leak")

    bad = copy.deepcopy(valid_response)
    bad["status"] = "succeeded"
    must_reject(response, bad, "success_without_result")

    bad = copy.deepcopy(valid_response)
    bad["status"] = "succeeded"
    bad["result"] = {"summary": "Prioritize pending reviews", "recommendations": []}
    response.validate(bad)

    bad = copy.deepcopy(valid_response)
    bad["result"] = {"summary": "OK", "recommendations": [], "toolCalls": []}
    must_reject(response, bad, "tool_calls_leak")

    # Semantic constraints: never claim that a pending, failed or denied
    # execution produced a successful advisory analysis.
    bad = copy.deepcopy(valid_response)
    bad["result"] = {"summary": "Unexpected advice", "recommendations": []}
    must_reject(response, bad, "accepted_with_result")

    bad = copy.deepcopy(valid_response)
    bad["status"] = "failed"
    bad["result"] = {"summary": "Unexpected success", "recommendations": []}
    bad["error_code"] = "EXECUTION_FAILED"
    must_reject(response, bad, "failed_with_success_result")

    bad = copy.deepcopy(valid_response)
    bad["status"] = "succeeded"
    bad["result"] = {"summary": "OK", "recommendations": []}
    bad["error_code"] = "EXECUTION_FAILED"
    must_reject(response, bad, "success_with_error_code")

    bad = copy.deepcopy(valid_response)
    bad["status"] = "denied"
    must_reject(response, bad, "denied_without_error_code")

    bad = copy.deepcopy(valid_response)
    bad["status"] = "abstained"
    bad["result"] = {"summary": "Unexpected advice", "recommendations": []}
    must_reject(response, bad, "abstained_with_result")

    bad = copy.deepcopy(valid_response)
    bad["status"] = "failed"
    bad["error_code"] = "EXECUTION_FAILED"
    response.validate(bad)

    # Accepted operation can be durably reserved before Fenix allocates agent_run.
    accepted_without_run = copy.deepcopy(valid_response)
    accepted_without_run["run_id"] = None
    response.validate(accepted_without_run)

    rejected_before_run = copy.deepcopy(accepted_without_run)
    rejected_before_run["status"] = "denied"
    rejected_before_run["error_code"] = "POLICY_DENIED"
    response.validate(rejected_before_run)

    bad = copy.deepcopy(valid_response)
    bad["status"] = "succeeded"
    bad["result"] = {"summary": "Done", "recommendations": []}
    bad["run_id"] = None
    must_reject(response, bad, "success_without_agent_run")

    bad = copy.deepcopy(valid_response)
    bad["status"] = "running"
    bad["run_id"] = None
    must_reject(response, bad, "running_without_agent_run")

    print(json.dumps({
        "kind": "FENIX_DUBBRIDGE_B1_CONTRACT_SELFTEST_V1",
        "status": "PASS",
        "positive": 6,
        "negative": len(negatives),
        "scope": "offline JSON schema only; NOT live API, NOT S2S, NOT E2E",
    }))
    return 0


if __name__ == "__main__":
    sys.exit(main())
