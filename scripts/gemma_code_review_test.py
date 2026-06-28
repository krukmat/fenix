#!/usr/bin/env python3
"""Unit tests for gemma-code-review.py."""

import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "gemma-code-review.py")
_spec = importlib.util.spec_from_file_location("gemma_code_review", _SCRIPT)
_mod = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_mod)


def _packet():
    return "\n".join([
        "# Review packet",
        "```diff",
        "diff --git a/scripts/a.py b/scripts/a.py",
        "--- a/scripts/a.py",
        "+++ b/scripts/a.py",
        "@@ -1 +1 @@",
        "-old",
        "+new",
        "```",
    ])


def _response(status="FINDINGS", finding_path="scripts/a.py", severity="major"):
    lines = [
        f"STATUS: {status}",
        "SUMMARY: reviewed",
    ]
    if status == "FINDINGS":
        lines.extend([
            "=== FINDING START ===",
            f"PATH: {finding_path}",
            "LINE: 12",
            f"SEVERITY: {severity}",
            "DETAIL: concrete issue",
            "SUGGESTION: fix it",
            "=== FINDING END ===",
        ])
    return "\n".join(lines)


class ChangedPaths(unittest.TestCase):
    def test_changed_paths_from_packet(self):
        self.assertEqual(_mod.changed_paths_from_packet(_packet()), ["scripts/a.py"])

    def test_changed_paths_handles_dev_null(self):
        packet = "diff --git a/scripts/a.py b/scripts/a.py\n--- /dev/null\n+++ b/scripts/a.py"
        self.assertEqual(_mod.changed_paths_from_packet(packet), ["scripts/a.py"])


class BuildReviewPayload(unittest.TestCase):
    def test_prompt_is_read_only(self):
        payload = _mod.build_review_payload("model", "packet", 16384, 4096, 0.1, False)
        system = payload["messages"][0]["content"]
        self.assertIn("read-only", system)
        self.assertIn("Do not approve", system)
        self.assertIn("output file bodies", system)
        self.assertIn("STATUS: PASS", system)

    def test_prompt_identifies_as_fenix(self):
        payload = _mod.build_review_payload("model", "packet", 16384, 4096, 0.1, False)
        system = payload["messages"][0]["content"]
        self.assertIn("Fenix", system)
        self.assertNotIn("DubBridge", system)

    def test_generation_options_are_shared_shape(self):
        payload = _mod.build_review_payload("model", "packet", 8192, 2048, 0.25, True)
        self.assertTrue(payload["stream"])
        self.assertTrue(payload["think"])
        self.assertEqual(payload["options"]["temperature"], 0.25)
        self.assertEqual(payload["options"]["num_ctx"], 8192)
        self.assertEqual(payload["options"]["num_predict"], 2048)


