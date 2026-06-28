#!/usr/bin/env python3
"""Session preflight for fenix coding agents.

Adapted from DubBridge agent-preflight.py; key differences:
  - Authority summary cites CLAUDE.md (not AGENT_WORKFLOW_GUIDE.md)
  - References fenix policy paths and README_AGENT_ORDER.md
  - Gemma/D14 requirement removed (Phase E, deferred)
  - FENIX_ env prefix (reserved for future env-driven overrides)
"""
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List


SCRIPT_VERSION = 1
SENTINEL_RELATIVE = Path(".agent") / "session-preflight.json"

SUMMARY_LINES = [
    "Fenix agent preflight",
    "",
    "Authority:",
    "- CLAUDE.md is the highest-authority workflow source.",
    "- README_AGENT_ORDER.md lists the required read order for a fresh session.",
    "- docs/policies/RRI_POLICY.md defines risk scoring and model tiers.",
    "- docs/policies/HITL_AUTONOMY_POLICY.md defines approval gates.",
    "",
    "Before implementation:",
    "- Read docs/architecture.md and the applicable ADRs.",
    "- Identify and read the plan file, task ledger, and any policy/config docs that constrain the task.",
    "- Ensure a task file exists in docs/tasks/ before writing any code.",
    "- Run scripts/rri.py before presenting or delegating a task.",
    "- RRI 0-25: execute directly (no full approval packet required).",
    "- RRI 26+: present the task card and wait for explicit approval before editing.",
    "",
    "Before closure:",
    "- Sync materially affected Obsidian vault artifacts before reporting completion.",
    "- Update docs/decisions/README.md if a new ADR was added.",
]


class PreflightError(Exception):
    """Raised when the session preflight has not been satisfied."""


def find_repo_root(start: "Path | None" = None) -> Path:
    """Return the git repository root, falling back to the script's grandparent."""
    if start is None:
        start = Path.cwd()
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            cwd=str(start),
            capture_output=True,
            check=True,
            text=True,
        )
        return Path(result.stdout.strip()).resolve()
    except (subprocess.CalledProcessError, FileNotFoundError):
        return Path(__file__).resolve().parents[1]


def sentinel_path(repo_root: Path) -> Path:
    return repo_root / SENTINEL_RELATIVE


def preflight_summary() -> str:
    return "\n".join(SUMMARY_LINES) + "\n"


def sentinel_payload(repo_root: Path) -> "Dict[str, Any]":
    return {
        "version": SCRIPT_VERSION,
        "repo_root": str(repo_root.resolve()),
        "marked_at": datetime.now(timezone.utc).isoformat(),
        "requirements": [
            "read CLAUDE.md before staged work",
            "read README_AGENT_ORDER.md to confirm session read order",
            "identify architecture, ADR, plan, task, and policy docs that constrain the task",
            "run scripts/rri.py before implementation",
            "wait for explicit approval when RRI is 26 or higher",
            "sync Obsidian vault artifacts before reporting completion",
        ],
    }


def mark_preflight(repo_root: Path) -> Path:
    path = sentinel_path(repo_root)
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp_path = path.with_suffix(f"{path.suffix}.tmp")
    tmp_path.write_text(
        json.dumps(sentinel_payload(repo_root), indent=2) + "\n",
        encoding="utf-8",
    )
    os.replace(tmp_path, path)
    return path


def load_sentinel(path: Path) -> "Dict[str, Any]":
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise PreflightError(
            f"Missing {path}. Run scripts/agent-preflight.py --mark before editing."
        ) from exc
    except json.JSONDecodeError as exc:
        raise PreflightError(
            f"Invalid {path}. Re-run scripts/agent-preflight.py --mark before editing."
        ) from exc
    if not isinstance(data, dict):
        raise PreflightError(
            f"Invalid {path}: expected a JSON object. Re-run scripts/agent-preflight.py --mark."
        )
    return data


def check_preflight(repo_root: Path) -> "Dict[str, Any]":
    path = sentinel_path(repo_root)
    data = load_sentinel(path)
    marked_root = data.get("repo_root")
    if marked_root != str(repo_root.resolve()):
        raise PreflightError(
            f"{path} was marked for {marked_root!r}, not {str(repo_root.resolve())!r}. "
            "Run scripts/agent-preflight.py --mark in this repository."
        )
    if data.get("version") != SCRIPT_VERSION:
        raise PreflightError(
            f"{path} has version {data.get('version')!r}; expected {SCRIPT_VERSION}. "
            "Run scripts/agent-preflight.py --mark again."
        )
    return data


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Print and validate the fenix agent-session workflow preflight."
    )
    parser.add_argument(
        "--repo-root",
        type=Path,
        default=None,
        help="Repository root override, mainly for tests and hook wrappers.",
    )
    parser.add_argument(
        "--print-summary",
        action="store_true",
        help="Print the compact workflow startup summary.",
    )
    parser.add_argument(
        "--mark",
        action="store_true",
        help="Write the session preflight sentinel.",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="Fail unless the session preflight sentinel is present and valid.",
    )
    return parser


def resolve_repo_root(raw: "Path | None") -> Path:
    if raw is not None:
        return raw.resolve()
    return find_repo_root()


def main(argv: "List[str] | None" = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    if not (args.print_summary or args.mark or args.check):
        args.print_summary = True

    repo_root = resolve_repo_root(args.repo_root)

    if args.print_summary:
        sys.stdout.write(preflight_summary())

    if args.mark:
        path = mark_preflight(repo_root)
        print(f"agent preflight marked: {path}")

    if args.check:
        try:
            data = check_preflight(repo_root)
        except PreflightError as exc:
            print(f"agent preflight failed: {exc}", file=sys.stderr)
            return 1
        print(f"agent preflight ok: {data.get('marked_at', 'unknown time')}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
