#!/usr/bin/env python3
"""Unit tests for gemma-push-review.py (T1: collector + packet, T1B: model invocation + grounding)."""

import argparse
import importlib.util
import io
import json
import os
import sys
import tempfile
import unittest
from unittest.mock import patch

_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "gemma-push-review.py")
_spec = importlib.util.spec_from_file_location("gemma_push_review", _SCRIPT)
_mod = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_mod)

import gemma_local as _gl


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def _completed_run(conclusion="success", status="completed"):
    return {
        "run_id": 987654,
        "workflow_name": "ci",
        "event": "push",
        "branch": "main",
        "head_sha": "abcdef1234567890abcdef1234567890abcdef12",
        "run_attempt": 1,
        "status": status,
        "conclusion": conclusion,
        "url": "https://github.com/owner/repo/actions/runs/987654",
        "created_at": "2026-06-25T00:00:00Z",
        "updated_at": "2026-06-25T00:01:00Z",
    }


def _sample_diff():
    return "\n".join([
        "diff --git a/scripts/example.py b/scripts/example.py",
        "index aaa..bbb 100644",
        "--- a/scripts/example.py",
        "+++ b/scripts/example.py",
        "@@ -1,3 +1,3 @@",
        "-old_line = 1",
        "+new_line = 1",
        " # context",
    ])


def _make_packet(**kwargs):
    defaults = dict(
        run_info=_completed_run(),
        jobs=[],
        annotations=[],
        log_paths=[],
        artifact_paths=[],
        before_sha="aaa0001",
        after_sha="abcdef1234567890abcdef1234567890abcdef12",
        diff=_sample_diff(),
        changed_paths=["scripts/example.py"],
        pipeline_evidence_partial=False,
        logs_truncated=False,
        repo="owner/repo",
    )
    defaults.update(kwargs)
    return _mod.build_packet(**defaults)


def _model_args(**kwargs):
    defaults = dict(
        host="http://localhost:11434",
        model="gemma4:26b-a4b-it-qat",
        num_ctx=32768,
        num_predict=4096,
        temperature=0.1,
        think=True,
        idle_timeout=60,
        max_wall=900,
        dry_run=False,
        collect_only=False,
    )
    defaults.update(kwargs)
    return argparse.Namespace(**defaults)


def _findings_response(path="scripts/example.py", line=1, severity="minor",
                       with_rri_hint=False):
    lines = [
        "STATUS: FINDINGS",
        "SUMMARY: found one issue",
        "=== FINDING START ===",
        f"PATH: {path}",
        f"LINE: {line}",
        f"SEVERITY: {severity}",
        "DETAIL: style issue",
        "SUGGESTION: rename variable",
    ]
    if with_rri_hint:
        lines.append("RRI_HINT: D=2 T=1 K=3 cc=5")
    lines.append("=== FINDING END ===")
    return "\n".join(lines)


# ===========================================================================
# T1: GitHub run resolution and packet builder
# ===========================================================================

class NormalizeRun(unittest.TestCase):
    def test_normalize_from_api_response(self):
        raw = {
            "databaseId": 123, "workflowName": "ci", "event": "push",
            "headBranch": "main", "headSha": "abc123", "runAttempt": 1,
            "status": "completed", "conclusion": "success",
            "url": "https://github.com/owner/repo/actions/runs/123",
            "createdAt": "2026-06-25T00:00:00Z", "updatedAt": "2026-06-25T00:01:00Z",
        }
        result = _mod._normalize_run(raw)
        self.assertEqual(result["run_id"], 123)
        self.assertEqual(result["workflow_name"], "ci")
        self.assertEqual(result["head_sha"], "abc123")
        self.assertEqual(result["conclusion"], "success")
        self.assertEqual(result["status"], "completed")

    def test_normalize_workflow_run_event_shape(self):
        raw = {
            "id": 999, "name": "ci", "event": "push",
            "head_branch": "main", "head_sha": "deadbeef", "run_attempt": 2,
            "status": "completed", "conclusion": "failure",
            "html_url": "https://github.com/owner/repo/actions/runs/999",
            "created_at": "2026-06-25T00:00:00Z", "updated_at": "2026-06-25T00:02:00Z",
        }
        result = _mod._normalize_run(raw)
        self.assertEqual(result["run_id"], 999)
        self.assertEqual(result["run_attempt"], 2)
        self.assertEqual(result["conclusion"], "failure")
        self.assertEqual(result["url"], "https://github.com/owner/repo/actions/runs/999")

    def test_normalize_attempt_field_fallback(self):
        raw = {
            "databaseId": 321, "workflowName": "ci", "event": "push",
            "headBranch": "main", "headSha": "abc999", "attempt": 4,
            "status": "completed", "conclusion": "success",
        }
        result = _mod._normalize_run(raw)
        self.assertEqual(result["run_attempt"], 4)


class PendingRun(unittest.TestCase):
    def test_in_progress_returns_sentinel_path(self):
        run = _completed_run(status="in_progress", conclusion=None)
        with tempfile.TemporaryDirectory() as tmp:
            path = _mod.write_sentinel("pipeline_pending", run, tmp, run["head_sha"])
            self.assertTrue(os.path.isfile(path))
            with open(path) as fh:
                data = json.load(fh)
            self.assertEqual(data["sentinel"], "pipeline_pending")
            self.assertEqual(data["role"], "gemma-push-reviewer")

    def test_queued_status_is_pending(self):
        run = _completed_run(status="queued", conclusion=None)
        with tempfile.TemporaryDirectory() as tmp:
            path = _mod.write_sentinel("pipeline_pending", run, tmp, run["head_sha"])
            with open(path) as fh:
                data = json.load(fh)
            self.assertEqual(data["sentinel"], "pipeline_pending")


class DocsOnlyDetection(unittest.TestCase):
    def test_docs_only_md_and_txt(self):
        self.assertTrue(_mod.is_docs_only(["docs/plan/foo.md", "README.md", "docs/notes.txt"]))

    def test_code_path_is_not_docs_only(self):
        self.assertFalse(_mod.is_docs_only(["docs/plan/foo.md", "scripts/example.py"]))

    def test_empty_paths_not_docs_only(self):
        self.assertFalse(_mod.is_docs_only([]))

    def test_image_files_are_docs_only(self):
        self.assertTrue(_mod.is_docs_only(["docs/images/diagram.png", "docs/logo.svg"]))

    def test_go_source_not_docs_only(self):
        self.assertFalse(_mod.is_docs_only(["internal/core/service.go"]))


class EvidencePartial(unittest.TestCase):
    def test_packet_records_partial_flag(self):
        p = _make_packet(pipeline_evidence_partial=True)
        self.assertTrue(p["push"]["pipeline_evidence_partial"])

    def test_packet_records_logs_truncated(self):
        p = _make_packet(logs_truncated=True)
        self.assertTrue(p["push"]["logs_truncated"])


