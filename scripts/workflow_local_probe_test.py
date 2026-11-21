"""Offline tests for the final Mac-only probe; never contact Ollama."""
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import workflow_local_probe as probe


class LocalProbeTest(unittest.TestCase):
    def test_only_loopback_allowed(self):
        for host in ("https://localhost:11434", "http://example.com:11434",
                     "http://127.0.0.1:11434/path", "http://localhost:11434?q=1"):
            with self.assertRaises(ValueError):
                probe.validate_host(host)
        self.assertEqual(probe.validate_host("http://127.0.0.1:11434"),
                         "http://127.0.0.1:11434")

    def test_missing_model_skips_generation(self):
        with patch.object(probe, "get_installed_models", return_value=set()), \
             patch.object(probe, "generate_metrics") as gen, \
             patch.object(probe, "memory_snapshot", return_value=None):
            result = probe.run_probe(host="http://127.0.0.1:11434",
                                     models=["missing:tag"], prompt="Safe", runs=2,
                                     timeout=10, max_tokens=32)
            self.assertFalse(result["models"][0]["available"])
            gen.assert_not_called()

    def test_available_model_captures_repetitions(self):
        with patch.object(probe, "get_installed_models", return_value={"m:tag"}), \
             patch.object(probe, "generate_metrics", return_value={"status": "complete", "wall_seconds": 1.0}) as gen, \
             patch.object(probe, "memory_snapshot", return_value=None):
            result = probe.run_probe(host="http://127.0.0.1:11434",
                                     models=["m:tag"], prompt="Safe", runs=3,
                                     timeout=10, max_tokens=32)
            self.assertEqual(len(result["models"][0]["measurements"]), 3)
            self.assertEqual(gen.call_count, 3)

    def test_invalid_run_count_fails(self):
        with self.assertRaises(ValueError):
            probe.run_probe(host="http://127.0.0.1:11434",
                            models=["m"], prompt="Safe", runs=11,
                            timeout=10, max_tokens=32)

    def test_dry_run_does_not_contact_ollama(self):
        with tempfile.TemporaryDirectory() as td:
            prompt = Path(td) / "safe.txt"
            out = Path(td) / "out.json"
            prompt.write_text("Hello", encoding="utf-8")
            with patch.object(probe, "get_installed_models") as tags:
                self.assertEqual(probe.main([
                    "--models", "m:tag", "--prompt-file", str(prompt),
                    "--output", str(out), "--dry-run"
                ]), 0)
                tags.assert_not_called()
            data = json.loads(out.read_text(encoding="utf-8"))
            self.assertNotIn("Hello", json.dumps(data))
            self.assertEqual(data["status"], "dry_run")


if __name__ == "__main__":
    unittest.main()
