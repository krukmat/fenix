#!/usr/bin/env python3
"""Read-only audit report tool for logs/gemma-audit/*.jsonl.

Emits per-role process metrics and D4 calibration signals (threshold-driven,
not gates). Tolerates null optional fields and skips malformed lines.
"""

import argparse
import json
import os
import sys
from pathlib import Path


# D4 thresholds (calibration signals only — not hard gates)
_THRESHOLD_ESCALATION_RATE = 0.20
_THRESHOLD_DESTRUCTIVE_REMOVED_RATIO = 3.0   # diff_removed / diff_added > 3×
_THRESHOLD_OUT_OF_SCOPE_RATE = 0.10
_THRESHOLD_DISMISSED_MAJOR_RATE = 0.50
_THRESHOLD_CONSENSUS_RATE = 0.50


def parse_args():
    parser = argparse.ArgumentParser(
        description="Emit per-role process metrics from logs/gemma-audit/*.jsonl.",
    )
    parser.add_argument(
        "--log-dir",
        default="logs/gemma-audit",
        metavar="DIR",
        help="Directory containing YYYY-MM.jsonl files (default: logs/gemma-audit).",
    )
    parser.add_argument(
        "--role",
        choices=["developer", "reviewer", "all"],
        default="all",
        help="Filter records by role (default: all).",
    )
    parser.add_argument(
        "--since",
        default=None,
        metavar="YYYY-MM",
        help="Include only files whose month stem >= YYYY-MM.",
    )
    parser.add_argument(
        "--format",
        choices=["text", "json"],
        default="text",
        dest="fmt",
        help="Output format (default: text).",
    )
    return parser.parse_args()