class OperationalFailure(unittest.TestCase):
    def test_failure_artifact_schema(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = _mod.write_failure("gh CLI failed", tmp, "abc1234")
            self.assertTrue(os.path.isfile(path))
            with open(path) as fh:
                data = json.load(fh)
            self.assertEqual(data["sentinel"], "operational_failure")
            self.assertEqual(data["error"], "gh CLI failed")
            self.assertEqual(data["role"], "gemma-push-reviewer")


class BuildPacket(unittest.TestCase):
    def test_top_level_fields(self):
        p = _make_packet()
        self.assertEqual(p["role"], "gemma-push-reviewer")
        self.assertEqual(p["schema_version"], 1)
        self.assertEqual(p["repo"], "owner/repo")
        self.assertEqual(p["branch"], "main")
        for key in ("pipeline", "push", "audit", "candidates",
                    "developer_dispatch", "post_development_review", "deployer_followup"):
            self.assertIn(key, p)

    def test_pipeline_section(self):
        p = _make_packet()
        pip = p["pipeline"]
        self.assertEqual(pip["run_id"], 987654)
        self.assertEqual(pip["workflow_name"], "ci")
        self.assertEqual(pip["conclusion"], "success")
        self.assertIn("jobs", pip)
        self.assertIn("log_paths", pip)

    def test_push_section(self):
        p = _make_packet()
        push = p["push"]
        self.assertIn("scripts/example.py", push["changed_paths"])
        self.assertIn("diff --git", push["diff"])
        self.assertFalse(push["pipeline_evidence_partial"])

    def test_audit_section_starts_pending(self):
        p = _make_packet()
        self.assertEqual(p["audit"]["quorum"], "pending")
        self.assertEqual(p["audit"]["passes_run"], 0)

    def test_candidates_empty(self):
        self.assertEqual(_make_packet()["candidates"], [])

    def test_no_raw_file_bodies(self):
        p = _make_packet()
        self.assertNotIn("file_bodies", p)
        self.assertNotIn("file_bodies", p.get("push", {}))

    def test_packet_uses_push_reviewer_contract(self):
        p = _make_packet()
        self.assertNotIn("findings", p)
        self.assertNotIn("reconciliation", p)
        self.assertNotIn("summary", p)


class ChangedPathsFromDiff(unittest.TestCase):
    def test_extracts_paths(self):
        self.assertIn("scripts/example.py", _mod.changed_paths_from_diff(_sample_diff()))

    def test_empty_diff_returns_empty(self):
        self.assertEqual(_mod.changed_paths_from_diff(""), [])

    def test_ignores_dev_null(self):
        diff = "diff --git a/new.py b/new.py\n--- /dev/null\n+++ b/new.py\n"
        paths = _mod.changed_paths_from_diff(diff)
        self.assertIn("new.py", paths)
        self.assertNotIn("/dev/null", paths)


class ShortSha(unittest.TestCase):
    def test_returns_seven_chars(self):
        self.assertEqual(_mod._short_sha("abcdef1234567890"), "abcdef1")

    def test_short_sha_handles_none(self):
        self.assertEqual(_mod._short_sha(None), "unknown")

    def test_short_sha_handles_short_input(self):
        self.assertEqual(_mod._short_sha("abc"), "abc")


class ResolveRunFromEvent(unittest.TestCase):
    def _make_event_file(self, tmp):
        event = {
            "action": "completed",
            "workflow_run": {
                "id": 555, "name": "ci", "event": "push",
                "head_branch": "main",
                "head_sha": "feed1234feed1234feed1234feed1234feed1234",
                "run_attempt": 1, "status": "completed", "conclusion": "success",
                "html_url": "https://github.com/owner/repo/actions/runs/555",
                "created_at": "2026-06-25T00:00:00Z",
                "updated_at": "2026-06-25T00:01:00Z",
            },
        }
        path = os.path.join(tmp, "event.json")
        with open(path, "w") as fh:
            json.dump(event, fh)
        return path

    def test_workflow_run_event_resolves_without_gh_call(self):
        with tempfile.TemporaryDirectory() as tmp:
            event_path = self._make_event_file(tmp)
            args = argparse.Namespace(
                run_id=None, workflow=None, branch=None,
                before=None, after=None, event_path=event_path,
            )
            run = _mod.resolve_run(args)
            self.assertEqual(run["run_id"], 555)
            self.assertEqual(run["conclusion"], "success")
            self.assertEqual(run["branch"], "main")
            self.assertEqual(run["head_sha"], "feed1234feed1234feed1234feed1234feed1234")

    def test_gh_not_called_when_event_present(self):
        with tempfile.TemporaryDirectory() as tmp:
            event_path = self._make_event_file(tmp)
            args = argparse.Namespace(
                run_id=None, workflow=None, branch=None,
                before=None, after=None, event_path=event_path,
            )
            with patch.object(_mod, "_run_gh") as mock_gh:
                _mod.resolve_run(args)
                mock_gh.assert_not_called()


class ResolveRunById(unittest.TestCase):
    def test_calls_gh_with_run_id(self):
        raw = {
            "databaseId": 777, "workflowName": "ci", "event": "push",
            "headBranch": "main", "headSha": "cafe0001", "attempt": 1,
            "status": "completed", "conclusion": "success",
            "url": "https://github.com/owner/repo/actions/runs/777",
            "createdAt": "2026-06-25T00:00:00Z", "updatedAt": "2026-06-25T00:01:00Z",
        }
        args = argparse.Namespace(
            run_id="777", workflow=None, branch=None,
            before=None, after=None, event_path=None,
        )
        with patch.object(_mod, "_run_gh", return_value=(json.dumps(raw), 0)):
            run = _mod.resolve_run(args)
        self.assertEqual(run["run_id"], 777)
        self.assertEqual(run["head_sha"], "cafe0001")

    def test_calls_gh_with_attempt_field(self):
        args = argparse.Namespace(
            run_id="777", workflow=None, branch=None,
            before=None, after=None, event_path=None,
        )
        with patch.object(_mod, "_run_gh", return_value=(json.dumps({}), 0)) as mock_gh:
            _mod.resolve_run(args)
        gh_args = mock_gh.call_args.args[0]
        self.assertIn("attempt", gh_args[-1])
        self.assertNotIn("runAttempt", gh_args[-1])


class ResolveRunUnavailable(unittest.TestCase):
    def test_no_runs_returns_unavailable_sentinel(self):
        args = argparse.Namespace(
            run_id=None, workflow=None, branch=None,
            before=None, after=None, event_path=None,
        )
        with patch.object(_mod, "_run_gh", return_value=(json.dumps([]), 0)):
            run = _mod.resolve_run(args)
        self.assertEqual(run.get("_sentinel"), "pipeline_unavailable")

    def test_gh_failure_raises_runtime_error(self):
        args = argparse.Namespace(
            run_id="bad-id", workflow=None, branch=None,
            before=None, after=None, event_path=None,
        )
        with patch.object(_mod, "_run_gh", side_effect=RuntimeError("gh run view failed")):
            with self.assertRaises(RuntimeError):
                _mod.resolve_run(args)


class CollectJobs(unittest.TestCase):
    def test_jobs_parsed_correctly(self):
        raw = {"jobs": [
            {"name": "test", "status": "completed", "conclusion": "success",
             "startedAt": "2026-06-25T00:00:00Z", "completedAt": "2026-06-25T00:01:00Z",
             "steps": [{"name": "checkout", "status": "completed", "conclusion": "success", "number": 1}]},
            {"name": "build", "status": "completed", "conclusion": "failure",
             "startedAt": "2026-06-25T00:01:00Z", "completedAt": "2026-06-25T00:02:00Z",
             "steps": [{"name": "compile", "status": "completed", "conclusion": "failure", "number": 1}]},
        ]}
        with patch.object(_mod, "_run_gh", return_value=(json.dumps(raw), 0)):
            jobs, partial = _mod.collect_jobs(123)
        self.assertFalse(partial)
        self.assertEqual(len(jobs), 2)
        failed = [j for j in jobs if j["conclusion"] == "failure"]
        self.assertEqual(len(failed), 1)
        self.assertEqual(len(failed[0]["failed_steps"]), 1)

    def test_gh_failure_returns_partial(self):
        with patch.object(_mod, "_run_gh", return_value=("", 1)):
            jobs, partial = _mod.collect_jobs(123)
        self.assertEqual(jobs, [])
        self.assertTrue(partial)


# ===========================================================================
# T1B: push-audit parser
# ===========================================================================

class PushAuditSystemPrompt(unittest.TestCase):
    def test_prompt_is_read_only(self):
        prompt = _mod.build_push_audit_system_prompt()
        self.assertIn("read-only", prompt)
        self.assertIn("STATUS: PASS", prompt)
        self.assertIn("RRI_HINT", prompt)
        self.assertIn("advisory only", prompt)

    def test_prompt_forbids_patches(self):
        prompt = _mod.build_push_audit_system_prompt()
        self.assertIn("no diff", prompt.lower())
        self.assertIn("no patch", prompt.lower())

    def test_prompt_identifies_as_fenix(self):
        prompt = _mod.build_push_audit_system_prompt()
        self.assertIn("Fenix", prompt)
        self.assertNotIn("DubBridge", prompt)


class ParsePushAuditResponse(unittest.TestCase):
    def test_pass_no_findings(self):
        resp = "STATUS: PASS\nSUMMARY: no issues found"
        result = _mod.parse_push_audit_response(resp, ["scripts/a.py"])
        self.assertEqual(result["status"], "pass")
        self.assertEqual(result["findings"], [])
        self.assertEqual(result["summary"], "no issues found")

    def test_findings_with_rri_hint(self):
        resp = _findings_response(with_rri_hint=True)
        result = _mod.parse_push_audit_response(resp, ["scripts/example.py"])
        self.assertEqual(result["status"], "findings")
        f = result["findings"][0]
        self.assertNotIn("rri_hint", f)
        self.assertIn("rri_input_proposal", f)
        self.assertEqual(f["rri_input_proposal"]["D"], 2)
        self.assertEqual(f["rri_input_proposal"]["K"], 3)

    def test_findings_without_rri_hint(self):
        resp = _findings_response(with_rri_hint=False)
        result = _mod.parse_push_audit_response(resp, ["scripts/example.py"])
        self.assertEqual(result["findings"][0]["rri_input_proposal"], {})

    def test_rri_hint_never_used_as_score(self):
        resp = _findings_response(with_rri_hint=True)
        result = _mod.parse_push_audit_response(resp, ["scripts/example.py"])
        f = result["findings"][0]
        self.assertNotIn("canonical_rri", f)
        self.assertNotIn("final_rri", f)

    def test_rejects_patch_like_diff_git(self):
        resp = "STATUS: FINDINGS\nSUMMARY: x\ndiff --git a/x b/x"
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_push_audit_response(resp, [])
        self.assertIn("patch-like", str(ctx.exception))

    def test_rejects_patch_like_hunk_marker(self):
        resp = "STATUS: PASS\nSUMMARY: ok\n@@ -1 +1 @@"
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_push_audit_response(resp, [])
        self.assertIn("patch-like", str(ctx.exception))

    def test_blocked_status(self):
        resp = "STATUS: BLOCKED\nSUMMARY: packet not auditable"
        result = _mod.parse_push_audit_response(resp, [])
        self.assertEqual(result["status"], "blocked")

    def test_missing_status_raises(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_push_audit_response("SUMMARY: x", [])
        self.assertIn("missing STATUS", str(ctx.exception))

    def test_missing_summary_raises(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_push_audit_response("STATUS: PASS", [])
        self.assertIn("missing SUMMARY", str(ctx.exception))

    def test_pass_with_findings_raises(self):
        resp = "\n".join([
            "STATUS: PASS", "SUMMARY: ok",
            "=== FINDING START ===",
            "PATH: scripts/a.py", "LINE: 1", "SEVERITY: minor",
            "DETAIL: d", "SUGGESTION: s",
            "=== FINDING END ===",
        ])
        with self.assertRaises(RuntimeError):
            _mod.parse_push_audit_response(resp, ["scripts/a.py"])

    def test_non_integer_line_normalized_to_one(self):
        resp = "\n".join([
            "STATUS: FINDINGS", "SUMMARY: one issue",
            "=== FINDING START ===",
            "PATH: scripts/gemma-push-review.py", "LINE: N/A", "SEVERITY: minor",
            "DETAIL: missing check", "SUGGESTION: add check",
            "=== FINDING END ===",
        ])
        result = _mod.parse_push_audit_response(resp, ["scripts/gemma-push-review.py"])
        self.assertEqual(result["status"], "findings")
        self.assertEqual(result["findings"][0]["line"], 1)
        self.assertTrue(any("N/A" in w for w in result["format_warnings"]))

    def test_parser_is_separate_from_code_review(self):
        import inspect
        src = inspect.getfile(_mod.parse_push_audit_response)
        self.assertIn("gemma-push-review", src)

    def test_finding_scope_in_vs_out(self):
        resp = _findings_response(path="scripts/example.py")
        result = _mod.parse_push_audit_response(resp, ["scripts/example.py"])
        self.assertEqual(result["findings"][0]["scope"], "in-scope")

        result2 = _mod.parse_push_audit_response(resp, ["scripts/other.py"])
        self.assertEqual(result2["findings"][0]["scope"], "out-of-scope")


# ===========================================================================
# T1B: diff hunk parsing
# ===========================================================================

class ParseDiffHunks(unittest.TestCase):
    def test_basic_hunk(self):
        diff = (
            "diff --git a/scripts/a.py b/scripts/a.py\n"
            "--- a/scripts/a.py\n"
            "+++ b/scripts/a.py\n"
            "@@ -1,3 +1,3 @@\n"
            "-old\n"
            "+new\n"
        )
        hunks = _mod.parse_diff_hunks(diff)
        self.assertIn("scripts/a.py", hunks)
        self.assertEqual(hunks["scripts/a.py"], [(1, 3)])

    def test_empty_diff(self):
        self.assertEqual(_mod.parse_diff_hunks(""), {})

    def test_multiple_hunks_same_file(self):
        diff = (
            "+++ b/scripts/a.py\n"
            "@@ -1,2 +1,2 @@\n"
            "@@ -10,3 +10,3 @@\n"
        )
        hunks = _mod.parse_diff_hunks(diff)
        self.assertEqual(len(hunks["scripts/a.py"]), 2)

    def test_multiple_files(self):
        diff = (
            "+++ b/scripts/a.py\n"
            "@@ -1,2 +1,2 @@\n"
            "+++ b/scripts/b.py\n"
            "@@ -5,3 +5,3 @@\n"
        )
        hunks = _mod.parse_diff_hunks(diff)
        self.assertIn("scripts/a.py", hunks)
        self.assertIn("scripts/b.py", hunks)

    def test_dev_null_excluded(self):
        diff = "+++ /dev/null\n@@ -1,2 +0,0 @@\n"
        hunks = _mod.parse_diff_hunks(diff)
        self.assertNotIn("/dev/null", hunks)


# ===========================================================================
# T1B: evidence grounding
# ===========================================================================

class GroundFindings(unittest.TestCase):
    def _diff(self):
        return (
            "diff --git a/scripts/a.py b/scripts/a.py\n"
            "--- a/scripts/a.py\n"
            "+++ b/scripts/a.py\n"
            "@@ -1,5 +1,5 @@\n"
            "-old\n"
            "+new\n"
            " context\n"
        )

    def test_grounded_finding_in_hunk(self):
        findings = [{"path": "scripts/a.py", "line": 3, "severity": "major"}]
        result = _mod.ground_findings(findings, self._diff(), ["scripts/a.py"])
        self.assertTrue(result[0]["evidence_grounded"])
        self.assertNotIn("routing", result[0])

    def test_ungrounded_wrong_path(self):
        findings = [{"path": "scripts/other.py", "line": 1, "severity": "minor"}]
        result = _mod.ground_findings(findings, self._diff(), ["scripts/a.py"])
        self.assertFalse(result[0]["evidence_grounded"])
        self.assertEqual(result[0]["routing"], "observe")

    def test_ungrounded_line_far_outside_hunk(self):
        findings = [{"path": "scripts/a.py", "line": 999, "severity": "minor"}]
        result = _mod.ground_findings(findings, self._diff(), ["scripts/a.py"])
        self.assertFalse(result[0]["evidence_grounded"])
        self.assertEqual(result[0]["routing"], "observe")

    def test_grounding_slack_allows_nearby_lines(self):
        findings = [{"path": "scripts/a.py", "line": 12, "severity": "minor"}]
        result = _mod.ground_findings(findings, self._diff(), ["scripts/a.py"], slack=10)
        self.assertTrue(result[0]["evidence_grounded"])

    def test_all_observe_does_not_raise(self):
        findings = [
            {"path": "scripts/other.py", "line": 1, "severity": "minor"},
            {"path": "docs/foo.md", "line": 5, "severity": "nit"},
        ]
        result = _mod.ground_findings(findings, self._diff(), ["scripts/a.py"])
        self.assertTrue(all(not f["evidence_grounded"] for f in result))
        self.assertTrue(all(f["routing"] == "observe" for f in result))


# ===========================================================================
# T1B: write_blocked artifact
# ===========================================================================

class WriteBlocked(unittest.TestCase):
    def test_blocked_artifact_schema(self):
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            path = _mod.write_blocked("idle_timeout", "timed out after 60s", run, tmp, run["head_sha"])
            with open(path) as fh:
                data = json.load(fh)
        self.assertEqual(data["sentinel"], "blocked")
        self.assertEqual(data["blocked_reason"], "idle_timeout")
        self.assertEqual(data["role"], "gemma-push-reviewer")

    def test_blocked_includes_full_run_context(self):
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            path = _mod.write_blocked("ollama_unavailable", "model not found", run, tmp, run["head_sha"])
            with open(path) as fh:
                data = json.load(fh)
        ctx = data["run_context"]
        self.assertEqual(ctx["run_id"], 987654)
        self.assertIn("head_sha", ctx)
        self.assertIn("branch", ctx)
        self.assertIn("url", ctx)
        self.assertIn("conclusion", ctx)

    def test_different_reasons_produce_different_artifacts(self):
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            for reason in ("idle_timeout", "wall_timeout", "ollama_unavailable", "patch_like_output"):
                path = _mod.write_blocked(reason, "msg", run, tmp, run["head_sha"])
                with open(path) as fh:
                    data = json.load(fh)
                self.assertEqual(data["blocked_reason"], reason)


# ===========================================================================
# T1B: run_push_audit
# ===========================================================================

class RunPushAuditDryRun(unittest.TestCase):
    def test_dry_run_prints_model_payload(self):
        args = _model_args(dry_run=True)
        packet = _make_packet()
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            with patch("sys.stdout", new_callable=io.StringIO) as mock_out:
                with patch.object(_mod.gemma_local, "append_audit_log") as mock_audit:
                    rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
        self.assertEqual(rc, 0)
        mock_audit.assert_not_called()
        output = mock_out.getvalue()
        parsed = json.loads(output)
        self.assertIn("model", parsed)
        self.assertIn("messages", parsed)

    def test_dry_run_no_blocked_artifact(self):
        args = _model_args(dry_run=True)
        packet = _make_packet()
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            with patch("sys.stdout", new_callable=io.StringIO):
                with patch.object(_mod.gemma_local, "append_audit_log"):
                    _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertFalse(os.path.isfile(os.path.join(tmp, "blocked.json")))


class RunPushAuditOllamaUnavailable(unittest.TestCase):
    def test_ec4_writes_blocked_with_run_context(self):
        args = _model_args()
        packet = _make_packet()
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available",
                              side_effect=RuntimeError("model not installed")):
                rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 2)
            with open(os.path.join(tmp, "blocked.json")) as fh:
                data = json.load(fh)
        self.assertEqual(data["sentinel"], "blocked")
        self.assertEqual(data["blocked_reason"], "ollama_unavailable")
        self.assertIn("run_id", data["run_context"])
        self.assertIn("head_sha", data["run_context"])
        self.assertIn("branch", data["run_context"])
        self.assertIn("url", data["run_context"])

    def test_ec4_no_audit_log_on_unavailable(self):
        args = _model_args()
        packet = _make_packet()
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available",
                              side_effect=RuntimeError("missing")):
                with patch.object(_mod.gemma_local, "append_audit_log") as mock_audit:
                    _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
        mock_audit.assert_not_called()


