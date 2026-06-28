#!/usr/bin/env python3
# RRI calculator — deterministic Required Reasoning Index engine. [rri-calculator-script T1]
# Source of truth: docs/policies/RRI_POLICY.md. The agent measures raw inputs
# (paths, raw CC, coverage); this script owns every numeric->score mapping and the
# entire formula, anchor-rubric, penalty, band, and decomposition logic.
"""Compute the Required Reasoning Index (RRI) for a fenix task.

Usage (task-presentation time, no diff yet):
  python3 scripts/rri.py --touches internal/domain/auth/service.go \\
      --cc 18 --T 3 --A 0 --X 2 --D 1 --K 1 --P 1

Usage (post-implementation, measured from the working branch):
  python3 scripts/rri.py --cc 12 --T 2 --A 0 --X 2 --D 2 --K 2 --P 2
"""
import argparse
import json
import shutil
import subprocess
import sys
from dataclasses import dataclass, field
from fnmatch import fnmatchcase
from pathlib import Path
from typing import Callable, Optional

# --- Formula weights (sum = 1.00; verified in RRI_POLICY.md) ---------------------
WEIGHTS = {"C": 0.18, "F": 0.12, "D": 0.15, "T": 0.15,
           "A": 0.12, "K": 0.12, "P": 0.10, "X": 0.06}

VARS = list(WEIGHTS.keys())

# --- Per-variable numeric->score tables -----------------------------------------
# C: raw cyclomatic complexity -> score (RRI_POLICY.md "C" band table).
CC_TABLE = [(5, 0), (10, 1), (20, 2), (30, 3), (50, 4)]  # else 5

# F: file count -> score (RRI_POLICY.md "F" band table).
F_TABLE = [(1, 0), (2, 1), (5, 2), (10, 3), (20, 4)]  # else 5


def cc_to_score(cc):
    """Map a raw cyclomatic-complexity value to its 0-5 RRI C score."""
    for upper, score in CC_TABLE:
        if cc <= upper:
            return score
    return 5


# --- Platform C measurers (Strategy) --------------------------------------------
# Every measurer has the same signature: (paths) -> (raw_cc | None, evidence).
# A None result means "tool absent or no matching files"; the caller falls back to
# score 0 + Low-confidence. No measurer requires its tool to be installed.

def _filter_existing(paths, suffixes):
    """Return the paths whose suffix is in `suffixes` and that exist on disk."""
    return [p for p in paths
            if Path(p).suffix in suffixes and Path(p).exists()]


def measure_cc_radon(paths):
    """Return (max_cc, evidence) by running radon over .py files in paths."""
    py_files = _filter_existing(paths, (".py",))
    if not py_files:
        return None, "no local .py files in --touches; radon skipped"
    try:
        out = subprocess.run(
            ["radon", "cc", "--min", "A", "--show-complexity", "--total-average",
             *py_files],
            capture_output=True, text=True, check=True)
    except FileNotFoundError:
        return None, "radon not installed (pip install radon); --auto-cc skipped"
    except subprocess.CalledProcessError as exc:
        return None, f"radon error: {exc.stderr.strip()[:120]}"

    # Parse "M <file>:<line> <name> - <grade> (<cc>)" lines.
    max_cc = 1
    for line in out.stdout.splitlines():
        line = line.strip()
        if not line or line.startswith("Average"):
            continue
        if "(" in line and line.endswith(")"):
            try:
                raw = int(line[line.rfind("(") + 1:-1])
                max_cc = max(max_cc, raw)
            except ValueError:
                continue
    return max_cc, f"radon cc over {len(py_files)} file(s) -> max CC {max_cc}"


