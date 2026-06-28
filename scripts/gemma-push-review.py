#!/usr/bin/env python3
"""Orchestrate a post-pipeline GitHub push audit using local Ollama/Gemma.

T1: Resolves the latest completed GitHub Actions push run, collects available
run metadata, job logs, annotations, and artifacts, then builds a push-audit
packet for the dedicated Push Reviewer role.

T1B: Sends the packet to the local model in a single reflexive pass (think=true),
parses the response with a dedicated push-audit parser, and applies evidence
grounding: each finding is verified against the diff before advancing to T2.
Findings not grounded are tagged observe and routed to daily non-Gemma review.
A blocked artifact (with full run_context) is written on timeout, missing Ollama,
or parser rejection so a non-Gemma agent can take over without re-running the CLI.
"""

import argparse
import datetime
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gemma_local


SCHEMA_VERSION = 1
DOCS_ONLY_EXTENSIONS = frozenset({
    ".md", ".txt", ".rst", ".adoc", ".asciidoc",
    ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
    ".pdf",
})
DOCS_ONLY_PATH_PREFIXES = ("docs/", "README", "CHANGELOG", "LICENSE", "CONTRIBUTING")

# T1B constants
DEFAULT_NUM_CTX_PUSH_REVIEW = 32768  # higher than code-review; CI logs included in packet
GROUNDING_SLACK = 10  # lines of tolerance around hunk boundaries
PATCH_LIKE_PATTERNS = (
    "diff --git ",
    "--- ",
    "+++ ",
    "@@ ",
    "=== FILE START ===",
    "=== REPLACEMENT START ===",
    "ACTION: ",
)
PUSH_AUDIT_STATUS_VALUES = {"PASS": "pass", "FINDINGS": "findings", "BLOCKED": "blocked"}
PUSH_AUDIT_SEVERITY_VALUES = {"blocking", "major", "minor", "nit"}
FINDING_START_MARKER = "=== FINDING START ==="
FINDING_END_MARKER = "=== FINDING END ==="
DELEGATE_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "delegate-low-rri.py")
FULL_FILE_MAX_LINES = 400
PURE_LOW_CODE_EXTENSIONS = {
    ".py", ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
}
PURE_LOW_EDITORIAL_PREFIXES = (
    "docs/",
    ".github/workflows/",
    ".github/actions/",
)
PURE_LOW_EDITORIAL_FILES = {
    "AGENTS.md", "CLAUDE.md", "README.md", "DESIGN.md", "Makefile",
}
PURE_LOW_HIGH_IMPACT_SEGMENTS = {
    "auth", "security", "rights", "rights-ledger", "ledger", "schema",
    "migrations", "ownership", "governance", "policy", "policies",
    "workflow", "workflows", "adr", "adrs",
}


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def parse_args():
    parser = argparse.ArgumentParser(
        description="Collect GitHub push pipeline evidence and run a push-audit review.",
    )
    # T1: collection args
    parser.add_argument("--run-id", dest="run_id", default=None, metavar="ID",
        help="Explicit GitHub Actions workflow run ID.")
    parser.add_argument("--workflow", default=None, metavar="NAME",
        help="Workflow name filter for automatic run resolution.")
    parser.add_argument("--branch", default=None, metavar="REF",
        help="Branch filter for automatic run resolution.")
    parser.add_argument("--before", default=None, metavar="SHA",
        help="Override/replay before-push SHA.")
    parser.add_argument("--after", default=None, metavar="SHA",
        help="Override/replay after-push SHA (head SHA).")
    parser.add_argument("--event-path", dest="event_path",
        default=os.environ.get("GITHUB_EVENT_PATH"), metavar="FILE",
        help="Path to GitHub push/workflow_run event JSON.")
    parser.add_argument("--out-dir", dest="out_dir", default=None, metavar="DIR",
        help="Directory to write push-audit artifacts.")
    parser.add_argument("--collect-only", dest="collect_only", action="store_true",
        help="Collect evidence and write the packet, then stop. No model invocation.")
    parser.add_argument("--dry-run", dest="dry_run", action="store_true",
        help="Print the assembled model payload without invoking Gemma or writing audit records.")
    parser.add_argument("--force", action="store_true",
        help="Continue even when all changed paths are docs-only.")

    # T1B: model-call args (FENIX env namespace)
    parser.add_argument("--host",
        default=os.environ.get("OLLAMA_HOST", gemma_local.DEFAULT_HOST),
        help=f"Ollama host (env OLLAMA_HOST, default {gemma_local.DEFAULT_HOST}).")
    parser.add_argument("--model",
        default=os.environ.get(
            "FENIX_PUSH_REVIEW_MODEL",
            os.environ.get("FENIX_LOW_RRI_MODEL", gemma_local.DEFAULT_MODEL),
        ),
        help="Push-review model (env FENIX_PUSH_REVIEW_MODEL -> FENIX_LOW_RRI_MODEL -> default).")
    parser.add_argument("--num-ctx", type=int, dest="num_ctx",
        default=int(os.environ.get(
            "FENIX_PUSH_REVIEW_NUM_CTX",
            os.environ.get("FENIX_LOW_RRI_NUM_CTX", str(DEFAULT_NUM_CTX_PUSH_REVIEW)),
        )),
        help="Context window size (env FENIX_PUSH_REVIEW_NUM_CTX, default 32768).")
    parser.add_argument("--num-predict", type=int, dest="num_predict",
        default=int(os.environ.get(
            "FENIX_PUSH_REVIEW_NUM_PREDICT",
            os.environ.get("FENIX_LOW_RRI_NUM_PREDICT", "8192"),
        )),
        help="Max tokens to generate (env FENIX_PUSH_REVIEW_NUM_PREDICT, default 8192).")
    parser.add_argument("--temperature", type=float,
        default=float(os.environ.get(
            "FENIX_PUSH_REVIEW_TEMPERATURE",
            os.environ.get("FENIX_LOW_RRI_TEMPERATURE", str(gemma_local.DEFAULT_TEMPERATURE)),
        )),
        help="Sampling temperature (env FENIX_PUSH_REVIEW_TEMPERATURE, default 0.1).")
    parser.add_argument("--think", action="store_true",
        default=gemma_local.bool_from_env(
            "FENIX_PUSH_REVIEW_THINK",
            gemma_local.bool_from_env("FENIX_LOW_RRI_THINK", True),
        ),
        help="Enable per-pass reflexion (default on for push-reviewer).")
    parser.add_argument("--no-think", action="store_false", dest="think",
        help="Disable think mode for this invocation.")
    parser.add_argument("--idle-timeout", type=int, dest="idle_timeout",
        default=int(os.environ.get(
            "FENIX_PUSH_REVIEW_IDLE_TIMEOUT_SECONDS",
            os.environ.get("FENIX_LOW_RRI_IDLE_TIMEOUT_SECONDS",
                           str(gemma_local.DEFAULT_IDLE_TIMEOUT_SECONDS)),
        )),
        help="Seconds without a token before treating Gemma as stalled.")
    parser.add_argument("--max-wall", type=int, dest="max_wall",
        default=int(os.environ.get(
            "FENIX_PUSH_REVIEW_MAX_WALL_SECONDS",
            os.environ.get("FENIX_LOW_RRI_MAX_WALL_SECONDS",
                           str(gemma_local.DEFAULT_MAX_WALL_SECONDS)),
        )),
        help="Hard wall-time cap in seconds.")

    return parser.parse_args()


