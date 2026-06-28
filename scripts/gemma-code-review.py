#!/usr/bin/env python3
"""Run a read-only local Gemma code review over a pre-built packet.

The caller builds the packet, including the diff to review. This wrapper owns
only transport, contract parsing, out-of-scope labeling, and result writing.
It never applies patches or approves task completion.
"""

import argparse
import datetime
import json
import os
import re
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gemma_local


STATUS_VALUES = {"PASS": "pass", "FINDINGS": "findings", "BLOCKED": "blocked"}
SEVERITY_VALUES = {"blocking", "major", "minor", "nit"}
FINDING_START_MARKER = "=== FINDING START ==="
FINDING_END_MARKER = "=== FINDING END ==="
PATCH_LIKE_PATTERNS = (
    "diff --git ",
    "--- ",
    "+++ ",
    "@@ ",
    "=== FILE START ===",
    "=== REPLACEMENT START ===",
    "ACTION: ",
)


def parse_args():
    parser = argparse.ArgumentParser(
        description="Send a read-only code review packet to local Ollama/Gemma.",
    )
    parser.add_argument(
        "packet",
        nargs="?",
        help="Packet file to send. Reads stdin when omitted or set to '-'.",
    )
    parser.add_argument(
        "--host",
        default=os.environ.get("OLLAMA_HOST", gemma_local.DEFAULT_HOST),
        help=f"Ollama host; defaults to OLLAMA_HOST or {gemma_local.DEFAULT_HOST}.",
    )
    parser.add_argument(
        "--model",
        default=os.environ.get(
            "FENIX_REVIEW_MODEL",
            os.environ.get("FENIX_LOW_RRI_MODEL", gemma_local.DEFAULT_MODEL),
        ),
        help=(
            "Review model; defaults to FENIX_REVIEW_MODEL, then "
            "FENIX_LOW_RRI_MODEL, then the repo local Gemma default."
        ),
    )
    parser.add_argument(
        "--idle-timeout",
        type=int,
        dest="idle_timeout",
        default=int(
            os.environ.get(
                "FENIX_REVIEW_IDLE_TIMEOUT_SECONDS",
                os.environ.get(
                    "FENIX_LOW_RRI_IDLE_TIMEOUT_SECONDS",
                    str(gemma_local.DEFAULT_IDLE_TIMEOUT_SECONDS),
                ),
            )
        ),
        help="Seconds without a new token before treating Gemma as stalled.",
    )
    parser.add_argument(
        "--max-wall",
        type=int,
        dest="max_wall",
        default=int(
            os.environ.get(
                "FENIX_REVIEW_MAX_WALL_SECONDS",
                os.environ.get(
                    "FENIX_LOW_RRI_MAX_WALL_SECONDS",
                    str(gemma_local.DEFAULT_MAX_WALL_SECONDS),
                ),
            )
        ),
        help="Hard wall-time cap in seconds.",
    )
    parser.add_argument(
        "--num-ctx",
        type=int,
        dest="num_ctx",
        default=int(
            os.environ.get(
                "FENIX_REVIEW_NUM_CTX",
                os.environ.get("FENIX_LOW_RRI_NUM_CTX", str(gemma_local.DEFAULT_NUM_CTX)),
            )
        ),
        help="Context window size for Ollama.",
    )
    parser.add_argument(
        "--num-predict",
        type=int,
        dest="num_predict",
        default=int(
            os.environ.get(
                "FENIX_REVIEW_NUM_PREDICT",
                os.environ.get(
                    "FENIX_LOW_RRI_NUM_PREDICT",
                    str(gemma_local.DEFAULT_NUM_PREDICT),
                ),
            )
        ),
        help="Max tokens Ollama may generate.",
    )
    parser.add_argument(
        "--temperature",
        type=float,
        default=float(
            os.environ.get(
                "FENIX_REVIEW_TEMPERATURE",
                os.environ.get(
                    "FENIX_LOW_RRI_TEMPERATURE",
                    str(gemma_local.DEFAULT_TEMPERATURE),
                ),
            )
        ),
        help="Sampling temperature for Ollama.",
    )
    parser.add_argument(
        "--think",
        action="store_true",
        default=gemma_local.bool_from_env(
            "FENIX_REVIEW_THINK",
            gemma_local.bool_from_env(
                "FENIX_LOW_RRI_THINK",
                gemma_local.DEFAULT_THINK,
            ),
        ),
        help="Enable Ollama thinking mode for models that support it.",
    )
    parser.add_argument(
        "--no-think",
        action="store_false",
        dest="think",
        help="Disable thinking mode for this invocation, overriding the environment.",
    )
    parser.add_argument(
        "--out",
        default=None,
        metavar="FILE",
        help="Write the validated result JSON atomically to FILE instead of stdout.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print the Ollama request payload without sending it.",
    )
    parser.add_argument(
        "--passes",
        type=int,
        default=int(os.environ.get("FENIX_REVIEW_PASSES", "3")),
        metavar="N",
        help="Number of sequential review passes (default: 3; env FENIX_REVIEW_PASSES).",
    )
    parser.add_argument(
        "--task-id",
        default=None,
        dest="task_id",
        metavar="ID",
        help="Optional task ID recorded in the audit log.",
    )
    parser.add_argument(
        "--attempt",
        type=int,
        default=None,
        metavar="N",
        help="Optional attempt number recorded in the audit log.",
    )
    return parser.parse_args()


