#!/usr/bin/env python3
"""Exhaustive neutral S2S envelope status/result/run/error matrix (offline only).

This proves schema invariants, not S2S authorization or real workflow execution.
"""
import json
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker

SCHEMA = Path(__file__).resolve().parents[1] / 'docs/contracts/dubbridge-fenix-v1-envelope-response.schema.json'
SUCCESS = {'succeeded', 'completed_with_warnings'}
ERROR = {'failed', 'denied', 'indeterminate'}
STATUSES = ('accepted','running','awaiting_approval','handed_off','succeeded',
            'completed_with_warnings','abstained','failed','denied','indeterminate')


def main():
    raw = json.loads(SCHEMA.read_text(encoding='utf-8'))
    Draft202012Validator.check_schema(raw)
    validator = Draft202012Validator(raw, format_checker=FormatChecker())
    base = {
        'contract_version': '1',
        'operation_id': 'dubbridge-request-0001',
        'capability_key': 'capability.example',
        'trace_id': 'fenix-trace-0001',
    }
    seen = {'accepted': 0, 'rejected': 0}
    for status in STATUSES:
        for run_id in (None, '550e8400-e29b-41d4-a716-446655440002'):
            for shape in ('null', 'valid', 'unsafe'):
                for with_error in (False, True):
                    result = (None if shape == 'null' else
                              {'schema_version': '1', 'data': {}} if shape == 'valid' else
                              {'reasoningTrace': ['private']})
                    item = {**base, 'status': status, 'run_id': run_id, 'result': result}
                    if with_error:
                        item['error_code'] = 'EXECUTION_FAILED'
                    expected = (
                        (status in SUCCESS and run_id is not None and shape == 'valid' and not with_error)
                        or (status in ERROR and shape == 'null' and with_error)
                        or (status not in SUCCESS | ERROR and shape == 'null' and not with_error
                            and (status == 'accepted' or run_id is not None))
                    )
                    actual = validator.is_valid(item)
                    if expected != actual:
                        raise AssertionError(f'mismatch status={status}, run_id={run_id}, '
                                             f'result={shape}, error={with_error}: '
                                             f'expected {expected}, actual {actual}')
                    seen['accepted' if actual else 'rejected'] += 1
    assert sum(seen.values()) == 120
    print(json.dumps({'kind': 'DUBBRIDGE_FENIX_NEUTRAL_ENVELOPE_MATRIX_V1',
                      'status': 'PASS', 'cases': sum(seen.values()),
                      **seen, 'scope': 'offline schema only; NOT S2S/E2E'}, sort_keys=True))


if __name__ == '__main__':
    main()