# ---------------------------------------------------------------------------
# T1: GitHub run resolution and evidence collection
# ---------------------------------------------------------------------------

def _run_gh(args, *, check=True):
    try:
        result = subprocess.run(
            ["gh"] + args, capture_output=True, text=True, check=False,
        )
        if check and result.returncode != 0:
            raise RuntimeError(
                f"gh {' '.join(args[:3])} failed (rc={result.returncode}): "
                f"{result.stderr.strip()}"
            )
        return result.stdout, result.returncode
    except FileNotFoundError as exc:
        raise RuntimeError("gh CLI not found; install GitHub CLI to use this tool") from exc


def _load_event(event_path):
    if not event_path:
        return {}
    try:
        with open(event_path, encoding="utf-8") as fh:
            return json.load(fh)
    except (OSError, json.JSONDecodeError):
        return {}


def _first_present(raw, *keys):
    for key in keys:
        if key in raw and raw.get(key) is not None:
            return raw.get(key)
    return None


def resolve_run(args):
    event = _load_event(args.event_path)

    if event.get("action") in ("completed", "requested") and "workflow_run" in event:
        return _normalize_run(event["workflow_run"])

    if args.run_id:
        stdout, _ = _run_gh(
            ["run", "view", str(args.run_id), "--json",
             "databaseId,name,event,headBranch,headSha,attempt,status,conclusion,url,"
             "workflowName,createdAt,updatedAt"],
        )
        return _normalize_run(json.loads(stdout))

    extra = []
    if args.workflow:
        extra += ["--workflow", args.workflow]
    if args.branch:
        extra += ["--branch", args.branch]
    stdout, _ = _run_gh(
        ["run", "list", "--event", "push", "--status", "completed",
         "--limit", "5", "--json",
         "databaseId,name,event,headBranch,headSha,attempt,status,conclusion,url,"
         "workflowName,createdAt,updatedAt"] + extra,
    )
    runs = json.loads(stdout)
    if not runs:
        return {"_sentinel": "pipeline_unavailable"}
    return _normalize_run(runs[0])


def _normalize_run(raw):
    head_commit = raw.get("head_commit") or {}
    return {
        "run_id": _first_present(raw, "databaseId", "id"),
        "workflow_name": _first_present(raw, "workflowName", "workflow_name", "name"),
        "event": _first_present(raw, "event"),
        "branch": _first_present(raw, "headBranch", "head_branch"),
        "head_sha": _first_present(raw, "headSha", "head_sha") or head_commit.get("id"),
        "run_attempt": _first_present(raw, "runAttempt", "run_attempt", "attempt") or 1,
        "status": _first_present(raw, "status"),
        "conclusion": _first_present(raw, "conclusion"),
        "url": _first_present(raw, "url", "html_url", "htmlUrl"),
        "created_at": _first_present(raw, "createdAt", "created_at"),
        "updated_at": _first_present(raw, "updatedAt", "updated_at"),
    }


def collect_jobs(run_id):
    stdout, rc = _run_gh(["run", "view", str(run_id), "--json", "jobs"], check=False)
    if rc != 0:
        return [], True
    raw_jobs = json.loads(stdout).get("jobs", [])
    jobs = []
    for j in raw_jobs:
        steps = [
            {"name": s.get("name"), "status": s.get("status"),
             "conclusion": s.get("conclusion"), "number": s.get("number")}
            for s in j.get("steps", [])
        ]
        jobs.append({
            "name": j.get("name"),
            "status": j.get("status"),
            "conclusion": j.get("conclusion"),
            "started_at": j.get("startedAt"),
            "completed_at": j.get("completedAt"),
            "steps": steps,
            "failed_steps": [s for s in steps if s.get("conclusion") == "failure"],
        })
    return jobs, False


def collect_annotations(run_id):
    _, rc = _run_gh(["run", "view", str(run_id), "--json", "checkRunUrl"], check=False)
    return [], rc != 0


def collect_logs(run_id, jobs, out_dir):
    failed_jobs = [j for j in jobs if j.get("conclusion") == "failure"]
    if not failed_jobs:
        return [], False
    log_dir = os.path.join(out_dir, "logs")
    os.makedirs(log_dir, exist_ok=True)
    log_paths, partial = [], False
    for job in failed_jobs:
        job_name = re.sub(r"[^\w\-.]", "_", job.get("name", "unknown"))
        log_path = os.path.join(log_dir, f"{job_name}.log")
        stdout, rc = _run_gh(["run", "view", str(run_id), "--log-failed"], check=False)
        if rc != 0:
            partial = True
            continue
        safe = gemma_local._redact(stdout)
        try:
            with open(log_path, "w", encoding="utf-8") as fh:
                fh.write(safe)
            log_paths.append(log_path)
        except OSError:
            partial = True
    return log_paths, partial


def resolve_shas(run_info, event, args):
    before = args.before
    after = args.after or run_info.get("head_sha")
    if not before:
        # event["before"] exists in push events; workflow_run events have no "before" field.
        # head_commit.id in workflow_run payloads equals head_sha (the "after" SHA) — never use it here.
        before = event.get("before")
    if not before and after:
        try:
            result = subprocess.run(
                ["git", "log", "--pretty=%P", "-n", "1", after],
                capture_output=True, text=True, check=False,
            )
            parts = result.stdout.strip().split()
            before = parts[0] if parts else None
        except (FileNotFoundError, IndexError):
            before = None
    return before, after


def build_diff(before, after):
    if not before or not after:
        return ""
    try:
        result = subprocess.run(
            ["git", "diff", f"{before}..{after}"],
            capture_output=True, text=True, check=False,
        )
        if result.returncode != 0:
            return ""
        return gemma_local._redact(result.stdout)
    except FileNotFoundError:
        return ""


def changed_paths_from_diff(diff):
    paths = set()
    for line in diff.splitlines():
        if line.startswith("diff --git "):
            m = re.match(r"diff --git a/(.+?) b/(.+)$", line)
            if m:
                paths.add(m.group(1))
                paths.add(m.group(2))
        elif line.startswith("+++ b/"):
            paths.add(line[len("+++ b/"):].strip())
        elif line.startswith("--- a/"):
            paths.add(line[len("--- a/"):].strip())
    paths.discard("/dev/null")
    return sorted(paths)


def is_docs_only(paths):
    if not paths:
        return False
    for path in paths:
        _, ext = os.path.splitext(path)
        if ext.lower() in DOCS_ONLY_EXTENSIONS:
            continue
        if any(path.startswith(pfx) for pfx in DOCS_ONLY_PATH_PREFIXES):
            continue
        return False
    return True


def _short_sha(sha):
    return sha[:7] if sha and len(sha) >= 7 else (sha or "unknown")


def _out_dir_for(after_sha, out_dir):
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
        return out_dir
    date = datetime.datetime.utcnow().strftime("%Y-%m-%d")
    short = _short_sha(after_sha)
    base = os.path.join("logs", "gemma-push-review", date, short)
    os.makedirs(base, exist_ok=True)
    return base


def _detect_repo():
    try:
        result = subprocess.run(
            ["gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner"],
            capture_output=True, text=True, check=False,
        )
        repo = result.stdout.strip()
        if repo:
            return repo
    except FileNotFoundError:
        pass
    return "unknown/unknown"