class RunPushAuditTimeout(unittest.TestCase):
    def test_ec3_idle_timeout_writes_blocked(self):
        args = _model_args()
        packet = _make_packet()
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(_mod.gemma_local, "stream_chat",
                                  side_effect=_gl.GemmaIdleTimeout(60)):
                    rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 2)
            with open(os.path.join(tmp, "blocked.json")) as fh:
                data = json.load(fh)
        self.assertEqual(data["blocked_reason"], "idle_timeout")
        self.assertIn("run_id", data["run_context"])

    def test_ec3_wall_timeout_writes_blocked(self):
        args = _model_args()
        packet = _make_packet()
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(_mod.gemma_local, "stream_chat",
                                  side_effect=_gl.GemmaWallTimeout(900)):
                    rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 2)
            with open(os.path.join(tmp, "blocked.json")) as fh:
                data = json.load(fh)
        self.assertEqual(data["blocked_reason"], "wall_timeout")
        self.assertIn("run_id", data["run_context"])

    def test_ec3_stream_error_writes_blocked(self):
        args = _model_args()
        packet = _make_packet()
        run = _completed_run()
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(_mod.gemma_local, "stream_chat",
                                  side_effect=RuntimeError("response cut by token limit; output may be truncated")):
                    rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 2)
            with open(os.path.join(tmp, "blocked.json")) as fh:
                data = json.load(fh)
        self.assertEqual(data["blocked_reason"], "stream_error")
        self.assertIn("FENIX_PUSH_REVIEW_NUM_PREDICT", data["blocked_message"])
        self.assertIn("run_id", data["run_context"])