def measure_cc_clippy(paths):
    """Return (max_cc, evidence) via `cargo clippy` cognitive_complexity warnings.

    Runs once over the whole crate graph (clippy cannot target single files), then
    keeps only diagnostics whose span file is in the requested .rs path set. Slow:
    it compiles. Returns (None, reason) when cargo is unavailable or no .rs files.
    """
    rs_files = _filter_existing(paths, (".rs",))
    if not rs_files:
        return None, "no local .rs files in --touches; clippy skipped"
    cargo_bin = shutil.which("cargo") or str(Path.home() / ".cargo" / "bin" / "cargo")
    if not Path(cargo_bin).exists():
        return None, "cargo not installed; clippy --auto-cc skipped"
    try:
        out = subprocess.run(
            [cargo_bin, "clippy", "--message-format=json", "--quiet",
             "--", "-W", "clippy::cognitive_complexity"],
            capture_output=True, text=True, check=False)
    except FileNotFoundError:
        return None, "cargo not installed; clippy --auto-cc skipped"

    wanted = {str(Path(p)) for p in rs_files}
    max_cc = 1
    found = 0
    for line in out.stdout.splitlines():
        line = line.strip()
        if not line.startswith("{"):
            continue
        try:
            msg = json.loads(line).get("message", {})
        except json.JSONDecodeError:
            continue
        text = msg.get("message", "")
        if "cognitive complexity" not in text:
            continue
        # Diagnostic spans tell us which file the complex fn lives in.
        in_scope = any(str(Path(s.get("file_name", ""))) in wanted
                       for s in msg.get("spans", []))
        if not in_scope:
            continue
        # Message form: "the function has a cognitive complexity of (N/M)".
        cc = _parse_clippy_cc(text)
        if cc is not None:
            max_cc = max(max_cc, cc)
            found += 1
    if found == 0:
        return 1, (f"cargo clippy over crate graph -> no cognitive-complexity "
                   f"warnings in {len(rs_files)} touched file(s) -> CC 1")
    return max_cc, (f"cargo clippy cognitive_complexity -> max CC {max_cc} "
                    f"across {found} warning(s) in touched .rs files")


def _parse_clippy_cc(text):
    """Extract N from 'cognitive complexity of (N/M)' in a clippy message."""
    marker = "complexity of ("
    i = text.find(marker)
    if i == -1:
        return None
    rest = text[i + len(marker):]
    num = rest.split("/", 1)[0].strip()
    try:
        return int(num)
    except ValueError:
        return None


def measure_cc_gocyclo(paths):
    """Return (max_cc, evidence) via `gocyclo` over .go files in paths."""
    go_files = _filter_existing(paths, (".go",))
    if not go_files:
        return None, "no local .go files in --touches; gocyclo skipped"
    try:
        out = subprocess.run(
            ["gocyclo", "-over", "0", *go_files],
            capture_output=True, text=True, check=False)
    except FileNotFoundError:
        return None, "gocyclo not installed (go install ...gocyclo); skipped"

    # Each line: "<cc> <package> <func> <file:line:col>".
    max_cc = 1
    for line in out.stdout.splitlines():
        head = line.strip().split(" ", 1)[0]
        try:
            max_cc = max(max_cc, int(head))
        except ValueError:
            continue
    return max_cc, f"gocyclo over {len(go_files)} file(s) -> max CC {max_cc}"


def measure_cc_eslint(paths):
    """Return (max_cc, evidence) via ESLint's `complexity` rule over JS/TS files."""
    js_files = _filter_existing(paths, (".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"))
    if not js_files:
        return None, "no local JS/TS files in --touches; eslint skipped"
    try:
        out = subprocess.run(
            ["eslint", "--no-eslintrc", "--format", "json",
             "--rule", '{"complexity":["warn",0]}', *js_files],
            capture_output=True, text=True, check=False)
    except FileNotFoundError:
        return None, "eslint not installed (npm i -D eslint); --auto-cc skipped"

    try:
        report = json.loads(out.stdout or "[]")
    except json.JSONDecodeError:
        return None, f"eslint produced no JSON ({(out.stderr or '').strip()[:80]})"

    # complexity messages read: "... has a complexity of N. Maximum allowed is 0."
    max_cc = 1
    found = 0
    for file_report in report:
        for m in file_report.get("messages", []):
            if m.get("ruleId") != "complexity":
                continue
            cc = _parse_eslint_cc(m.get("message", ""))
            if cc is not None:
                max_cc = max(max_cc, cc)
                found += 1
    if found == 0:
        return 1, (f"eslint complexity over {len(js_files)} file(s) -> "
                   f"no complexity warnings -> CC 1")
    return max_cc, (f"eslint complexity -> max CC {max_cc} "
                    f"across {found} warning(s) in {len(js_files)} file(s)")