def write_sentinel(kind, run_info, out_dir, after_sha):
    base = _out_dir_for(after_sha, out_dir)
    artifact = {
        "role": "gemma-push-reviewer",
        "schema_version": SCHEMA_VERSION,
        "sentinel": kind,
        "run_info": run_info,
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
    }
    path = os.path.join(base, f"{kind}.json")
    gemma_local.write_result(artifact, path)
    return path


def write_failure(message, out_dir, after_sha):
    base = _out_dir_for(after_sha, out_dir)
    artifact = {
        "role": "gemma-push-reviewer",
        "schema_version": SCHEMA_VERSION,
        "sentinel": "operational_failure",
        "error": message,
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
    }
    path = os.path.join(base, "operational_failure.json")
    gemma_local.write_result(artifact, path)
    return path


def build_packet(
    *,
    run_info,
    jobs,
    annotations,
    log_paths,
    artifact_paths,
    before_sha,
    after_sha,
    diff,
    changed_paths,
    pipeline_evidence_partial,
    logs_truncated,
    repo,
):
    return {
        "role": "gemma-push-reviewer",
        "schema_version": SCHEMA_VERSION,
        "repo": repo,
        "branch": run_info.get("branch"),
        "before": before_sha,
        "after": after_sha,
        "pipeline": {
            "workflow_name": run_info.get("workflow_name"),
            "run_id": run_info.get("run_id"),
            "run_attempt": run_info.get("run_attempt", 1),
            "event": run_info.get("event"),
            "status": run_info.get("status"),
            "conclusion": run_info.get("conclusion"),
            "url": run_info.get("url"),
            "jobs": jobs,
            "annotations_count": len(annotations),
            "log_paths": log_paths,
            "artifact_paths": artifact_paths,
        },
        "push": {
            "changed_paths": changed_paths,
            "diff": diff,
            "pipeline_evidence_partial": pipeline_evidence_partial,
            "logs_truncated": logs_truncated,
        },
        "audit": {
            "passes_run": 0,
            "passes_succeeded": 0,
            "quorum": "pending",
            "degraded": False,
            "aggregate_path": None,
        },
        "candidates": [],
        "developer_dispatch": {
            "attempted_count": 0,
            "succeeded_count": 0,
            "blocked_count": 0,
            "development_reports": [],
        },
        "post_development_review": {
            "required_count": 0,
            "in_review_count": 0,
            "pending_count": 0,
        },
        "deployer_followup": {
            "pure_low_dispatched_count": 0,
            "deferred_due_complexity_count": 0,
            "needs_hitl_count": 0,
        },
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
    }


# ---------------------------------------------------------------------------
# T1B: model invocation, parser, evidence grounding
# ---------------------------------------------------------------------------

def build_push_audit_system_prompt():
    return (
        "You are Gemma Push Reviewer for Fenix. You are read-only.\n"
        "Audit the supplied GitHub push evidence packet. Do not approve, close tasks, "
        "modify files, emit patches, emit unified diffs, or output file bodies.\n"
        "Return ONLY tagged text in this exact shape:\n"
        "STATUS: PASS\n"
        "SUMMARY: short push-audit summary\n"
        "=== FINDING START ===\n"
        "PATH: repo/relative/path.ext\n"
        "LINE: 123\n"
        "SEVERITY: blocking|major|minor|nit\n"
        "DETAIL: concrete risk introduced by this push or surfaced by the pipeline\n"
        "SUGGESTION: concise fix direction\n"
        "RRI_HINT: D=1 T=2 A=1 K=1 P=1 X=2 cc=6\n"
        "=== FINDING END ===\n"
        "Rules: use exactly one STATUS value: PASS, FINDINGS, or BLOCKED. "
        "Use PASS with no finding blocks when no issues are found. Use FINDINGS "
        "with one or more finding blocks. Use BLOCKED only when the packet is not "
        "auditable. RRI_HINT is optional and advisory only — never a final score. "
        "No markdown fences, no JSON, no diff, no patch, no file bodies. "
        "LINE must be a positive integer (the line number where the issue begins). "
        "If you cannot determine the exact line number, use 1."
    )


def _parse_rri_hint(hint_str):
    result = {}
    for part in hint_str.split():
        if "=" in part:
            k, _, v = part.partition("=")
            try:
                result[k] = int(v)
            except ValueError:
                result[k] = v
    return result


def parse_push_audit_response(content, changed_paths):
    """Dedicated push-audit parser. Not imported from gemma-code-review.py (D1a)."""
    content = gemma_local.normalize_tagged_content(content, "push-audit")
    for pattern in PATCH_LIKE_PATTERNS:
        if pattern in content:
            raise RuntimeError(
                f"invalid push-audit response: patch-like output is forbidden ({pattern!r})"
            )

    lines = content.split("\n")
    status = None
    summary = None
    findings = []
    format_warnings = []
    idx = 0

    def skip_blank(i):
        while i < len(lines) and not lines[i].strip():
            i += 1
        return i

    idx = skip_blank(idx)
    while idx < len(lines):
        line = lines[idx]
        if line == FINDING_START_MARKER:
            finding, idx = _parse_push_finding_block(lines, idx, format_warnings)
            finding["scope"] = (
                "in-scope" if finding["path"] in changed_paths else "out-of-scope"
            )
            findings.append(finding)
            idx = skip_blank(idx)
            continue
        if line.startswith("STATUS: ") or line.strip() in PUSH_AUDIT_STATUS_VALUES:
            if status is not None:
                raise RuntimeError("invalid push-audit response: duplicate STATUS header")
            if not line.startswith("STATUS: "):
                msg = f"bare STATUS value accepted (non-standard format): {line!r}"
                print(f"[push-audit] warning: {msg}", file=sys.stderr)
                format_warnings.append(msg)
            raw = (line[len("STATUS: "):] if line.startswith("STATUS: ") else line).strip()
            if raw not in PUSH_AUDIT_STATUS_VALUES:
                raise RuntimeError(f"invalid push-audit response: unknown STATUS {raw!r}")
            status = PUSH_AUDIT_STATUS_VALUES[raw]
        elif line.startswith("SUMMARY: "):
            if summary is not None:
                raise RuntimeError("invalid push-audit response: duplicate SUMMARY header")
            summary = line[len("SUMMARY: "):].strip()
        elif line == FINDING_END_MARKER:
            msg = f"stray {FINDING_END_MARKER!r} outside finding block, skipping"
            print(f"[push-audit] warning: {msg}", file=sys.stderr)
            format_warnings.append(msg)
        else:
            raise RuntimeError(
                f"invalid push-audit response: unexpected text outside sections: {line!r}"
            )
        idx += 1
        idx = skip_blank(idx)

    if status is None:
        raise RuntimeError("invalid push-audit response: missing STATUS header")
    if summary is None:
        raise RuntimeError("invalid push-audit response: missing SUMMARY header")
    if status == "pass" and findings:
        raise RuntimeError("invalid push-audit response: STATUS PASS cannot include findings")
    if status == "findings" and not findings:
        raise RuntimeError("invalid push-audit response: STATUS FINDINGS requires findings")

    return {
        "status": status,
        "summary": summary,
        "changed_paths": changed_paths,
        "findings": findings,
        "format_warnings": format_warnings,
    }


