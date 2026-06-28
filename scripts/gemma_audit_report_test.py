#!/usr/bin/env python3
"""Unit tests for gemma-audit-report.py."""

import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import unittest

_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "gemma-audit-report.py")
_spec = importlib.util.spec_from_file_location("gemma_audit_report", _SCRIPT)
_mod = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_mod)


def _write_jsonl(directory, filename, records):
    path = os.path.join(directory, filename)
    with open(path, "w", encoding="utf-8") as f:
        for r in records:
            f.write(json.dumps(r) + "\n")
    return path


def _dev_record(**kwargs):
    base = {
        "ts": "2026-06-01T10:00:00Z",
        "role": "developer",
        "outcome": "PATCH",
        "done_reason": "stop",
        "mode": "full-file",
        "elapsed_s": 5.0,
        "escalated": False,
        "diff_added": 10,
        "diff_removed": 2,
        "scope_violations": 0,
        "apply_result": "clean",
        "verify_ok": None,
        "packet_tokens_est": None,
        "response_tokens": None,
        "task_id": None,
        "rri": None,
        "band": None,
        "attempt": None,
    }
    base.update(kwargs)
    return base


def _rev_record(**kwargs):
    base = {
        "ts": "2026-06-01T11:00:00Z",
        "role": "reviewer",
        "outcome": "PASS",
        "done_reason": "stop",
        "mode": "n/a",
        "elapsed_s": 8.0,
        "escalated": False,
        "findings_count": 0,
        "findings_by_severity": {"blocking": 0, "major": 0, "minor": 0, "nit": 0},
        "out_of_scope": 0,
        "dispositions": None,
        "disposition_divergence": None,
        "packet_tokens_est": None,
        "response_tokens": None,
        "task_id": None,
        "rri": None,
        "band": None,
        "attempt": None,
    }
    base.update(kwargs)
    return base


class LoadRecords(unittest.TestCase):
    def test_hp2_empty_directory_returns_no_records(self):
        with tempfile.TemporaryDirectory() as tmp:
            records, skipped = _mod.load_records(tmp, "all", None)
        self.assertEqual(records, [])
        self.assertEqual(skipped, 0)

    def test_hp2_missing_directory_returns_no_records(self):
        records, skipped = _mod.load_records("/nonexistent/path/xyz", "all", None)
        self.assertEqual(records, [])
        self.assertEqual(skipped, 0)

    def test_loads_records_from_jsonl(self):
        with tempfile.TemporaryDirectory() as tmp:
            _write_jsonl(tmp, "2026-06.jsonl", [_dev_record(), _rev_record()])
            records, skipped = _mod.load_records(tmp, "all", None)
        self.assertEqual(len(records), 2)
        self.assertEqual(skipped, 0)

    def test_role_filter_developer_excludes_reviewer(self):
        with tempfile.TemporaryDirectory() as tmp:
            _write_jsonl(tmp, "2026-06.jsonl", [_dev_record(), _rev_record()])
            records, _ = _mod.load_records(tmp, "developer", None)
        self.assertEqual(len(records), 1)
        self.assertEqual(records[0]["role"], "developer")

    def test_since_filter_excludes_older_months(self):
        with tempfile.TemporaryDirectory() as tmp:
            _write_jsonl(tmp, "2026-04.jsonl", [_dev_record()])
            _write_jsonl(tmp, "2026-06.jsonl", [_dev_record()])
            records, _ = _mod.load_records(tmp, "all", "2026-06")
        self.assertEqual(len(records), 1)

    def test_ec1_malformed_line_skipped_with_count(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "2026-06.jsonl")
            with open(path, "w") as f:
                f.write(json.dumps(_dev_record()) + "\n")
                f.write("NOT_JSON_AT_ALL\n")
                f.write(json.dumps(_rev_record()) + "\n")
            records, skipped = _mod.load_records(tmp, "all", None)
        self.assertEqual(len(records), 2)
        self.assertEqual(skipped, 1)