def _parse_eslint_cc(text):
    """Extract N from 'complexity of N' in an ESLint complexity message."""
    marker = "complexity of "
    i = text.find(marker)
    if i == -1:
        return None
    rest = text[i + len(marker):]
    num = ""
    for ch in rest:
        if ch.isdigit():
            num += ch
        else:
            break
    return int(num) if num else None


def measure_cc_none(paths):
    """No-op measurer for the generic profile: always defers to agent judgment."""
    return None, "generic platform: no automatic CC measurer; pass --cc or --C"


def count_to_f(n):
    """Map an affected-file count to its 0-5 RRI F score."""
    for upper, score in F_TABLE:
        if n <= upper:
            return score
    return 5


# --- Anchor rubric: path glob -> (D, P, K) floor + ADR -------------------------
# Each row maps a path glob to D/P/K floors plus an ADR citation and a label.
# Rows are ordered most-specific-first; the first matching row wins. fnmatchcase
# '*' spans '/', so a single '*' covers a subtree.
@dataclass(frozen=True)
class RubricRow:
    glob: str
    d: int
    p: int
    k: int
    adr: str
    label: str


# Fenix rubric (RRI_POLICY.md "Fenix anchor rubric"). ADR-anchored.
# High floors: governance, audit, auth, tool execution — these are the product moat.
# Medium floors: API surfaces, agent runtime, knowledge pipeline.
# Low floors: BFF, mobile, scripts, docs — low blast radius.
_FENIX_RUBRIC = [
    RubricRow("infra/migrations/*",                4, 5, 4, "ADR-017",      "migrations (schema/data)"),
    RubricRow("internal/domain/audit/*",           4, 5, 4, "ADR-017",      "internal/domain/audit"),
    RubricRow("internal/domain/policy/*",          3, 3, 3, "ADR-017",      "internal/domain/policy"),
    RubricRow("internal/domain/auth/*",            3, 4, 3, "ADR-017",      "internal/domain/auth"),
    RubricRow("internal/domain/tool/*",            3, 3, 3, "ADR-017",      "internal/domain/tool"),
    RubricRow("internal/domain/agent/*",           3, 2, 3, "ADR-100",      "internal/domain/agent"),
    RubricRow("internal/domain/workflow/*",        3, 2, 3, "ADR-100",      "internal/domain/workflow"),
    RubricRow("internal/domain/knowledge/*",       2, 2, 3, "ADR-012",      "internal/domain/knowledge"),
    RubricRow("internal/domain/copilot/*",         2, 2, 2, "—",            "internal/domain/copilot"),
    RubricRow("internal/domain/eval/*",            2, 2, 2, "ADR-102",      "internal/domain/eval"),
    RubricRow("internal/domain/*",                 2, 2, 2, "—",            "internal/domain (other)"),
    RubricRow("internal/api/*",                    2, 2, 2, "ADR-008",      "internal/api"),
    RubricRow("internal/infra/*",                  2, 2, 2, "ADR-010",      "internal/infra"),
    RubricRow("pkg/*",                             2, 2, 2, "—",            "pkg"),
    RubricRow("mobile/*",                          2, 1, 2, "ADR-022",      "mobile"),
    RubricRow("bff/*",                             1, 1, 1, "ADR-009",      "bff"),
    RubricRow("scripts/*",                         1, 0, 0, "—",            "scripts"),
    RubricRow("docs/*",                            0, 0, 0, "—",            "docs"),
    RubricRow("features/*",                        0, 0, 0, "ADR-018",      "features (BDD)"),
]