class RunPushAuditParserRejection(unittest.TestCase):
    def test_ec2_patch_like_writes_blocked(self):
        args = _model_args()
        packet = _make_packet()
        run = _completed_run()
        patch_resp = "STATUS: FINDINGS\nSUMMARY: x\ndiff --git a/x b/x"
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(_mod.gemma_local, "stream_chat", return_value=patch_resp):
                    rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 2)
            with open(os.path.join(tmp, "blocked.json")) as fh:
                data = json.load(fh)
        self.assertIn(data["blocked_reason"], ("parser_rejection", "patch_like_output"))
        self.assertIn("run_context", data)
        self.assertIn("run_id", data["run_context"])

    def test_ec2_blocked_report_keeps_workflow_run_identity(self):
        args = _model_args()
        raw_run = {
            "id": 555,
            "name": "ci",
            "event": "push",
            "head_branch": "main",
            "head_sha": "feed1234feed1234feed1234feed1234feed1234",
            "run_attempt": 1,
            "status": "completed",
            "conclusion": "failure",
            "html_url": "https://github.com/owner/repo/actions/runs/555",
        }
        run = _mod._normalize_run(raw_run)
        packet = _make_packet(
            run_info=run,
            after_sha=run["head_sha"],
            changed_paths=["scripts/example.py"],
        )
        invalid_resp = "\n".join([
            "STATUS: FINDINGS",
            "SUMMARY: x",
            "=== FINDING START ===",
            "PATH: scripts/example.py",
            "LINE: 0",
            "SEVERITY: minor",
            "DETAIL: style issue",
            "SUGGESTION: rename variable",
            "=== FINDING END ===",
        ])
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(_mod.gemma_local, "stream_chat", return_value=invalid_resp):
                    rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 2)
            with open(os.path.join(tmp, "blocked.json"), encoding="utf-8") as fh:
                data = json.load(fh)
            self.assertEqual(data["run_context"]["run_id"], 555)
            self.assertEqual(data["run_context"]["branch"], "main")
            self.assertEqual(
                data["run_context"]["head_sha"],
                "feed1234feed1234feed1234feed1234feed1234",
            )
            markdown_path = data["reports"]["markdown_summary_path"]
            self.assertTrue(os.path.isfile(markdown_path))
            self.assertTrue(markdown_path.endswith("feed123.md"))


