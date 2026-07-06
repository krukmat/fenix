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
        self.assertIn("--model", cmd)
        self.assertIn("claude-sonnet-4-6", cmd)
        self.assertIn("--output-format", cmd)
        self.assertIn("json", cmd)

    def test_codex_exec_command_is_read_only(self):
        cmd = _mod.codex_exec_command("PROMPT")
        self.assertEqual(cmd[:2], ["codex", "exec"])
        self.assertIn("--sandbox", cmd)
        self.assertIn("read-only", cmd)

    def test_codex_review_command_uses_base_ref(self):
        # Codex CLI 0.142.5: `--base` and a positional PROMPT are mutually
        # exclusive, so the review command carries only the base ref.
        cmd = _mod.codex_review_command("main")
        self.assertEqual(cmd, ["codex", "review", "--base", "main"])
        self.assertNotIn("--instructions", cmd)

    def test_codex_review_command_without_base(self):
        cmd = _mod.codex_review_command(None)
        self.assertEqual(cmd, ["codex", "review"])

    def test_custom_executable_flows_into_commands(self):
        self.assertEqual(
            _mod.claude_command("PROMPT", executable="/x/claude")[0], "/x/claude"
        )
        self.assertEqual(
            _mod.codex_exec_command("PROMPT", executable="/x/codex")[0], "/x/codex"
        )
        self.assertEqual(
            _mod.codex_review_command("main", executable="/x/codex")[0],
            "/x/codex",
        )

    def test_parse_args_accepts_post_code_review_criticality(self):
        ns = _mod.parse_args(
            [
                "post-code-review",
                "--task",
                "task.md",
                "--plan",
                "plan.md",
                "--base",
                "main",
                "--criticality",
                "critical",
            ]
        )
        self.assertEqual(ns.criticality, "critical")


class CodexReviewOutputParsingTest(unittest.TestCase):
    def test_findings_map_to_needs_changes(self):
        raw = (
            "Two issues found.\n\nFull review comments:\n\n"
            "- [P1] Fix the diff scope — scripts/x.py:10-12\n"
            "  detail line\n"
            "- [P2] Honor the timeout — scripts/x.py:40\n"
        )
        v = _mod.parse_verdict(_mod.parse_codex_review_output(raw))
        self.assertEqual(v["status"], "needs_changes")
        self.assertEqual(len(v["findings"]), 2)
        self.assertTrue(v["findings"][0].startswith("[P1]"))
        self.assertEqual(v["summary"], "Two issues found.")

    def test_no_findings_map_to_pass(self):
        raw = "No blocking issues. The change looks correct and well tested.\n"
        v = _mod.parse_verdict(_mod.parse_codex_review_output(raw))
        self.assertEqual(v["status"], "pass")
        self.assertEqual(v["findings"], [])

    def test_empty_output_maps_to_pass_with_default_summary(self):
        v = _mod.parse_verdict(_mod.parse_codex_review_output(""))
        self.assertEqual(v["status"], "pass")
        self.assertEqual(v["summary"], "codex review completed")


