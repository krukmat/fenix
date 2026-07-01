#!/usr/bin/env python3
"""Operational wiring tests for the push-review make target and workflow."""

import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


REPO_ROOT = Path(__file__).resolve().parents[1]
MAKEFILE = REPO_ROOT / "Makefile"
WORKFLOW = REPO_ROOT / ".github" / "workflows" / "push-review.yml"
PUSH_REVIEW_COMMIT = REPO_ROOT / "scripts" / "push_review_commit.py"

_SPEC = importlib.util.spec_from_file_location("push_review_commit", PUSH_REVIEW_COMMIT)
_MOD = importlib.util.module_from_spec(_SPEC)
_SPEC.loader.exec_module(_MOD)


class PushReviewOpsWiring(unittest.TestCase):
    def test_make_target_exists_and_is_skippable(self):
        text = MAKEFILE.read_text(encoding="utf-8")
        self.assertIn("qa-gemma-push-review:", text)
        self.assertIn("FENIX_SKIP_GEMMA_PUSH_REVIEW", text)
        self.assertIn("[gemma-push-review] skipped", text)

    def test_make_target_maps_env_to_cli_flags(self):
        text = MAKEFILE.read_text(encoding="utf-8")
        self.assertIn("python3 scripts/gemma-push-review.py", text)
        self.assertIn("FENIX_PUSH_REVIEW_RUN_ID", text)
        self.assertIn("--run-id", text)
        self.assertIn("FENIX_PUSH_REVIEW_WORKFLOW", text)
        self.assertIn("--workflow", text)
        self.assertIn("FENIX_PUSH_REVIEW_BRANCH", text)
        self.assertIn("--branch", text)
        self.assertIn("FENIX_PUSH_REVIEW_DRY_RUN", text)
        self.assertIn("--dry-run", text)
        self.assertIn("FENIX_PUSH_REVIEW_COLLECT_ONLY", text)
        self.assertIn("--collect-only", text)

    def test_workflow_is_post_pipeline_self_hosted_and_advisory(self):
        text = WORKFLOW.read_text(encoding="utf-8")
        self.assertIn("workflow_run:", text)
        self.assertIn('workflows: ["ci"]', text)
        self.assertIn("types: [completed]", text)
        self.assertIn("self-hosted", text)
        self.assertNotIn("continue-on-error: true", text)
        self.assertIn("make qa-gemma-push-review", text)
        self.assertIn("FENIX_PUSH_REVIEW_EVENT_PATH", text)
        self.assertIn("FENIX_PUSH_REVIEW_RUN_ID", text)
        self.assertIn("FENIX_PUSH_REVIEW_WORKFLOW", text)
        self.assertIn("FENIX_PUSH_REVIEW_BRANCH", text)
        self.assertIn("FENIX_PUSH_REVIEW_AFTER", text)
        self.assertIn("FENIX_PUSH_REVIEW_OUT_DIR", text)
        self.assertIn("${{ github.run_id }}", text)
        self.assertIn("actions/upload-artifact@v4", text)
        self.assertIn("name: push-review-${{ github.event.workflow_run.head_sha }}-${{ github.run_id }}", text)
        self.assertIn("path: logs/gemma-push-review/${{ github.event.workflow_run.head_sha }}/${{ github.run_id }}/", text)
        self.assertNotIn("docs/reports/push-review/", text)
        self.assertIn("if: always()", text)
        self.assertIn("steps.push_review.outcome", text)
        self.assertIn("blocked/degraded result or operational failure", text)
        self.assertIn("Primary CI remains authoritative.", text)


class PushReviewCommitBehavior(unittest.TestCase):
    def test_build_commit_message_uses_push_review_prefix(self):
        message = _MOD.build_commit_message("abcdef1234567890", "2026-07-01")
        self.assertEqual(
            message,
            "[push-review] report abcdef1 + daily 2026-07-01 entry [skip ci]",
        )

    def test_main_invokes_git_commit_with_push_review_prefix(self):
        recorded = []

        def fake_run(args, capture_output=False, text=False, check=False, **kwargs):
            recorded.append(args)
            if args[:3] == ["git", "rev-parse", "--abbrev-ref"]:
                return subprocess.CompletedProcess(args, 0, stdout="main\n", stderr="")
            if args[:3] == ["git", "diff", "--cached"]:
                return subprocess.CompletedProcess(args, 1, stdout="", stderr="")
            return subprocess.CompletedProcess(args, 0, stdout="", stderr="")

        with tempfile.TemporaryDirectory() as tmp:
            out_dir = Path(tmp) / "out"
            reports = out_dir / "reports"
            reports.mkdir(parents=True)
            (reports / "2026-07-01-abcdef1.md").write_text("report", encoding="utf-8")
            (out_dir / "aggregate.json").write_text(
                '{"status":"pass","audit":{"quorum":"met","passes_succeeded":1,"passes_run":1},"candidates":[],"pipeline":{"conclusion":"success"}}',
                encoding="utf-8",
            )

            repo = Path(tmp) / "repo"
            repo.mkdir()
            prev = os.getcwd()
            os.chdir(repo)
            try:
                with patch.object(_MOD, "today_utc", return_value="2026-07-01"), \
                     patch.object(_MOD.subprocess, "run", side_effect=fake_run), \
                     patch("sys.argv", ["push_review_commit.py", str(out_dir), "abcdef1234567890", "12345"]):
                    _MOD.main()
            finally:
                os.chdir(prev)

        self.assertIn(
            ["git", "commit", "-m", "[push-review] report abcdef1 + daily 2026-07-01 entry [skip ci]"],
            recorded,
        )


if __name__ == "__main__":
    unittest.main()