class ParseReviewResponse(unittest.TestCase):
    def test_pass_without_findings(self):
        result = _mod.parse_review_response(
            "STATUS: PASS\nSUMMARY: clean",
            ["scripts/a.py"],
        )
        self.assertEqual(result["status"], "pass")
        self.assertEqual(result["findings"], [])

    def test_status_without_prefix_accepted(self):
        result = _mod.parse_review_response(
            "PASS\nSUMMARY: clean",
            ["scripts/a.py"],
        )
        self.assertEqual(result["status"], "pass")
        self.assertEqual(result["findings"], [])

    def test_duplicate_identical_status_header_is_accepted_with_warning(self):
        result = _mod.parse_review_response(
            "STATUS: PASS\nSTATUS: PASS\nSUMMARY: clean",
            ["scripts/a.py"],
        )
        self.assertEqual(result["status"], "pass")
        self.assertIn("duplicate STATUS header repeated with same value 'PASS', skipping", result["format_warnings"])

    def test_conflicting_status_headers_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_review_response(
                "STATUS: PASS\nSTATUS: FINDINGS\nSUMMARY: clean",
                ["scripts/a.py"],
            )
        self.assertIn("conflicting STATUS headers", str(ctx.exception))

    def test_duplicate_identical_summary_header_is_accepted_with_warning(self):
        result = _mod.parse_review_response(
            "STATUS: PASS\nSUMMARY: clean\nSUMMARY: clean",
            ["scripts/a.py"],
        )
        self.assertEqual(result["summary"], "clean")
        self.assertIn(
            "duplicate SUMMARY header repeated with same value 'clean', skipping",
            result["format_warnings"],
        )

    def test_conflicting_summary_headers_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_review_response(
                "STATUS: PASS\nSUMMARY: clean\nSUMMARY: changed",
                ["scripts/a.py"],
            )
        self.assertIn("conflicting SUMMARY headers", str(ctx.exception))

    def test_finding_in_scope(self):
        result = _mod.parse_review_response(_response(), ["scripts/a.py"])
        self.assertEqual(result["status"], "findings")
        self.assertEqual(result["findings"][0]["scope"], "in-scope")
        self.assertEqual(result["findings"][0]["line"], 12)

    def test_finding_out_of_scope_is_labeled_not_dropped(self):
        result = _mod.parse_review_response(
            _response(finding_path="scripts/other.py"),
            ["scripts/a.py"],
        )
        self.assertEqual(result["findings"][0]["scope"], "out-of-scope")
        self.assertEqual(result["findings"][0]["path"], "scripts/other.py")

    def test_blocked_allowed_without_findings(self):
        result = _mod.parse_review_response(
            "STATUS: BLOCKED\nSUMMARY: packet incomplete",
            ["scripts/a.py"],
        )
        self.assertEqual(result["status"], "blocked")

    def test_invalid_severity_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_review_response(_response(severity="critical"), ["scripts/a.py"])
        self.assertIn("invalid severity", str(ctx.exception))

    def test_patch_like_output_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_review_response(
                "STATUS: FINDINGS\nSUMMARY: x\n"
                "=== FINDING START ===\n"
                "PATH: scripts/a.py\n"
                "LINE: 1\n"
                "SEVERITY: major\n"
                "DETAIL: diff --git a/x b/x\n"
                "SUGGESTION: no\n"
                "=== FINDING END ===",
                ["scripts/a.py"],
            )
        self.assertIn("patch-like", str(ctx.exception))

    def test_pass_with_findings_coerced_to_findings(self):
        content = "\n".join([
            "STATUS: PASS",
            "SUMMARY: x",
            "=== FINDING START ===",
            "PATH: scripts/a.py",
            "LINE: 1",
            "SEVERITY: minor",
            "DETAIL: issue",
            "SUGGESTION: fix",
            "=== FINDING END ===",
        ])
        result = _mod.parse_review_response(content, ["scripts/a.py"])
        self.assertEqual(result["status"], "findings")
        self.assertEqual(len(result["findings"]), 1)
        self.assertIn(
            "STATUS PASS with findings coerced to FINDINGS",
            result["format_warnings"],
        )

    def test_findings_without_finding_block_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_review_response("STATUS: FINDINGS\nSUMMARY: x", ["scripts/a.py"])
        self.assertIn("requires findings", str(ctx.exception))

    def test_missing_end_marker_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_review_response(
                "\n".join([
                    "STATUS: FINDINGS",
                    "SUMMARY: x",
                    "=== FINDING START ===",
                    "PATH: scripts/a.py",
                    "LINE: 1",
                    "SEVERITY: minor",
                    "DETAIL: issue",
                    "SUGGESTION: fix",
                ]),
                ["scripts/a.py"],
            )
        self.assertIn("finding end", str(ctx.exception))