class ReviewerExecutableResolutionTest(unittest.TestCase):
    def test_explicit_codex_override_wins(self):
        with patch.dict(os.environ, {"FENIX_CODEX_BIN": "/tmp/codex-bin"}, clear=False):
            with patch.object(_mod, "_is_executable", return_value=True), \
                 patch.object(_mod.shutil, "which", return_value="/wrong/path/codex"):
                self.assertEqual(
                    _mod.resolve_reviewer_executable("codex"),
                    "/tmp/codex-bin",
                )

    def test_claude_uses_path_lookup_when_available(self):
        with patch.dict(os.environ, {}, clear=True):
            with patch.object(
                _mod.shutil, "which", side_effect=lambda name: "/opt/homebrew/bin/claude" if name == "claude" else None
            ):
                self.assertEqual(
                    _mod.resolve_reviewer_executable("claude"),
                    "/opt/homebrew/bin/claude",
                )

    def test_codex_uses_known_install_location_when_path_missing(self):
        known_path = "/Users/example/.vscode/extensions/openai.chatgpt-1/bin/macos/codex"
        with patch.dict(os.environ, {}, clear=True):
            with patch.object(_mod.shutil, "which", return_value=None), \
                 patch.object(_mod.glob, "glob", side_effect=lambda pattern: [known_path] if "openai.chatgpt-" in pattern else []), \
                 patch.object(_mod, "_is_executable", side_effect=lambda path: path == known_path):
                self.assertEqual(
                    _mod.resolve_reviewer_executable("codex"),
                    known_path,
                )

    def test_invalid_override_raises_actionable_error(self):
        with patch.dict(os.environ, {"FENIX_CODEX_BIN": "/missing/codex"}, clear=False):
            with patch.object(_mod, "_is_executable", return_value=False):
                with self.assertRaises(_mod.ReviewerUnavailable) as ctx:
                    _mod.resolve_reviewer_executable("codex")
        self.assertIn("FENIX_CODEX_BIN=/missing/codex", str(ctx.exception))

    def test_missing_codex_reports_lookup_attempts(self):
        with patch.dict(os.environ, {}, clear=True):
            with patch.object(_mod.shutil, "which", return_value=None), \
                 patch.object(_mod.glob, "glob", return_value=[]):
                with self.assertRaises(_mod.ReviewerUnavailable) as ctx:
                    _mod.resolve_reviewer_executable("codex")
        self.assertIn("PATH:codex", str(ctx.exception))
        self.assertIn("reviewer CLI not found for codex", str(ctx.exception))

    def test_missing_claude_reports_lookup_attempts(self):
        with patch.dict(os.environ, {}, clear=True):
            with patch.object(_mod.shutil, "which", return_value=None), \
                 patch.object(_mod.glob, "glob", return_value=[]):
                with self.assertRaises(_mod.ReviewerUnavailable) as ctx:
                    _mod.resolve_reviewer_executable("claude")
        self.assertIn("PATH:claude", str(ctx.exception))
        self.assertIn("reviewer CLI not found for claude", str(ctx.exception))


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


class PromptBuildingTest(unittest.TestCase):
    def test_task_readiness_prompt_includes_criticality_concurrence_instruction(self):
        prompt = _mod.build_prompt(
            "task-readiness",
            {"task": "criticality: standard\ncriticality_basis: docs-only"},
        )
        self.assertIn("explicitly assess the declared task `criticality` label", prompt)
        self.assertIn("record that dispute as a reviewer finding", prompt)

    def test_post_code_review_prompt_does_not_include_criticality_concurrence_instruction(self):
        prompt = _mod.build_prompt("post-code-review", {"diff": "diff --git a b"})
        self.assertNotIn(
            "explicitly assess the declared task `criticality` label", prompt
        )


class LocalPromptCompactionTest(unittest.TestCase):
    def test_compaction_preserves_acceptance_criteria_under_budget(self):
        task = """---
criticality: standard
criticality_basis: workflow only
---

# Task Example

## Summary
Short summary.

## Acceptance Criteria
1. Keep the acceptance criteria.
2. Keep the critical task contract visible.

## Narrative Log
%s
""" % ("noise " * 2000)
        plan = """# Plan

## Purpose
Restore reliable local review.

## Verification Plan
Run the targeted tests.

## Status Updates
%s
""" % ("history " * 2000)
        with patch.object(_mod, "LOCAL_REVIEW_MAX_PROMPT_CHARS", 2200):
            compacted = _mod.compact_packet_for_local_review(
                "task-readiness",
                {"task": task, "plan": plan, "task_card": "Task: Example\nSummary: OK"},
            )
            prompt = _mod.build_prompt("task-readiness", compacted)
        self.assertLessEqual(len(prompt), 2200)
        self.assertIn("## Acceptance Criteria", compacted["task"])
        self.assertIn("criticality: standard", compacted["task"])
        self.assertIn("## Verification Plan", compacted["plan"])
        self.assertIn("truncated for local review", prompt)

