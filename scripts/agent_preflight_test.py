#!/usr/bin/env python3
"""Tests for agent-preflight.py — adapted for fenix authority model."""
from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path


SPEC = importlib.util.spec_from_file_location(
    "agent_preflight",
    Path(__file__).parent / "agent-preflight.py",
)
agent_preflight = importlib.util.module_from_spec(SPEC)  # type: ignore[arg-type]
SPEC.loader.exec_module(agent_preflight)  # type: ignore[union-attr]


class AgentPreflightTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def test_hp1_mark_then_check_passes(self):
        path = agent_preflight.mark_preflight(self.root)

        self.assertTrue(path.exists())
        data = agent_preflight.check_preflight(self.root)

        self.assertEqual(data["repo_root"], str(self.root.resolve()))
        self.assertEqual(data["version"], agent_preflight.SCRIPT_VERSION)

    def test_hp2_summary_names_required_fenix_rules(self):
        summary = agent_preflight.preflight_summary()

        self.assertIn("CLAUDE.md", summary)
        self.assertIn("README_AGENT_ORDER.md", summary)
        self.assertIn("docs/policies/RRI_POLICY.md", summary)
        self.assertIn("docs/policies/HITL_AUTONOMY_POLICY.md", summary)
        self.assertIn("docs/architecture.md", summary)
        self.assertIn("scripts/rri.py", summary)
        self.assertIn("RRI 26+", summary)

    def test_hp3_sentinel_payload_contains_required_fields(self):
        payload = agent_preflight.sentinel_payload(self.root)

        self.assertEqual(payload["version"], agent_preflight.SCRIPT_VERSION)
        self.assertIn("repo_root", payload)
        self.assertIn("marked_at", payload)
        self.assertIn("requirements", payload)
        self.assertIsInstance(payload["requirements"], list)
        self.assertGreater(len(payload["requirements"]), 0)

    def test_ec1_check_fails_when_sentinel_missing(self):
        with self.assertRaises(agent_preflight.PreflightError) as ctx:
            agent_preflight.check_preflight(self.root)

        self.assertIn("Missing", str(ctx.exception))
        self.assertIn("--mark", str(ctx.exception))

    def test_ec2_check_fails_for_different_repo_root(self):
        other_root = self.root / "other"
        other_root.mkdir()
        agent_preflight.mark_preflight(other_root)
        sentinel = agent_preflight.sentinel_path(other_root)
        local_sentinel = agent_preflight.sentinel_path(self.root)
        local_sentinel.parent.mkdir(parents=True, exist_ok=True)
        local_sentinel.write_text(sentinel.read_text(encoding="utf-8"), encoding="utf-8")

        with self.assertRaises(agent_preflight.PreflightError) as ctx:
            agent_preflight.check_preflight(self.root)

        self.assertIn("was marked for", str(ctx.exception))
        self.assertIn(str(self.root.resolve()), str(ctx.exception))

    def test_ec3_check_fails_for_stale_version(self):
        import json
        path = agent_preflight.mark_preflight(self.root)
        data = json.loads(path.read_text(encoding="utf-8"))
        data["version"] = 999
        path.write_text(json.dumps(data) + "\n", encoding="utf-8")

        with self.assertRaises(agent_preflight.PreflightError) as ctx:
            agent_preflight.check_preflight(self.root)

        self.assertIn("version", str(ctx.exception))

    def test_ec4_check_fails_for_malformed_json(self):
        sentinel = agent_preflight.sentinel_path(self.root)
        sentinel.parent.mkdir(parents=True, exist_ok=True)
        sentinel.write_text("not json{{{", encoding="utf-8")

        with self.assertRaises(agent_preflight.PreflightError) as ctx:
            agent_preflight.check_preflight(self.root)

        self.assertIn("Invalid", str(ctx.exception))

    def test_cli_check_returns_nonzero_without_sentinel(self):
        result = agent_preflight.main(["--repo-root", str(self.root), "--check"])

        self.assertEqual(result, 1)

    def test_cli_mark_and_check_returns_zero(self):
        mark_result = agent_preflight.main(["--repo-root", str(self.root), "--mark"])
        check_result = agent_preflight.main(["--repo-root", str(self.root), "--check"])

        self.assertEqual(mark_result, 0)
        self.assertEqual(check_result, 0)

    def test_cli_no_args_defaults_to_print_summary(self):
        import io
        import contextlib
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            result = agent_preflight.main(["--repo-root", str(self.root)])
        self.assertEqual(result, 0)
        self.assertIn("CLAUDE.md", buf.getvalue())


if __name__ == "__main__":
    unittest.main()