class CliBehavior(unittest.TestCase):
    def run_cli(self, *args, stdin=None, env=None):
        return subprocess.run(
            [sys.executable, _SCRIPT, *args],
            capture_output=True,
            text=True,
            input=stdin,
            env=env,
        )

    def test_dry_run_uses_review_model_env(self):
        env = os.environ.copy()
        env["FENIX_REVIEW_MODEL"] = "review-model"
        r = self.run_cli("-", "--dry-run", stdin=_packet(), env=env)
        self.assertEqual(r.returncode, 0, r.stderr)
        payload = json.loads(r.stdout)
        self.assertEqual(payload["model"], "review-model")

    def test_dry_run_falls_back_to_low_rri_model_env(self):
        env = os.environ.copy()
        env.pop("FENIX_REVIEW_MODEL", None)
        env["FENIX_LOW_RRI_MODEL"] = "low-model"
        r = self.run_cli("-", "--dry-run", stdin=_packet(), env=env)
        self.assertEqual(r.returncode, 0, r.stderr)
        payload = json.loads(r.stdout)
        self.assertEqual(payload["model"], "low-model")

    def test_empty_packet_exits_1(self):
        r = self.run_cli("-", stdin=" \n")
        self.assertEqual(r.returncode, 1)
        self.assertIn("empty", r.stderr)

    def test_dry_run_think_false_by_default(self):
        r = self.run_cli("-", "--dry-run", stdin=_packet())
        self.assertEqual(r.returncode, 0, r.stderr)
        payload = json.loads(r.stdout)
        self.assertFalse(payload["think"])

    def test_no_dubbridge_env_refs_in_script(self):
        with open(_SCRIPT) as fh:
            src = fh.read()
        self.assertNotIn("DUBBRIDGE_", src)


# ---------------------------------------------------------------------------
# AuditEmission — verify append_audit_log is called with the right shape
# ---------------------------------------------------------------------------
class AuditEmission(unittest.TestCase):
    def _run(self, stream_response, extra_args=None):
        captured = []
        with tempfile.TemporaryDirectory() as tmp:
            out_path = os.path.join(tmp, "result.json")
            packet_file = os.path.join(tmp, "packet.md")
            with open(packet_file, "w") as f:
                f.write(_packet())

            argv = [_SCRIPT, packet_file, "--out", out_path, "--passes", "1"] + (extra_args or [])
            with patch("sys.argv", argv), \
                 patch.object(_mod.gemma_local, "ensure_model_available"), \
                 patch.object(_mod.gemma_local, "stream_chat",
                              return_value=stream_response), \
                 patch.object(_mod.gemma_local, "append_audit_log",
                              side_effect=lambda r: captured.append(r)):
                _mod.main()
        return captured

    def test_pass_emits_one_record_with_reviewer_role(self):
        records = self._run(_mod.gemma_local.StreamChatResult(
            content="STATUS: PASS\nSUMMARY: clean",
            usage=_mod.gemma_local.StreamUsage(response_tokens=17),
        ))
        self.assertEqual(len(records), 1)
        r = records[0]
        self.assertEqual(r["role"], "reviewer")
        self.assertEqual(r["outcome"], "PASS")
        self.assertEqual(r["done_reason"], "stop")
        self.assertFalse(r["escalated"])
        self.assertEqual(r["findings_count"], 0)
        self.assertEqual(r["findings_by_severity"], {"blocking": 0, "major": 0, "minor": 0, "nit": 0})
        self.assertEqual(r["out_of_scope"], 0)
        self.assertIsNone(r["dispositions"])
        self.assertIn("system_prompt", r)
        self.assertIn("user_prompt", r)
        self.assertEqual(r["response_tokens"], 17)
        self.assertGreater(r["packet_tokens_est"], 0)

    def test_findings_counts_by_severity(self):
        records = self._run(_response("FINDINGS", severity="major"))
        r = records[0]
        self.assertEqual(r["outcome"], "FINDINGS")
        self.assertEqual(r["findings_count"], 1)
        self.assertEqual(r["findings_by_severity"]["major"], 1)
        self.assertEqual(r["findings_by_severity"]["blocking"], 0)

    def test_legacy_string_response_keeps_response_tokens_null(self):
        records = self._run("STATUS: PASS\nSUMMARY: clean")
        self.assertEqual(len(records), 1)
        self.assertIsNone(records[0]["response_tokens"])

    def test_out_of_scope_counted(self):
        records = self._run(_response("FINDINGS", finding_path="scripts/other.py"))
        r = records[0]
        self.assertEqual(r["out_of_scope"], 1)

    def test_blocked_emits_escalated_true(self):
        records = self._run("STATUS: BLOCKED\nSUMMARY: cannot review")
        r = records[0]
        self.assertEqual(r["outcome"], "BLOCKED")
        self.assertTrue(r["escalated"])
        self.assertEqual(r["findings_count"], 0)

    def test_task_id_and_attempt_passed_through(self):
        records = self._run(
            "STATUS: PASS\nSUMMARY: ok",
            extra_args=["--task-id", "T3", "--attempt", "1"],
        )
        r = records[0]
        self.assertEqual(r["task_id"], "T3")
        self.assertEqual(r["attempt"], 1)