def build_review_payload(model, packet, num_ctx, num_predict, temperature, think):
    system_prompt = (
        "You are Gemma Reviewer for Fenix. You are read-only.\n"
        "Review the supplied packet and diff. Do not approve, close tasks, "
        "modify files, emit patches, emit unified diffs, or output file bodies.\n"
        "Return ONLY tagged text in this exact shape:\n"
        "STATUS: PASS\n"
        "SUMMARY: short review summary\n"
        "=== FINDING START ===\n"
        "PATH: repo/relative/path.ext\n"
        "LINE: 123\n"
        "SEVERITY: blocking|major|minor|nit\n"
        "DETAIL: concrete bug or risk\n"
        "SUGGESTION: concise fix direction\n"
        "=== FINDING END ===\n"
        "Rules: use exactly one STATUS value: PASS, FINDINGS, or BLOCKED. "
        "Use PASS with no finding blocks when no issues are found. Use FINDINGS "
        "with one or more finding blocks. Use BLOCKED only when the packet is not "
        "reviewable. No markdown fences, no JSON, no diff, no patch, no extra text."
    )
    return gemma_local.build_chat_payload(
        model=model,
        system_prompt=system_prompt,
        packet=packet,
        num_ctx=num_ctx,
        num_predict=num_predict,
        temperature=temperature,
        think=think,
    )


def changed_paths_from_packet(packet):
    paths = set()
    for line in packet.splitlines():
        if line.startswith("diff --git "):
            match = re.match(r"diff --git a/(.+?) b/(.+)$", line)
            if match:
                paths.add(match.group(1))
                paths.add(match.group(2))
        elif line.startswith("+++ b/"):
            paths.add(line[len("+++ b/"):].strip())
        elif line.startswith("--- a/"):
            paths.add(line[len("--- a/"):].strip())
    paths.discard("/dev/null")
    return sorted(paths)


