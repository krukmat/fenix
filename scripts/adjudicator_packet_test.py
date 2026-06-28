"""Unit tests for adjudicator-packet.py."""

import importlib.util
import sys
import os
import unittest

# Load module from hyphenated filename.
_spec = importlib.util.spec_from_file_location(
    "adjudicator_packet",
    os.path.join(os.path.dirname(__file__), "adjudicator-packet.py"),
)
_mod = importlib.util.module_from_spec(_spec)  # type: ignore[arg-type]
_spec.loader.exec_module(_mod)  # type: ignore[union-attr]

should_adjudicate = _mod.should_adjudicate
build_adjudicator_packet = _mod.build_adjudicator_packet
_assert_packet_isolation = _mod._assert_packet_isolation
ALLOWED_PACKET_SECTIONS = _mod.ALLOWED_PACKET_SECTIONS
BANDS_MED_HIGH_OR_ABOVE = _mod.BANDS_MED_HIGH_OR_ABOVE


class TestShouldAdjudicate(unittest.TestCase):

    def _agg(self, findings=None, recon=None):
        return {
            "findings": findings or [],
            "reconciliation": recon or {},
        }

    # --- True paths ---

    def test_gemma_blocked_returns_true(self):
        self.assertTrue(should_adjudicate(self._agg(), band="Low", gemma_blocked=True))

    def test_band_med_high_returns_true(self):
        self.assertTrue(should_adjudicate(self._agg(), band="Med-high"))

    def test_band_complex_returns_true(self):
        self.assertTrue(should_adjudicate(self._agg(), band="Complex"))

    def test_band_high_returns_true(self):
        self.assertTrue(should_adjudicate(self._agg(), band="High"))

    def test_band_very_high_returns_true(self):
        self.assertTrue(should_adjudicate(self._agg(), band="Very high"))

    def test_band_excessive_returns_true(self):
        self.assertTrue(should_adjudicate(self._agg(), band="Excessive"))

    def test_blocking_finding_returns_true(self):
        agg = self._agg(findings=[{"severity": "blocking", "msg": "null deref"}])
        self.assertTrue(should_adjudicate(agg, band="Low"))

    def test_major_finding_returns_true(self):
        agg = self._agg(findings=[{"severity": "major", "msg": "data race"}])
        self.assertTrue(should_adjudicate(agg, band="Moderate"))

    def test_severity_inconsistent_returns_true(self):
        agg = self._agg(recon={"severity_inconsistent": 1, "location_inconsistent": 0})
        self.assertTrue(should_adjudicate(agg, band="Low"))

    def test_location_inconsistent_returns_true(self):
        agg = self._agg(recon={"severity_inconsistent": 0, "location_inconsistent": 2})
        self.assertTrue(should_adjudicate(agg, band="Low"))

    # --- False paths ---

    def test_low_band_no_issues_returns_false(self):
        self.assertFalse(should_adjudicate(self._agg(), band="Low"))

    def test_moderate_band_no_issues_returns_false(self):
        self.assertFalse(should_adjudicate(self._agg(), band="Moderate"))

    def test_moderate_with_minor_finding_returns_false(self):
        agg = self._agg(findings=[{"severity": "minor", "msg": "style nit"}])
        self.assertFalse(should_adjudicate(agg, band="Moderate"))

    def test_moderate_gemma_available_no_disagreement_returns_false(self):
        agg = self._agg(
            findings=[{"severity": "minor"}],
            recon={"severity_inconsistent": 0, "location_inconsistent": 0},
        )
        self.assertFalse(should_adjudicate(agg, band="Moderate", gemma_blocked=False))

    def test_missing_findings_key_returns_false(self):
        self.assertFalse(should_adjudicate({}, band="Low"))


class TestBuildAdjudicatorPacket(unittest.TestCase):

    def test_valid_packet_returned(self):
        packet = build_adjudicator_packet(
            diff="--- a/foo.go\n+++ b/foo.go",
            criteria={"max_severity": "blocking"},
            reconciled_findings=[{"severity": "minor", "msg": "nit"}],
        )
        self.assertEqual(set(packet.keys()), ALLOWED_PACKET_SECTIONS)
        self.assertIn("diff", packet)
        self.assertIn("criteria", packet)
        self.assertIn("reconciled_findings", packet)

    def test_empty_findings_valid(self):
        packet = build_adjudicator_packet(diff="", criteria={}, reconciled_findings=[])
        self.assertEqual(set(packet.keys()), ALLOWED_PACKET_SECTIONS)

    def test_disposition_divergence_none_accepted(self):
        # None is explicitly allowed
        packet = build_adjudicator_packet("diff", {}, [], disposition_divergence=None)
        self.assertIsNotNone(packet)

    def test_disposition_divergence_valid_values(self):
        for val in ("none", "partial", "full"):
            with self.subTest(val=val):
                packet = build_adjudicator_packet("d", {}, [], disposition_divergence=val)
                self.assertIsNotNone(packet)

    def test_disposition_divergence_invalid_raises(self):
        with self.assertRaises(ValueError):
            build_adjudicator_packet("d", {}, [], disposition_divergence="unknown")


class TestAssertPacketIsolation(unittest.TestCase):

    def test_valid_packet_passes(self):
        _assert_packet_isolation({"diff": "x", "criteria": {}, "reconciled_findings": []})

    def test_injected_key_raises(self):
        bad_packet = {
            "diff": "x",
            "criteria": {},
            "reconciled_findings": [],
            "extra_context": "should not be here",
        }
        with self.assertRaises(RuntimeError) as ctx:
            _assert_packet_isolation(bad_packet)
        self.assertIn("isolation violation", str(ctx.exception))
        self.assertIn("extra_context", str(ctx.exception))

    def test_multiple_injected_keys_raises(self):
        bad_packet = {"diff": "x", "criteria": {}, "reconciled_findings": [], "a": 1, "b": 2}
        with self.assertRaises(RuntimeError):
            _assert_packet_isolation(bad_packet)

    def test_empty_packet_subset_raises(self):
        # Missing keys are fine for isolation (allowlist only rejects extras)
        _assert_packet_isolation({"diff": "x"})

    def test_empty_packet_passes(self):
        _assert_packet_isolation({})


class TestAllowedSections(unittest.TestCase):

    def test_allowed_sections_value(self):
        self.assertEqual(ALLOWED_PACKET_SECTIONS, {"diff", "criteria", "reconciled_findings"})


class TestNoDubbridgeRefs(unittest.TestCase):

    def test_no_dubbridge_in_module_source(self):
        path = os.path.join(os.path.dirname(__file__), "adjudicator-packet.py")
        with open(path) as f:
            source = f.read()
        self.assertNotIn("DUBBRIDGE", source)
        self.assertNotIn("dubbridge", source)


if __name__ == "__main__":
    unittest.main()
