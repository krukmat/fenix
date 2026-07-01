#!/usr/bin/env python3
"""Unit tests for peer-workflow-review.py (Phase F peer review gate).

All external CLI calls are mocked; no live Claude/Codex invocation occurs.

Run: python3 scripts/peer_workflow_review_test.py
     python3 -m unittest scripts/peer_workflow_review_test.py
"""

import importlib.util
import json
import os
import subprocess
import tempfile
import unittest
from unittest.mock import MagicMock, patch

_SCRIPT = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "peer-workflow-review.py"
)
_spec = importlib.util.spec_from_file_location("peer_workflow_review", _SCRIPT)
_mod = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_mod)


def _completed(returncode=0, stdout="", stderr=""):
    proc = MagicMock()
    proc.returncode = returncode
    proc.stdout = stdout
    proc.stderr = stderr
    return proc


def _verdict_json(status="pass", summary="ok", findings=None):
    return json.dumps(
        {"status": status, "summary": summary, "findings": findings or []}
    )


class ReviewerResolutionTest(unittest.TestCase):
    def test_claude_code_resolves_to_codex(self):
        self.assertEqual(_mod.resolve_reviewer("claude-code"), "codex")

    def test_codex_resolves_to_claude(self):
        self.assertEqual(_mod.resolve_reviewer("codex"), "claude")

    def test_local_remote_unknown_default_to_claude(self):
        for kind in ("local-provider", "remote-provider", "unknown", "bogus"):
            self.assertEqual(_mod.resolve_reviewer(kind), "claude")

    def test_normalize_prefers_valid_kind(self):
        self.assertEqual(_mod.normalize_caller_kind("Codex"), "codex")

    def test_normalize_falls_back_to_provider(self):
        self.assertEqual(
            _mod.normalize_caller_kind("weird", "claude-code"), "claude-code"
        )

    def test_normalize_defaults_to_unknown(self):
        self.assertEqual(_mod.normalize_caller_kind(None, None), "unknown")

    def test_resolve_fallback_reviewer_is_local_gemma(self):
        self.assertEqual(_mod.resolve_fallback_reviewer(), "local-gemma")


class CommandConstructionTest(unittest.TestCase):
    def test_claude_command_is_non_mutating_json(self):
        cmd = _mod.claude_command("PROMPT")
        self.assertEqual(cmd[0], "claude")
        self.assertIn("-p", cmd)
        self.assertIn("--output-format", cmd)
        self.assertIn("json", cmd)

    def test_codex_exec_command_is_read_only(self):
        cmd = _mod.codex_exec_command("PROMPT")
        self.assertEqual(cmd[:2], ["codex", "exec"])
        self.assertIn("--sandbox", cmd)
        self.assertIn("read-only", cmd)

    def test_codex_review_command_uses_base_ref(self):
        cmd = _mod.codex_review_command("main", "PROMPT")
        self.assertEqual(cmd[:2], ["codex", "review"])
        self.assertIn("--base", cmd)
        self.assertIn("main", cmd)


class VerdictParsingTest(unittest.TestCase):
    def test_parses_plain_verdict(self):
        v = _mod.parse_verdict(_verdict_json("pass"))
        self.assertEqual(v["status"], "pass")

    def test_unwraps_claude_envelope(self):
        raw = json.dumps({"result": _verdict_json("needs_changes", "fix it")})
        v = _mod.parse_verdict(raw)
        self.assertEqual(v["status"], "needs_changes")
        self.assertEqual(v["summary"], "fix it")

    def test_extracts_embedded_json(self):
        raw = "here is my review:\n" + _verdict_json("pass") + "\nthanks"
        self.assertEqual(_mod.parse_verdict(raw)["status"], "pass")

    def test_invalid_json_raises(self):
        with self.assertRaises(_mod.InvalidVerdict):
            _mod.parse_verdict("not json at all")

    def test_unknown_status_raises(self):
        with self.assertRaises(_mod.InvalidVerdict):
            _mod.parse_verdict(_verdict_json("approved"))