def parse_review_response(content, changed_paths):
    content = gemma_local.normalize_tagged_content(content, "review")
    for pattern in PATCH_LIKE_PATTERNS:
        if pattern in content:
            raise RuntimeError(
                f"invalid review response: patch-like output is forbidden ({pattern!r})"
            )

    lines = content.split("\n")
    status = None
    summary = None
    findings = []
    format_warnings = []
    idx = 0

    def warn(msg):
        print(f"[review] warning: {msg}", file=sys.stderr)
        format_warnings.append(msg)

    def skip_blank(i):
        while i < len(lines) and not lines[i].strip():
            i += 1
        return i

    idx = skip_blank(idx)
    while idx < len(lines):
        line = lines[idx]
        if line == FINDING_START_MARKER:
            finding, idx = parse_finding_block(lines, idx)
            finding["scope"] = (
                "in-scope" if finding["path"] in changed_paths else "out-of-scope"
            )
            findings.append(finding)
            idx = skip_blank(idx)
            continue
        if line.startswith("STATUS: ") or line.strip() in STATUS_VALUES:
            raw = (line[len("STATUS: "):] if line.startswith("STATUS: ") else line).strip()
            if raw not in STATUS_VALUES:
                raise RuntimeError(f"invalid review response: unknown STATUS {raw!r}")
            parsed_status = STATUS_VALUES[raw]
            if status is not None:
                if status != parsed_status:
                    raise RuntimeError("invalid review response: conflicting STATUS headers")
                warn(f"duplicate STATUS header repeated with same value {raw!r}, skipping")
                idx += 1
                idx = skip_blank(idx)
                continue
            if not line.startswith("STATUS: "):
                warn(f"bare STATUS value accepted (non-standard format): {line!r}")
            status = parsed_status
        elif line.startswith("SUMMARY: "):
            parsed_summary = line[len("SUMMARY: "):].strip()
            if summary is not None:
                if summary != parsed_summary:
                    raise RuntimeError("invalid review response: conflicting SUMMARY headers")
                warn(f"duplicate SUMMARY header repeated with same value {parsed_summary!r}, skipping")
                idx += 1
                idx = skip_blank(idx)
                continue
            summary = parsed_summary
        elif line == FINDING_END_MARKER:
            # stray end marker without a preceding start — model format artifact, skip
            warn(f"stray {FINDING_END_MARKER!r} outside finding block, skipping")
        else:
            raise RuntimeError(
                "invalid review response: unexpected text outside sections: "
                f"{line!r}"
            )
        idx += 1
        idx = skip_blank(idx)

    if status is None:
        raise RuntimeError("invalid review response: missing STATUS header")
    if summary is None:
        raise RuntimeError("invalid review response: missing SUMMARY header")
    if status == "pass" and findings:
        warn("STATUS PASS with findings coerced to FINDINGS")
        status = "findings"
    if status == "findings" and not findings:
        raise RuntimeError("invalid review response: STATUS FINDINGS requires findings")

    return {
        "status": status,
        "summary": summary,
        "changed_paths": changed_paths,
        "findings": findings,
        "format_warnings": format_warnings,
    }


def parse_finding_block(lines, start_idx):
    idx = start_idx + 1
    fields = {}
    while idx < len(lines) and lines[idx] != FINDING_END_MARKER:
        line = lines[idx]
        if not line.strip():
            idx += 1
            continue
        for label in ("PATH", "LINE", "SEVERITY", "DETAIL", "SUGGESTION"):
            prefix = f"{label}: "
            if line.startswith(prefix):
                key = label.lower()
                if key in fields:
                    raise RuntimeError(
                        f"invalid review response: duplicate {label} in finding"
                    )
                fields[key] = line[len(prefix):].strip()
                break
        else:
            raise RuntimeError(
                f"invalid review response: unexpected finding text: {line!r}"
            )
        idx += 1

    if idx >= len(lines):
        raise RuntimeError(
            "invalid review response: missing finding end marker "
            "(response may be truncated)"
        )
    idx += 1

    required = ["path", "line", "severity", "detail", "suggestion"]
    missing = [key for key in required if not fields.get(key)]
    if missing:
        raise RuntimeError(f"invalid review response: finding missing {missing}")
    if fields["severity"] not in SEVERITY_VALUES:
        raise RuntimeError(
            f"invalid review response: invalid severity {fields['severity']!r}"
        )
    try:
        fields["line"] = int(fields["line"])
    except ValueError as exc:
        raise RuntimeError("invalid review response: LINE must be an integer") from exc
    if fields["line"] < 1:
        raise RuntimeError("invalid review response: LINE must be >= 1")
    return fields, idx