class InvokeReviewerTest(unittest.TestCase):
    def test_invoker_resolves_executable_before_running(self):
        with patch.object(_mod, "resolve_reviewer_executable", return_value="/x/codex"), \
             patch.object(
                 _mod.subprocess, "run", return_value=_completed(stdout=_verdict_json())
             ) as run:
            _mod.invoke_reviewer("codex", "task-readiness", {}, None, 5)
            argv = run.call_args[0][0]
            self.assertEqual(argv[0], "/x/codex")

    def test_unavailable_cli_raises(self):
        with patch.object(_mod, "resolve_reviewer_executable", return_value="/x/claude"), \
             patch.object(_mod.subprocess, "run", side_effect=FileNotFoundError()):
            with self.assertRaises(_mod.ReviewerUnavailable):
                _mod.invoke_reviewer("claude", "task-readiness", {}, None, 5)

    def test_timeout_raises(self):
        exc = subprocess.TimeoutExpired(cmd="claude", timeout=5)
        with patch.object(_mod, "resolve_reviewer_executable", return_value="/x/claude"), \
             patch.object(_mod.subprocess, "run", side_effect=exc):
            with self.assertRaises(_mod.ReviewerTimeout):
                _mod.invoke_reviewer("claude", "task-readiness", {}, None, 5)

    def test_nonzero_exit_raises_unavailable(self):
        with patch.object(_mod, "resolve_reviewer_executable", return_value="/x/claude"), \
             patch.object(
                 _mod.subprocess, "run", return_value=_completed(returncode=1, stderr="auth")
             ):
            with self.assertRaises(_mod.ReviewerUnavailable):
                _mod.invoke_reviewer("claude", "task-readiness", {}, None, 5)

    def test_codex_post_code_dispatches_review(self):
        review_text = "- [P1] Something to fix — scripts/x.py:1\n"
        with patch.object(_mod, "resolve_reviewer_executable", return_value="/x/codex"), \
             patch.object(
                 _mod.subprocess, "run", return_value=_completed(stdout=review_text)
             ) as run:
            out = _mod.invoke_reviewer("codex", "post-code-review", {}, "main", 5)
            argv = run.call_args[0][0]
            # `--base` is present; no positional PROMPT is passed alongside it.
            self.assertEqual(argv, ["/x/codex", "review", "--base", "main"])
            # Native codex-review text is normalized into the gate JSON verdict.
            self.assertEqual(_mod.parse_verdict(out)["status"], "needs_changes")


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

    def test_standard_post_code_review_does_not_add_advisory_attempt(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args("post-code-review", d, extra=["--base", "main"])
            with patch.object(_mod, "read_diff", return_value="diff --git ..."), \
                 patch.object(
                     _mod.subprocess,
                     "run",
                     return_value=_completed(stdout=_verdict_json("pass")),
                 ), \
                 patch.object(_mod, "invoke_advisory_qwen_reviewer") as advisory:
                self.assertEqual(_mod.main(argv), 0)
            advisory.assert_not_called()
            artifacts = os.listdir(os.path.join(d, "artifacts"))
            with open(
                os.path.join(d, "artifacts", artifacts[0]), "r", encoding="utf-8"
            ) as fh:
                data = json.load(fh)
            self.assertEqual(len(data["attempts"]), 1)
            self.assertEqual(data["attempts"][0]["role"], "primary")

    def test_critical_post_code_review_adds_advisory_attempt_without_changing_pass(self):
        with tempfile.TemporaryDirectory() as d:
            argv = self._args(
                "post-code-review",
                d,
                extra=["--base", "main", "--criticality", "critical"],
            )
            advisory_attempt = {
                "role": "advisory-local",
                "reviewer": "local-qwen",
                "verdict": {"status": "blocked", "reason": "advisory_blocked"},
            }
            with patch.object(_mod, "read_diff", return_value="diff --git ..."), \
                 patch.object(
                     _mod.subprocess,
                     "run",
                     return_value=_completed(stdout=_verdict_json("pass")),
                 ), \
                 patch.object(
                     _mod,
                     "invoke_advisory_qwen_reviewer",
                     return_value=advisory_attempt,
                 ) as advisory:
                self.assertEqual(_mod.main(argv), 0)
            advisory.assert_called_once()
            artifacts = os.listdir(os.path.join(d, "artifacts"))
            with open(
                os.path.join(d, "artifacts", artifacts[0]), "r", encoding="utf-8"
            ) as fh:
                data = json.load(fh)
            self.assertEqual(data["verdict"]["status"], "pass")
            self.assertEqual(len(data["attempts"]), 2)
            self.assertEqual(data["attempts"][1]["role"], "advisory-local")
            self.assertEqual(data["attempts"][1]["reviewer"], "local-qwen")


class ReadDiffTest(unittest.TestCase):
    def test_empty_base_returns_empty(self):
        self.assertEqual(_mod.read_diff(None), "")
        self.assertEqual(_mod.read_diff(""), "")

    def test_uses_two_dot_working_tree_diff(self):
        # Must compare base vs working tree (two-dot), not commits only
        # (three-dot), so uncommitted changes are included in the review.
        with patch.object(
            _mod.subprocess, "run", return_value=_completed(stdout="diff --git a b")
        ) as run:
            out = _mod.read_diff("main")
        argv = run.call_args[0][0]
        self.assertEqual(argv, ["git", "diff", "--no-color", "main"])
        self.assertNotIn("main...HEAD", argv)
        self.assertEqual(out, "diff --git a b")

    def test_nonzero_returncode_returns_empty(self):
        with patch.object(
            _mod.subprocess, "run", return_value=_completed(returncode=128, stderr="bad rev")
        ):
            self.assertEqual(_mod.read_diff("nope"), "")

    def test_git_oserror_returns_empty(self):
        with patch.object(_mod.subprocess, "run", side_effect=OSError("no git")):
            self.assertEqual(_mod.read_diff("main"), "")


class FallbackTimeoutTest(unittest.TestCase):
    def _run_fallback(self, timeout, env=None):
        captured = {}

        def fake_stream_chat(endpoint, payload, idle_timeout, max_wall, progress_label):
            captured["idle_timeout"] = idle_timeout
            captured["max_wall"] = max_wall
            return object()

        gemma = _mod.gemma_local
        with patch.dict(os.environ, env or {}, clear=False), \
             patch.object(gemma, "ensure_model_available", return_value=None), \
             patch.object(gemma, "build_chat_payload", return_value={}), \
             patch.object(gemma, "stream_chat", side_effect=fake_stream_chat), \
             patch.object(gemma, "stream_result_content", return_value=_verdict_json("pass")):
            _mod.invoke_local_fallback_reviewer("task-readiness", {"task": "x"}, timeout)
        return captured

    def test_short_timeout_caps_idle_and_wall(self):
        cap = self._run_fallback(timeout=5)
        self.assertLessEqual(cap["idle_timeout"], 5)
        self.assertLessEqual(cap["max_wall"], 5)

    def test_zero_timeout_leaves_env_defaults(self):
        # A falsy timeout must not shrink the limits to 0.
        cap = self._run_fallback(timeout=0)
        self.assertGreater(cap["idle_timeout"], 0)
        self.assertGreater(cap["max_wall"], 0)

    def test_large_timeout_does_not_raise_env_limits(self):
        # timeout larger than the env/default limit must not increase them.
        env = {
            "FENIX_REVIEW_IDLE_TIMEOUT_SECONDS": "30",
            "FENIX_REVIEW_MAX_WALL_SECONDS": "60",
        }
        cap = self._run_fallback(timeout=9999, env=env)
        self.assertEqual(cap["idle_timeout"], 30)
        self.assertEqual(cap["max_wall"], 60)

    def test_local_fallback_payload_uses_compacted_prompt(self):
        captured = {}

        def fake_build_chat_payload(**kwargs):
            captured["packet"] = kwargs["packet"]
            return {}

        task = """# Task

## Acceptance Criteria
- preserve this requirement

## Narrative
%s
""" % ("noise " * 2500)
        plan = """# Plan

## Purpose
Restore local review.

## Status Updates
%s
""" % ("history " * 2500)
        gemma = _mod.gemma_local
        with patch.object(_mod, "LOCAL_REVIEW_MAX_PROMPT_CHARS", 2600), \
             patch.object(gemma, "ensure_model_available", return_value=None), \
             patch.object(gemma, "build_chat_payload", side_effect=fake_build_chat_payload), \
             patch.object(gemma, "stream_chat", return_value=object()), \
             patch.object(gemma, "stream_result_content", return_value=_verdict_json("pass")):
            _mod.invoke_local_fallback_reviewer(
                "task-readiness",
                {"task": task, "plan": plan, "task_card": "Task: Demo\nSummary: compact"},
                timeout=5,
            )
        self.assertLessEqual(len(captured["packet"]), 2600)
        self.assertIn("## Acceptance Criteria", captured["packet"])
        self.assertIn("Task: Demo", captured["packet"])
        self.assertIn("truncated for local review", captured["packet"])


class AdvisoryQwenTest(unittest.TestCase):
    def test_advisory_qwen_uses_keep_alive_zero(self):
        captured = {}

        def fake_build_chat_payload(**kwargs):
            captured["keep_alive"] = kwargs["keep_alive"]
            return {}

        gemma = _mod.gemma_local
        with patch.object(gemma, "ensure_model_available", return_value=None), \
             patch.object(gemma, "build_chat_payload", side_effect=fake_build_chat_payload), \
             patch.object(gemma, "stream_chat", return_value=object()), \
             patch.object(gemma, "stream_result_content", return_value=_verdict_json("pass")):
            attempt = _mod.invoke_advisory_qwen_reviewer(
                "post-code-review", {"diff": "x"}, timeout=5
            )
        self.assertEqual(captured["keep_alive"], 0)
        self.assertEqual(attempt["role"], "advisory-local")
        self.assertEqual(attempt["reviewer"], "local-qwen")
        self.assertEqual(attempt["verdict"]["status"], "pass")

    def test_advisory_qwen_retries_once_then_blocks(self):
        gemma = _mod.gemma_local
        with patch.object(gemma, "ensure_model_available", return_value=None), \
             patch.object(gemma, "build_chat_payload", return_value={}), \
             patch.object(
                 gemma,
                 "stream_chat",
                 side_effect=[gemma.GemmaIdleTimeout(5), RuntimeError("oom")],
             ) as stream_chat:
            attempt = _mod.invoke_advisory_qwen_reviewer(
                "post-code-review", {"diff": "x"}, timeout=5
            )
        self.assertEqual(stream_chat.call_count, 2)
        self.assertEqual(attempt["role"], "advisory-local")
        self.assertEqual(attempt["reviewer"], "local-qwen")
        self.assertEqual(attempt["verdict"]["status"], "blocked")
        self.assertEqual(attempt["verdict"]["reason"], "advisory_blocked")
        self.assertEqual(attempt["verdict"]["blocked_reason"], "reviewer_unavailable")

    def test_advisory_qwen_payload_keeps_diff_and_verification_under_budget(self):
        captured = {}

        def fake_build_chat_payload(**kwargs):
            captured["packet"] = kwargs["packet"]
            return {}

        diff = "diff --git a/x b/x\n" + ("+line\n" * 1200)
        verification_log = "verify\n" + ("step ok\n" * 600)
        task = """# Task

## Acceptance Criteria
- keep diff and verification visible

## Narrative
%s
""" % ("noise " * 2000)
        gemma = _mod.gemma_local
        with patch.object(_mod, "LOCAL_REVIEW_MAX_PROMPT_CHARS", 4200), \
             patch.object(gemma, "ensure_model_available", return_value=None), \
             patch.object(gemma, "build_chat_payload", side_effect=fake_build_chat_payload), \
             patch.object(gemma, "stream_chat", return_value=object()), \
             patch.object(gemma, "stream_result_content", return_value=_verdict_json("pass")):
            attempt = _mod.invoke_advisory_qwen_reviewer(
                "post-code-review",
                {
                    "task": task,
                    "plan": "# Plan\n\n## Purpose\nKeep reviewable evidence.\n",
                    "verification_log": verification_log,
                    "diff": diff,
                },
                timeout=5,
            )
        self.assertEqual(attempt["verdict"]["status"], "pass")
        self.assertLessEqual(len(captured["packet"]), 4200)
        self.assertIn("=== DIFF ===", captured["packet"])
        self.assertIn("=== VERIFICATION_LOG ===", captured["packet"])
        self.assertIn("truncated for local review", captured["packet"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