class InvokeReviewerTest(unittest.TestCase):
    def test_unavailable_cli_raises(self):
        with patch.object(_mod.subprocess, "run", side_effect=FileNotFoundError()):
            with self.assertRaises(_mod.ReviewerUnavailable):
                _mod.invoke_reviewer("claude", "task-readiness", {}, None, 5)

    def test_timeout_raises(self):
        exc = subprocess.TimeoutExpired(cmd="claude", timeout=5)
        with patch.object(_mod.subprocess, "run", side_effect=exc):
            with self.assertRaises(_mod.ReviewerTimeout):
                _mod.invoke_reviewer("claude", "task-readiness", {}, None, 5)

    def test_nonzero_exit_raises_unavailable(self):
        with patch.object(
            _mod.subprocess, "run", return_value=_completed(returncode=1, stderr="auth")
        ):
            with self.assertRaises(_mod.ReviewerUnavailable):
                _mod.invoke_reviewer("claude", "task-readiness", {}, None, 5)

    def test_codex_post_code_dispatches_review(self):
        with patch.object(
            _mod.subprocess, "run", return_value=_completed(stdout=_verdict_json())
        ) as run:
            _mod.invoke_reviewer("codex", "post-code-review", {}, "main", 5)
            argv = run.call_args[0][0]
            self.assertEqual(argv[:2], ["codex", "review"])


class RedactionTest(unittest.TestCase):
    def test_redacts_key_value_secrets(self):
        text = "api_key = sk-abcdef123456 and password: hunter2"
        out = _mod.redact_text(text)
        self.assertNotIn("hunter2", out)
        self.assertNotIn("sk-abcdef123456", out)
        self.assertIn(_mod.REDACTED, out)

    def test_redacts_bearer_token(self):
        out = _mod.redact_text("Authorization: Bearer abc.def.ghi")
        self.assertNotIn("abc.def.ghi", out)

    def test_redact_packet_masks_string_values(self):
        packet = {"task": "token=SEKRET99999", "mode": "task-readiness"}
        red = _mod.redact_packet(packet)
        self.assertNotIn("SEKRET99999", red["task"])
        self.assertEqual(red["mode"], "task-readiness")


class ArtifactTest(unittest.TestCase):
    def test_writes_redacted_artifact(self):
        with tempfile.TemporaryDirectory() as d:
            packet = {"task": "password: leaky_secret_value", "mode": "x"}
            verdict = {"status": "pass", "summary": "ok", "findings": []}
            path = _mod.write_artifact(
                d, "task-readiness", "claude-code", "codex", packet, verdict
            )
            self.assertTrue(os.path.exists(path))
            with open(path, "r", encoding="utf-8") as fh:
                data = json.load(fh)
            self.assertEqual(data["reviewer"], "codex")
            self.assertEqual(data["verdict"]["status"], "pass")
            self.assertNotIn("leaky_secret_value", json.dumps(data))

    def test_writes_attempts_when_provided(self):
        with tempfile.TemporaryDirectory() as d:
            packet = {"task": "task body", "mode": "x"}
            verdict = {"status": "pass", "summary": "ok", "findings": []}
            attempts = [
                {"role": "primary", "reviewer": "claude", "verdict": {"status": "blocked"}},
                {"role": "fallback", "reviewer": "local-gemma", "verdict": {"status": "pass"}},
            ]
            path = _mod.write_artifact(
                d, "task-readiness", "codex", "local-gemma", packet, verdict, attempts
            )
            with open(path, "r", encoding="utf-8") as fh:
                data = json.load(fh)
            self.assertEqual(len(data["attempts"]), 2)
            self.assertEqual(data["attempts"][0]["reviewer"], "claude")
            self.assertEqual(data["attempts"][1]["reviewer"], "local-gemma")