def _parse_push_finding_block(lines, start_idx, format_warnings=None):
    if format_warnings is None:
        format_warnings = []
    idx = start_idx + 1
    fields = {}
    while idx < len(lines) and lines[idx] != FINDING_END_MARKER:
        line = lines[idx]
        if not line.strip():
            idx += 1
            continue
        for label in ("PATH", "LINE", "SEVERITY", "DETAIL", "SUGGESTION", "RRI_HINT"):
            prefix = f"{label}: "
            if line.startswith(prefix):
                key = label.lower()
                if key in fields:
                    raise RuntimeError(
                        f"invalid push-audit response: duplicate {label} in finding"
                    )
                fields[key] = line[len(prefix):].strip()
                break
        else:
            raise RuntimeError(
                f"invalid push-audit response: unexpected finding text: {line!r}"
            )
        idx += 1

    if idx >= len(lines):
        raise RuntimeError(
            "invalid push-audit response: missing finding end marker "
            "(response may be truncated)"
        )
    idx += 1

    required = ["path", "line", "severity", "detail", "suggestion"]
    missing = [k for k in required if not fields.get(k)]
    if missing:
        raise RuntimeError(f"invalid push-audit response: finding missing {missing}")
    if fields["severity"] not in PUSH_AUDIT_SEVERITY_VALUES:
        raise RuntimeError(
            f"invalid push-audit response: invalid severity {fields['severity']!r}"
        )
    try:
        fields["line"] = int(fields["line"])
    except ValueError:
        # Gemma returned a non-integer (e.g. "N/A"); normalize to 1 so the finding
        # still reaches evidence grounding instead of blocking the entire response.
        msg = f"non-integer LINE value {fields['line']!r} normalized to 1"
        print(f"[push-audit] warning: {msg}", file=sys.stderr)
        format_warnings.append(msg)
        fields["line"] = 1
    if fields["line"] < 1:
        raise RuntimeError("invalid push-audit response: LINE must be >= 1")

    # RRI_HINT is advisory only — never a final score (D2)
    rri_hint_raw = fields.pop("rri_hint", None)
    fields["rri_input_proposal"] = _parse_rri_hint(rri_hint_raw) if rri_hint_raw else {}

    return fields, idx


def parse_diff_hunks(diff):
    """Return {path: [(start_line, end_line), ...]} from a unified diff."""
    hunks = {}
    current_path = None
    for line in diff.splitlines():
        if line.startswith("+++ b/"):
            current_path = line[len("+++ b/"):].strip()
            if current_path and current_path != "/dev/null":
                hunks.setdefault(current_path, [])
        elif line.startswith("@@ ") and current_path:
            m = re.search(r"\+(\d+)(?:,(\d+))?", line)
            if m:
                start = int(m.group(1))
                length = int(m.group(2) if m.group(2) is not None else 1)
                end = start + max(length - 1, 0)
                hunks[current_path].append((start, end))
    return hunks


def ground_findings(findings, diff, changed_paths, slack=GROUNDING_SLACK):
    """Tag each finding as evidence_grounded. Ungrounded findings get routing=observe."""
    hunks = parse_diff_hunks(diff)
    for f in findings:
        path_ok = f["path"] in changed_paths
        line_ok = any(
            start <= f["line"] <= end + slack
            for (start, end) in hunks.get(f["path"], [])
        )
        f["evidence_grounded"] = path_ok and line_ok
        if not f["evidence_grounded"]:
            f["routing"] = "observe"
    return findings


def write_blocked(reason, message, run_info, out_dir, after_sha):
    """Write a blocked artifact for model-invocation failures.

    Includes full run_context so the non-Gemma daily agent can reconstruct
    the push context without re-running the CLI.
    """
    base = _out_dir_for(after_sha, out_dir)
    artifact = {
        "role": "gemma-push-reviewer",
        "schema_version": SCHEMA_VERSION,
        "sentinel": "blocked",
        "blocked_reason": reason,
        "blocked_message": message,
        "run_context": {
            "run_id": run_info.get("run_id"),
            "head_sha": run_info.get("head_sha"),
            "branch": run_info.get("branch"),
            "conclusion": run_info.get("conclusion"),
            "url": run_info.get("url"),
        },
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
    }
    path = os.path.join(base, "blocked.json")
    gemma_local.write_result(artifact, path)
    return path