# Generic cross-language rubric by directory convention (no project-specific ADRs).
# Used by the rust/python/go/rn profiles so the tool is useful in any repo.
_GENERIC_RUBRIC = [
    RubricRow("*/migrations/*", 4, 5, 4, "—", "migrations (data/schema)"),
    RubricRow("migrations/*", 4, 5, 4, "—", "migrations (data/schema)"),
    RubricRow("*/auth/*", 4, 4, 4, "—", "auth"),
    RubricRow("auth/*", 4, 4, 4, "—", "auth"),
    RubricRow("*/security/*", 4, 4, 4, "—", "security"),
    RubricRow("security/*", 4, 4, 4, "—", "security"),
    RubricRow("*/crypto/*", 4, 4, 4, "—", "crypto"),
    RubricRow("crypto/*", 4, 4, 4, "—", "crypto"),
    RubricRow("*/payments/*", 3, 4, 3, "—", "payments"),
    RubricRow("*/db/*", 3, 3, 3, "—", "db"),
    RubricRow("*/database/*", 3, 3, 3, "—", "database"),
    RubricRow("*/api/*", 3, 3, 3, "—", "api surface"),
    RubricRow("*/services/*", 3, 3, 3, "—", "services"),
    RubricRow("*/test*/*", 0, 0, 0, "—", "tests"),
    RubricRow("test*/*", 0, 0, 0, "—", "tests"),
    RubricRow("docs/*", 0, 0, 0, "—", "docs"),
]


def first_matching_row(path, rubric):
    """Return the most-specific rubric row matching path, or None."""
    for row in rubric:
        if fnmatchcase(path, row.glob):
            return row
    return None


def match_rubric(paths, rubric):
    """Derive D/P/K floors from the paths a task touches, using `rubric`.

    Returns (floors, rows, advisories, matched_auth) where floors maps each of
    D/P/K to its highest floor across all paths, rows records which rubric label
    set each floor, advisories holds content-dependent notes, and matched_auth is
    True when any touched path carries a P floor >= 4 (auth/authz/ownership/data).
    """
    floors = {"D": 0, "P": 0, "K": 0}
    rows = {"D": None, "P": None, "K": None}
    advisories = []
    for path in paths:
        row = first_matching_row(path, rubric)
        if row is None:
            advisories.append(
                f"{path}: no anchor-rubric match — agent judgment governs D/P/K")
            continue
        for dim, val in (("D", row.d), ("P", row.p), ("K", row.k)):
            if val > floors[dim]:
                floors[dim] = val
                rows[dim] = (row.label, row.adr)
        if fnmatchcase(path, "config/*.toml"):
            advisories.append(
                f"{path}: config floor is 0 (non-secret); if it wires env/secrets, "
                f"raise D/P/K to >= 1")
    matched_auth = floors["P"] >= 4
    return floors, rows, advisories, matched_auth


# --- Platform profiles (Strategy + Registry) ------------------------------------
@dataclass(frozen=True)
class PlatformProfile:
    """Bundles a platform's CC measurer, anchor rubric, and detection markers."""
    name: str
    markers: tuple = ()
    source_suffixes: tuple = ()
    measure_cc: Callable = measure_cc_none
    rubric: list = field(default_factory=list)


PROFILES = {
    # Fenix: this repo. go.mod + docs/policies/RRI_POLICY.md marker -> fenix profile.
    # The policy-file marker makes auto-detection unambiguous even if go.mod exists.
    "fenix": PlatformProfile(
        name="fenix",
        markers=("docs/policies/RRI_POLICY.md",),
        source_suffixes=(".go",),
        measure_cc=measure_cc_gocyclo,
        rubric=_FENIX_RUBRIC),
    "rust": PlatformProfile(
        name="rust", markers=("Cargo.toml",), source_suffixes=(".rs",),
        measure_cc=measure_cc_clippy, rubric=_GENERIC_RUBRIC),
    "go": PlatformProfile(
        name="go", markers=("go.mod",), source_suffixes=(".go",),
        measure_cc=measure_cc_gocyclo, rubric=_GENERIC_RUBRIC),
    "rn": PlatformProfile(
        name="rn", markers=("package.json",),
        source_suffixes=(".js", ".jsx", ".ts", ".tsx"),
        measure_cc=measure_cc_eslint, rubric=_GENERIC_RUBRIC),
    "python": PlatformProfile(
        name="python", markers=("pyproject.toml", "setup.py", "setup.cfg"),
        source_suffixes=(".py",), measure_cc=measure_cc_radon,
        rubric=_GENERIC_RUBRIC),
    # Fallback when nothing is detected: no measurer, empty rubric -> agent judgment.
    "generic": PlatformProfile(name="generic"),
}

