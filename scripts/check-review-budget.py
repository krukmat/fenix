#!/usr/bin/env python3
"""Pre-delegation reviewability budget gate for non-Gemma agents.

Local Gemma reviewer (`gemma-code-review.py`) and developer
(`delegate-low-rri.py`) roles evaluate a change inside a fixed context window
(`DEFAULT_NUM_CTX`) while reserving generation headroom (`DEFAULT_NUM_PREDICT`).
A change whose diff is larger than that effective window either overflows the
context silently or truncates Gemma's response (`done_reason == "length"`).

This gate is the *proactive* counterpart to those fail-closed paths: it runs
before delegation and fails closed when the added/changed lines of the diff
exceed a budget derived from the context window, so the delivering (non-Gemma)
agent is forced to split the change or escalate to a non-Gemma reviewer (D14).

The budget is derived from the same env vars Gemma reads, not hardcoded, so it
tracks the context window instead of drifting from it. Only code paths that
Gemma actually receives are counted; docs/config/markdown are excluded, mirroring
the `qa-gemma-review` packet filter and `check-maintainability.classify_path`.
"""

from __future__ import annotations

import argparse
import importlib.util
import os
import re
import sys
from pathlib import Path


# Reuse the diff parser and code-path classifier already proven by the
# maintainability gate instead of re-deriving them here. Both modules use
# hyphenated/underscored filenames, so load them by path.
def _load(module_name: str, filename: str):
    spec = importlib.util.spec_from_file_location(module_name, Path(__file__).with_name(filename))
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[module_name] = module
    spec.loader.exec_module(module)
    return module


_maint = _load("check_maintainability", "check-maintainability.py")
_gemma = _load("gemma_local", "gemma_local.py")


# Marker the delivering agent records (commit body or task entry) to take the
# documented escape when a change is legitimately irreducible. The reason is
# captured for the audit log; an empty reason does not satisfy the escape.
D14_OVERRIDE_RE = re.compile(r"D14-OVERRIDE:\s*(?P<reason>\S.*)$", re.MULTILINE)

# Token budget consumed by the fixed parts of a delegation packet (system prompt
# + contract + acceptance criteria) before any diff is added. Conservative
# overhead estimate so the derived line budget errs toward keeping changes
# reviewable rather than toward the raw ceiling.
PACKET_OVERHEAD_TOKENS = 1300

# A diff line averages well under the raw `num_ctx / num_predict` token cost; we
# use the same ~4-chars-per-token basis as gemma_local.estimate_text_tokens and
# assume an ~80-char average reviewable line ≈ 20 tokens/line.
TOKENS_PER_DIFF_LINE = 20

# Floor so a misconfigured tiny context window can never produce an absurd
# 1- or 2-line budget that blocks every change.
MIN_DERIVED_BUDGET = 120


def packet_overhead_tokens() -> int:
    """Fixed token cost of a delegation packet before the diff is added.

    `FENIX_REVIEW_PACKET_OVERHEAD_TOKENS` tunes this without code changes
    when the prompt/contract template drifts from the default estimate.
    """
    return _env_int("FENIX_REVIEW_PACKET_OVERHEAD_TOKENS", PACKET_OVERHEAD_TOKENS)


def _env_int(name: str, default: int) -> int:
    raw = os.environ.get(name)
    if raw is None or not raw.strip():
        return default
    try:
        return int(raw.strip())
    except ValueError:
        return default


def derive_budget() -> int:
    """Return the max reviewable diff lines, derived from Gemma's context window.

    budget = (num_ctx - num_predict - packet_overhead) / tokens_per_line

    `FENIX_REVIEW_MAX_DIFF_LINES` overrides the derived value outright when
    an operator needs an explicit ceiling.
    """
    explicit = os.environ.get("FENIX_REVIEW_MAX_DIFF_LINES")
    if explicit is not None and explicit.strip():
        try:
            return max(1, int(explicit.strip()))
        except ValueError:
            pass

    num_ctx = _env_int(
        "FENIX_REVIEW_NUM_CTX",
        _env_int("FENIX_LOW_RRI_NUM_CTX", _gemma.DEFAULT_NUM_CTX),
    )
    num_predict = _env_int(
        "FENIX_REVIEW_NUM_PREDICT",
        _env_int("FENIX_LOW_RRI_NUM_PREDICT", _gemma.DEFAULT_NUM_PREDICT),
    )
    usable_tokens = num_ctx - num_predict - packet_overhead_tokens()
    derived = usable_tokens // TOKENS_PER_DIFF_LINE
    return max(MIN_DERIVED_BUDGET, derived)


def count_reviewable_lines(base: "str | None", files: "list[str]") -> int:
    """Count added/changed code lines Gemma would receive for this change."""
    return len(_maint.added_lines_for(base, files))


def find_override(text: str) -> "str | None":
    """Return the documented D14 override reason from `text`, or None."""
    match = D14_OVERRIDE_RE.search(text or "")
    if match is None:
        return None
    reason = match.group("reason").strip()
    return reason or None


def _override_text(explicit_message: "str | None") -> str:
    """Assemble the text searched for a D14 override marker.

    Sources, in order: an explicit `--override-message`, then the tip commit
    message (so the marker can live in the commit body that ships the change).
    """
    parts = []
    if explicit_message:
        parts.append(explicit_message)
    if os.environ.get("FENIX_REVIEW_OVERRIDE"):
        parts.append(os.environ["FENIX_REVIEW_OVERRIDE"])
    try:
        parts.append(_maint.run_git(["log", "-1", "--pretty=%B"]))
    except RuntimeError:
        pass
    return "\n".join(parts)


def render_report(count: int, budget: int, override_reason: "str | None") -> "tuple[str, int]":
    """Return (report, exit_code) for the given counts and override state."""
    if count <= budget:
        return (f"Reviewability budget gate passed ({count}/{budget} reviewable diff lines).", 0)
    if override_reason is not None:
        return (
            "Reviewability budget exceeded but D14 override is documented "
            f"({count}/{budget} lines).\n"
            f"- override reason: {override_reason}\n"
            "This change must be reviewed by a non-Gemma (D14) reviewer; "
            "record disposition_divergence in the audit log.",
            0,
        )
    return (
        "Reviewability budget gate failed: "
        f"{count} added/changed code lines exceed the budget of {budget}.\n"
        "Gemma cannot evaluate a change this large in-context with full fidelity.\n"
        "Split the change into smaller delegation units, or — if it is genuinely "
        "irreducible — escalate to a non-Gemma (D14) reviewer and record a\n"
        "  D14-OVERRIDE: <reason>\n"
        "line in the commit body or task entry.",
        1,
    )


def main(argv: "list[str]") -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", help="git revision to diff against")
    parser.add_argument("--files", nargs="*", help="explicit files to inspect")
    parser.add_argument(
        "--override-message",
        help="text to scan for a D14-OVERRIDE marker (in addition to the tip commit)",
    )
    args = parser.parse_args(argv)

    base = _maint.discover_base(args.base)
    files = args.files if args.files is not None else _maint.changed_files(base)
    count = count_reviewable_lines(base, files)
    budget = derive_budget()
    override_reason = None
    if count > budget:
        override_reason = find_override(_override_text(args.override_message))

    report, code = render_report(count, budget, override_reason)
    print(report)
    return code


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