def run_push_audit(packet, run_info, args, out_dir, repo_root="."):
    """Single reflexive Gemma pass + evidence grounding (T1B). Returns exit code."""
    changed_paths = packet["push"]["changed_paths"]
    diff = packet["push"]["diff"]
    after_sha = packet.get("after") or run_info.get("head_sha", "unknown")
    short_sha = _short_sha(after_sha)

    payload = gemma_local.build_chat_payload(
        model=args.model,
        system_prompt=build_push_audit_system_prompt(),
        packet=json.dumps(packet),
        num_ctx=args.num_ctx,
        num_predict=args.num_predict,
        temperature=args.temperature,
        think=args.think,
    )

    if args.dry_run:
        # No model call, no audit record
        print(json.dumps(payload, indent=2, sort_keys=True))
        return 0

    system_prompt = payload["messages"][0]["content"]
    user_prompt = payload["messages"][1]["content"]

    try:
        gemma_local.ensure_model_available(args.host, args.model, args.idle_timeout)
    except RuntimeError as exc:
        path = write_blocked("ollama_unavailable", str(exc), run_info, out_dir, after_sha)
        write_blocked_report(path, _load_json(path), repo_root=repo_root)
        print(f"[push-audit] blocked (Ollama unavailable): {exc}", file=sys.stderr)
        print(f"[push-audit] non-Gemma agent should review this push manually", file=sys.stderr)
        print(f"[push-audit] blocked artifact: {path}", file=sys.stderr)
        return 2

    wall_start = time.monotonic()
    try:
        stream_result = gemma_local.stream_chat(
            gemma_local.endpoint(args.host, "/api/chat"),
            payload,
            idle_timeout=args.idle_timeout,
            max_wall=args.max_wall,
            progress_label="push-audit",
        )
    except gemma_local.GemmaIdleTimeout as exc:
        path = write_blocked("idle_timeout", str(exc), run_info, out_dir, after_sha)
        write_blocked_report(path, _load_json(path), repo_root=repo_root)
        print(f"[push-audit] blocked (idle timeout): {exc}", file=sys.stderr)
        print(f"[push-audit] non-Gemma agent should review this push manually", file=sys.stderr)
        print(f"[push-audit] blocked artifact: {path}", file=sys.stderr)
        return 2
    except gemma_local.GemmaWallTimeout as exc:
        path = write_blocked("wall_timeout", str(exc), run_info, out_dir, after_sha)
        write_blocked_report(path, _load_json(path), repo_root=repo_root)
        print(f"[push-audit] blocked (wall timeout): {exc}", file=sys.stderr)
        print(f"[push-audit] non-Gemma agent should review this push manually", file=sys.stderr)
        print(f"[push-audit] blocked artifact: {path}", file=sys.stderr)
        return 2
    except RuntimeError as exc:
        msg = str(exc) + " — increase FENIX_PUSH_REVIEW_NUM_PREDICT or review packet manually"
        path = write_blocked("stream_error", msg, run_info, out_dir, after_sha)
        write_blocked_report(path, _load_json(path), repo_root=repo_root)
        print(f"[push-audit] blocked (stream error): {exc}", file=sys.stderr)
        print(f"[push-audit] non-Gemma agent should review this push manually", file=sys.stderr)
        print(f"[push-audit] blocked artifact: {path}", file=sys.stderr)
        return 2

    elapsed = time.monotonic() - wall_start
    content = gemma_local.stream_result_content(stream_result)
    usage = gemma_local.stream_result_usage(stream_result)

    try:
        result = parse_push_audit_response(content, changed_paths)
    except RuntimeError as exc:
        reason = "patch_like_output" if "patch-like" in str(exc) else "parser_rejection"
        path = write_blocked(reason, str(exc), run_info, out_dir, after_sha)
        write_blocked_report(path, _load_json(path), repo_root=repo_root)
        print(f"[push-audit] blocked (parser): {exc}", file=sys.stderr)
        print(f"[push-audit] non-Gemma agent should review this push manually", file=sys.stderr)
        print(f"[push-audit] blocked artifact: {path}", file=sys.stderr)
        return 2

    # Evidence grounding — must happen before finding_id assignment
    result["findings"] = ground_findings(result["findings"], diff, changed_paths)

    fid = 1
    for f in result["findings"]:
        if f["evidence_grounded"]:
            f["finding_id"] = f"push-{short_sha}-F{fid:03d}"
            fid += 1

    grounded = [f for f in result["findings"] if f["evidence_grounded"]]
    observe = [f for f in result["findings"] if not f["evidence_grounded"]]

    # T2: canonical RRI scoring for each grounded finding
    pre_aggregate = {
        "role": "gemma-push-reviewer",
        "schema_version": SCHEMA_VERSION,
        "findings": result["findings"],
    }
    candidates = score_candidates(pre_aggregate, changed_paths)
    dispatch_summary = dispatch_pure_low_candidates(candidates, out_dir, repo_root=".")
    followup_counts = compute_followup_counts(candidates)

    aggregate = {
        "role": "gemma-push-reviewer",
        "schema_version": SCHEMA_VERSION,
        "repo": packet.get("repo"),
        "branch": packet.get("branch"),
        "before": packet.get("before"),
        "after": after_sha,
        "status": result["status"],
        "summary": result["summary"],
        "findings": result["findings"],
        "grounded_count": len(grounded),
        "observe_count": len(observe),
        "candidates": candidates,
        "candidates_scored_count": sum(1 for c in candidates if c.get("canonical_rri")),
        "candidates_pure_low_count": sum(1 for c in candidates if c.get("pure_low_eligible")),
        "changed_paths": changed_paths,
        "format_warnings": result.get("format_warnings") or None,
        "pipeline": packet["pipeline"],
        "audit": {
            "passes_run": 1,
            "passes_succeeded": 1,
            "quorum": "met",
            "degraded": False,
            "aggregate_path": None,
        },
        "developer_dispatch": {
            "attempted_count": dispatch_summary["attempted_count"],
            "succeeded_count": dispatch_summary["succeeded_count"],
            "blocked_count": dispatch_summary["blocked_count"],
            "development_reports": dispatch_summary["development_reports"],
        },
        "post_development_review": {
            "required_count": dispatch_summary["post_development_review_required_count"],
            "in_review_count": dispatch_summary["in_review_count"],
            "pending_count": dispatch_summary["pending_count"],
        },
        "deployer_followup": followup_counts,
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
    }

    aggregate_path, markdown_path = write_push_reports(
        aggregate,
        out_dir,
        repo_root=repo_root,
    )
    print(
        f"[push-audit] {len(grounded)} grounded, {len(observe)} observe, "
        f"{aggregate['candidates_scored_count']} scored -> {aggregate_path}",
        file=sys.stderr,
    )

    # D13 audit record — one per real run, after reconciliation
    gemma_local.append_audit_log({
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
        "role": "push-reviewer",
        "outcome": result["status"].upper(),
        "elapsed_s": round(elapsed, 3),
        "run_id": run_info.get("run_id"),
        "head_sha": run_info.get("head_sha"),
        "branch": run_info.get("branch"),
        "conclusion": run_info.get("conclusion"),
        "grounded_count": len(grounded),
        "observe_count": len(observe),
        "candidates_count": len(candidates),
        "candidates_scored_count": aggregate["candidates_scored_count"],
        "candidates_pure_low_count": aggregate["candidates_pure_low_count"],
        "system_prompt": system_prompt,
        "user_prompt": user_prompt,
        "markdown_summary_path": markdown_path,
        "file_tokens_est": None,
        "packet_tokens_est": gemma_local.estimate_payload_tokens(payload),
        "response_tokens": usage.response_tokens,
    })

    return 0


# ---------------------------------------------------------------------------
# T2: Canonical RRI scoring adapter
# ---------------------------------------------------------------------------

# Mapping from RRI_HINT keys to rri.py CLI flag names (D/K/P are judgment flags).
_RRI_HINT_FLAG_MAP = {
    "D": "--D", "K": "--K", "P": "--P",
    "T": "--T", "A": "--A", "X": "--X",
}

_BAND_TO_ROUTING = {
    "Low":      "gemma-developer-dispatch",
    "Moderate": "daily-non-gemma-review",
    "Med-high": "daily-non-gemma-review",
    "Complex":  "daily-non-gemma-review",
}


def _routing_from_band(label):
    return _BAND_TO_ROUTING.get(label, "daily-non-gemma-review")


def _normalize_repo_rel_path(path):
    norm = os.path.normpath(path or "")
    return "" if norm == "." else norm.replace("\\", "/")


def _is_editorial_or_workflow_path(path):
    norm = _normalize_repo_rel_path(path)
    if norm in PURE_LOW_EDITORIAL_FILES:
        return True
    return any(norm.startswith(prefix) for prefix in PURE_LOW_EDITORIAL_PREFIXES)


def _is_high_impact_path(path):
    norm = _normalize_repo_rel_path(path)
    parts = {segment.lower() for segment in Path(norm).parts}
    return bool(parts & PURE_LOW_HIGH_IMPACT_SEGMENTS)


def _is_code_or_test_path(path):
    norm = _normalize_repo_rel_path(path)
    return os.path.splitext(norm)[1].lower() in PURE_LOW_CODE_EXTENSIONS


def _read_repo_file(repo_root, rel_path):
    with open(os.path.join(repo_root, _normalize_repo_rel_path(rel_path)), encoding="utf-8") as fh:
        return fh.read()


def _write_text(path, content):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as fh:
        fh.write(content)


def _load_json(path):
    with open(path, encoding="utf-8") as fh:
        return json.load(fh)


def _report_date(ts=None):
    if ts:
        return ts[:10]
    return datetime.datetime.utcnow().strftime("%Y-%m-%d")


def _push_report_markdown_path(out_dir, after_sha, ts=None):
    filename = f"{_report_date(ts)}-{_short_sha(after_sha)}.md"
    return os.path.join(out_dir, "reports", filename)


def _repo_rel_or_raw(path, repo_root):
    if not path:
        return ""
    abs_path = os.path.abspath(path)
    abs_root = os.path.abspath(repo_root)
    try:
        rel = os.path.relpath(abs_path, abs_root)
    except ValueError:
        return path
    if rel.startswith(".."):
        return path
    return rel


def _audit_section_for_report(aggregate, aggregate_path):
    audit = dict(aggregate.get("audit") or {})
    audit.setdefault("passes_run", 1)
    audit.setdefault("passes_succeeded", 1)
    audit.setdefault("quorum", "met")
    audit.setdefault("degraded", False)
    audit["aggregate_path"] = aggregate_path
    return audit