class RunPushAuditHappyPath(unittest.TestCase):
    def test_hp1_grounded_finding_written(self):
        args = _model_args()
        packet = _make_packet(diff=_sample_diff(), changed_paths=["scripts/example.py"])
        run = _completed_run()
        response = _findings_response(path="scripts/example.py", line=1, with_rri_hint=True)
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(
                    _mod.gemma_local,
                    "stream_chat",
                    return_value=_mod.gemma_local.StreamChatResult(
                        content=response,
                        usage=_mod.gemma_local.StreamUsage(response_tokens=19),
                    ),
                ):
                    with patch.object(_mod.gemma_local, "append_audit_log") as mock_audit:
                        rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 0)
            with open(os.path.join(tmp, "aggregate.json")) as fh:
                agg = json.load(fh)

        grounded = [f for f in agg["findings"] if f["evidence_grounded"]]
        self.assertEqual(len(grounded), 1)
        self.assertTrue(grounded[0]["finding_id"].startswith("push-abcdef1-F"))
        self.assertEqual(agg["grounded_count"], 1)
        self.assertEqual(agg["observe_count"], 0)

        mock_audit.assert_called_once()
        record = mock_audit.call_args[0][0]
        self.assertEqual(record["role"], "push-reviewer")
        self.assertIn("run_id", record)
        self.assertIn("head_sha", record)
        self.assertIn("branch", record)
        self.assertIn("conclusion", record)
        self.assertEqual(record["grounded_count"], 1)
        self.assertEqual(record["response_tokens"], 19)
        self.assertGreater(record["packet_tokens_est"], 0)
        self.assertIn("file_tokens_est", record)

    def test_ec1_all_observe_does_not_block(self):
        args = _model_args()
        packet = _make_packet(diff=_sample_diff(), changed_paths=["scripts/example.py"])
        run = _completed_run()
        response = _findings_response(path="scripts/unrelated.py", line=1)
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(_mod.gemma_local, "stream_chat", return_value=response):
                    with patch.object(_mod.gemma_local, "append_audit_log"):
                        rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 0)
            self.assertFalse(os.path.isfile(os.path.join(tmp, "blocked.json")))
            with open(os.path.join(tmp, "aggregate.json")) as fh:
                agg = json.load(fh)
        self.assertEqual(agg["grounded_count"], 0)
        self.assertEqual(agg["observe_count"], 1)

    def test_pass_response_no_findings(self):
        args = _model_args()
        packet = _make_packet()
        run = _completed_run()
        response = "STATUS: PASS\nSUMMARY: no issues"
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(_mod.gemma_local, "stream_chat", return_value=response):
                    with patch.object(_mod.gemma_local, "append_audit_log"):
                        rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
            self.assertEqual(rc, 0)
            with open(os.path.join(tmp, "aggregate.json")) as fh:
                agg = json.load(fh)
        self.assertEqual(agg["grounded_count"], 0)
        self.assertEqual(agg["observe_count"], 0)
        self.assertEqual(agg["findings"], [])

    def test_legacy_string_response_keeps_response_tokens_null(self):
        args = _model_args()
        packet = _make_packet()
        run = _completed_run()
        response = "STATUS: PASS\nSUMMARY: no issues"
        with tempfile.TemporaryDirectory() as tmp:
            with patch.object(_mod.gemma_local, "ensure_model_available"):
                with patch.object(_mod.gemma_local, "stream_chat", return_value=response):
                    with patch.object(_mod.gemma_local, "append_audit_log") as mock_audit:
                        rc = _mod.run_push_audit(packet, run, args, tmp, repo_root=tmp)
        self.assertEqual(rc, 0)
        record = mock_audit.call_args[0][0]
        self.assertIsNone(record["response_tokens"])