class MainTest(unittest.TestCase):
    def _args(self, mode, out_dir, extra=None):
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".md", delete=False, dir=out_dir
        ) as t:
            t.write("task body\n")
            task = t.name
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".md", delete=False, dir=out_dir
        ) as p:
            p.write("plan body\n")
            plan = p.name
        argv = [
            mode,
            "--caller-kind",
            "claude-code",
            "--task",
            task,
            "--plan",
            plan,
            "--out-dir",
            os.path.join(out_dir, "artifacts"),
            "--timeout",
            "5",
        ]
        if extra:
            argv += extra
        return argv

    def test_pass_verdict_exits_zero(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args("task-readiness", d)
            with patch.object(
                _mod.subprocess,
                "run",
                return_value=_completed(stdout=_verdict_json("pass")),
            ):
                self.assertEqual(_mod.main(argv), 0)

    def test_needs_changes_exits_nonzero(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args("task-readiness", d)
            with patch.object(
                _mod.subprocess,
                "run",
                return_value=_completed(stdout=_verdict_json("needs_changes")),
            ):
                self.assertEqual(_mod.main(argv), 1)

    def test_unavailable_cli_writes_blocked_and_exits_nonzero(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args("task-readiness", d)
            exc = _mod.ReviewerUnavailable("reviewer CLI not found: codex")
            with patch.object(_mod, "invoke_reviewer", side_effect=exc), \
                 patch.object(
                     _mod,
                     "invoke_local_fallback_reviewer",
                     side_effect=_mod.ReviewerUnavailable("local fallback unavailable"),
                 ):
                self.assertEqual(_mod.main(argv), 1)
            artifacts = os.listdir(os.path.join(d, "artifacts"))
            self.assertEqual(len(artifacts), 1)
            with open(
                os.path.join(d, "artifacts", artifacts[0]), "r", encoding="utf-8"
            ) as fh:
                data = json.load(fh)
            self.assertEqual(data["verdict"]["status"], "blocked")
            self.assertEqual(data["verdict"]["reason"], "reviewer_unavailable")
            self.assertEqual(data["reviewer"], "local-gemma")
            self.assertEqual(data["attempts"][0]["reviewer"], "codex")
            self.assertEqual(data["attempts"][1]["reviewer"], "local-gemma")

    def test_invalid_json_writes_blocked_and_exits_nonzero(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args("task-readiness", d)
            with patch.object(
                _mod.subprocess,
                "run",
                return_value=_completed(stdout="garbage not json"),
            ):
                self.assertEqual(_mod.main(argv), 1)
            artifacts = os.listdir(os.path.join(d, "artifacts"))
            with open(
                os.path.join(d, "artifacts", artifacts[0]), "r", encoding="utf-8"
            ) as fh:
                data = json.load(fh)
            self.assertEqual(data["verdict"]["reason"], "invalid_verdict")

    def test_primary_timeout_falls_back_to_local_and_passes(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args("task-readiness", d, extra=["--caller-kind", "codex"])
            exc = _mod.ReviewerTimeout("reviewer timed out after 5s")
            with patch.object(_mod, "invoke_reviewer", side_effect=exc), \
                 patch.object(_mod, "invoke_local_fallback_reviewer", return_value=_verdict_json("pass", "fallback ok")):
                self.assertEqual(_mod.main(argv), 0)
            artifacts = os.listdir(os.path.join(d, "artifacts"))
            with open(
                os.path.join(d, "artifacts", artifacts[0]), "r", encoding="utf-8"
            ) as fh:
                data = json.load(fh)
            self.assertEqual(data["reviewer"], "local-gemma")
            self.assertEqual(data["verdict"]["status"], "pass")
            self.assertEqual(len(data["attempts"]), 2)
            self.assertEqual(data["attempts"][0]["verdict"]["reason"], "reviewer_timeout")
            self.assertEqual(data["attempts"][1]["reviewer"], "local-gemma")

    def test_primary_unavailable_and_invalid_fallback_blocks(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args("task-readiness", d, extra=["--caller-kind", "codex"])
            exc = _mod.ReviewerUnavailable("reviewer CLI not found: claude")
            with patch.object(_mod, "invoke_reviewer", side_effect=exc), \
                 patch.object(_mod, "invoke_local_fallback_reviewer", return_value="not json"):
                self.assertEqual(_mod.main(argv), 1)
            artifacts = os.listdir(os.path.join(d, "artifacts"))
            with open(
                os.path.join(d, "artifacts", artifacts[0]), "r", encoding="utf-8"
            ) as fh:
                data = json.load(fh)
            self.assertEqual(data["reviewer"], "local-gemma")
            self.assertEqual(data["verdict"]["status"], "blocked")
            self.assertEqual(data["verdict"]["reason"], "invalid_verdict")
            self.assertEqual(data["attempts"][0]["verdict"]["reason"], "reviewer_unavailable")
            self.assertEqual(data["attempts"][1]["reviewer"], "local-gemma")

    def test_post_code_review_reads_base_and_passes(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args("post-code-review", d, extra=["--base", "main"])
            with patch.object(_mod, "read_diff", return_value="diff --git ..."):
                with patch.object(
                    _mod.subprocess,
                    "run",
                    return_value=_completed(stdout=_verdict_json("pass")),
                ):
                    self.assertEqual(_mod.main(argv), 0)


if __name__ == "__main__":
    unittest.main(verbosity=2)