# ---------------------------------------------------------------------------
# Reconcile — unit tests for the D8 deterministic reconciliation function
# ---------------------------------------------------------------------------

def _finding(path="scripts/a.py", line=10, severity="major", scope="in-scope", **kw):
    return {"path": path, "line": line, "severity": severity,
            "scope": scope, "detail": "d", "suggestion": "s", **kw}


def _pass_result(status="pass", findings=None, summary="ok"):
    return {"status": status, "summary": summary,
            "findings": findings or [], "changed_paths": ["scripts/a.py"]}


class ReconcileUnit(unittest.TestCase):
    def test_hp1_three_pass_all_pass_no_findings(self):
        passes = [_pass_result("pass"), _pass_result("pass"), _pass_result("pass")]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["status"], "pass")
        self.assertEqual(agg["findings"], [])
        self.assertEqual(agg["reconciliation"]["consensus_count"], 0)

    def test_consensus_finding_present_in_two_passes(self):
        f = _finding()
        passes = [
            _pass_result("findings", [f]),
            _pass_result("findings", [f]),
            _pass_result("pass"),
        ]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["reconciliation"]["consensus_count"], 1)
        self.assertEqual(agg["reconciliation"]["pass_specific_count"], 0)
        self.assertEqual(agg["findings"], agg["reconciliation"]["consensus"])

    def test_pass_specific_finding_in_one_pass_only(self):
        f = _finding()
        passes = [
            _pass_result("findings", [f]),
            _pass_result("pass"),
        ]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["reconciliation"]["pass_specific_count"], 1)
        self.assertEqual(agg["reconciliation"]["consensus_count"], 0)

    def test_severity_inconsistent_same_path_line_different_severity(self):
        f_major = _finding(severity="major")
        f_minor = _finding(severity="minor")
        passes = [
            _pass_result("findings", [f_major]),
            _pass_result("findings", [f_minor]),
        ]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["reconciliation"]["severity_inconsistent_count"], 2)
        self.assertEqual(agg["reconciliation"]["pass_specific_count"], 0)

    def test_location_inconsistent_within_three_lines_different_passes(self):
        f1 = _finding(line=10)
        f2 = _finding(line=12)
        passes = [
            _pass_result("findings", [f1]),
            _pass_result("findings", [f2]),
        ]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["reconciliation"]["location_inconsistent_count"], 2)

    def test_location_not_clustered_beyond_three_lines(self):
        f1 = _finding(line=10)
        f2 = _finding(line=14)
        passes = [
            _pass_result("findings", [f1]),
            _pass_result("findings", [f2]),
        ]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["reconciliation"]["location_inconsistent_count"], 0)
        self.assertEqual(agg["reconciliation"]["pass_specific_count"], 2)

    def test_likely_false_positive_pass_specific_and_out_of_scope(self):
        f = _finding(path="scripts/other.py", scope="out-of-scope")
        passes = [
            _pass_result("findings", [f]),
            _pass_result("pass"),
        ]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["reconciliation"]["likely_false_positive_count"], 1)
        self.assertEqual(agg["reconciliation"]["pass_specific_count"], 0)

    def test_aggregate_status_findings_when_any_pass_has_findings(self):
        passes = [
            _pass_result("findings", [_finding()]),
            _pass_result("pass"),
        ]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["status"], "findings")

    def test_aggregate_status_pass_when_all_passes_pass(self):
        passes = [_pass_result("pass"), _pass_result("pass")]
        agg = _mod.reconcile(passes, ["scripts/a.py"])
        self.assertEqual(agg["status"], "pass")