# ===========================================================================
# T1B: env namespace (FENIX_ prefix)
# ===========================================================================

class EnvNamespace(unittest.TestCase):
    def test_push_review_model_takes_priority_over_low_rri(self):
        env = {
            "FENIX_PUSH_REVIEW_MODEL": "gemma-push-specific",
            "FENIX_LOW_RRI_MODEL": "fallback-model",
        }
        with patch.dict(os.environ, env, clear=False):
            model = os.environ.get(
                "FENIX_PUSH_REVIEW_MODEL",
                os.environ.get("FENIX_LOW_RRI_MODEL", _mod.gemma_local.DEFAULT_MODEL),
            )
        self.assertEqual(model, "gemma-push-specific")

    def test_fallback_to_low_rri_when_push_review_absent(self):
        clean = {k: v for k, v in os.environ.items()
                 if k != "FENIX_PUSH_REVIEW_MODEL"}
        clean["FENIX_LOW_RRI_MODEL"] = "fallback-model"
        with patch.dict(os.environ, clean, clear=True):
            model = os.environ.get(
                "FENIX_PUSH_REVIEW_MODEL",
                os.environ.get("FENIX_LOW_RRI_MODEL", _mod.gemma_local.DEFAULT_MODEL),
            )
        self.assertEqual(model, "fallback-model")

    def test_num_ctx_default_higher_than_code_review(self):
        self.assertGreater(
            _mod.DEFAULT_NUM_CTX_PUSH_REVIEW,
            _mod.gemma_local.DEFAULT_NUM_CTX,
        )

    def test_think_defaults_true_for_push_reviewer(self):
        clean = {k: v for k, v in os.environ.items()
                 if k not in ("FENIX_PUSH_REVIEW_THINK", "FENIX_LOW_RRI_THINK")}
        with patch.dict(os.environ, clean, clear=True):
            think = _mod.gemma_local.bool_from_env(
                "FENIX_PUSH_REVIEW_THINK",
                _mod.gemma_local.bool_from_env("FENIX_LOW_RRI_THINK", True),
            )
        self.assertTrue(think)


# ===========================================================================
# T2: canonical RRI scoring adapter
# ===========================================================================

def _rri_json(final=18, label="Low", penalties=None, triggers=None):
    return json.dumps({
        "platform": "generic",
        "variables": {},
        "base": final,
        "penalties": penalties or [],
        "penalty_total": 0,
        "final": final,
        "band": {"range": "0-25", "label": label, "effort": "S", "gate": "Gemma OK"},
        "triggers": triggers or [],
        "advisories": [],
    })


def _grounded_finding(path="scripts/example.py", finding_id="push-abc1234-F001",
                      rri_proposal=None):
    return {
        "finding_id": finding_id,
        "path": path,
        "line": 1,
        "severity": "minor",
        "detail": "style issue",
        "suggestion": "rename var",
        "evidence_grounded": True,
        "scope": "in-scope",
        "rri_input_proposal": rri_proposal or {"D": 1, "K": 1, "P": 1, "T": 1, "A": 0, "X": 0, "cc": 5},
    }


def _aggregate_with_findings(findings):
    return {"role": "gemma-push-reviewer", "schema_version": 1, "findings": findings}


def _report_candidate(label="Low", routing="gemma-developer-dispatch", pure=True,
                      path="scripts/example.py", status="patched"):
    return {
        "finding_id": "push-abcdef1-F001",
        "gemma_finding": {
            "path": path,
            "line": 1,
            "severity": "minor" if label == "Low" else "major",
            "detail": f"{label} issue",
            "suggestion": "fix it",
        },
        "canonical_rri": {
            "source": "scripts/rri.py --json",
            "final": 18 if label == "Low" else 43,
            "band": {"label": label},
            "raw": {"penalties": [], "triggers": []},
        },
        "routing": routing,
        "pure_low_eligible": pure,
        "developer_dispatch": {
            "status": status,
            "result_path": "logs/gemma-push-review/2026-06-25/abcdef1/developer/result.json",
            "development_report_path": "logs/gemma-push-review/2026-06-25/abcdef1/developer/report.json",
            "post_development_review_required": status in ("patched", "blocked"),
            "review_status": "in_review" if status in ("patched", "blocked") else None,
            "review_method": "gemma-code-review-triple-quorum" if status in ("patched", "blocked") else None,
            "review_orchestrator": "non-gemma-agent" if status in ("patched", "blocked") else None,
        },
    }