class ComputeMetrics(unittest.TestCase):
    def test_hp2_empty_records_returns_zero_total(self):
        m = _mod.compute_metrics([])
        self.assertEqual(m["total_records"], 0)

    def test_hp1_mixed_roles_counted_separately(self):
        records = [_dev_record(), _dev_record(), _rev_record()]
        m = _mod.compute_metrics(records)
        self.assertEqual(m["total_records"], 3)
        self.assertEqual(m["by_role"]["developer"], 2)
        self.assertEqual(m["by_role"]["reviewer"], 1)

    def test_escalated_count_and_blocked_rate(self):
        records = [
            _dev_record(escalated=True),
            _dev_record(escalated=True),
            _dev_record(escalated=False),
            _dev_record(escalated=False),
            _dev_record(escalated=False),
        ]
        m = _mod.compute_metrics(records)
        self.assertEqual(m["escalated_count"], 2)
        self.assertAlmostEqual(m["blocked_rate"], 0.4, places=3)

    def test_threshold_flag_escalation_rate(self):
        records = [_dev_record(escalated=True)] * 3 + [_dev_record(escalated=False)] * 2
        m = _mod.compute_metrics(records)
        self.assertTrue(any("escalation_rate" in f for f in m["threshold_flags"]))

    def test_no_threshold_flags_when_all_clean(self):
        records = [_dev_record(), _rev_record()]
        m = _mod.compute_metrics(records)
        self.assertEqual(m["threshold_flags"], [])

    def test_ec2_null_optional_fields_tolerated(self):
        record = _dev_record(elapsed_s=None, diff_added=None, diff_removed=None)
        m = _mod.compute_metrics([record])
        self.assertIsNone(m["mean_elapsed_s"])
        self.assertEqual(m["destructive_diff_count"], 0)

    def test_findings_by_severity_aggregated(self):
        rev1 = _rev_record(
            outcome="FINDINGS",
            findings_count=2,
            findings_by_severity={"blocking": 1, "major": 1, "minor": 0, "nit": 0},
        )
        rev2 = _rev_record(
            outcome="FINDINGS",
            findings_count=1,
            findings_by_severity={"blocking": 0, "major": 0, "minor": 1, "nit": 0},
        )
        m = _mod.compute_metrics([rev1, rev2])
        self.assertEqual(m["findings_by_severity"]["blocking"], 1)
        self.assertEqual(m["findings_by_severity"]["major"], 1)
        self.assertEqual(m["findings_by_severity"]["minor"], 1)
        self.assertEqual(m["total_findings"], 3)

    def test_out_of_scope_rate_threshold_flag(self):
        records = [
            _rev_record(
                outcome="FINDINGS",
                findings_count=10,
                out_of_scope=2,
                findings_by_severity={"blocking": 0, "major": 2, "minor": 8, "nit": 0},
            )
        ]
        m = _mod.compute_metrics(records)
        self.assertAlmostEqual(m["out_of_scope_rate"], 0.2, places=3)
        self.assertTrue(any("out_of_scope_rate" in f for f in m["threshold_flags"]))

    def test_truncation_rate_flag(self):
        records = [_dev_record(done_reason="length"), _dev_record()]
        m = _mod.compute_metrics(records)
        self.assertEqual(m["truncated_count"], 1)
        self.assertTrue(any("truncation_rate" in f for f in m["threshold_flags"]))

    def test_destructive_diff_detected(self):
        records = [
            _dev_record(diff_added=1, diff_removed=10, outcome="PATCH"),
        ]
        m = _mod.compute_metrics(records)
        self.assertEqual(m["destructive_diff_count"], 1)
        self.assertTrue(any("destructive_diff" in f for f in m["threshold_flags"]))

    def test_mean_elapsed_s(self):
        records = [_dev_record(elapsed_s=4.0), _dev_record(elapsed_s=6.0)]
        m = _mod.compute_metrics(records)
        self.assertAlmostEqual(m["mean_elapsed_s"], 5.0, places=3)

    def test_token_telemetry_summaries_ignore_nulls(self):
        records = [
            _dev_record(packet_tokens_est=100, response_tokens=30),
            _rev_record(packet_tokens_est=None, response_tokens=None),
            _rev_record(packet_tokens_est=140, response_tokens=50),
        ]
        m = _mod.compute_metrics(records)
        self.assertEqual(m["packet_tokens_est"]["records_with_data"], 2)
        self.assertEqual(m["packet_tokens_est"]["sum"], 240)
        self.assertAlmostEqual(m["packet_tokens_est"]["mean"], 120.0, places=3)
        self.assertEqual(m["response_tokens"]["records_with_data"], 2)
        self.assertEqual(m["response_tokens"]["sum"], 80)
        self.assertAlmostEqual(m["response_tokens"]["mean"], 40.0, places=3)

    def test_token_telemetry_all_null_stays_none(self):
        m = _mod.compute_metrics([_dev_record(), _rev_record()])
        self.assertEqual(m["packet_tokens_est"]["records_with_data"], 0)
        self.assertIsNone(m["packet_tokens_est"]["sum"])
        self.assertIsNone(m["packet_tokens_est"]["mean"])
        self.assertEqual(m["response_tokens"]["records_with_data"], 0)
        self.assertIsNone(m["response_tokens"]["sum"])
        self.assertIsNone(m["response_tokens"]["mean"])