# ---------------------------------------------------------------------------
# MultiPassCli — integration tests for the N-pass loop and quorum via CLI
# ---------------------------------------------------------------------------

class MultiPassCli(unittest.TestCase):
    def _run_multi(self, pass_responses, extra_args=None):
        responses = list(pass_responses)
        call_count = [0]

        def _mock_stream(*a, **kw):
            idx = call_count[0]
            call_count[0] += 1
            resp = responses[idx] if idx < len(responses) else "STATUS: PASS\nSUMMARY: ok"
            if isinstance(resp, Exception):
                raise resp
            return resp

        with tempfile.TemporaryDirectory() as tmp:
            out_path = os.path.join(tmp, "result.json")
            packet_file = os.path.join(tmp, "packet.md")
            with open(packet_file, "w") as f:
                f.write(_packet())

            n = len(pass_responses)
            argv = ([_SCRIPT, packet_file, "--out", out_path, "--passes", str(n)]
                    + (extra_args or []))
            with patch("sys.argv", argv), \
                 patch.object(_mod.gemma_local, "ensure_model_available"), \
                 patch.object(_mod.gemma_local, "stream_chat", side_effect=_mock_stream), \
                 patch.object(_mod.gemma_local, "append_audit_log"):
                exit_code = _mod.main()

            aggregate = None
            if os.path.exists(out_path):
                with open(out_path) as f:
                    aggregate = json.load(f)

            pass_artifacts = []
            for k in range(1, n + 1):
                p = out_path.replace(".json", f".pass{k}.json")
                if os.path.exists(p):
                    with open(p) as f:
                        pass_artifacts.append(json.load(f))
                else:
                    pass_artifacts.append(None)

        return exit_code, aggregate, pass_artifacts

    def test_hp1_three_of_three_pass_exit_zero_no_degraded(self):
        ec, agg, _ = self._run_multi([
            "STATUS: PASS\nSUMMARY: ok",
            "STATUS: PASS\nSUMMARY: ok",
            "STATUS: PASS\nSUMMARY: ok",
        ])
        self.assertEqual(ec, 0)
        self.assertFalse(agg["degraded"])
        self.assertEqual(agg["passes_run"], 3)
        self.assertEqual(agg["passes_succeeded"], 3)

    def test_two_of_three_degraded_exit_zero(self):
        ec, agg, _ = self._run_multi([
            "STATUS: PASS\nSUMMARY: ok",
            RuntimeError("simulated failure"),
            "STATUS: PASS\nSUMMARY: ok",
        ])
        self.assertEqual(ec, 0)
        self.assertTrue(agg["degraded"])
        self.assertEqual(agg["passes_succeeded"], 2)

    def test_one_of_three_quorum_fails_exit_nonzero_no_aggregate(self):
        ec, agg, _ = self._run_multi([
            "STATUS: PASS\nSUMMARY: ok",
            RuntimeError("fail"),
            RuntimeError("fail"),
        ])
        self.assertNotEqual(ec, 0)
        self.assertIsNone(agg)

    def test_truncation_fails_pass(self):
        ec, agg, _ = self._run_multi([
            "STATUS: PASS\nSUMMARY: ok",
            RuntimeError("done_reason='length' truncated"),
            "STATUS: PASS\nSUMMARY: ok",
        ])
        self.assertEqual(ec, 0)
        self.assertTrue(agg["degraded"])

    def test_per_pass_artifacts_written(self):
        _, _, artifacts = self._run_multi([
            "STATUS: PASS\nSUMMARY: ok",
            "STATUS: PASS\nSUMMARY: ok",
            "STATUS: PASS\nSUMMARY: ok",
        ])
        self.assertEqual(len([a for a in artifacts if a is not None]), 3)

    def test_passes_1_backward_compat_no_reconciliation(self):
        with tempfile.TemporaryDirectory() as tmp:
            out_path = os.path.join(tmp, "result.json")
            packet_file = os.path.join(tmp, "packet.md")
            with open(packet_file, "w") as f:
                f.write(_packet())
            argv = [_SCRIPT, packet_file, "--out", out_path, "--passes", "1"]
            with patch("sys.argv", argv), \
                 patch.object(_mod.gemma_local, "ensure_model_available"), \
                 patch.object(_mod.gemma_local, "stream_chat",
                              return_value="STATUS: PASS\nSUMMARY: ok"), \
                 patch.object(_mod.gemma_local, "append_audit_log"):
                ec = _mod.main()
            with open(out_path) as f:
                result = json.load(f)
        self.assertEqual(ec, 0)
        self.assertNotIn("reconciliation", result)
        self.assertNotIn("passes_run", result)

    def test_aggregate_has_reconciliation_block(self):
        ec, agg, _ = self._run_multi([
            "STATUS: PASS\nSUMMARY: ok",
            "STATUS: PASS\nSUMMARY: ok",
        ])
        self.assertEqual(ec, 0)
        self.assertIn("reconciliation", agg)
        rec = agg["reconciliation"]
        for key in ("consensus", "pass_specific", "severity_inconsistent",
                    "location_inconsistent", "likely_false_positive",
                    "consensus_count", "pass_specific_count"):
            self.assertIn(key, rec)

    def test_consensus_finding_in_aggregate(self):
        finding_response = _response("FINDINGS", severity="major")
        ec, agg, _ = self._run_multi([finding_response, finding_response])
        self.assertEqual(ec, 0)
        self.assertEqual(agg["reconciliation"]["consensus_count"], 1)
        self.assertEqual(agg["reconciliation"]["pass_specific_count"], 0)