def _aggregate_report_fixture(candidates=None, status="findings", summary="summary", degraded=False,
                              quorum="met", passes_run=1, passes_succeeded=1):
    return {
        "role": "gemma-push-reviewer",
        "schema_version": 1,
        "repo": "owner/repo",
        "branch": "main",
        "before": "aaa0001",
        "after": "abcdef1234567890abcdef1234567890abcdef12",
        "status": status,
        "summary": summary,
        "findings": [],
        "grounded_count": len(candidates or []),
        "observe_count": 0,
        "candidates": candidates or [],
        "candidates_scored_count": len(candidates or []),
        "candidates_pure_low_count": sum(1 for c in (candidates or []) if c.get("pure_low_eligible")),
        "changed_paths": [c["gemma_finding"]["path"] for c in (candidates or [])] or ["scripts/example.py"],
        "pipeline": {
            "run_id": 987654,
            "conclusion": "success",
        },
        "audit": {
            "passes_run": passes_run,
            "passes_succeeded": passes_succeeded,
            "quorum": quorum,
            "degraded": degraded,
            "aggregate_path": None,
        },
        "developer_dispatch": {
            "attempted_count": sum(1 for c in (candidates or []) if (c.get("developer_dispatch") or {}).get("status") in ("patched", "blocked")),
            "succeeded_count": sum(1 for c in (candidates or []) if (c.get("developer_dispatch") or {}).get("status") == "patched"),
            "blocked_count": sum(1 for c in (candidates or []) if (c.get("developer_dispatch") or {}).get("status") == "blocked"),
            "development_reports": [c["developer_dispatch"]["development_report_path"] for c in (candidates or []) if (c.get("developer_dispatch") or {}).get("development_report_path")],
        },
        "post_development_review": {
            "required_count": sum(1 for c in (candidates or []) if (c.get("developer_dispatch") or {}).get("post_development_review_required")),
            "in_review_count": sum(1 for c in (candidates or []) if (c.get("developer_dispatch") or {}).get("review_status") == "in_review"),
            "pending_count": sum(1 for c in (candidates or []) if (c.get("developer_dispatch") or {}).get("review_status") == "in_review"),
        },
        "deployer_followup": {
            "pure_low_dispatched_count": sum(1 for c in (candidates or []) if (c.get("developer_dispatch") or {}).get("status") == "patched"),
            "deferred_due_complexity_count": sum(1 for c in (candidates or []) if c.get("routing") == "daily-non-gemma-review"),
            "needs_hitl_count": sum(1 for c in (candidates or []) if ((c.get("canonical_rri") or {}).get("band") or {}).get("label") in ("Moderate", "Med-high", "Complex", "High", "Very high")),
        },
        "ts": "2026-06-25T10:00:00Z",
    }


class BuildRriCmd(unittest.TestCase):
    def test_uses_cc_when_provided(self):
        proposal = {"cc": 12, "D": 1, "K": 1, "P": 1, "T": 2, "A": 0, "X": 0}
        cmd = _mod._build_rri_cmd("scripts/example.py", proposal)
        self.assertIn("--cc", cmd)
        idx = cmd.index("--cc")
        self.assertEqual(cmd[idx + 1], "12")
        self.assertNotIn("--C", cmd)

    def test_falls_back_to_C_when_no_cc(self):
        proposal = {"C": 2, "D": 1, "K": 1, "P": 1, "T": 2, "A": 0, "X": 0}
        cmd = _mod._build_rri_cmd("scripts/example.py", proposal)
        self.assertIn("--C", cmd)
        self.assertNotIn("--cc", cmd)

    def test_C_defaults_to_zero_when_absent(self):
        proposal = {"D": 1, "K": 1, "P": 1, "T": 2, "A": 0, "X": 0}
        cmd = _mod._build_rri_cmd("scripts/example.py", proposal)
        self.assertIn("--C", cmd)
        idx = cmd.index("--C")
        self.assertEqual(cmd[idx + 1], "0")

    def test_all_required_flags_present(self):
        proposal = {"cc": 5, "D": 1, "K": 2, "P": 1, "T": 3, "A": 1, "X": 0}
        cmd = _mod._build_rri_cmd("scripts/example.py", proposal)
        for flag in ("--D", "--K", "--P", "--T", "--A", "--X"):
            self.assertIn(flag, cmd)

    def test_touches_path_included(self):
        cmd = _mod._build_rri_cmd("scripts/foo.py", {})
        self.assertIn("--touches", cmd)
        idx = cmd.index("--touches")
        self.assertEqual(cmd[idx + 1], "scripts/foo.py")

    def test_json_flag_present(self):
        cmd = _mod._build_rri_cmd("scripts/foo.py", {})
        self.assertIn("--json", cmd)

    def test_cc_zero_falls_back_to_C_flag(self):
        proposal = {"cc": 0, "D": 1, "K": 1, "P": 1, "T": 1, "A": 0, "X": 0}
        cmd = _mod._build_rri_cmd("scripts/foo.py", proposal)
        self.assertNotIn("--cc", cmd)
        self.assertIn("--C", cmd)
        idx = cmd.index("--C")
        self.assertEqual(cmd[idx + 1], "0")


class RoutingFromBand(unittest.TestCase):
    def test_low_routes_to_gemma_dispatch(self):
        self.assertEqual(_mod._routing_from_band("Low"), "gemma-developer-dispatch")

    def test_moderate_routes_to_daily(self):
        self.assertEqual(_mod._routing_from_band("Moderate"), "daily-non-gemma-review")

    def test_med_high_routes_to_daily(self):
        self.assertEqual(_mod._routing_from_band("Med-high"), "daily-non-gemma-review")

    def test_complex_routes_to_daily(self):
        self.assertEqual(_mod._routing_from_band("Complex"), "daily-non-gemma-review")

    def test_unknown_defaults_to_daily(self):
        self.assertEqual(_mod._routing_from_band("UnknownBand"), "daily-non-gemma-review")


class ScoreCandidatesHP1(unittest.TestCase):
    def test_hp1_canonical_rri_source_is_rri_py(self):
        agg = _aggregate_with_findings([_grounded_finding()])
        rri_out = _rri_json(final=18, label="Low")
        with patch.object(_mod.subprocess, "run") as mock_run:
            mock_run.return_value = type("R", (), {"returncode": 0, "stdout": rri_out, "stderr": ""})()
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        self.assertEqual(len(candidates), 1)
        self.assertEqual(candidates[0]["canonical_rri"]["source"], "scripts/rri.py --json")

    def test_hp1_final_and_band_extracted(self):
        agg = _aggregate_with_findings([_grounded_finding()])
        rri_out = _rri_json(final=18, label="Low")
        with patch.object(_mod.subprocess, "run") as mock_run:
            mock_run.return_value = type("R", (), {"returncode": 0, "stdout": rri_out, "stderr": ""})()
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        c = candidates[0]
        self.assertEqual(c["canonical_rri"]["final"], 18)
        self.assertEqual(c["canonical_rri"]["band"]["label"], "Low")

    def test_hp1_proposal_never_overwrites_canonical(self):
        proposal = {"D": 5, "K": 5, "P": 5, "T": 5, "A": 5, "X": 5, "cc": 99}
        agg = _aggregate_with_findings([_grounded_finding(rri_proposal=proposal)])
        rri_out = _rri_json(final=18, label="Low")
        with patch.object(_mod.subprocess, "run") as mock_run:
            mock_run.return_value = type("R", (), {"returncode": 0, "stdout": rri_out, "stderr": ""})()
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        self.assertEqual(candidates[0]["canonical_rri"]["final"], 18)
        self.assertEqual(candidates[0]["canonical_rri"]["source"], "scripts/rri.py --json")