# Detection order: fenix before go (the policy marker is more specific than go.mod),
# then the remaining ecosystems.
DETECTION_ORDER = ["fenix", "rust", "go", "rn", "python"]


def detect_platform(start_dir="."):
    """Return the PlatformProfile for the nearest marker found walking up from start_dir.

    Walks from start_dir toward the filesystem root; at each level, the first profile
    in DETECTION_ORDER whose marker exists wins. Falls back to the generic profile.
    """
    base = Path(start_dir).resolve()
    for cur in [base, *base.parents]:
        for name in DETECTION_ORDER:
            prof = PROFILES[name]
            if any((cur / marker).exists() for marker in prof.markers):
                return prof
    return PROFILES["generic"]


def resolve_platform(name):
    """Resolve a --platform value to a profile; 'auto' triggers detection."""
    if name in (None, "auto"):
        return detect_platform()
    return PROFILES[name]


# --- Penalties (RRI_POLICY.md "Penalties") --------------------------------------
PENALTY_VALUES = {
    "refactor_and_behavior": 8,
    "no_tests_high_impact": 10,
    "complex_and_domain": 10,
    "auth_security": 10,
    "many_files": 8,
    "arch_decision": 12,
    "no_verification": 15,
}
# Penalties an agent may pass via --penalty. The other three are auto-detected.
MANUAL_PENALTIES = {"refactor_and_behavior", "arch_decision",
                    "no_verification", "auth_security"}


def detect_penalties(scores, matched_auth, manual):
    """Return {name: (value, reason)} merging auto-detected and manual penalties.

    Each penalty is applied at most once; an auto reason wins over a manual one.
    """
    applied = {}
    if scores["F"] >= 4:
        applied["many_files"] = (8, f"F={scores['F']} >= 4")
    if scores["C"] >= 4 and scores["D"] >= 3:
        applied["complex_and_domain"] = (
            10, f"C={scores['C']} >= 4 and D={scores['D']} >= 3")
    if scores["T"] >= 4 and scores["P"] >= 4:
        applied["no_tests_high_impact"] = (
            10, f"T={scores['T']} >= 4 and P={scores['P']} >= 4")
    if matched_auth:
        applied["auth_security"] = (10, "anchor-rubric P floor >= 4 (auth/audit/rights/secrets)")
    for name in manual:
        if name not in applied:
            applied[name] = (PENALTY_VALUES[name], "manual flag")
    return applied


# --- Bands crosswalk (RRI_POLICY.md "Bands, autonomy gates, and model tiers") ---
# Model tiers: economy=claude-haiku-4-5, balanced=claude-sonnet-4-6, premium=claude-opus-4-8
# (upper_inclusive, label, effort, economy_model, balanced_model, thinking, gate)
BANDS = [
    (25, "Low", "S", "claude-sonnet-4-6", "claude-sonnet-4-6", "Off",
     "Execute directly. No full approval packet required."),
    (40, "Moderate", "M", "claude-sonnet-4-6", "claude-sonnet-4-6", "Off",
     "Present task card and wait for explicit approval. Confirm tests exist in the affected area."),
    (55, "Med-high", "L", "claude-sonnet-4-6", "claude-opus-4-8", "On",
     "Plan + explicit acceptance criteria required before approval."),
    (70, "Complex", "L", "claude-opus-4-8", "claude-opus-4-8", "On",
     "Plan first. Human reviews the plan before any implementation."),
    (85, "High", "XL", "claude-opus-4-8", "claude-opus-4-8", "On",
     "Characterization tests + explicit acceptance criteria + human reviews the diff."),
    (100, "Very high", "XL", "claude-opus-4-8", "claude-opus-4-8", "On",
     "Do not implement directly. Produce an ADR + risk analysis + decompose into subtasks."),
    (float("inf"), "Excessive", "XL", "claude-opus-4-8", "claude-opus-4-8", "On",
     "Architecture/design work must happen first. Re-scope before any implementation."),
]