def _render_table(headers, rows):
    if not rows:
        return ""
    lines = [
        "| " + " | ".join(headers) + " |",
        "|" + "|".join(["---"] * len(headers)) + "|",
    ]
    for row in rows:
        lines.append("| " + " | ".join(str(cell) for cell in row) + " |")
    return "\n".join(lines)


def _render_push_report_markdown(aggregate, *, repo_root):
    after_sha = aggregate.get("after") or "unknown"
    audit = aggregate.get("audit") or {}
    pipeline = aggregate.get("pipeline") or {}
    delegated_rows = []
    deferred_rows = []
    hitl_rows = []
    optimization_rows = []

    for candidate in aggregate.get("candidates", []):
        band = ((candidate.get("canonical_rri") or {}).get("band") or {}).get("label", "")
        finding = candidate.get("gemma_finding") or {}
        dispatch = candidate.get("developer_dispatch") or {}
        summary = finding.get("detail") or finding.get("suggestion") or ""
        path = finding.get("path", "")
        report_path = _repo_rel_or_raw(dispatch.get("development_report_path"), repo_root)

        if dispatch.get("status") in ("patched", "blocked"):
            delegated_rows.append([
                candidate.get("finding_id", ""),
                path,
                dispatch.get("status", ""),
                dispatch.get("review_status", ""),
                report_path or "n/a",
            ])

        if candidate.get("routing") == "daily-non-gemma-review":
            deferred_rows.append([
                candidate.get("finding_id", ""),
                band or "unscored",
                path,
                finding.get("severity", ""),
                "do not auto-apply" if band in ("Complex", "High", "Very high") else "non-Gemma review",
            ])

        if band in ("Moderate", "Med-high", "Complex", "High", "Very high"):
            hitl_rows.append([
                candidate.get("finding_id", ""),
                band,
                path,
                "decompose before implementation" if band in ("Complex", "High", "Very high")
                else "approval before implementation",
            ])

        if finding.get("severity") in ("minor", "nit"):
            optimization_rows.append([
                candidate.get("finding_id", ""),
                path,
                finding.get("severity", ""),
                summary,
            ])

    lines = [
        f"# Push Review Summary — {_report_date(aggregate.get('ts'))} {_short_sha(after_sha)}",
        "",
        "## Audit outcome",
        "",
        _render_table(
            ["Field", "Value"],
            [
                ["Status", aggregate.get("status", "")],
                ["Summary", aggregate.get("summary", "")],
                ["Quorum", audit.get("quorum", "")],
                ["Degraded", str(audit.get("degraded", False)).lower()],
                ["Passes", f"{audit.get('passes_succeeded', 0)}/{audit.get('passes_run', 0)}"],
                ["Branch", aggregate.get("branch", "")],
                ["Push range", f"{aggregate.get('before', '')} -> {aggregate.get('after', '')}"],
                ["Run ID", pipeline.get("run_id", "")],
                ["Conclusion", pipeline.get("conclusion", "")],
            ],
        ),
        "",
        "## Delegated development reports",
        "",
    ]

    if delegated_rows:
        lines.extend([
            _render_table(
                ["Finding", "Path", "Dispatch", "Review", "Development report"],
                delegated_rows,
            ),
            "",
        ])
    else:
        lines.extend(["none", ""])

    lines.extend(["## Non-Low deferred items", ""])
    if deferred_rows:
        lines.extend([
            _render_table(
                ["Finding", "Band", "Path", "Severity", "Action"],
                deferred_rows,
            ),
            "",
        ])
    else:
        lines.extend(["none", ""])

    lines.extend(["## HITL decisions", ""])
    if hitl_rows:
        lines.extend([
            _render_table(
                ["Finding", "Band", "Path", "Required next step"],
                hitl_rows,
            ),
            "",
        ])
    else:
        lines.extend(["none", ""])

    lines.extend(["## Optimizaciones y mejoras", ""])
    if optimization_rows:
        lines.extend([
            _render_table(
                ["Finding", "Path", "Severity", "Signal"],
                optimization_rows,
            ),
            "",
        ])
    else:
        lines.extend(["none", ""])

    return "\n".join(lines).rstrip() + "\n"


def write_push_reports(aggregate, out_dir, *, repo_root="."):
    after_sha = aggregate.get("after") or "unknown"
    aggregate_path = os.path.join(out_dir, "aggregate.json")
    markdown_path = _push_report_markdown_path(out_dir, after_sha, aggregate.get("ts"))

    aggregate["push_range"] = {
        "before": aggregate.get("before"),
        "after": aggregate.get("after"),
    }
    aggregate["audit"] = _audit_section_for_report(aggregate, aggregate_path)
    aggregate["reports"] = {
        "aggregate_json_path": aggregate_path,
        "markdown_summary_path": markdown_path,
    }

    gemma_local.write_result(aggregate, aggregate_path)
    _write_text(markdown_path, _render_push_report_markdown(aggregate, repo_root=repo_root))
    return aggregate_path, markdown_path


def _render_blocked_report_markdown(artifact, *, blocked_path, repo_root):
    ctx = artifact.get("run_context") or {}
    lines = [
        f"# Push Review Summary — {_report_date(artifact.get('ts'))} {_short_sha(ctx.get('head_sha'))}",
        "",
        "## Audit outcome",
        "",
        _render_table(
            ["Field", "Value"],
            [
                ["Status", "blocked"],
                ["Reason", artifact.get("blocked_reason", "")],
                ["Message", artifact.get("blocked_message", "")],
                ["Branch", ctx.get("branch", "")],
                ["Head SHA", ctx.get("head_sha", "")],
                ["Run ID", ctx.get("run_id", "")],
                ["Conclusion", ctx.get("conclusion", "")],
                ["Fallback packet", _repo_rel_or_raw(blocked_path, repo_root)],
            ],
        ),
        "",
        "## Non-Low deferred items",
        "",
        "none",
        "",
        "## HITL decisions",
        "",
        "non-Gemma agent must inspect the fallback packet before any implementation.",
        "",
    ]
    return "\n".join(lines).rstrip() + "\n"


def write_blocked_report(blocked_path, artifact, *, repo_root="."):
    ctx = artifact.get("run_context") or {}
    markdown_path = _push_report_markdown_path(
        os.path.dirname(blocked_path),
        ctx.get("head_sha"),
        artifact.get("ts"),
    )
    artifact["reports"] = {
        "blocked_json_path": blocked_path,
        "markdown_summary_path": markdown_path,
        "fallback_packet_path": blocked_path,
    }
    gemma_local.write_result(artifact, blocked_path)
    _write_text(
        markdown_path,
        _render_blocked_report_markdown(
            artifact,
            blocked_path=blocked_path,
            repo_root=repo_root,
        ),
    )
    return markdown_path


def _developer_dir(out_dir):
    path = os.path.join(out_dir, "developer")
    os.makedirs(path, exist_ok=True)
    return path


