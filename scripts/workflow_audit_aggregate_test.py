"""Offline tests for the read-only workflow performance audit."""
import json
import tempfile
import unittest
from pathlib import Path

import workflow_audit_aggregate as aggregate


class AggregateTest(unittest.TestCase):
    def test_empty_dirs_do_not_invent_metrics(self):
        with tempfile.TemporaryDirectory() as td:
            result = aggregate.aggregate(Path(td) / "local", Path(td) / "peer")
            self.assertEqual(result["status"], "no_records")
            self.assertIsNone(result["local"]["prompt_tokens"]["total"])
            self.assertIsNone(result["local"]["elapsed_seconds"]["p95"])

    def test_measured_tokens_and_peer_attempts(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / "local").mkdir()
            (root / "peer").mkdir()
            with (root / "local" / "2026-11.jsonl").open("w") as f:
                f.write(json.dumps({"model": "m", "role": "developer", "outcome": "PASS",
                    "prompt_tokens": 30, "packet_tokens_est": 48, "response_tokens": 7, "elapsed_s": 2.0,
                    "user_prompt": "VERY_PRIVATE_TOKEN"}) + "\n")
                f.write(json.dumps({"model": "m", "role": "reviewer", "outcome": "BLOCKED",
                    "response_tokens": None, "elapsed_s": 3.0}) + "\n")
            (root / "peer" / "a.json").write_text(json.dumps({
                "mode": "post-code-review", "verdict": {"status": "pass"},
                "packet": {"secret": "VERY_PRIVATE_TOKEN"},
                "attempts": [{"role": "primary"}, {"role": "fallback"}]
            }))
            result = aggregate.aggregate(root / "local", root / "peer")
            self.assertEqual(result["local"]["prompt_tokens"]["coverage_ratio"], 0.5)
            self.assertEqual(result["local"]["prompt_tokens"]["total"], 30)
            self.assertEqual(result["local"]["packet_tokens_est"]["total"], 48)
            self.assertEqual(result["local"]["by_model_metrics"]["m"]["p95_elapsed_seconds"], 3.0)
            self.assertEqual(result["local"]["by_model_metrics"]["m"]["prompt_tokens"]["total"], 30)
            self.assertEqual(result["local"]["elapsed_seconds"]["p95"], 3.0)
            self.assertEqual(result["peer"]["by_attempt_role"]["fallback"], 1)
            self.assertNotIn("VERY_PRIVATE_TOKEN", json.dumps(result))

    def test_invalid_lines_ignored(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / "2026-11.jsonl").write_text('not json\n[]\n{"role":"x","elapsed_s":-1}\n')
            result = aggregate.aggregate(root, root / "missing")
            self.assertEqual(result["evidence"]["local_records"], 1)
            self.assertEqual(result["local"]["elapsed_seconds"]["sample_count"], 0)

    def test_boolean_token_count_is_not_numeric(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / "2026-11.jsonl").write_text('{"prompt_tokens":true}\n')
            self.assertIsNone(aggregate.aggregate(root, root / "missing")["local"]["prompt_tokens"]["total"])

    def test_artifact_without_attempts_is_reported(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / "a.json").write_text('{"mode":"task-readiness","verdict":{"status":"blocked"}}')
            self.assertEqual(aggregate.aggregate(root / "missing", root)["peer"]["artifacts_missing_attempts"], 1)


if __name__ == "__main__":
    unittest.main()
