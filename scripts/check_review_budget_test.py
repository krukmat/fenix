#!/usr/bin/env python3

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path
from unittest import mock


SCRIPT = Path(__file__).with_name("check-review-budget.py")
SPEC = importlib.util.spec_from_file_location("check_review_budget", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
gate = importlib.util.module_from_spec(SPEC)
sys.modules["check_review_budget"] = gate
SPEC.loader.exec_module(gate)


class DeriveBudgetTest(unittest.TestCase):
    def test_default_window_yields_reviewable_budget(self) -> None:
        # 16384 ctx − 4096 predict − 1300 overhead = 10988 / 20 ≈ 549 lines.
        with mock.patch.dict(gate.os.environ, {}, clear=True):
            self.assertEqual(gate.derive_budget(), 549)

    def test_explicit_override_wins(self) -> None:
        with mock.patch.dict(gate.os.environ, {"FENIX_REVIEW_MAX_DIFF_LINES": "200"}, clear=True):
            self.assertEqual(gate.derive_budget(), 200)

    def test_budget_tracks_larger_context_window(self) -> None:
        with mock.patch.dict(gate.os.environ, {"FENIX_REVIEW_NUM_CTX": "32768"}, clear=True):
            self.assertGreater(gate.derive_budget(), 549)

    def test_tiny_window_clamped_to_floor(self) -> None:
        with mock.patch.dict(
            gate.os.environ,
            {"FENIX_REVIEW_NUM_CTX": "4096", "FENIX_REVIEW_NUM_PREDICT": "4000"},
            clear=True,
        ):
            self.assertEqual(gate.derive_budget(), gate.MIN_DERIVED_BUDGET)

    def test_non_integer_explicit_falls_back_to_derived(self) -> None:
        with mock.patch.dict(gate.os.environ, {"FENIX_REVIEW_MAX_DIFF_LINES": "lots"}, clear=True):
            self.assertEqual(gate.derive_budget(), 549)

    def test_packet_overhead_is_env_tunable(self) -> None:
        # Lower overhead → more usable tokens → larger budget.
        with mock.patch.dict(
            gate.os.environ, {"FENIX_REVIEW_PACKET_OVERHEAD_TOKENS": "300"}, clear=True
        ):
            self.assertEqual(gate.derive_budget(), (16384 - 4096 - 300) // gate.TOKENS_PER_DIFF_LINE)

    def test_no_dubbridge_env_refs_in_script(self) -> None:
        with open(SCRIPT) as fh:
            src = fh.read()
        self.assertNotIn("DUBBRIDGE_", src)


class FindOverrideTest(unittest.TestCase):
    def test_marker_with_reason_is_extracted(self) -> None:
        text = "fix(x): mechanical rename\n\nD14-OVERRIDE: irreducible 900-line rename, see ADR-XX"
        self.assertEqual(gate.find_override(text), "irreducible 900-line rename, see ADR-XX")

    def test_marker_without_reason_does_not_satisfy(self) -> None:
        self.assertIsNone(gate.find_override("D14-OVERRIDE:   "))

    def test_no_marker_returns_none(self) -> None:
        self.assertIsNone(gate.find_override("ordinary commit body"))


class RenderReportTest(unittest.TestCase):
    def test_hp_under_budget_passes(self) -> None:
        report, code = gate.render_report(count=100, budget=549, override_reason=None)
        self.assertEqual(code, 0)
        self.assertIn("passed", report)

    def test_hp_exactly_at_budget_passes(self) -> None:
        report, code = gate.render_report(count=549, budget=549, override_reason=None)
        self.assertEqual(code, 0)

    def test_ec_over_budget_no_override_fails_closed(self) -> None:
        report, code = gate.render_report(count=900, budget=549, override_reason=None)
        self.assertEqual(code, 1)
        self.assertIn("Split the change", report)
        self.assertIn("D14-OVERRIDE", report)

    def test_ec_over_budget_with_override_passes_and_logs(self) -> None:
        report, code = gate.render_report(count=900, budget=549, override_reason="irreducible rename")
        self.assertEqual(code, 0)
        self.assertIn("irreducible rename", report)
        self.assertIn("non-Gemma (D14) reviewer", report)


class MainIntegrationTest(unittest.TestCase):
    def test_main_over_budget_without_override_exits_1(self) -> None:
        with mock.patch.object(gate, "count_reviewable_lines", return_value=900), \
             mock.patch.object(gate, "derive_budget", return_value=549), \
             mock.patch.object(gate._maint, "discover_base", return_value=None), \
             mock.patch.object(gate._maint, "changed_files", return_value=["internal/domain/policy/a.go"]), \
             mock.patch.object(gate, "_override_text", return_value="no marker here"):
            self.assertEqual(gate.main([]), 1)

    def test_main_over_budget_with_override_exits_0(self) -> None:
        with mock.patch.object(gate, "count_reviewable_lines", return_value=900), \
             mock.patch.object(gate, "derive_budget", return_value=549), \
             mock.patch.object(gate._maint, "discover_base", return_value=None), \
             mock.patch.object(gate._maint, "changed_files", return_value=["internal/domain/policy/a.go"]), \
             mock.patch.object(gate, "_override_text", return_value="body\nD14-OVERRIDE: irreducible"):
            self.assertEqual(gate.main([]), 0)

    def test_main_under_budget_exits_0(self) -> None:
        with mock.patch.object(gate, "count_reviewable_lines", return_value=10), \
             mock.patch.object(gate, "derive_budget", return_value=549), \
             mock.patch.object(gate._maint, "discover_base", return_value=None), \
             mock.patch.object(gate._maint, "changed_files", return_value=["internal/domain/policy/a.go"]):
            self.assertEqual(gate.main([]), 0)


if __name__ == "__main__":
    unittest.main()