def _build_delegation_packet(candidate, repo_root):
    finding = candidate["gemma_finding"]
    rel_path = _normalize_repo_rel_path(finding["path"])
    if not rel_path:
        raise RuntimeError("candidate path is empty")
    if not _is_code_or_test_path(rel_path):
        raise RuntimeError(f"path {rel_path!r} is not a pure code/test file")
    if _is_editorial_or_workflow_path(rel_path):
        raise RuntimeError(f"path {rel_path!r} is editorial/workflow scope")
    if _is_high_impact_path(rel_path):
        raise RuntimeError(f"path {rel_path!r} is high-impact scope")

    current = _read_repo_file(repo_root, rel_path)
    line_count = len(current.splitlines())
    if line_count > FULL_FILE_MAX_LINES:
        raise RuntimeError(
            f"path {rel_path!r} has {line_count} lines; full-file Low-RRI packet not allowed"
        )

    canonical = candidate.get("canonical_rri") or {}
    final_rri = canonical.get("final")
    band_label = (canonical.get("band") or {}).get("label")
    detail = finding.get("detail", "").strip() or "Apply the requested narrow fix."
    suggestion = finding.get("suggestion", "").strip() or "Implement the smallest safe fix."
    finding_id = candidate.get("finding_id") or "push-unknown-F000"

    packet = "\n".join([
        f"# Low-RRI packet: {finding_id}",
        "",
        "## Goal",
        detail,
        suggestion,
        "",
        "## Allowed files",
        f"- {rel_path}",
        "",
        "## Do not change",
        "- Any file outside the allowed set.",
        "- Public behavior outside this finding's narrow scope.",
        "- Docs, plans, ledgers, workflow, policy, auth, schema, or ownership logic.",
        "",
        "## Acceptance criteria",
        f"- Modify only `{rel_path}`.",
        "- Return tagged text only; no JSON and no unified diff.",
        "- Keep the patch narrow and mechanical.",
        "- Preserve existing behavior outside the addressed finding.",
        "",
        "## RRI",
        "- Canonical source: `scripts/rri.py --json`",
        f"- Final RRI: `{final_rri}`",
        f"- Band: `{band_label}`",
        "",
        "## Stop condition",
        f"- Stop after the minimal in-scope patch for `{rel_path}`. Do not touch any other file.",
        "",
        "## Current file content",
        f"```text\n{current}\n```",
        "",
    ])
    return {
        "packet_text": packet,
        "allowed_paths": [rel_path],
    }


def _extract_changed_files_from_result(candidate, result_payload):
    if isinstance(result_payload.get("files"), list):
        paths = [entry.get("path") for entry in result_payload["files"] if entry.get("path")]
        if paths:
            return paths
    finding_path = candidate.get("gemma_finding", {}).get("path")
    return [finding_path] if finding_path else []


def _build_development_report(candidate, result_payload, dispatch_meta):
    return {
        "role": "gemma-push-reviewer",
        "schema_version": SCHEMA_VERSION,
        "finding_id": candidate.get("finding_id"),
        "canonical_rri": candidate.get("canonical_rri"),
        "allowed_paths": dispatch_meta["allowed_paths"],
        "packet_path": dispatch_meta["packet_path"],
        "result_path": dispatch_meta["result_path"],
        "delegate_exit_code": dispatch_meta["delegate_exit_code"],
        "developer_status": dispatch_meta["developer_status"],
        "delegate_status": result_payload.get("status"),
        "files_changed": _extract_changed_files_from_result(candidate, result_payload),
        "apply_result": result_payload.get("apply_result", "skipped"),
        "verification_intent": result_payload.get("test_commands", []),
        "repair_cycle_needed": False,
        "post_development_review_required": True,
        "review_status": "in_review",
        "review_method": "gemma-code-review-triple-quorum",
        "review_orchestrator": "non-gemma-agent",
        "risk_notes": result_payload.get("risk_notes", []),
        "summary": result_payload.get("summary"),
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
    }


def dispatch_pure_low_candidates(candidates, out_dir, repo_root="."):
    developer_dir = _developer_dir(out_dir)
    summary = {
        "attempted_count": 0,
        "succeeded_count": 0,
        "blocked_count": 0,
        "development_reports": [],
        "post_development_review_required_count": 0,
        "in_review_count": 0,
        "pending_count": 0,
    }

    for candidate in candidates:
        candidate.setdefault("developer_dispatch", {
            "status": "not_started",
            "result_path": None,
            "development_report_path": None,
            "post_development_review_required": False,
            "review_status": None,
            "review_method": None,
            "review_orchestrator": None,
        })

        if candidate.get("routing") != "gemma-developer-dispatch" or not candidate.get("pure_low_eligible"):
            continue

        try:
            packet_meta = _build_delegation_packet(candidate, repo_root)
        except (OSError, RuntimeError) as exc:
            candidate["routing"] = "daily-non-gemma-review"
            candidate["pure_low_eligible"] = False
            candidate["developer_dispatch"].update({
                "status": "blocked",
                "blocked_reason": str(exc),
            })
            continue

        summary["attempted_count"] += 1
        finding_id = candidate.get("finding_id", "push-unknown-F000")
        packet_path = os.path.join(developer_dir, f"{finding_id}-packet.md")
        result_path = os.path.join(developer_dir, f"{finding_id}-result.json")
        report_path = os.path.join(developer_dir, f"{finding_id}-development.json")
        _write_text(packet_path, packet_meta["packet_text"])

        cmd = [
            sys.executable, DELEGATE_SCRIPT, packet_path,
            "--out", result_path,
            "--repo-root", repo_root,
            "--task-id", finding_id,
            "--allow-path", packet_meta["allowed_paths"][0],
            "--apply",
        ]
        proc = subprocess.run(cmd, capture_output=True, text=True, check=False)

        result_payload = {
            "status": "blocked" if proc.returncode else "patch",
            "summary": proc.stderr.strip() or proc.stdout.strip() or "delegate invocation finished",
            "test_commands": [],
            "risk_notes": [proc.stderr.strip()] if proc.returncode and proc.stderr.strip() else [],
            "apply_result": "skipped",
        }
        if proc.returncode == 0 and os.path.exists(result_path):
            result_payload = _load_json(result_path)
        elif proc.returncode != 0 and not os.path.exists(result_path):
            gemma_local.write_result(result_payload, result_path)

        patched = result_payload.get("status") == "patch" and result_payload.get("apply_result") == "applied"
        developer_status = "patched" if patched else "blocked"
        report = _build_development_report(candidate, result_payload, {
            "allowed_paths": packet_meta["allowed_paths"],
            "packet_path": packet_path,
            "result_path": result_path,
            "delegate_exit_code": proc.returncode,
            "developer_status": developer_status,
        })
        gemma_local.write_result(report, report_path)

        candidate["developer_dispatch"].update({
            "status": developer_status,
            "result_path": result_path,
            "development_report_path": report_path,
            "post_development_review_required": True,
            "review_status": "in_review",
            "review_method": "gemma-code-review-triple-quorum",
            "review_orchestrator": "non-gemma-agent",
        })

        summary["development_reports"].append(report_path)
        summary["post_development_review_required_count"] += 1
        summary["in_review_count"] += 1
        summary["pending_count"] += 1

        if patched:
            summary["succeeded_count"] += 1
        else:
            candidate["routing"] = "daily-non-gemma-review"
            summary["blocked_count"] += 1

    return summary


def compute_followup_counts(candidates):
    return {
        "pure_low_dispatched_count": sum(
            1 for c in candidates
            if (c.get("developer_dispatch") or {}).get("status") == "patched"
        ),
        "deferred_due_complexity_count": sum(
            1 for c in candidates
            if c.get("canonical_rri")
            and (c["canonical_rri"].get("band") or {}).get("label") not in ("", "Low")
            and c.get("routing") == "daily-non-gemma-review"
        ),
        "needs_hitl_count": sum(
            1 for c in candidates
            if c.get("canonical_rri")
            and (c["canonical_rri"].get("band") or {}).get("label")
            in ("Moderate", "Med-high", "Complex", "High", "Very high")
        ),
    }