def resolve_band(rri):
    """Return the crosswalk row (as a dict) for a final RRI value."""
    for upper, label, effort, economy, balanced, thinking, gate in BANDS:
        if rri <= upper:
            lower = {"Low": 0, "Moderate": 26, "Med-high": 41, "Complex": 56,
                     "High": 71, "Very high": 86, "Excessive": 101}[label]
            rng = f"{lower}-{upper}" if upper != float("inf") else ">100"
            return {"range": rng, "label": label, "effort": effort,
                    "economy": economy, "balanced": balanced,
                    "thinking": thinking, "gate": gate}
    raise AssertionError("unreachable: bands cover all values")


def detect_triggers(rri, base, scores, applied):
    """Return the list of decomposition triggers that fired (RRI_POLICY.md)."""
    triggers = []
    if rri > 70:
        triggers.append("RRI > 70")
    if base > 100:
        triggers.append("base RRI > 100")
    if scores["F"] >= 4 and scores["K"] >= 3:
        triggers.append("F >= 4 and K >= 3")
    if scores["C"] >= 4 and scores["D"] >= 3:
        triggers.append("C >= 4 and D >= 3")
    if "refactor_and_behavior" in applied:
        triggers.append("refactor + behavior (+8) active")
    if scores["T"] >= 4 and scores["P"] >= 4:
        triggers.append("T >= 4 and P >= 4")
    return triggers


# --- Core evaluation ------------------------------------------------------------
def git_diff_paths(base):
    """Return the files changed vs base, or raise RuntimeError if git is unavailable."""
    try:
        out = subprocess.run(
            ["git", "diff", "--name-only", f"{base}...HEAD"],
            capture_output=True, text=True, check=True)
    except (FileNotFoundError, subprocess.CalledProcessError) as exc:
        raise RuntimeError(
            "git diff unavailable; pass --touches <path>... or --F <0-5>") from exc
    return [line for line in out.stdout.splitlines() if line.strip()]


def evaluate(*, cc=None, c_score=None, auto_cc=False, touches=None,
             f_override=None, base="main",
             d=0, k=0, p=0, t=0, a=0, x=0, manual_penalties=None, low_conf=None,
             git=git_diff_paths, profile=None):
    """Compute a full RRI result dict from resolved inputs.

    Exactly one of cc / c_score / auto_cc must drive C. F comes from touches,
    else f_override, else git. D/K/P are raised to their anchor-rubric floors;
    low-confidence vars are bumped +1 (capped at 5). The active platform `profile`
    (auto-detected when None) supplies the --auto-cc measurer and the anchor rubric.
    Returns a dict ready for rendering.
    """
    manual_penalties = set(manual_penalties or [])
    low_conf = list(low_conf or [])
    if profile is None:
        profile = detect_platform()

    # F + the path set used for the anchor rubric.
    # Resolved first so --auto-cc can reuse the same list without a second git call.
    if touches:
        paths = list(touches)
        f = count_to_f(len(paths))
        f_ev = f"--touches -> {len(paths)} files"
    elif f_override is not None:
        paths = []
        f = f_override
        f_ev = f"--F override = {f_override}"
    else:
        paths = git(base)
        f = count_to_f(len(paths))
        f_ev = f"git diff --name-only {base}...HEAD -> {len(paths)} files"

    # C: raw CC mapped via the policy table, else the pre-computed score, else
    # auto-measured by the active platform's measurer over the resolved path list.
    if cc is not None:
        c = cc_to_score(cc)
        c_ev = f"raw CC {cc} -> score {c} (policy CC table)"
    elif auto_cc:
        measured, measure_ev = profile.measure_cc(paths)
        if measured is not None:
            c = cc_to_score(measured)
            c_ev = f"{measure_ev} -> score {c} (policy CC table)"
        else:
            # Fallback: score 0 marked Low-confidence so the output is honest.
            c = 0
            c_ev = f"auto-cc fallback (score=0): {measure_ev}"
            if "C" not in low_conf:
                low_conf = list(low_conf) + ["C"]
    else:
        c = c_score
        c_ev = "agent-supplied score"

    floors, rows, advisories, matched_auth = match_rubric(paths, profile.rubric)

    # Agent judgments, with D/P/K raised to the rubric floor (never lowered).
    agent = {"D": d, "K": k, "P": p}
    scores = {"C": c, "F": f, "T": t, "A": a, "X": x}
    floor_ev = {}
    for dim in ("D", "K", "P"):
        raised = max(agent[dim], floors[dim])
        scores[dim] = raised
        if rows[dim] is not None:
            label, adr = rows[dim]
            if raised > agent[dim]:
                floor_ev[dim] = (f"anchor rubric: {label} ({adr}) -> floor {floors[dim]}; "
                                 f"raised from {agent[dim]}")
            else:
                floor_ev[dim] = (f"anchor rubric: {label} ({adr}) -> floor {floors[dim]} "
                                 f"(agent {agent[dim]} kept)")
        else:
            floor_ev[dim] = "agent-supplied (no rubric match)"

    # Confidence + low-confidence +1 bump (mechanical; RRI_POLICY.md).
    confidence = {v: "High" for v in VARS}
    for v in low_conf:
        scores[v] = min(5, scores[v] + 1)
        confidence[v] = "Low"

    # Base value: weighted sum, normalized by /5, scaled x100, rounded to nearest.
    weighted = sum(WEIGHTS[v] * scores[v] for v in VARS)
    base_val = round(100 * weighted / 5)

    applied = detect_penalties(scores, matched_auth, manual_penalties)
    penalty_total = sum(val for val, _ in applied.values())
    final = base_val + penalty_total

    band = resolve_band(final)
    triggers = detect_triggers(final, base_val, scores, applied)

    evidence = {
        "C": c_ev, "F": f_ev, "T": "agent-supplied", "A": "agent-supplied",
        "X": "agent-supplied", "D": floor_ev["D"], "K": floor_ev["K"],
        "P": floor_ev["P"],
    }
    return {
        "scores": scores, "evidence": evidence, "confidence": confidence,
        "base": base_val, "penalties": applied, "penalty_total": penalty_total,
        "final": final, "band": band, "triggers": triggers,
        "advisories": advisories, "platform": profile.name,
    }


