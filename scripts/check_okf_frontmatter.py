#!/usr/bin/env python3
"""Fenix doc_type frontmatter validator — enforces the closed doc_type vocabulary.

Adapted from DubBridge check_okf_frontmatter.py; key differences:
  - YAML key is 'doc_type' (not 'type')
  - ADRs live in docs/decisions/, plans in docs/plans/
  - Vocabulary: task, adr, summary, audit, handoff, plan, policy, playbook, proposal, roadmap
  - ADR status sync checks '## Status' prose section (not '- **Status:**' bullet)
  - --report-only: print violations but exit 0 (safe introduction against existing corpus)
  - --changed-only: validate only files changed vs. BASE (default: main)
"""
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Dict, List, Optional

import yaml

REPO_ROOT = Path(__file__).resolve().parent.parent

# Closed doc_type vocabulary: value -> path pattern relative to repo root.
VOCAB: Dict[str, re.Pattern] = {
    "roadmap":  re.compile(r"^docs/plans/roadmap\.md$"),
    "adr":      re.compile(r"^docs/decisions/ADR-\d+.*\.md$"),
    "plan":     re.compile(r"^docs/plans/(?!roadmap\.md)[^/]+\.md$"),
    "policy":   re.compile(r"^docs/policies/[^/]+\.md$"),
    "playbook": re.compile(r"^docs/playbooks/[^/]+\.md$"),
    "proposal": re.compile(r"^docs/proposals/[^/]+\.md$"),
    "task":     re.compile(r"^docs/tasks/[^/]+\.md$"),
    "summary":  re.compile(r"^docs/summaries/[^/]+\.md$"),
    "audit":    re.compile(r"^docs/audit/[^/]+\.md$"),
    "handoff":  re.compile(r"^docs/handoffs/[^/]+\.md$"),
}

# Navigation READMEs — pure index files, no frontmatter required.
INDEX_READMES = {
    "docs/decisions/README.md",
    "docs/plans/README.md",
    "docs/policies/README.md",
    "docs/playbooks/README.md",
    "docs/proposals/README.md",
    "docs/tasks/README.md",
    "docs/summaries/README.md",
    "docs/audit/README.md",
    "docs/handoffs/README.md",
}

SKIP_PATTERNS = [
    re.compile(r"(^|/)TEMPLATE\.md$"),
]

# ADR status values recognised in the prose '## Status' section.
ADR_STATUSES = {"proposed", "accepted", "superseded", "deprecated"}


def _rel(path: Path) -> str:
    return str(path.relative_to(REPO_ROOT))


def should_skip(rel: str) -> bool:
    if rel in INDEX_READMES:
        return True
    return any(p.search(rel) for p in SKIP_PATTERNS)


def parse_frontmatter(text: str) -> Optional[dict]:
    """Return parsed YAML frontmatter dict, or None if absent/malformed."""
    if not text.startswith("---"):
        return None
    end = text.find("\n---", 3)
    if end == -1:
        return None
    block = text[3:end].strip()
    try:
        data = yaml.safe_load(block)
        if isinstance(data, dict):
            return data
        return None
    except yaml.YAMLError:
        return None


def extract_prose_adr_status(text: str) -> Optional[str]:
    """Return normalised status from the '## Status' prose section, or None."""
    m = re.search(r"^## Status\s*\n+[`'\"]?(\w+)[`'\"]?", text, re.MULTILINE)
    if not m:
        return None
    return m.group(1).strip().lower()


def doc_type_matches_location(doc_type: str, rel: str) -> bool:
    pattern = VOCAB.get(doc_type)
    if pattern is None:
        return False
    return bool(pattern.match(rel))


def collect_in_scope_files() -> List[Path]:
    """All markdown files whose path falls under a known vocab location."""
    docs = REPO_ROOT / "docs"
    files = []
    for path in sorted(docs.rglob("*.md")):
        rel = _rel(path)
        if should_skip(rel):
            continue
        if any(p.match(rel) for p in VOCAB.values()):
            files.append(path)
    return files


def collect_changed_files(base: str = "main") -> List[Path]:
    """Markdown files changed vs. base branch that fall under a vocab location."""
    try:
        result = subprocess.run(
            ["git", "diff", "--name-only", "--diff-filter=ACM", base + "...HEAD"],
            capture_output=True,
            text=True,
            cwd=str(REPO_ROOT),
        )
        changed = result.stdout.strip().splitlines()
    except Exception:
        changed = []
    files = []
    for rel in changed:
        if not rel.endswith(".md"):
            continue
        if should_skip(rel):
            continue
        if any(p.match(rel) for p in VOCAB.values()):
            path = REPO_ROOT / rel
            if path.exists():
                files.append(path)
    return files


def validate(files: List[Path]) -> List[str]:
    errors: List[str] = []

    for path in files:
        rel = _rel(path)
        text = path.read_text(encoding="utf-8")

        fm = parse_frontmatter(text)
        if fm is None:
            errors.append(f"{rel}: missing or malformed frontmatter block")
            continue

        doc_type = fm.get("doc_type")
        if doc_type not in VOCAB:
            errors.append(
                f"{rel}: 'doc_type' value {doc_type!r} is not in the closed vocabulary "
                f"({', '.join(sorted(VOCAB))})"
            )
            continue

        if not doc_type_matches_location(doc_type, rel):
            errors.append(
                f"{rel}: 'doc_type: {doc_type}' does not match file location"
            )
            continue

        if doc_type == "adr":
            fm_status = str(fm.get("status", "")).lower()
            prose_status = extract_prose_adr_status(text)
            if prose_status is not None and fm_status != prose_status:
                errors.append(
                    f"{rel}: frontmatter status={fm_status!r} does not match "
                    f"prose status={prose_status!r}"
                )

    return errors


def main() -> int:
    args = sys.argv[1:]
    report_only = "--report-only" in args
    changed_only = "--changed-only" in args
    base = "main"
    for arg in args:
        if arg.startswith("--base="):
            base = arg.split("=", 1)[1]

    files = collect_changed_files(base) if changed_only else collect_in_scope_files()
    errors = validate(files)

    if errors:
        print("doc_type frontmatter check failed:")
        for e in errors:
            print(f"  - {e}")
        if report_only:
            print("(report-only mode: exit 0)")
            return 0
        return 1

    mode = " (changed-only)" if changed_only else ""
    print(f"doc_type frontmatter check passed{mode}.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