def _build_rri_cmd(path, proposal):
    """Build a rri.py --json invocation for a single candidate path.

    Proposal hints are passed as input to rri.py but rri.py output is
    always the canonical result (D2). The C dimension uses --C fallback
    because raw CC is not available without the source file.
    """
    cmd = [sys.executable, os.path.join(os.path.dirname(os.path.abspath(__file__)), "rri.py"),
           "--json", "--touches", path]
    cc = proposal.get("cc")
    if cc is not None and int(cc) >= 1:
        cmd += ["--cc", str(int(cc))]
    else:
        cmd += ["--C", str(int(proposal.get("C", 0)))]
    for hint_key, flag in _RRI_HINT_FLAG_MAP.items():
        val = proposal.get(hint_key)
        cmd += [flag, str(int(val)) if val is not None else "0"]
    return cmd


def score_candidates(aggregate, changed_paths):
    """Normalize grounded findings into candidate records with canonical RRI (T2).

    Only evidence_grounded findings are scored. Observe findings remain in
    aggregate["findings"] and are not included in the returned candidates list.
    Returns a list of candidate dicts.
    """
    findings = aggregate.get("findings", [])
    candidates = []

    for f in findings:
        if not f.get("evidence_grounded"):
            continue

        path = f.get("path", "")
        if path not in changed_paths:
            candidates.append({
                "finding_id": f.get("finding_id"),
                "gemma_finding": f,
                "rri_input_proposal": f.get("rri_input_proposal", {}),
                "routing": "dismiss-candidate",
                "rri_unavailable": False,
                "canonical_rri": None,
                "pure_low_eligible": False,
            })
            continue

        proposal = f.get("rri_input_proposal", {})
        cmd = _build_rri_cmd(path, proposal)

        try:
            result = subprocess.run(cmd, capture_output=True, text=True, check=False)
        except OSError as exc:
            candidates.append({
                "finding_id": f.get("finding_id"),
                "gemma_finding": f,
                "rri_input_proposal": proposal,
                "routing": "daily-non-gemma-review",
                "rri_unavailable": True,
                "rri_error": str(exc),
                "canonical_rri": None,
                "pure_low_eligible": False,
            })
            continue

        if result.returncode != 0:
            candidates.append({
                "finding_id": f.get("finding_id"),
                "gemma_finding": f,
                "rri_input_proposal": proposal,
                "routing": "daily-non-gemma-review",
                "rri_unavailable": True,
                "rri_error": result.stderr.strip(),
                "canonical_rri": None,
                "pure_low_eligible": False,
            })
            continue

        try:
            rri_raw = json.loads(result.stdout)
        except json.JSONDecodeError as exc:
            candidates.append({
                "finding_id": f.get("finding_id"),
                "gemma_finding": f,
                "rri_input_proposal": proposal,
                "routing": "daily-non-gemma-review",
                "rri_unavailable": True,
                "rri_error": f"rri.py JSON parse error: {exc}",
                "canonical_rri": None,
                "pure_low_eligible": False,
            })
            continue

        if not isinstance(rri_raw, dict):
            candidates.append({
                "finding_id": f.get("finding_id"),
                "gemma_finding": f,
                "rri_input_proposal": proposal,
                "routing": "daily-non-gemma-review",
                "rri_unavailable": True,
                "rri_error": f"rri.py returned non-dict JSON: {type(rri_raw).__name__}",
                "canonical_rri": None,
                "pure_low_eligible": False,
            })
            continue

        canonical = {
            "source": "scripts/rri.py --json",
            "final": rri_raw.get("final"),
            "band": rri_raw.get("band", {}),
            "raw": rri_raw,
        }
        band_label = canonical["band"].get("label", "")
        routing = _routing_from_band(band_label)
        pure_low_eligible = (
            band_label == "Low"
            and not rri_raw.get("penalties")
            and not rri_raw.get("triggers")
            and _is_code_or_test_path(path)
            and not _is_editorial_or_workflow_path(path)
            and not _is_high_impact_path(path)
        )

        candidates.append({
            "finding_id": f.get("finding_id"),
            "gemma_finding": f,
            "rri_input_proposal": proposal,
            "routing": routing,
            "rri_unavailable": False,
            "canonical_rri": canonical,
            "pure_low_eligible": pure_low_eligible,
        })

    return candidates


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    args = parse_args()

    try:
        run_info = resolve_run(args)
    except RuntimeError as exc:
        after_sha = args.after
        path = write_failure(str(exc), args.out_dir, after_sha)
        print(f"[push-review] run resolution failed: {exc}", file=sys.stderr)
        print(f"[push-review] operational failure written to {path}", file=sys.stderr)
        return 1

    sentinel = run_info.get("_sentinel")
    after_sha = run_info.get("head_sha") or args.after

    if sentinel == "pipeline_unavailable":
        path = write_sentinel("pipeline_unavailable", run_info, args.out_dir, after_sha)
        print(f"[push-review] no completed push run found; written to {path}", file=sys.stderr)
        return 0

    status = run_info.get("status")
    if status in ("queued", "in_progress", "waiting", "pending"):
        path = write_sentinel("pipeline_pending", run_info, args.out_dir, after_sha)
        print(f"[push-review] pipeline still {status!r}; written to {path}", file=sys.stderr)
        return 0

    out_dir = _out_dir_for(after_sha, args.out_dir)

    event = _load_event(args.event_path)
    before_sha, after_sha = resolve_shas(run_info, event, args)
    diff = build_diff(before_sha, after_sha)
    changed_paths = changed_paths_from_diff(diff)

    if changed_paths and is_docs_only(changed_paths) and not args.force:
        base = _out_dir_for(after_sha, args.out_dir)
        artifact = {
            "role": "gemma-push-reviewer",
            "schema_version": SCHEMA_VERSION,
            "sentinel": "audit_skipped",
            "reason": "docs_only",
            "changed_paths": changed_paths,
            "run_info": run_info,
            "ts": datetime.datetime.utcnow().isoformat() + "Z",
        }
        path = os.path.join(base, "audit_skipped.json")
        gemma_local.write_result(artifact, path)
        print(f"[push-review] docs-only push; skipped (use --force to override)", file=sys.stderr)
        return 0

    jobs, jobs_partial = collect_jobs(run_info["run_id"])
    annotations, ann_partial = collect_annotations(run_info["run_id"])
    log_paths, logs_partial = collect_logs(run_info["run_id"], jobs, out_dir)
    pipeline_evidence_partial = jobs_partial or ann_partial or logs_partial

    repo = _detect_repo()

    packet = build_packet(
        run_info=run_info,
        jobs=jobs,
        annotations=annotations,
        log_paths=log_paths,
        artifact_paths=[],
        before_sha=before_sha,
        after_sha=after_sha,
        diff=diff,
        changed_paths=changed_paths,
        pipeline_evidence_partial=pipeline_evidence_partial,
        logs_truncated=False,
        repo=repo,
    )

    packet_path = os.path.join(out_dir, "packet.json")
    gemma_local.write_result(packet, packet_path)
    print(f"[push-review] packet written to {packet_path}", file=sys.stderr)

    if args.collect_only:
        return 0

    return run_push_audit(packet, run_info, args, out_dir)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except RuntimeError as exc:
        print(f"[push-review] fatal: {exc}", file=sys.stderr)
        raise SystemExit(1)
    except KeyboardInterrupt:
        raise SystemExit(130)