# ---------------------------------------------------------------------------
# MultiPassCliAudit — verify append_audit_log is called with D12 fields
# ---------------------------------------------------------------------------

class MultiPassCliAudit(unittest.TestCase):
    def _run_audit(self, pass_responses, extra_args=None):
        responses = list(pass_responses)
        call_count = [0]
        captured = []

        def _mock_stream(*a, **kw):
            idx = call_count[0]
            call_count[0] += 1
            resp = responses[idx] if idx < len(responses) else "STATUS: PASS\nSUMMARY: ok"
            if isinstance(resp, Exception):
                raise resp
            return resp

        with tempfile.TemporaryDirectory() as tmp:
            out_path = os.path.join(tmp, "result.json")
            packet_file = os.path.join(tmp, "packet.md")
            with open(packet_file, "w") as f:
                f.write(_packet())

            n = len(pass_responses)
            argv = ([_SCRIPT, packet_file, "--out", out_path, "--passes", str(n)]
                    + (extra_args or []))
            with patch("sys.argv", argv), \
                 patch.object(_mod.gemma_local, "ensure_model_available"), \
                 patch.object(_mod.gemma_local, "stream_chat", side_effect=_mock_stream), \
                 patch.object(_mod.gemma_local, "append_audit_log",
                              side_effect=lambda r: captured.append(r)):
                _mod.main()

        return captured

    def test_multipass_pass_emits_one_record_with_d12_fields(self):
        records = self._run_audit([
            _mod.gemma_local.StreamChatResult(
                content="STATUS: PASS\nSUMMARY: ok",
                usage=_mod.gemma_local.StreamUsage(response_tokens=5),
            ),
            _mod.gemma_local.StreamChatResult(
                content="STATUS: PASS\nSUMMARY: ok",
                usage=_mod.gemma_local.StreamUsage(response_tokens=7),
            ),
            _mod.gemma_local.StreamChatResult(
                content="STATUS: PASS\nSUMMARY: ok",
                usage=_mod.gemma_local.StreamUsage(response_tokens=11),
            ),
        ])
        self.assertEqual(len(records), 1)
        r = records[0]
        self.assertEqual(r["role"], "reviewer")
        self.assertEqual(r["outcome"], "PASS")
        self.assertEqual(r["passes_run"], 3)
        self.assertEqual(r["passes_succeeded"], 3)
        self.assertFalse(r["degraded"])
        self.assertIn("consensus_count", r)
        self.assertIn("pass_specific_count", r)
        self.assertIn("severity_inconsistent_count", r)
        self.assertIn("likely_false_positive_count", r)
        self.assertEqual(r["response_tokens"], 23)
        self.assertGreater(r["packet_tokens_est"], 0)

    def test_degraded_run_reflected_in_audit_record(self):
        records = self._run_audit([
            _mod.gemma_local.StreamChatResult(
                content="STATUS: PASS\nSUMMARY: ok",
                usage=_mod.gemma_local.StreamUsage(response_tokens=3),
            ),
            RuntimeError("fail"),
            _mod.gemma_local.StreamChatResult(
                content="STATUS: PASS\nSUMMARY: ok",
                usage=_mod.gemma_local.StreamUsage(response_tokens=4),
            ),
        ])
        self.assertEqual(len(records), 1)
        r = records[0]
        self.assertTrue(r["degraded"])
        self.assertEqual(r["passes_succeeded"], 2)
        self.assertEqual(r["passes_run"], 3)

    def test_multipass_partial_usage_keeps_response_tokens_null(self):
        records = self._run_audit([
            _mod.gemma_local.StreamChatResult(
                content="STATUS: PASS\nSUMMARY: ok",
                usage=_mod.gemma_local.StreamUsage(response_tokens=3),
            ),
            _mod.gemma_local.StreamChatResult(
                content="STATUS: PASS\nSUMMARY: ok",
                usage=_mod.gemma_local.StreamUsage(response_tokens=None),
            ),
        ])
        self.assertEqual(len(records), 1)
        self.assertIsNone(records[0]["response_tokens"])

    def test_quorum_failure_emits_no_audit_record(self):
        records = self._run_audit([
            "STATUS: PASS\nSUMMARY: ok",
            RuntimeError("fail"),
            RuntimeError("fail"),
        ])
        self.assertEqual(records, [])

    def test_consensus_count_in_audit_record(self):
        finding_response = _response("FINDINGS", severity="major")
        records = self._run_audit([finding_response, finding_response])
        self.assertEqual(len(records), 1)
        self.assertEqual(records[0]["consensus_count"], 1)
        self.assertEqual(records[0]["pass_specific_count"], 0)

    def test_passes_1_audit_has_no_d12_fields(self):
        captured = []
        with tempfile.TemporaryDirectory() as tmp:
            out_path = os.path.join(tmp, "result.json")
            packet_file = os.path.join(tmp, "packet.md")
            with open(packet_file, "w") as f:
                f.write(_packet())
            argv = [_SCRIPT, packet_file, "--out", out_path, "--passes", "1"]
            with patch("sys.argv", argv), \
                 patch.object(_mod.gemma_local, "ensure_model_available"), \
                 patch.object(_mod.gemma_local, "stream_chat",
                              return_value="STATUS: PASS\nSUMMARY: ok"), \
                 patch.object(_mod.gemma_local, "append_audit_log",
                              side_effect=lambda r: captured.append(r)):
                _mod.main()
        self.assertEqual(len(captured), 1)
        r = captured[0]
        self.assertNotIn("passes_run", r)
        self.assertNotIn("consensus_count", r)


if __name__ == "__main__":
    unittest.main(verbosity=2)