# --- Rendering ------------------------------------------------------------------
VAR_LABELS = {"C": "C cyclomatic", "F": "F files", "D": "D domain",
              "T": "T coverage", "A": "A ambiguity", "K": "K coupling",
              "P": "P impact", "X": "X context"}
TABLE_ORDER = ["C", "F", "D", "T", "A", "K", "P", "X"]


def render_markdown(r):
    """Render the result as the RRI_POLICY.md reporting-format markdown block."""
    lines = []
    if r.get("platform"):
        lines.append(f"**Platform:** {r['platform']}")
        lines.append("")
    lines += ["| Variable | Score | Evidence | Confidence |", "|---|---|---|---|"]
    for v in TABLE_ORDER:
        lines.append(
            f"| {VAR_LABELS[v]} | {r['scores'][v]} | {r['evidence'][v]} | {r['confidence'][v]} |")
    lines.append("")
    if r["penalties"]:
        pen = "; ".join(f"{name} (+{val}, {reason})"
                        for name, (val, reason) in sorted(r["penalties"].items()))
    else:
        pen = "none"
    b = r["band"]
    lines.append(f"**Base value:** 100 x (weighted / 5) = {r['base']}")
    lines.append(f"**Penalties applied:** {pen}")
    lines.append(
        f"**Final RRI:** {r['final']} -> band {b['label']} ({b['range']}) -> "
        f"Effort {b['effort']} . Economy {b['economy']} . Balanced {b['balanced']} . "
        f"thinking {b['thinking']}")
    lines.append(f"**Gates for this band:** {b['gate']}")
    if r["triggers"]:
        lines.append(f"**Decomposition:** triggered by {', '.join(r['triggers'])} "
                     f"— split before implementing")
    else:
        lines.append("**Decomposition:** not triggered")
    for note in r["advisories"]:
        lines.append(f"**Advisory:** {note}")
    return "\n".join(lines)


