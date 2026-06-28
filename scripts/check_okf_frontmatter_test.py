#!/usr/bin/env python3
"""Tests for check_okf_frontmatter.py — adapted for fenix doc_type vocabulary.

Key differences from DubBridge source:
  - YAML key is 'doc_type' (not 'type')
  - ADRs in docs/decisions/, plans in docs/plans/
  - ADR status parsed from '## Status\\n\\n`<value>`' section (not bullet)
  - No governed_by / adr_exists checks (removed in fenix adaptation)
  - Python 3.9 compatible (no match, no X | Y unions)
"""
import sys
import textwrap
import unittest
from pathlib import Path
from unittest.mock import MagicMock, patch

import importlib.util

_spec = importlib.util.spec_from_file_location(
    "check_okf_frontmatter",
    Path(__file__).parent / "check_okf_frontmatter.py",
)
_mod = importlib.util.module_from_spec(_spec)  # type: ignore[arg-type]
_spec.loader.exec_module(_mod)  # type: ignore[union-attr]

parse_frontmatter = _mod.parse_frontmatter
extract_prose_adr_status = _mod.extract_prose_adr_status
doc_type_matches_location = _mod.doc_type_matches_location
should_skip = _mod.should_skip
validate = _mod.validate
REPO_ROOT = _mod.REPO_ROOT


def _run_validate(rel, text):
    """Run validate() against a single fake file."""
    p = MagicMock(spec=Path)
    p.read_text.return_value = text
    p.relative_to.return_value = Path(rel)
    with patch.object(_mod, "_rel", return_value=rel):
        return validate([p])


def _make_adr(fm_status, prose_status):
    return textwrap.dedent("""\
        ---
        doc_type: adr
        id: ADR-999
        title: "Test ADR"
        status: {fm}
        ---
        # ADR-999

        ## Status

        `{prose}`

        ## Context
        Some context.
    """).format(fm=fm_status, prose=prose_status)


# ---------------------------------------------------------------------------
# parse_frontmatter
# ---------------------------------------------------------------------------

class TestParseFrontmatter(unittest.TestCase):
    def test_valid_block(self):
        text = "---\ndoc_type: adr\nstatus: accepted\n---\n# body"
        fm = parse_frontmatter(text)
        self.assertEqual(fm, {"doc_type": "adr", "status": "accepted"})

    def test_missing_block(self):
        self.assertIsNone(parse_frontmatter("# no frontmatter"))

    def test_malformed_yaml(self):
        self.assertIsNone(parse_frontmatter("---\n: bad: yaml: [\n---\n"))

    def test_non_dict_yaml(self):
        self.assertIsNone(parse_frontmatter("---\n- list item\n---\n"))

    def test_unclosed_block(self):
        self.assertIsNone(parse_frontmatter("---\ndoc_type: adr\n"))


# ---------------------------------------------------------------------------
# extract_prose_adr_status
# ---------------------------------------------------------------------------

class TestExtractProseADRStatus(unittest.TestCase):
    def test_backtick_format(self):
        text = "## Status\n\n`accepted`\n"
        self.assertEqual(extract_prose_adr_status(text), "accepted")

    def test_plain_format(self):
        text = "## Status\n\naccepted\n"
        self.assertEqual(extract_prose_adr_status(text), "accepted")

    def test_proposed(self):
        text = "## Status\n\n`proposed`\n"
        self.assertEqual(extract_prose_adr_status(text), "proposed")

    def test_missing_section(self):
        self.assertIsNone(extract_prose_adr_status("No status section"))


# ---------------------------------------------------------------------------
# doc_type_matches_location
# ---------------------------------------------------------------------------

class TestDocTypeMatchesLocation(unittest.TestCase):
    def test_adr_match(self):
        self.assertTrue(doc_type_matches_location("adr", "docs/decisions/ADR-006-foo.md"))

    def test_plan_match(self):
        self.assertTrue(doc_type_matches_location("plan", "docs/plans/my-plan.md"))

    def test_roadmap_match(self):
        self.assertTrue(doc_type_matches_location("roadmap", "docs/plans/roadmap.md"))

    def test_plan_does_not_match_roadmap(self):
        self.assertFalse(doc_type_matches_location("plan", "docs/plans/roadmap.md"))

    def test_task_match(self):
        self.assertTrue(doc_type_matches_location("task", "docs/tasks/task_001.md"))

    def test_policy_match(self):
        self.assertTrue(doc_type_matches_location("policy", "docs/policies/RRI_POLICY.md"))

    def test_wrong_location(self):
        self.assertFalse(doc_type_matches_location("adr", "docs/plans/my-plan.md"))

    def test_unknown_doc_type(self):
        self.assertFalse(doc_type_matches_location("bogus", "docs/decisions/ADR-001.md"))


# ---------------------------------------------------------------------------
# should_skip
# ---------------------------------------------------------------------------

class TestShouldSkip(unittest.TestCase):
    def test_template(self):
        self.assertTrue(should_skip("docs/TEMPLATE.md"))

    def test_index_readme(self):
        self.assertTrue(should_skip("docs/decisions/README.md"))

    def test_tasks_readme(self):
        self.assertTrue(should_skip("docs/tasks/README.md"))

    def test_normal_adr_not_skipped(self):
        self.assertFalse(should_skip("docs/decisions/ADR-006-complexity-gate.md"))

    def test_normal_task_not_skipped(self):
        self.assertFalse(should_skip("docs/tasks/task_paw_b1.md"))


