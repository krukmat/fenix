#!/usr/bin/env python3
"""Offline exhaustive status/result/error matrix for proposed S2S v1 (NOT E2E)."""
import json
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker

SCHEMA = Path(__file__).resolve().parents[1] / "docs/contracts/dubbridge-fenix-v1-run.schema.json"
SUCCESS = {"succeeded", "completed_with_warnings"}
ERROR = {"failed", "denied", "indeterminate"}
STATUSES = [
    "accepted", "running", "awaiting_approval", "handed_off", "succeeded",
    "completed_with_warnings", "abstained", "failed", "denied", "indeterminate",
]


def main():
    raw = json.loads(SCHEMA.read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(raw)
    validator = Draft202012Validator(raw, format_checker=FormatChecker())
    base = {
        "contract_version": "1",
        "operation_id": "analysis-operation-0001",
        "run_id": "550e8400-e29b-41d4-a716-446655440002",
        "trace_id": "trace-0001",
    }
    allowed = rejected = 0
    for status in STATUSES:
        for shape in ("null", "typed", "malformed"):
            for with_error in (False, True):
                result = (
                    None if shape == "null" else
                    {"summary": "Completed", "recommendations": []} if shape == "typed" else
                    {"unexpected_payload": 1}
                )
                candidate = {**base, "status": status, "result": result}
                if with_error:
                    candidate["error_code"] = "EXECUTION_FAILED"
                expected = (
                    (status in SUCCESS and shape == "typed" and not with_error)
                    or (status in ERROR and shape == "null" and with_error)
                    or (status not in SUCCESS | ERROR and shape == "null" and not with_error)
                )
                actual = validator.is_valid(candidate)
                if actual != expected:
                    raise AssertionError(
                        f"state={status} shape={shape} error={with_error}: "
                        f"expected={expected}, got={actual}"
                    )
                allowed += int(actual)
                rejected += int(not actual)
    print(json.dumps({
        "kind": "FENIX_B1_STATUS_MATRIX_V1",
        "status": "PASS",
        "allowed": allowed,
        "rejected": rejected,
        "combinations": allowed + rejected,
        "scope": "offline schema only; NOT provider runtime, NOT E2E",
    }))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