def _pass_artifact_path(out_path, k):
    stem, ext = os.path.splitext(out_path)
    return f"{stem}.pass{k}{ext}"


def _empty_reconciliation():
    return {
        "consensus": [],
        "pass_specific": [],
        "severity_inconsistent": [],
        "location_inconsistent": [],
        "likely_false_positive": [],
        "consensus_count": 0,
        "pass_specific_count": 0,
        "severity_inconsistent_count": 0,
        "location_inconsistent_count": 0,
        "likely_false_positive_count": 0,
    }


def reconcile(pass_results, changed_paths):
    """Deterministic D8 reconciliation of ≥2 successful pass results.

    Returns an aggregate result dict with a reconciliation block. The top-level
    findings list contains only consensus findings (≥2 passes exact match).
    The reconciliation block contains all five classified finding sets.
    """
    import collections

    tagged = []
    for pass_idx, result in enumerate(pass_results):
        for f in result["findings"]:
            tagged.append({**f, "_pass_idx": pass_idx})

    if not tagged:
        statuses = {r["status"] for r in pass_results}
        agg_status = "findings" if "findings" in statuses else "pass"
        return {
            "status": agg_status,
            "summary": pass_results[0].get("summary", ""),
            "changed_paths": changed_paths,
            "findings": [],
            "reconciliation": _empty_reconciliation(),
        }

    # Step 1: exact consensus — same (path, line, severity) in ≥2 passes.
    exact_groups = collections.defaultdict(list)
    for f in tagged:
        exact_groups[(f["path"], f["line"], f["severity"])].append(f)

    consensus_tagged = []
    solo_tagged = []
    for group in exact_groups.values():
        if len(group) >= 2:
            consensus_tagged.append(group[0])
        else:
            solo_tagged.append(group[0])

    # Step 2: severity-inconsistent — same (path, line), different severity among solo.
    line_groups = collections.defaultdict(list)
    for f in solo_tagged:
        line_groups[(f["path"], f["line"])].append(f)

    severity_inconsistent_tagged = []
    remaining_solo = []
    for group in line_groups.values():
        severities = {f["severity"] for f in group}
        if len(severities) > 1 and len(group) >= 2:
            severity_inconsistent_tagged.extend(group)
        else:
            remaining_solo.extend(group)

    # Step 3: location-inconsistent — same path, line ±3, from different passes.
    path_groups = collections.defaultdict(list)
    for f in remaining_solo:
        path_groups[f["path"]].append(f)

    location_inconsistent_tagged = []
    truly_solo = []
    for findings in path_groups.values():
        if len(findings) < 2:
            truly_solo.extend(findings)
            continue
        sorted_f = sorted(findings, key=lambda x: x["line"])
        clustered = set()
        for i in range(len(sorted_f)):
            for j in range(i + 1, len(sorted_f)):
                fi, fj = sorted_f[i], sorted_f[j]
                if (abs(fi["line"] - fj["line"]) <= 3
                        and fi["_pass_idx"] != fj["_pass_idx"]):
                    clustered.add(i)
                    clustered.add(j)
        for i, f in enumerate(sorted_f):
            (location_inconsistent_tagged if i in clustered else truly_solo).append(f)

    # Step 4: pass_specific vs likely_false_positive.
    pass_specific_tagged = []
    likely_fp_tagged = []
    for f in truly_solo:
        (likely_fp_tagged if f.get("scope") == "out-of-scope" else pass_specific_tagged).append(f)

    def _clean(findings):
        return [{k: v for k, v in f.items() if k != "_pass_idx"} for f in findings]

    consensus = _clean(consensus_tagged)
    severity_inconsistent = _clean(severity_inconsistent_tagged)
    location_inconsistent = _clean(location_inconsistent_tagged)
    pass_specific = _clean(pass_specific_tagged)
    likely_false_positive = _clean(likely_fp_tagged)

    statuses = {r["status"] for r in pass_results}
    agg_status = "findings" if "findings" in statuses else "pass"

    return {
        "status": agg_status,
        "summary": pass_results[0].get("summary", ""),
        "changed_paths": changed_paths,
        "findings": consensus,
        "reconciliation": {
            "consensus": consensus,
            "pass_specific": pass_specific,
            "severity_inconsistent": severity_inconsistent,
            "location_inconsistent": location_inconsistent,
            "likely_false_positive": likely_false_positive,
            "consensus_count": len(consensus),
            "pass_specific_count": len(pass_specific),
            "severity_inconsistent_count": len(severity_inconsistent),
            "location_inconsistent_count": len(location_inconsistent),
            "likely_false_positive_count": len(likely_false_positive),
        },
    }


