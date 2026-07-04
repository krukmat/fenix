#!/usr/bin/env python3
# Unit tests for the RRI calculator. [rri-calculator-script T1]
# Run: python3 scripts/rri_test.py   (or: python3 -m unittest scripts/rri_test.py)
import json
import os
import subprocess
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import rri  # noqa: E402

SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "rri.py")


def base_of(**kw):
    """Evaluate with safe defaults and return the base value (no git, no rubric)."""
    defaults = dict(f_override=0, d=0, k=0, p=0, t=0, a=0, x=0)
    defaults.update(kw)
    return rri.evaluate(**defaults)["base"]


class CcMapping(unittest.TestCase):
    def test_boundaries(self):
        cases = {5: 0, 6: 1, 10: 1, 11: 2, 20: 2, 21: 3, 30: 3, 31: 4, 50: 4, 51: 5}
        for raw, score in cases.items():
            self.assertEqual(rri.cc_to_score(raw), score, f"cc {raw}")


class FMapping(unittest.TestCase):
    def test_boundaries(self):
        cases = {1: 0, 2: 1, 3: 2, 5: 2, 6: 3, 10: 3, 11: 4, 20: 4, 21: 5, 99: 5}
        for n, score in cases.items():
            self.assertEqual(rri.count_to_f(n), score, f"count {n}")


class BaseFormula(unittest.TestCase):
    def test_all_zero(self):
        self.assertEqual(base_of(c_score=0), 0)

    def test_all_five(self):
        # 100 * (5 * sum(weights=1.00) / 5) = 100.
        self.assertEqual(base_of(c_score=5, f_override=5, d=5, k=5, p=5, t=5, a=5, x=5), 100)

    def test_t1_vector(self):
        # C1 F1 D1 T2 A0 K1 P0 X2 -> 0.99 / 5 * 100 = 19.8 -> 20.
        self.assertEqual(
            base_of(c_score=1, f_override=1, d=1, k=1, p=0, t=2, a=0, x=2), 20)

    def test_paw_a1_rri(self):
        # PAW-A1 scored: C2 F2 D1 T1 A1 K1 P0 X1 -> RRI 24 (Low).
        r = rri.evaluate(c_score=2, f_override=2, d=1, k=1, p=0, t=1, a=1, x=1)
        self.assertEqual(r["final"], 24)
        self.assertEqual(r["band"]["label"], "Low")