def render_json(r):
    """Render the result as a machine-parseable JSON object."""
    return json.dumps({
        "platform": r.get("platform"),
        "variables": {v: {"score": r["scores"][v], "evidence": r["evidence"][v],
                          "confidence": r["confidence"][v]} for v in TABLE_ORDER},
        "base": r["base"],
        "penalties": [{"name": n, "value": v, "reason": why}
                      for n, (v, why) in sorted(r["penalties"].items())],
        "penalty_total": r["penalty_total"],
        "final": r["final"],
        "band": r["band"],
        "triggers": r["triggers"],
        "advisories": r["advisories"],
    }, indent=2)


# --- CLI ------------------------------------------------------------------------
def _score_arg(name):
    def parse(raw):
        try:
            val = int(raw)
        except ValueError:
            raise argparse.ArgumentTypeError(f"{name} must be an integer 0-5, got {raw!r}")
        if not 0 <= val <= 5:
            raise argparse.ArgumentTypeError(f"{name} must be in range 0-5, got {val}")
        return val
    return parse


def build_parser():
    p = argparse.ArgumentParser(
        prog="rri.py",
        description="Deterministic RRI calculator (docs/policies/RRI_POLICY.md).")
    cgroup = p.add_mutually_exclusive_group(required=True)
    cgroup.add_argument("--cc", type=int, metavar="RAW",
                        help="Raw cyclomatic complexity (radon/mccabe/clippy) -> mapped to C score")
    cgroup.add_argument("--C", dest="c_score", type=_score_arg("--C"), metavar="0-5",
                        help="Pre-computed C score (use when raw CC is unavailable)")
    cgroup.add_argument("--auto-cc", action="store_true", default=False,
                        help="Measure CC automatically via the detected platform's "
                             "measurer (gocyclo/eslint/radon) over --touches "
                             "paths; falls back to score=0 + Low-confidence if the "
                             "tool is unavailable or no matching source files found)")
    for name in ("T", "A", "X"):
        p.add_argument(f"--{name}", required=True, type=_score_arg(f"--{name}"),
                       metavar="0-5")
    for name in ("D", "K", "P"):
        p.add_argument(f"--{name}", required=True, type=_score_arg(f"--{name}"),
                       metavar="0-5", help=f"{name} judgment (raised to anchor-rubric floor)")
    p.add_argument("--touches", action="append", metavar="PATH",
                   help="Affected path; repeatable. Drives F and the anchor rubric.")
    p.add_argument("--base", default="main", help="Base branch for git-diff fallback")
    p.add_argument("--F", dest="f_override", type=_score_arg("--F"), metavar="0-5",
                   help="Manual F score (overrides git; use with no --touches)")
    p.add_argument("--penalty", action="append", default=[], choices=sorted(MANUAL_PENALTIES),
                   metavar="KEY", help=f"Manual penalty; repeatable. One of: {sorted(MANUAL_PENALTIES)}")
    p.add_argument("--low-confidence", default="", metavar="VARS",
                   help="Comma list (e.g. D,K) bumped +1 and marked Low")
    p.add_argument("--platform", default="auto",
                   choices=["auto", *DETECTION_ORDER, "generic"],
                   help="Platform profile (CC measurer + anchor rubric). "
                        "'auto' detects by marker file (default).")
    p.add_argument("--json", action="store_true", help="Emit JSON instead of markdown")
    return p


def main(argv=None):
    parser = build_parser()
    args = parser.parse_args(argv)

    low = [v.strip().upper() for v in args.low_confidence.split(",") if v.strip()]
    bad = [v for v in low if v not in VARS]
    if bad:
        parser.error(f"--low-confidence has unknown variable(s) {bad}; valid: {VARS}")

    if args.cc is not None and args.cc < 1:
        parser.error("--cc must be >= 1 (cyclomatic complexity starts at 1)")

    profile = resolve_platform(args.platform)

    try:
        result = evaluate(
            cc=args.cc, c_score=args.c_score, auto_cc=args.auto_cc,
            touches=args.touches, f_override=args.f_override, base=args.base,
            d=args.D, k=args.K, p=args.P, t=args.T, a=args.A, x=args.X,
            manual_penalties=args.penalty, low_conf=low, profile=profile)
    except RuntimeError as exc:
        parser.error(str(exc))

    print(render_json(result) if args.json else render_markdown(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