class ScoreCandidatesHP2(unittest.TestCase):
    def test_hp2_low_no_penalties_is_pure_low_eligible(self):
        agg = _aggregate_with_findings([_grounded_finding()])
        rri_out = _rri_json(final=18, label="Low", penalties=[], triggers=[])
        with patch.object(_mod.subprocess, "run") as mock_run:
            mock_run.return_value = type("R", (), {"returncode": 0, "stdout": rri_out, "stderr": ""})()
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        self.assertTrue(candidates[0]["pure_low_eligible"])
        self.assertEqual(candidates[0]["routing"], "gemma-developer-dispatch")

    def test_hp2_moderate_not_pure_low(self):
        agg = _aggregate_with_findings([_grounded_finding()])
        rri_out = _rri_json(final=30, label="Moderate")
        with patch.object(_mod.subprocess, "run") as mock_run:
            mock_run.return_value = type("R", (), {"returncode": 0, "stdout": rri_out, "stderr": ""})()
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        self.assertFalse(candidates[0]["pure_low_eligible"])
        self.assertEqual(candidates[0]["routing"], "daily-non-gemma-review")


class ScoreCandidatesEC1(unittest.TestCase):
    def test_ec1_out_of_scope_path_dismissed(self):
        agg = _aggregate_with_findings([_grounded_finding(path="scripts/other.py")])
        with patch.object(_mod.subprocess, "run") as mock_run:
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        mock_run.assert_not_called()
        self.assertEqual(len(candidates), 1)
        self.assertEqual(candidates[0]["routing"], "dismiss-candidate")
        self.assertIsNone(candidates[0]["canonical_rri"])
        self.assertFalse(candidates[0]["pure_low_eligible"])

    def test_ec1_observe_findings_skipped(self):
        observe_f = {
            "path": "scripts/example.py", "line": 1, "severity": "minor",
            "detail": "d", "suggestion": "s",
            "evidence_grounded": False, "routing": "observe",
            "rri_input_proposal": {},
        }
        agg = _aggregate_with_findings([observe_f])
        with patch.object(_mod.subprocess, "run") as mock_run:
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        mock_run.assert_not_called()
        self.assertEqual(candidates, [])


class ScoreCandidatesEC2(unittest.TestCase):
    def test_ec2_rri_failure_marks_unavailable(self):
        agg = _aggregate_with_findings([_grounded_finding()])
        with patch.object(_mod.subprocess, "run") as mock_run:
            mock_run.return_value = type("R", (), {
                "returncode": 1, "stdout": "", "stderr": "missing required arg --D"
            })()
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        self.assertTrue(candidates[0]["rri_unavailable"])
        self.assertEqual(candidates[0]["routing"], "daily-non-gemma-review")
        self.assertIsNone(candidates[0]["canonical_rri"])

    def test_ec2_json_parse_error_marks_unavailable(self):
        agg = _aggregate_with_findings([_grounded_finding()])
        with patch.object(_mod.subprocess, "run") as mock_run:
            mock_run.return_value = type("R", (), {
                "returncode": 0, "stdout": "not-json", "stderr": ""
            })()
            candidates = _mod.score_candidates(agg, ["scripts/example.py"])
        self.assertTrue(candidates[0]["rri_unavailable"])
        self.assertIsNone(candidates[0]["canonical_rri"])


# ===========================================================================
# T4: report writer
# ===========================================================================

class PushAuditReports(unittest.TestCase):
    def test_report_paths_deterministic_from_date_and_short_sha(self):
        path = _mod._push_report_markdown_path(
            "/tmp/out",
            "abcdef1234567890abcdef1234567890abcdef12",
            "2026-06-25T10:00:00Z",
        )
        self.assertEqual(path, "/tmp/out/reports/2026-06-25-abcdef1.md")

    def test_hp1_delegated_low_and_moderate_rendered(self):
        low = _report_candidate(label="Low", routing="gemma-developer-dispatch", pure=True, status="patched")
        moderate = _report_candidate(
            label="Moderate", routing="daily-non-gemma-review", pure=False,
            path="scripts/moderate.py", status="not_started",
        )
        aggregate = _aggregate_report_fixture([low, moderate])
        with tempfile.TemporaryDirectory() as repo_root, tempfile.TemporaryDirectory() as out_dir:
            aggregate_path, markdown_path = _mod.write_push_reports(aggregate, out_dir, repo_root=repo_root)
            self.assertTrue(os.path.isfile(aggregate_path))
            self.assertTrue(os.path.isfile(markdown_path))
            with open(markdown_path, encoding="utf-8") as fh:
                text = fh.read()
        self.assertIn("Delegated development reports", text)
        self.assertIn("Non-Low deferred items", text)
        self.assertIn("approval before implementation", text)

    def test_hp2_no_findings_renders_empty_sections(self):
        aggregate = _aggregate_report_fixture([], status="pass", summary="no issues")
        with tempfile.TemporaryDirectory() as repo_root, tempfile.TemporaryDirectory() as out_dir:
            _, markdown_path = _mod.write_push_reports(aggregate, out_dir, repo_root=repo_root)
            with open(markdown_path, encoding="utf-8") as fh:
                text = fh.read()
        self.assertIn("| Status | pass |", text)
        self.assertIn("none", text)

    def test_ec1_degraded_report_marks_degraded_audit(self):
        aggregate = _aggregate_report_fixture(
            [_report_candidate(label="Low", routing="gemma-developer-dispatch", pure=True, status="patched")],
            degraded=True, passes_run=3, passes_succeeded=2,
        )
        with tempfile.TemporaryDirectory() as repo_root, tempfile.TemporaryDirectory() as out_dir:
            aggregate_path, markdown_path = _mod.write_push_reports(aggregate, out_dir, repo_root=repo_root)
            with open(aggregate_path, encoding="utf-8") as fh:
                data = json.load(fh)
            with open(markdown_path, encoding="utf-8") as fh:
                text = fh.read()
        self.assertTrue(data["audit"]["degraded"])
        self.assertIn("| Degraded | true |", text)
        self.assertIn("| Passes | 2/3 |", text)

    def test_ec3_complex_candidate_explicitly_not_auto_apply(self):
        complex_candidate = _report_candidate(
            label="Complex", routing="daily-non-gemma-review", pure=False,
            path="scripts/complex.py", status="not_started",
        )
        aggregate = _aggregate_report_fixture([complex_candidate])
        with tempfile.TemporaryDirectory() as repo_root, tempfile.TemporaryDirectory() as out_dir:
            _, markdown_path = _mod.write_push_reports(aggregate, out_dir, repo_root=repo_root)
            with open(markdown_path, encoding="utf-8") as fh:
                text = fh.read()
        self.assertIn("do not auto-apply", text)
        self.assertIn("decompose before implementation", text)


if __name__ == "__main__":
    unittest.main()