class AnchorRubric(unittest.TestCase):
    def test_floor_raises_audit(self):
        # internal/domain/audit has D=4, P=5, K=4 floor.
        r = rri.evaluate(c_score=0, touches=["internal/domain/audit/service.go"],
                         d=1, k=1, p=1, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertEqual(r["scores"]["D"], 4)
        self.assertEqual(r["scores"]["P"], 5)
        self.assertEqual(r["scores"]["K"], 4)

    def test_floor_raises_auth(self):
        # internal/domain/auth has D=3, P=4, K=3 floor.
        r = rri.evaluate(c_score=0, touches=["internal/domain/auth/jwt.go"],
                         d=1, k=1, p=1, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertEqual(r["scores"]["D"], 3)
        self.assertEqual(r["scores"]["P"], 4)
        self.assertEqual(r["scores"]["K"], 3)

    def test_floor_only_raises_never_lowers(self):
        # Agent D=4 against a floor-2 path (internal/domain/copilot) -> kept at 4.
        r = rri.evaluate(c_score=0, touches=["internal/domain/copilot/service.go"],
                         d=4, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertEqual(r["scores"]["D"], 4)

    def test_migrations_p_five(self):
        # migrations have P=5 floor.
        r = rri.evaluate(c_score=0, touches=["infra/migrations/001_init.up.sql"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertEqual(r["scores"]["P"], 5)

    def test_docs_floor_zero(self):
        r = rri.evaluate(c_score=0, touches=["docs/architecture.md"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertEqual(r["scores"]["P"], 0)
        self.assertEqual(r["scores"]["D"], 0)

    def test_scripts_floor_low(self):
        r = rri.evaluate(c_score=0, touches=["scripts/rri.py"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertEqual(r["scores"]["P"], 0)
        self.assertEqual(r["scores"]["D"], 1)

    def test_unmatched_path_advisory(self):
        r = rri.evaluate(c_score=0, touches=["cmd/serve/main.go"],
                         d=1, k=1, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertTrue(any("no anchor-rubric match" in n for n in r["advisories"]))


class Penalties(unittest.TestCase):
    def test_many_files_auto(self):
        r = rri.evaluate(c_score=0, f_override=4, d=0, k=0, p=0, t=0, a=0, x=0)
        self.assertIn("many_files", r["penalties"])
        self.assertEqual(r["penalties"]["many_files"][0], 8)

    def test_complex_and_domain_auto(self):
        r = rri.evaluate(c_score=4, f_override=0, d=3, k=0, p=0, t=0, a=0, x=0)
        self.assertIn("complex_and_domain", r["penalties"])

    def test_no_tests_high_impact_auto(self):
        r = rri.evaluate(c_score=0, f_override=0, d=0, k=0, p=4, t=4, a=0, x=0)
        self.assertIn("no_tests_high_impact", r["penalties"])

    def test_auth_security_auto_from_rubric(self):
        r = rri.evaluate(c_score=0, touches=["internal/domain/auth/service.go"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertIn("auth_security", r["penalties"])

    def test_manual_penalty_dedup(self):
        # Manual auth_security plus rubric auto -> applied once.
        r = rri.evaluate(c_score=0, touches=["internal/domain/auth/service.go"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         manual_penalties=["auth_security"],
                         profile=rri.PROFILES["fenix"])
        self.assertEqual(r["penalties"]["auth_security"][1],
                         "anchor-rubric P floor >= 4 (auth/audit/rights/secrets)")

    def test_manual_only_penalties(self):
        r = rri.evaluate(c_score=0, f_override=0, d=0, k=0, p=0, t=0, a=0, x=0,
                         manual_penalties=["arch_decision", "no_verification"])
        self.assertEqual(r["penalty_total"], 12 + 15)


class CriticalitySignal(unittest.TestCase):
    def test_agent_supplied_p_triggers_suggestion(self):
        r = rri.evaluate(c_score=0, f_override=0, d=0, k=0, p=4, t=0, a=0, x=0)
        self.assertTrue(r["criticality_suggested"])
        self.assertIn("agent-supplied P=4", r["criticality_reason"])

    def test_rubric_floor_triggers_suggestion(self):
        r = rri.evaluate(c_score=0, touches=["internal/domain/auth/jwt.go"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertTrue(r["criticality_suggested"])
        self.assertEqual(r["criticality_reason"],
                         "anchor-rubric P floor >= 4 (auth/audit/rights/secrets)")

    def test_below_threshold_no_suggestion(self):
        r = rri.evaluate(c_score=0, f_override=0, d=0, k=0, p=3, t=0, a=0, x=0)
        self.assertFalse(r["criticality_suggested"])
        self.assertIn("P < 4", r["criticality_reason"])

    def test_json_includes_criticality_fields(self):
        r = rri.evaluate(c_score=0, f_override=0, d=0, k=0, p=4, t=0, a=0, x=0)
        payload = json.loads(rri.render_json(r))
        self.assertTrue(payload["criticality_suggested"])
        self.assertIn("agent-supplied P=4", payload["criticality_reason"])

    def test_markdown_includes_criticality_line(self):
        r = rri.evaluate(c_score=0, f_override=0, d=0, k=0, p=4, t=0, a=0, x=0)
        md = rri.render_markdown(r)
        self.assertIn("**Criticality suggested:** yes", md)


class Bands(unittest.TestCase):
    def test_band_boundaries(self):
        self.assertEqual(rri.resolve_band(25)["label"], "Low")
        self.assertEqual(rri.resolve_band(26)["label"], "Moderate")
        self.assertEqual(rri.resolve_band(55)["label"], "Med-high")
        self.assertEqual(rri.resolve_band(56)["label"], "Complex")
        self.assertEqual(rri.resolve_band(70)["label"], "Complex")
        self.assertEqual(rri.resolve_band(71)["label"], "High")
        self.assertEqual(rri.resolve_band(100)["label"], "Very high")
        self.assertEqual(rri.resolve_band(101)["label"], "Excessive")

    def test_crosswalk_fields(self):
        b = rri.resolve_band(30)
        self.assertEqual(b["effort"], "M")
        self.assertEqual(b["thinking"], "Off")

    def test_low_band_model_tiers(self):
        b = rri.resolve_band(25)
        self.assertEqual(b["economy"], "claude-sonnet-4-6")
        self.assertEqual(b["balanced"], "claude-sonnet-4-6")
        self.assertIn("Execute directly", b["gate"])

    def test_high_band_uses_opus(self):
        b = rri.resolve_band(80)
        self.assertEqual(b["economy"], "claude-opus-4-8")
        self.assertEqual(b["balanced"], "claude-opus-4-8")


class Decomposition(unittest.TestCase):
    def test_complex_and_domain_triggers(self):
        r = rri.evaluate(c_score=4, f_override=0, d=3, k=0, p=0, t=0, a=0, x=0)
        self.assertTrue(any("C >= 4 and D >= 3" in t for t in r["triggers"]))

    def test_high_rri_triggers(self):
        r = rri.evaluate(c_score=5, f_override=5, d=5, k=5, p=5, t=5, a=5, x=5)
        self.assertTrue(any("RRI > 70" in t for t in r["triggers"]))

    def test_no_trigger_low(self):
        r = rri.evaluate(c_score=1, f_override=1, d=1, k=1, p=0, t=2, a=0, x=2)
        self.assertEqual(r["triggers"], [])


class LowConfidence(unittest.TestCase):
    def test_bump_marks_low_and_raises(self):
        r = rri.evaluate(c_score=2, f_override=0, d=2, k=2, p=2, t=2, a=2, x=2,
                         low_conf=["D"])
        self.assertEqual(r["scores"]["D"], 3)
        self.assertEqual(r["confidence"]["D"], "Low")

    def test_bump_capped_at_five(self):
        r = rri.evaluate(c_score=0, f_override=0, d=5, k=0, p=0, t=0, a=0, x=0,
                         low_conf=["D"])
        self.assertEqual(r["scores"]["D"], 5)


class CliBehavior(unittest.TestCase):
    """Exit-code and CLI-parsing behavior via subprocess (EC-1, EC-2, EC-3, EC-6)."""

    def run_cli(self, *args):
        return subprocess.run([sys.executable, SCRIPT, *args],
                              capture_output=True, text=True)

    def test_ok_markdown(self):
        r = self.run_cli("--cc", "12", "--F", "1", "--D", "2", "--K", "2",
                         "--P", "2", "--T", "2", "--A", "0", "--X", "2")
        self.assertEqual(r.returncode, 0)
        self.assertIn("Final RRI:", r.stdout)

    def test_json_output(self):
        import json
        r = self.run_cli("--cc", "12", "--F", "1", "--D", "2", "--K", "2",
                         "--P", "2", "--T", "2", "--A", "0", "--X", "2", "--json")
        self.assertEqual(r.returncode, 0)
        data = json.loads(r.stdout)
        for key in ("variables", "base", "penalties", "final", "band", "triggers"):
            self.assertIn(key, data)

    def test_score_out_of_range(self):  # EC-1
        r = self.run_cli("--C", "6", "--F", "1", "--D", "2", "--K", "2",
                         "--P", "2", "--T", "2", "--A", "0", "--X", "2")
        self.assertNotEqual(r.returncode, 0)
        self.assertIn("0-5", r.stderr)

    def test_unknown_penalty(self):  # EC-2
        r = self.run_cli("--cc", "12", "--F", "1", "--D", "2", "--K", "2",
                         "--P", "2", "--T", "2", "--A", "0", "--X", "2",
                         "--penalty", "bogus")
        self.assertNotEqual(r.returncode, 0)

    def test_both_cc_and_C(self):  # EC-6
        r = self.run_cli("--cc", "12", "--C", "2", "--F", "1", "--D", "2",
                         "--K", "2", "--P", "2", "--T", "2", "--A", "0", "--X", "2")
        self.assertNotEqual(r.returncode, 0)

    def test_neither_cc_nor_C(self):  # EC-6
        r = self.run_cli("--F", "1", "--D", "2", "--K", "2", "--P", "2",
                         "--T", "2", "--A", "0", "--X", "2")
        self.assertNotEqual(r.returncode, 0)

    def test_cc_below_one(self):
        r = self.run_cli("--cc", "0", "--F", "1", "--D", "2", "--K", "2",
                         "--P", "2", "--T", "2", "--A", "0", "--X", "2")
        self.assertNotEqual(r.returncode, 0)


class AutoCcMeasurement(unittest.TestCase):
    """Tests for --auto-cc wiring and graceful degradation."""

    def test_no_py_files_returns_none(self):
        cc, ev = rri.measure_cc_radon(["internal/domain/auth/service.go",
                                       "infra/migrations/001.sql"])
        self.assertIsNone(cc)
        self.assertIn("no local .py", ev)

    def test_nonexistent_file_treated_as_non_py(self):
        # A .py path that does not exist on disk is skipped (not an error).
        cc, ev = rri.measure_cc_radon(["scripts/does_not_exist.py"])
        self.assertIsNone(cc)

    def test_auto_cc_fallback_marks_low_confidence(self):
        # With the python profile and no .py files, radon is never called: score
        # defaults to 0 with C added to low_conf -> C score bumped to 1. Pinning the
        # profile keeps this deterministic regardless of the host toolchain.
        r = rri.evaluate(auto_cc=True, f_override=0, d=0, k=0, p=0, t=0, a=0, x=0,
                         touches=["internal/domain/auth/service.go"],  # no .py files
                         profile=rri.PROFILES["python"])
        self.assertEqual(r["confidence"]["C"], "Low")
        # Score is 0 bumped +1 -> 1.
        self.assertEqual(r["scores"]["C"], 1)
        self.assertIn("auto-cc fallback", r["evidence"]["C"])

    def test_auto_cc_cli_flag_accepted(self):
        # CLI: --auto-cc must not crash when there are no .go files in --touches.
        r = subprocess.run(
            [sys.executable, SCRIPT,
             "--auto-cc", "--touches", "docs/architecture.md",
             "--D", "0", "--K", "0", "--P", "0", "--T", "0", "--A", "0", "--X", "0"],
            capture_output=True, text=True)
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("Final RRI:", r.stdout)

    def test_auto_cc_mutually_exclusive_with_cc(self):
        r = subprocess.run(
            [sys.executable, SCRIPT,
             "--cc", "12", "--auto-cc",
             "--D", "0", "--K", "0", "--P", "0", "--T", "0", "--A", "0", "--X", "0"],
            capture_output=True, text=True)
        self.assertNotEqual(r.returncode, 0)

    def test_auto_cc_mutually_exclusive_with_C(self):
        r = subprocess.run(
            [sys.executable, SCRIPT,
             "--C", "2", "--auto-cc",
             "--D", "0", "--K", "0", "--P", "0", "--T", "0", "--A", "0", "--X", "0"],
            capture_output=True, text=True)
        self.assertNotEqual(r.returncode, 0)


class PlatformDetection(unittest.TestCase):
    """detect_platform resolves the right profile from marker files."""

    def _detect_in(self, marker):
        with tempfile.TemporaryDirectory() as d:
            open(os.path.join(d, marker), "w").close()
            return rri.detect_platform(d).name

    def test_cargo_toml_detects_rust(self):
        self.assertEqual(self._detect_in("Cargo.toml"), "rust")

    def test_go_mod_detects_go(self):
        self.assertEqual(self._detect_in("go.mod"), "go")

    def test_package_json_detects_rn(self):
        self.assertEqual(self._detect_in("package.json"), "rn")

    def test_pyproject_detects_python(self):
        self.assertEqual(self._detect_in("pyproject.toml"), "python")

    def test_no_marker_falls_back_to_generic(self):
        with tempfile.TemporaryDirectory() as d:
            self.assertEqual(rri.detect_platform(d).name, "generic")

    def test_fenix_marker_beats_generic_go(self):
        # A dir with both go.mod and the policy file must resolve to fenix.
        with tempfile.TemporaryDirectory() as d:
            open(os.path.join(d, "go.mod"), "w").close()
            os.makedirs(os.path.join(d, "docs", "policies"))
            open(os.path.join(d, "docs", "policies", "RRI_POLICY.md"), "w").close()
            self.assertEqual(rri.detect_platform(d).name, "fenix")

    def test_resolve_platform_explicit_overrides_detection(self):
        self.assertEqual(rri.resolve_platform("go").name, "go")

    def test_resolve_platform_auto_detects_fenix(self):
        # In a temp dir with both go.mod and the policy file, auto resolves to fenix.
        with tempfile.TemporaryDirectory() as d:
            open(os.path.join(d, "go.mod"), "w").close()
            os.makedirs(os.path.join(d, "docs", "policies"))
            open(os.path.join(d, "docs", "policies", "RRI_POLICY.md"), "w").close()
            self.assertEqual(rri.detect_platform(d).name, "fenix")


class PlatformMeasurers(unittest.TestCase):
    """Every measurer degrades gracefully when its tool/files are absent."""

    def test_clippy_no_rs_files(self):
        cc, ev = rri.measure_cc_clippy(["docs/x.md", "main.py"])
        self.assertIsNone(cc)
        self.assertIn("no local .rs", ev)

    def test_gocyclo_no_go_files(self):
        cc, ev = rri.measure_cc_gocyclo(["docs/x.md"])
        self.assertIsNone(cc)
        self.assertIn("no local .go", ev)

    def test_eslint_no_js_files(self):
        cc, ev = rri.measure_cc_eslint(["docs/x.md"])
        self.assertIsNone(cc)
        self.assertIn("no local JS/TS", ev)

    def test_generic_measurer_is_noop(self):
        cc, ev = rri.measure_cc_none(["anything.go"])
        self.assertIsNone(cc)
        self.assertIn("generic platform", ev)

    def test_filter_existing_by_suffix_and_disk(self):
        # This test file exists; a fabricated .rs path does not.
        kept = rri._filter_existing([__file__, "nope.rs"], (".py",))
        self.assertEqual(kept, [__file__])

    def test_clippy_cc_parser(self):
        self.assertEqual(
            rri._parse_clippy_cc("the function has a cognitive complexity of (12/7)"), 12)
        self.assertIsNone(rri._parse_clippy_cc("unrelated message"))

    def test_eslint_cc_parser(self):
        self.assertEqual(
            rri._parse_eslint_cc("Function 'f' has a complexity of 9. Maximum allowed is 0."), 9)
        self.assertIsNone(rri._parse_eslint_cc("no number here"))


class PlatformRubric(unittest.TestCase):
    """Fenix rubric vs generic rubric, selected by profile."""

    def test_generic_auth_raises_floors(self):
        r = rri.evaluate(c_score=0, touches=["internal/auth/token.go"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["go"])
        self.assertEqual(r["scores"]["P"], 4)
        self.assertEqual(r["scores"]["D"], 4)
        self.assertEqual(r["scores"]["K"], 4)
        self.assertIn("auth_security", r["penalties"])

    def test_generic_migrations_raises_p_to_five(self):
        r = rri.evaluate(c_score=0, touches=["db/migrations/001_init.sql"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["rn"])
        self.assertEqual(r["scores"]["P"], 5)

    def test_generic_docs_floor_zero(self):
        r = rri.evaluate(c_score=0, touches=["docs/guide.md"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["python"])
        self.assertEqual(r["scores"]["P"], 0)

    def test_fenix_rubric_audit_has_p_five(self):
        r = rri.evaluate(c_score=0, touches=["internal/domain/audit/service.go"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        self.assertEqual(r["scores"]["P"], 5)
        self.assertIn("ADR-017", r["evidence"]["P"])

    def test_platform_name_in_output(self):
        r = rri.evaluate(c_score=0, f_override=0, d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["go"])
        self.assertEqual(r["platform"], "go")
        self.assertIn("**Platform:** go", rri.render_markdown(r))

    def test_platform_in_json(self):
        import json
        r = rri.evaluate(c_score=0, f_override=0, d=0, k=0, p=0, t=0, a=0, x=0,
                         profile=rri.PROFILES["fenix"])
        data = json.loads(rri.render_json(r))
        self.assertEqual(data["platform"], "fenix")


class PlatformCli(unittest.TestCase):
    """--platform flag wiring via subprocess."""

    def run_cli(self, *args):
        return subprocess.run([sys.executable, SCRIPT, *args],
                              capture_output=True, text=True)

    def test_platform_go_forces_generic_rubric(self):
        r = self.run_cli("--platform", "go", "--C", "2",
                         "--touches", "internal/auth/token.go",
                         "--D", "0", "--K", "0", "--P", "0",
                         "--T", "1", "--A", "0", "--X", "1")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("**Platform:** go", r.stdout)
        self.assertIn("auth", r.stdout)

    def test_platform_fenix_explicit(self):
        # Explicit --platform fenix uses the fenix profile and rubric.
        r = self.run_cli("--platform", "fenix", "--C", "1", "--F", "1", "--D", "0",
                         "--K", "0", "--P", "0", "--T", "0", "--A", "0", "--X", "0")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("**Platform:** fenix", r.stdout)

    def test_unknown_platform_rejected(self):
        r = self.run_cli("--platform", "cobol", "--C", "1", "--F", "1",
                         "--D", "0", "--K", "0", "--P", "0",
                         "--T", "0", "--A", "0", "--X", "0")
        self.assertNotEqual(r.returncode, 0)


class GitDiffBehavior(unittest.TestCase):
    """Document the git-diff F=0 limitation explicitly."""

    def test_empty_diff_yields_f_zero(self):
        # When git diff returns no files (e.g. no commits ahead of base, or
        # all changes are uncommitted), the script correctly reports F=0.
        # This is expected behaviour — use --touches at task-presentation time
        # to declare paths before committing (documented in RRI_POLICY.md).
        empty_git = lambda base: []  # noqa: E731
        r = rri.evaluate(c_score=0, d=0, k=0, p=0, t=0, a=0, x=0,
                         git=empty_git)
        self.assertEqual(r["scores"]["F"], 0)
        self.assertIn("0 files", r["evidence"]["F"])

    def test_touches_overrides_git(self):
        # --touches takes precedence over git diff; git is never called.
        def git_should_not_be_called(base):
            raise AssertionError("git should not be called when --touches is given")
        r = rri.evaluate(c_score=0, touches=["internal/domain/copilot/service.go"],
                         d=0, k=0, p=0, t=0, a=0, x=0,
                         git=git_should_not_be_called)
        self.assertEqual(r["scores"]["F"], 0)  # 1 file -> F score 0


if __name__ == "__main__":
    unittest.main(verbosity=2)