def main():
    args = parse_args()
    packet = gemma_local.read_packet(args.packet).strip()
    if not packet:
        raise RuntimeError("review packet is empty")

    payload = build_review_payload(
        args.model,
        packet,
        args.num_ctx,
        args.num_predict,
        args.temperature,
        args.think,
    )
    if args.dry_run:
        print(json.dumps(payload, indent=2, sort_keys=True))
        return 0

    system_prompt = payload["messages"][0]["content"]
    user_prompt = payload["messages"][1]["content"]
    changed_paths = changed_paths_from_packet(packet)

    gemma_local.ensure_model_available(args.host, args.model, args.idle_timeout)

    if args.passes == 1:
        wall_start = time.monotonic()
        stream_result = gemma_local.stream_chat(
            gemma_local.endpoint(args.host, "/api/chat"),
            payload,
            idle_timeout=args.idle_timeout,
            max_wall=args.max_wall,
            progress_label="review",
        )
        content = gemma_local.stream_result_content(stream_result)
        usage = gemma_local.stream_result_usage(stream_result)
        result = parse_review_response(content, changed_paths)

        if args.out:
            gemma_local.write_result(result, args.out)
            print(f"[review] result written to {args.out}", file=sys.stderr)
        else:
            print(json.dumps(result, indent=2, sort_keys=True))

        findings_by_severity = {"blocking": 0, "major": 0, "minor": 0, "nit": 0}
        for f in result["findings"]:
            sev = f.get("severity", "")
            if sev in findings_by_severity:
                findings_by_severity[sev] += 1
        out_of_scope = sum(1 for f in result["findings"] if f.get("scope") == "out-of-scope")

        gemma_local.append_audit_log({
            "ts": datetime.datetime.utcnow().isoformat() + "Z",
            "role": "reviewer",
            "outcome": result["status"].upper(),
            "done_reason": "stop",
            "mode": "n/a",
            "elapsed_s": round(time.monotonic() - wall_start, 3),
            "escalated": result["status"] == "blocked",
            "system_prompt": system_prompt,
            "user_prompt": user_prompt,
            "task_id": args.task_id,
            "rri": None,
            "band": None,
            "attempt": args.attempt,
            "disposition": None,
            "findings_count": len(result["findings"]),
            "findings_by_severity": findings_by_severity,
            "out_of_scope": out_of_scope,
            "dispositions": None,
            "disposition_divergence": None,
            "format_warnings": result.get("format_warnings") or None,
            "file_lines": None,
            "file_tokens_est": None,
            "packet_tokens_est": gemma_local.estimate_payload_tokens(payload),
            "response_tokens": usage.response_tokens,
        })

        return 2 if result["status"] == "blocked" else 0

    # Multi-pass path (args.passes >= 2).
    wall_start = time.monotonic()
    pass_results = []
    usage_results = []
    per_call_packet_tokens = gemma_local.estimate_payload_tokens(payload)
    for k in range(1, args.passes + 1):
        format_retry_used = False
        for attempt in range(2):
            try:
                stream_result = gemma_local.stream_chat(
                    gemma_local.endpoint(args.host, "/api/chat"),
                    payload,
                    idle_timeout=args.idle_timeout,
                    max_wall=args.max_wall,
                    progress_label=f"review pass {k}/{args.passes}" + (" (retry)" if attempt else ""),
                )
                usage_results.append(gemma_local.stream_result_usage(stream_result))
                content = gemma_local.stream_result_content(stream_result)
                result = parse_review_response(content, changed_paths)
                if format_retry_used:
                    result["format_retry"] = True
                if args.out:
                    pass_out = _pass_artifact_path(args.out, k)
                    gemma_local.write_result(result, pass_out)
                    print(f"[review] pass {k} written to {pass_out}", file=sys.stderr)
                pass_results.append(("ok", result))
                break
            except RuntimeError as exc:
                msg = str(exc)
                if attempt == 0 and "STATUS PASS cannot include findings" in msg:
                    print(f"[review] pass {k} format error (STATUS PASS + findings), retrying: {exc}", file=sys.stderr)
                    format_retry_used = True
                    continue
                print(f"[review] pass {k} failed: {exc}", file=sys.stderr)
                pass_results.append(("fail", None))
                break
            except (gemma_local.GemmaIdleTimeout, gemma_local.GemmaWallTimeout) as exc:
                print(f"[review] pass {k} failed: {exc}", file=sys.stderr)
                pass_results.append(("fail", None))
                break

    succeeded = [r for status, r in pass_results if status == "ok"]
    all_format_warnings = [w for _, r in pass_results if r for w in r.get("format_warnings") or []]
    if len(succeeded) < 2:
        print(
            f"[review] quorum not met ({len(succeeded)}/{args.passes} passed)",
            file=sys.stderr,
        )
        return 3

    aggregate = reconcile(succeeded, changed_paths)
    aggregate["degraded"] = len(succeeded) < args.passes
    aggregate["passes_run"] = args.passes
    aggregate["passes_succeeded"] = len(succeeded)

    if args.out:
        gemma_local.write_result(aggregate, args.out)
        print(f"[review] aggregate written to {args.out}", file=sys.stderr)
    else:
        print(json.dumps(aggregate, indent=2, sort_keys=True))

    rec = aggregate.get("reconciliation", {})
    findings_by_severity = {"blocking": 0, "major": 0, "minor": 0, "nit": 0}
    for f in aggregate.get("findings", []):
        sev = f.get("severity", "")
        if sev in findings_by_severity:
            findings_by_severity[sev] += 1

    gemma_local.append_audit_log({
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
        "role": "reviewer",
        "outcome": aggregate["status"].upper(),
        "done_reason": "stop",
        "mode": "n/a",
        "elapsed_s": round(time.monotonic() - wall_start, 3),
        "escalated": aggregate["status"] == "blocked",
        "system_prompt": system_prompt,
        "user_prompt": user_prompt,
        "task_id": args.task_id,
        "rri": None,
        "band": None,
        "attempt": args.attempt,
        "disposition": None,
        "findings_count": len(aggregate.get("findings", [])),
        "findings_by_severity": findings_by_severity,
        "out_of_scope": rec.get("likely_false_positive_count", 0),
        "dispositions": None,
        "disposition_divergence": None,
        "format_warnings": all_format_warnings or None,
        "passes_run": aggregate["passes_run"],
        "passes_succeeded": aggregate["passes_succeeded"],
        "degraded": aggregate["degraded"],
        "consensus_count": rec.get("consensus_count", 0),
        "pass_specific_count": rec.get("pass_specific_count", 0),
        "severity_inconsistent_count": rec.get("severity_inconsistent_count", 0),
        "likely_false_positive_count": rec.get("likely_false_positive_count", 0),
        "file_lines": None,
        "file_tokens_est": None,
        "packet_tokens_est": per_call_packet_tokens * len(usage_results),
        "response_tokens": gemma_local.sum_measured_tokens(
            usage.response_tokens for usage in usage_results
        ),
    })

    return 2 if aggregate["status"] == "blocked" else 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except gemma_local.GemmaIdleTimeout as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(exc.exit_code)
    except gemma_local.GemmaWallTimeout as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(exc.exit_code)
    except (RuntimeError, OSError) as exc:
        print(f"Gemma review failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
