#!/usr/bin/env python3
"""Capability-neutral DubBridge→Fenix envelope schema checks (NOT live S2S/E2E).

Requirements: python3 + jsonschema>=4.18
Run: python3 scripts/int_b1_envelope_selftest.py
A PASS validates only structural schemas. Fenix must additionally enforce
registered capability input/output schemas and authorization at runtime.
"""
import copy
import json
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker

ROOT = Path(__file__).resolve().parents[1] / "docs" / "contracts"


def schema(name):
    raw = json.loads((ROOT / f"dubbridge-fenix-v1-envelope-{name}.schema.json").read_text())
    Draft202012Validator.check_schema(raw)
    return Draft202012Validator(raw, format_checker=FormatChecker())


def main():
    req, res, err = (schema(part) for part in ("request", "response", "error"))
    request = {
        "contract_version": "1",
        "operation_id": "dubbridge-request-0001",
        "capability_key": "capability.example",
        "input_schema_version": "1",
        "trace_id": "dubbridge-trace-0001",
        "source": {
            "product": "dubbridge",
            "org_id": "550e8400-e29b-41d4-a716-446655440000",
        },
        "input": {},
    }
    response = {
        "contract_version": "1",
        "operation_id": request["operation_id"],
        "capability_key": request["capability_key"],
        "status": "accepted",
        "run_id": None,
        "trace_id": "fenix-trace-0001",
        "result": None,
    }
    error = {
        "contract_version": "1", "error_code": "CAPABILITY_NOT_CONFIGURED",
        "message": "Capability unavailable",
    }
    positives = negatives = 0

    for validator, sample in ((req, request), (res, response), (err, error)):
        validator.validate(sample)
        positives += 1

    def should_reject(validator, sample, label):
        nonlocal negatives
        if validator.is_valid(sample):
            raise AssertionError("unexpectedly accepted: " + label)
        negatives += 1

    for field, value, label in (
        ("contract_version", "2", "contract_version"),
        ("capability_key", "unsafe workflow selection", "unregistered_syntax"),
        ("operation_id", "tiny", "short_operation_id"),
        ("input_schema_version", "0", "schema_version"),
    ):
        bad = copy.deepcopy(request)
        bad[field] = value
        should_reject(req, bad, label)

    bad = copy.deepcopy(request)
    bad["workflow_id"] = "some-secret-workflow"
    should_reject(req, bad, "direct_workflow_id")

    bad = copy.deepcopy(request)
    bad["source"]["org_id"] = "not-uuid"
    should_reject(req, bad, "invalid_org_id")

    bad = copy.deepcopy(request)
    bad["source"]["reviewer_email"] = "user@example.invalid"
    should_reject(req, bad, "unapproved_source_pii")

    bad = copy.deepcopy(response)
    bad["toolCalls"] = []
    should_reject(res, bad, "internal_tools_leak")

    bad = copy.deepcopy(response)
    bad["status"] = "succeeded"
    should_reject(res, bad, "fake_success_without_run_or_result")

    bad = copy.deepcopy(response)
    bad["status"] = "running"
    should_reject(res, bad, "running_without_run_id")

    bad = copy.deepcopy(response)
    bad["status"] = "failed"
    should_reject(res, bad, "failed_without_error_code")

    bad = copy.deepcopy(error)
    bad["error_code"] = "MODEL_STARTED"
    should_reject(err, bad, "undeclared_error")

    success = copy.deepcopy(response)
    success["status"] = "succeeded"
    success["run_id"] = "550e8400-e29b-41d4-a716-446655440002"
    success["result"] = {"schema_version": "1", "data": {}}
    res.validate(success)
    positives += 1

    failed = copy.deepcopy(response)
    failed["status"] = "denied"
    failed["error_code"] = "POLICY_DENIED"
    res.validate(failed)
    positives += 1

    print(json.dumps({
        "kind": "DUBBRIDGE_FENIX_NEUTRAL_ENVELOPE_SELFTEST",
        "result": "PASS",
        "positive": positives, "negative": negatives,
        "scope": "draft schema only; not capability auth, live runtime or E2E",
    }))


if __name__ == "__main__":
    main()
