#!/usr/bin/env python3
"""Operational wiring tests for the push-review make target and workflow."""

from pathlib import Path
import unittest


REPO_ROOT = Path(__file__).resolve().parents[1]
MAKEFILE = REPO_ROOT / "Makefile"
WORKFLOW = REPO_ROOT / ".github" / "workflows" / "push-review.yml"


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


if __name__ == "__main__":
    unittest.main()