# ---------------------------------------------------------------------------
# HP-1 — valid ADR with matching status
# ---------------------------------------------------------------------------

class TestHP1ValidADR(unittest.TestCase):
    def test_passes_when_status_matches(self):
        text = _make_adr("accepted", "accepted")
        errors = _run_validate("docs/decisions/ADR-999-test.md", text)
        self.assertEqual(errors, [])

    def test_passes_when_no_prose_status_section(self):
        text = "---\ndoc_type: adr\nstatus: accepted\n---\n# No status section\n"
        errors = _run_validate("docs/decisions/ADR-999-test.md", text)
        self.assertEqual(errors, [])


# ---------------------------------------------------------------------------
# HP-2 — valid task file
# ---------------------------------------------------------------------------

class TestHP2ValidTask(unittest.TestCase):
    def test_passes(self):
        text = "---\ndoc_type: task\nid: PAW-B1\ntitle: test\nstatus: pending\n---\n# Task\n"
        errors = _run_validate("docs/tasks/task_paw_b1.md", text)
        self.assertEqual(errors, [])

    def test_passes_plan(self):
        text = "---\ndoc_type: plan\ntitle: My Plan\nstatus: proposed\n---\n# Plan\n"
        errors = _run_validate("docs/plans/my-plan.md", text)
        self.assertEqual(errors, [])


# ---------------------------------------------------------------------------
# EC-1 — ADR status drift
# ---------------------------------------------------------------------------

class TestEC1ADRStatusDrift(unittest.TestCase):
    def test_fails_when_status_mismatch(self):
        text = _make_adr("proposed", "accepted")
        errors = _run_validate("docs/decisions/ADR-999-test.md", text)
        self.assertEqual(len(errors), 1)
        self.assertIn("proposed", errors[0])
        self.assertIn("accepted", errors[0])


# ---------------------------------------------------------------------------
# EC-2 — unknown or misplaced doc_type
# ---------------------------------------------------------------------------

class TestEC2BadDocType(unittest.TestCase):
    def test_unknown_doc_type(self):
        text = "---\ndoc_type: bogus\n---\n# body"
        errors = _run_validate("docs/decisions/ADR-999-test.md", text)
        self.assertEqual(len(errors), 1)
        self.assertIn("closed vocabulary", errors[0])

    def test_doc_type_wrong_location(self):
        text = "---\ndoc_type: plan\ntitle: x\nstatus: proposed\n---\n# body"
        errors = _run_validate("docs/decisions/ADR-999-test.md", text)
        self.assertEqual(len(errors), 1)
        self.assertIn("does not match file location", errors[0])

    def test_missing_doc_type_key(self):
        text = "---\ntitle: something\n---\n# body"
        errors = _run_validate("docs/decisions/ADR-999-test.md", text)
        self.assertEqual(len(errors), 1)
        self.assertIn("closed vocabulary", errors[0])


# ---------------------------------------------------------------------------
# EC-3 — missing frontmatter on in-scope file
# ---------------------------------------------------------------------------

class TestEC3MissingFrontmatter(unittest.TestCase):
    def test_no_frontmatter_fails(self):
        text = "# No frontmatter here\n"
        errors = _run_validate("docs/decisions/ADR-999-test.md", text)
        self.assertEqual(len(errors), 1)
        self.assertIn("missing or malformed", errors[0])


# ---------------------------------------------------------------------------
# EC-4 — report-only mode does not exit 1
# ---------------------------------------------------------------------------

class TestEC4ReportOnly(unittest.TestCase):
    def test_report_only_exits_zero_with_errors(self):
        with patch.object(_mod, "collect_in_scope_files", return_value=[]), \
             patch.object(_mod, "validate", return_value=["some error"]), \
             patch("sys.argv", ["check_okf_frontmatter.py", "--report-only"]):
            rc = _mod.main()
        self.assertEqual(rc, 0)

    def test_no_report_only_exits_one_with_errors(self):
        with patch.object(_mod, "collect_in_scope_files", return_value=[]), \
             patch.object(_mod, "validate", return_value=["some error"]), \
             patch("sys.argv", ["check_okf_frontmatter.py"]):
            rc = _mod.main()
        self.assertEqual(rc, 1)


# ---------------------------------------------------------------------------
# collect_in_scope_files smoke test
# ---------------------------------------------------------------------------

class TestCollectInScopeFiles(unittest.TestCase):
    def test_returns_list_of_paths(self):
        files = _mod.collect_in_scope_files()
        self.assertIsInstance(files, list)
        rels = [_mod._rel(f) for f in files]
        self.assertTrue(any(r.startswith("docs/decisions/ADR-") for r in rels))
        self.assertNotIn("docs/decisions/README.md", rels)

    def test_tasks_included(self):
        files = _mod.collect_in_scope_files()
        rels = [_mod._rel(f) for f in files]
        self.assertTrue(any(r.startswith("docs/tasks/") for r in rels))


# ---------------------------------------------------------------------------
# main() happy path
# ---------------------------------------------------------------------------

class TestMain(unittest.TestCase):
    def test_main_pass(self):
        with patch.object(_mod, "collect_in_scope_files", return_value=[]), \
             patch.object(_mod, "validate", return_value=[]), \
             patch("sys.argv", ["check_okf_frontmatter.py"]):
            rc = _mod.main()
        self.assertEqual(rc, 0)


if __name__ == "__main__":
    unittest.main()
