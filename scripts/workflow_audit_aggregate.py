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
    per_model = Counter(str(r.get("model") or "unknown") for r in local)
    durations = [_numeric(r, "elapsed_s") for r in local]
    durations = [x for x in durations if x is not None]

    def token_metric(key, records=None):
        dataset = local if records is None else records
        # Token counters are integers. Do not silently truncate malformed floats.
        present = [r[key] for r in dataset if type(r.get(key)) is int and r[key] >= 0]
        return {
            "records_with_data": len(present),
            "coverage_ratio": round(len(present) / len(dataset), 3) if dataset else None,
            "total": sum(present) if present else None,
        }

    groups = defaultdict(list)
    for record in local:
        groups[str(record.get("model") or "unknown")].append(record)
    model_metrics = {}
    for model, records in sorted(groups.items()):
        times = [_numeric(record, "elapsed_s") for record in records]
        times = [value for value in times if value is not None]
        model_metrics[model] = {
            "records": len(records),
            "mean_elapsed_seconds": round(sum(times) / len(times), 3) if times else None,
            "p95_elapsed_seconds": _percentile(times, 0.95),
            "prompt_tokens": token_metric("prompt_tokens", records),
            "response_tokens": token_metric("response_tokens", records),
        }

    reviewer_records = [r for r in local if r.get("role") == "reviewer"]
    measured_passes = [r["passes_run"] for r in reviewer_records
                       if type(r.get("passes_run")) is int and r["passes_run"] > 0]
    review_stage = Counter()
    review_attempt_roles = Counter()
    review_verdicts = Counter()
    missing_attempts = 0
    fallback_artifacts = 0
    for rec in peer:
        review_stage[str(rec.get("mode", "unknown"))] += 1
        verdict = rec.get("verdict", {})
        review_verdicts[str(verdict.get("status", "unknown")) if isinstance(verdict, dict) else "invalid"] += 1
        attempts = rec.get("attempts")
        if not isinstance(attempts, list):
            missing_attempts += 1
            continue
        if any(isinstance(a, dict) and a.get("role") == "fallback" for a in attempts):
            fallback_artifacts += 1
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
            "by_model_metrics": model_metrics,
            "outcomes": dict(sorted(outcomes.items())),
            "elapsed_seconds": {
                "sample_count": len(durations),
                "mean": round(sum(durations) / len(durations), 3) if durations else None,
                "p95": _percentile(durations, 0.95),
            },
            "prompt_tokens": token_metric("prompt_tokens"),
            "packet_tokens_est": token_metric("packet_tokens_est"),
            "response_tokens": token_metric("response_tokens"),
            "review_passes": {
                "review_records": len(reviewer_records),
                "records_with_data": len(measured_passes),
                "coverage_ratio": round(len(measured_passes) / len(reviewer_records), 3)
                                  if reviewer_records else None,
                "passes_total": sum(measured_passes) if measured_passes else None,
            },
        },
        "peer": {
            "by_stage": dict(sorted(review_stage.items())),
            "by_attempt_role": dict(sorted(review_attempt_roles.items())),
            "verdicts": dict(sorted(review_verdicts.items())),
            "artifacts_missing_attempts": missing_attempts,
            "fallback_artifacts": fallback_artifacts,
            "fallback_rate_observed": round(fallback_artifacts / (len(peer) - missing_attempts), 3)
                                       if len(peer) > missing_attempts else None,
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
    parser.add_argument("--w6-shadow-dir", type=Path, default=None,
                        help="Include separate W6 native shadow observations (opt-in).")
    args = parser.parse_args(argv)
    report = aggregate(args.gemma_log_dir, args.peer_log_dir)
    if args.w6_shadow_dir is not None:
        from workflow_shadow import aggregate as aggregate_shadow
        report["w6_shadow"] = aggregate_shadow(args.w6_shadow_dir)
    content = json.dumps(report, indent=2, sort_keys=True) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(content, encoding="utf-8")
    else:
        print(content, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