def load_records(log_dir, role_filter, since):
    log_path = Path(log_dir)
    if not log_path.is_dir():
        return [], 0

    records = []
    skipped = 0
    for jsonl_file in sorted(log_path.glob("*.jsonl")):
        month = jsonl_file.stem
        if since and month < since:
            continue
        with open(jsonl_file, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    record = json.loads(line)
                except json.JSONDecodeError:
                    skipped += 1
                    continue
                if role_filter != "all" and record.get("role") != role_filter:
                    continue
                records.append(record)
    return records, skipped


def _mean(values):
    values = [v for v in values if v is not None]
    return round(sum(values) / len(values), 3) if values else None


def _token_summary(records, field):
    values = [r.get(field) for r in records if r.get(field) is not None]
    return {
        "records_with_data": len(values),
        "sum": sum(values) if values else None,
        "mean": round(sum(values) / len(values), 3) if values else None,
    }


def compute_metrics(records):
    if not records:
        return {"total_records": 0}

    total = len(records)
    by_role = {}
    for r in records:
        role = r.get("role", "unknown")
        by_role[role] = by_role.get(role, 0) + 1

    outcome_counts = {}
    for r in records:
        outcome = r.get("outcome", "UNKNOWN")
        outcome_counts[outcome] = outcome_counts.get(outcome, 0) + 1

    escalated_count = sum(1 for r in records if r.get("escalated"))
    blocked_rate = escalated_count / total

    truncated_count = sum(1 for r in records if r.get("done_reason") == "length")
    truncation_rate = truncated_count / total

    elapsed_values = [r.get("elapsed_s") for r in records]
    mean_elapsed_s = _mean(elapsed_values)
    response_tokens = _token_summary(records, "response_tokens")
    packet_tokens_est = _token_summary(records, "packet_tokens_est")

    # Developer-specific metrics
    dev_records = [r for r in records if r.get("role") == "developer"]
    patch_records = [r for r in dev_records if r.get("outcome") == "PATCH"]
    destructive_diff_count = 0
    for r in patch_records:
        added = r.get("diff_added") or 0
        removed = r.get("diff_removed") or 0
        if added > 0 and removed / added > _THRESHOLD_DESTRUCTIVE_REMOVED_RATIO:
            destructive_diff_count += 1
        elif added == 0 and removed > 0:
            destructive_diff_count += 1

    mode_distribution = {}
    for r in dev_records:
        mode = r.get("mode", "unknown")
        mode_distribution[mode] = mode_distribution.get(mode, 0) + 1

    # Reviewer-specific metrics
    rev_records = [r for r in records if r.get("role") == "reviewer"]
    findings_by_severity = {"blocking": 0, "major": 0, "minor": 0, "nit": 0}
    total_findings = 0
    total_out_of_scope = 0
    for r in rev_records:
        fbs = r.get("findings_by_severity") or {}
        for sev in findings_by_severity:
            findings_by_severity[sev] += fbs.get(sev, 0)
        total_findings += r.get("findings_count") or 0
        total_out_of_scope += r.get("out_of_scope") or 0

    out_of_scope_rate = (
        total_out_of_scope / total_findings if total_findings > 0 else 0.0
    )

    # Inter-pass disagreement (records with pass-level fields from T6)
    consensus_records = [
        r for r in rev_records
        if r.get("consensus_count") is not None or r.get("pass_specific_count") is not None
    ]
    consensus_rate = None
    if consensus_records:
        total_c = sum((r.get("consensus_count") or 0) for r in consensus_records)
        total_ps = sum((r.get("pass_specific_count") or 0) for r in consensus_records)
        total_findings_multi = total_c + total_ps
        consensus_rate = (
            total_c / total_findings_multi if total_findings_multi > 0 else 1.0
        )

    # Threshold flags (D4 calibration signals)
    threshold_flags = []
    if blocked_rate > _THRESHOLD_ESCALATION_RATE:
        threshold_flags.append(
            f"escalation_rate={blocked_rate:.1%} > {_THRESHOLD_ESCALATION_RATE:.0%}"
        )
    if truncation_rate > 0:
        threshold_flags.append(
            f"truncation_rate={truncation_rate:.1%} > 0% — unsafe output sizing"
        )
    if destructive_diff_count > 0:
        threshold_flags.append(
            f"destructive_diff_count={destructive_diff_count} — silent corruption risk"
        )
    if total_findings > 0 and out_of_scope_rate > _THRESHOLD_OUT_OF_SCOPE_RATE:
        threshold_flags.append(
            f"out_of_scope_rate={out_of_scope_rate:.1%} > {_THRESHOLD_OUT_OF_SCOPE_RATE:.0%} — reviewer drift"
        )
    if (
        consensus_rate is not None
        and consensus_rate < _THRESHOLD_CONSENSUS_RATE
    ):
        threshold_flags.append(
            f"consensus_rate={consensus_rate:.1%} < {_THRESHOLD_CONSENSUS_RATE:.0%} — unstable review"
        )

    return {
        "total_records": total,
        "by_role": by_role,
        "outcome_counts": outcome_counts,
        "escalated_count": escalated_count,
        "blocked_rate": round(blocked_rate, 4),
        "truncated_count": truncated_count,
        "truncation_rate": round(truncation_rate, 4),
        "mean_elapsed_s": mean_elapsed_s,
        "mode_distribution": mode_distribution,
        "response_tokens": response_tokens,
        "packet_tokens_est": packet_tokens_est,
        "findings_by_severity": findings_by_severity,
        "total_findings": total_findings,
        "total_out_of_scope": total_out_of_scope,
        "out_of_scope_rate": round(out_of_scope_rate, 4),
        "consensus_rate": round(consensus_rate, 4) if consensus_rate is not None else None,
        "destructive_diff_count": destructive_diff_count,
        "threshold_flags": threshold_flags,
    }


def format_text(metrics, skipped):
    lines = []
    if metrics.get("total_records", 0) == 0:
        lines.append("no records found")
        if skipped:
            lines.append(f"  ({skipped} malformed lines skipped)")
        return "\n".join(lines)

    lines.append(f"total_records:      {metrics['total_records']}")
    if skipped:
        lines.append(f"malformed_skipped:  {skipped}")

    by_role = metrics.get("by_role", {})
    for role, count in sorted(by_role.items()):
        lines.append(f"  {role}:  {count}")

    lines.append("")
    lines.append("outcome_counts:")
    for outcome, count in sorted(metrics.get("outcome_counts", {}).items()):
        lines.append(f"  {outcome}: {count}")

    lines.append("")
    lines.append(f"escalated_count:    {metrics['escalated_count']}")
    lines.append(f"blocked_rate:       {metrics['blocked_rate']:.1%}")
    lines.append(f"truncated_count:    {metrics['truncated_count']}")
    lines.append(f"truncation_rate:    {metrics['truncation_rate']:.1%}")
    if metrics.get("mean_elapsed_s") is not None:
        lines.append(f"mean_elapsed_s:     {metrics['mean_elapsed_s']:.3f}")

    response_tokens = metrics.get("response_tokens", {})
    packet_tokens_est = metrics.get("packet_tokens_est", {})
    if response_tokens.get("records_with_data", 0) > 0 or packet_tokens_est.get("records_with_data", 0) > 0:
        lines.append("")
        lines.append("token_telemetry:")
        lines.append(
            f"  response_tokens:   records={response_tokens.get('records_with_data', 0)} "
            f"sum={response_tokens.get('sum')} mean={response_tokens.get('mean')}"
        )
        lines.append(
            f"  packet_tokens_est: records={packet_tokens_est.get('records_with_data', 0)} "
            f"sum={packet_tokens_est.get('sum')} mean={packet_tokens_est.get('mean')}"
        )

    if metrics.get("mode_distribution"):
        lines.append("")
        lines.append("mode_distribution (developer):")
        for mode, count in sorted(metrics["mode_distribution"].items()):
            lines.append(f"  {mode}: {count}")

    if metrics.get("total_findings", 0) > 0 or metrics.get("by_role", {}).get("reviewer", 0) > 0:
        lines.append("")
        lines.append("reviewer findings:")
        lines.append(f"  total:          {metrics['total_findings']}")
        fbs = metrics.get("findings_by_severity", {})
        for sev in ("blocking", "major", "minor", "nit"):
            lines.append(f"  {sev}:      {fbs.get(sev, 0)}")
        lines.append(f"  out_of_scope:   {metrics['total_out_of_scope']}")
        lines.append(f"  out_of_scope_rate: {metrics['out_of_scope_rate']:.1%}")
        if metrics.get("consensus_rate") is not None:
            lines.append(f"  consensus_rate: {metrics['consensus_rate']:.1%}")

    flags = metrics.get("threshold_flags", [])
    lines.append("")
    if flags:
        lines.append("threshold_flags (D4 calibration signals):")
        for flag in flags:
            lines.append(f"  ! {flag}")
    else:
        lines.append("threshold_flags:    none")

    return "\n".join(lines)


def format_json(metrics, skipped):
    output = dict(metrics)
    output["malformed_skipped"] = skipped
    return json.dumps(output, indent=2, sort_keys=True)


def main():
    args = parse_args()
    records, skipped = load_records(args.log_dir, args.role, args.since)
    metrics = compute_metrics(records)

    if args.fmt == "json":
        print(format_json(metrics, skipped))
    else:
        print(format_text(metrics, skipped))

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
