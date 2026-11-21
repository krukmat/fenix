#!/usr/bin/env python3
"""Read-only workflow-level latency/token audit. Never logs raw prompts or patches."""
from __future__ import annotations

import argparse
import json
import math
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any


def _read_jsonl(directory: Path):
    for path in sorted(directory.glob("*.jsonl")) if directory.is_dir() else ():
        if path.is_symlink():
            continue
        try:
            with path.open(encoding="utf-8") as stream:
                for line in stream:
                    try:
                        item = json.loads(line)
                    except (ValueError, TypeError):
                        continue
                    if isinstance(item, dict):
                        yield item
        except (OSError, UnicodeError):
            continue


def _read_review_artifacts(directory: Path):
    for path in sorted(directory.glob("*.json")) if directory.is_dir() else ():
        if path.is_symlink():
            continue
        try:
            item = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, UnicodeError, ValueError):
            continue
        if isinstance(item, dict) and "verdict" in item:
            yield item


def _numeric(record: dict, key: str):
    value = record.get(key)
    return float(value) if type(value) in (float, int) and math.isfinite(value) and value >= 0 else None


def _percentile(values: list[float], percentile: float):
    if not values:
        return None
    ordered = sorted(values)
    return ordered[max(0, math.ceil(percentile * len(ordered)) - 1)]


def aggregate(gemma_dir: Path, peer_dir: Path) -> dict[str, Any]:
    local = list(_read_jsonl(gemma_dir))
    peer = list(_read_review_artifacts(peer_dir))
    role_counts = Counter(str(r.get("role", "unknown")) for r in local)
    outcomes = Counter(str(r.get("outcome", "unknown")) for r in local)
    per_model = Counter(str(r.get("model", "unknown")) for r in local)
    durations = [_numeric(r, "elapsed_s") for r in local]
    durations = [x for x in durations if x is not None]

    def token_metric(key):
        measured = [_numeric(r, key) for r in local]
        present = [int(x) for x in measured if x is not None]
        return {
            "records_with_data": len(present),
            "coverage_ratio": round(len(present) / len(local), 3) if local else None,
            "total": sum(present) if present else None,
        }

    review_stage = Counter()
    review_attempt_roles = Counter()
    review_verdicts = Counter()
    missing_attempts = 0
    for rec in peer:
        review_stage[str(rec.get("mode", "unknown"))] += 1
        verdict = rec.get("verdict", {})
        review_verdicts[str(verdict.get("status", "unknown")) if isinstance(verdict, dict) else "invalid"] += 1
        attempts = rec.get("attempts")
        if not isinstance(attempts, list):
            missing_attempts += 1
            continue
        for a in attempts:
            if isinstance(a, dict):
                review_attempt_roles[str(a.get("role", "unknown"))] += 1

    return {
        "schema": "fenix-workflow-audit-v1",
        "status": "data_present" if local or peer else "no_records",
        "evidence": {"local_records": len(local), "peer_artifacts": len(peer)},
        "local": {
            "by_role": dict(sorted(role_counts.items())),
            "by_model": dict(sorted(per_model.items())),
            "outcomes": dict(sorted(outcomes.items())),
            "elapsed_seconds": {
                "sample_count": len(durations),
                "mean": round(sum(durations) / len(durations), 3) if durations else None,
                "p95": _percentile(durations, 0.95),
            },
            "prompt_tokens": token_metric("prompt_tokens"),
            "response_tokens": token_metric("response_tokens"),
        },
        "peer": {
            "by_stage": dict(sorted(review_stage.items())),
            "by_attempt_role": dict(sorted(review_attempt_roles.items())),
            "verdicts": dict(sorted(review_verdicts.items())),
            "artifacts_missing_attempts": missing_attempts,
        },
        "limitations": [
            "Results are aggregated observations, not matched end-to-end task traces.",
            "Missing token metrics are null, never estimated as zero.",
            "Raw prompts, patches and reviewer summaries are never exported.",
            "No source code changes, model calls, CI or QA gates are performed.",
        ],
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--gemma-log-dir", type=Path, default=Path("logs/gemma-audit"))
    parser.add_argument("--peer-log-dir", type=Path, default=Path("logs/peer-workflow-review"))
    parser.add_argument("--output", type=Path, default=None)
    args = parser.parse_args(argv)
    report = aggregate(args.gemma_log_dir, args.peer_log_dir)
    content = json.dumps(report, indent=2, sort_keys=True) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(content, encoding="utf-8")
    else:
        print(content, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