class FormatText(unittest.TestCase):
    def test_empty_records_prints_no_records(self):
        output = _mod.format_text({"total_records": 0}, skipped=0)
        self.assertIn("no records found", output)

    def test_skipped_count_shown_when_nonzero(self):
        output = _mod.format_text({"total_records": 0}, skipped=3)
        self.assertIn("3", output)

    def test_threshold_flags_shown(self):
        m = _mod.compute_metrics(
            [_dev_record(escalated=True)] * 5
        )
        output = _mod.format_text(m, skipped=0)
        self.assertIn("escalation_rate", output)

    def test_token_telemetry_block_shown_when_data_present(self):
        m = _mod.compute_metrics([
            _dev_record(packet_tokens_est=100, response_tokens=30),
            _rev_record(packet_tokens_est=140, response_tokens=50),
        ])
        output = _mod.format_text(m, skipped=0)
        self.assertIn("token_telemetry:", output)
        self.assertIn("response_tokens:", output)
        self.assertIn("packet_tokens_est:", output)

    def test_token_telemetry_block_omitted_when_no_data(self):
        m = _mod.compute_metrics([_dev_record(), _rev_record()])
        output = _mod.format_text(m, skipped=0)
        self.assertNotIn("token_telemetry:", output)

    def test_no_threshold_flags_label(self):
        m = _mod.compute_metrics([_dev_record()])
        output = _mod.format_text(m, skipped=0)
        self.assertIn("none", output)


class FormatJson(unittest.TestCase):
    def test_json_output_is_valid_and_has_required_keys(self):
        records = [_dev_record(), _rev_record()]
        m = _mod.compute_metrics(records)
        output = _mod.format_json(m, skipped=0)
        data = json.loads(output)
        for key in ("total_records", "by_role", "outcome_counts", "escalated_count",
                    "blocked_rate", "threshold_flags", "malformed_skipped"):
            self.assertIn(key, data, f"missing key: {key}")

    def test_malformed_skipped_in_json(self):
        m = _mod.compute_metrics([])
        output = _mod.format_json(m, skipped=7)
        data = json.loads(output)
        self.assertEqual(data["malformed_skipped"], 7)


class CliBehavior(unittest.TestCase):
    def _run(self, *args, log_dir=None):
        with tempfile.TemporaryDirectory() as tmp:
            if log_dir is None:
                log_dir = tmp
            cmd = [sys.executable, _SCRIPT, "--log-dir", log_dir] + list(args)
            result = subprocess.run(cmd, capture_output=True, text=True)
            return result

    def test_help_exits_zero(self):
        r = subprocess.run([sys.executable, _SCRIPT, "--help"],
                           capture_output=True, text=True)
        self.assertEqual(r.returncode, 0)

    def test_empty_directory_exits_zero_no_records(self):
        r = self._run()
        self.assertEqual(r.returncode, 0)
        self.assertIn("no records", r.stdout)

    def test_format_json_produces_valid_json(self):
        with tempfile.TemporaryDirectory() as tmp:
            _write_jsonl(tmp, "2026-06.jsonl", [_dev_record()])
            cmd = [sys.executable, _SCRIPT, "--log-dir", tmp, "--format", "json"]
            r = subprocess.run(cmd, capture_output=True, text=True)
        self.assertEqual(r.returncode, 0)
        data = json.loads(r.stdout)
        self.assertEqual(data["total_records"], 1)

    def test_role_filter_via_cli(self):
        with tempfile.TemporaryDirectory() as tmp:
            _write_jsonl(tmp, "2026-06.jsonl", [_dev_record(), _rev_record()])
            cmd = [sys.executable, _SCRIPT, "--log-dir", tmp,
                   "--role", "reviewer", "--format", "json"]
            r = subprocess.run(cmd, capture_output=True, text=True)
        self.assertEqual(r.returncode, 0)
        data = json.loads(r.stdout)
        self.assertEqual(data["total_records"], 1)
        self.assertEqual(data["by_role"], {"reviewer": 1})


if __name__ == "__main__":
    unittest.main(verbosity=2)
