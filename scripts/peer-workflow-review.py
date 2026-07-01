#!/usr/bin/env python3
"""peer-workflow-review.py — Phase F provider-aware peer workflow review gate.

Single local interface used by the portable agent workflow to request an
independent peer review of a task, in one of two modes:

  * task-readiness   — before presenting a task card (reviews task + plan + card)
  * post-code-review — before closing a development task (reviews the diff)

Reviewer resolution follows the Phase F contract
(docs/playbooks/AGENT_WORKFLOW_GUIDE.md#peer-review):

    claude-code      -> codex
    codex            -> claude
    local-provider   -> claude
    remote-provider  -> claude
    unknown          -> claude

The gate FAILS CLOSED: it exits 0 only for a `pass` verdict. Any other verdict,
an invalid/unparseable verdict, an unavailable/unauthenticated peer CLI, or a
timeout writes a `blocked`/`needs_changes` artifact and exits nonzero. The caller
must revise the work or obtain an explicit user waiver rather than self-review.

This script never mutates repository state: reviewer adapters run in read-only /
non-mutating mode and reviewer-generated patches are never applied.

Policy reference: docs/policies/HITL_AUTONOMY_POLICY.md (peer review does not
replace RRI/HITL human approval).

Run: python3 scripts/peer-workflow-review.py task-readiness --caller-kind ...
     python3 scripts/peer-workflow-review.py post-code-review --caller-kind ...
"""

from __future__ import annotations

import argparse
import datetime
import json
import os
import re
import subprocess
import sys
from typing import Dict, List, Optional

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gemma_local

VALID_CALLER_KINDS = {
    "claude-code",
    "codex",
    "local-provider",
    "remote-provider",
    "unknown",
}

VALID_STATUSES = {"pass", "needs_changes", "out_of_scope", "blocked"}

DEFAULT_OUT_DIR = os.environ.get(
    "FENIX_PEER_REVIEW_DIR", "logs/peer-workflow-review"
)
DEFAULT_TIMEOUT = int(os.environ.get("FENIX_PEER_REVIEW_TIMEOUT", "180"))
LOCAL_FALLBACK_REVIEWER = "local-gemma"

REVIEW_INSTRUCTIONS = (
    "You are an independent peer reviewer. Do not modify any files. Review the "
    "material below and reply with ONE JSON object and nothing else, matching:\n"
    '  {"status": "pass|needs_changes|out_of_scope|blocked", '
    '"summary": "<one line>", "findings": ["<finding>", ...]}\n'
    "Use \"pass\" only if the work is ready. Use \"needs_changes\" for required "
    "fixes, \"out_of_scope\" if the work exceeds the task contract, and "
    "\"blocked\" if you cannot review."
)


# --------------------------------------------------------------------------- #
# Errors — all map to a fail-closed blocked/needs_changes verdict.
# --------------------------------------------------------------------------- #
class ReviewerError(Exception):
    """Base class for reviewer failures. `reason` is stamped on the artifact."""

    reason = "reviewer_error"


class ReviewerUnavailable(ReviewerError):
    reason = "reviewer_unavailable"


class ReviewerTimeout(ReviewerError):
    reason = "reviewer_timeout"


class InvalidVerdict(ReviewerError):
    reason = "invalid_verdict"


# --------------------------------------------------------------------------- #
# Caller / reviewer resolution
# --------------------------------------------------------------------------- #
def normalize_caller_kind(kind, provider=None):
    # type: (Optional[str], Optional[str]) -> str
    """Return a canonical caller kind from a raw kind + optional provider."""
    if kind:
        low = kind.strip().lower()
        if low in VALID_CALLER_KINDS:
            return low
    if provider:
        prov = provider.strip().lower()
        if prov in ("claude-code", "codex"):
            return prov
    return "unknown"


def resolve_reviewer(kind):
    # type: (str) -> str
    """Resolve the independent reviewer for a caller kind (Phase F contract)."""
    if kind == "claude-code":
        return "codex"
    if kind == "codex":
        return "claude"
    return "claude"


def resolve_fallback_reviewer():
    # type: () -> str
    return LOCAL_FALLBACK_REVIEWER


def _env_first(*keys, default=None):
    # type: (*str, Optional[str]) -> Optional[str]
    for key in keys:
        value = os.environ.get(key)
        if value not in (None, ""):
            return value
    return default


# --------------------------------------------------------------------------- #
# Secret redaction
# --------------------------------------------------------------------------- #
_SECRET_KV = re.compile(
    r"(?i)("
    r"\b(?:token|api[_-]?key|apikey|key|password|passwd|secret|credential"
    r"|client[_-]?secret)\b\s*[:=]\s*)"
    r"(['\"]?)([^\s'\",]+)"
)
_SECRET_BEARER = re.compile(r"(?i)\b(bearer\s+)[A-Za-z0-9._\-]+")
_SECRET_PREFIX = re.compile(r"\b(sk|ghp|gho|xox[baprs]|AKIA)[A-Za-z0-9_\-]{8,}")

REDACTED = "[REDACTED]"


def redact_text(text):
    # type: (str) -> str
    """Mask token/key/password/secret/credential-like values in free text."""
    if not text:
        return text
    text = _SECRET_KV.sub(lambda m: m.group(1) + m.group(2) + REDACTED, text)
    text = _SECRET_BEARER.sub(lambda m: m.group(1) + REDACTED, text)
    text = _SECRET_PREFIX.sub(REDACTED, text)
    return text


def redact_packet(packet):
    # type: (dict) -> dict
    """Return a copy of the packet with every string value redacted."""
    out = {}  # type: Dict[str, object]
    for key, value in packet.items():
        if isinstance(value, str):
            out[key] = redact_text(value)
        else:
            out[key] = value
    return out


# --------------------------------------------------------------------------- #
# Packet building
# --------------------------------------------------------------------------- #
def _read_text(path):
    # type: (Optional[str]) -> str
    if not path:
        return ""
    try:
        with open(path, "r", encoding="utf-8") as fh:
            return fh.read()
    except OSError as err:
        return "<<unreadable: %s (%s)>>" % (path, err)


def read_diff(base):
    # type: (Optional[str]) -> str
    """Return the working-tree diff against `base` (read-only git call)."""
    if not base:
        return ""
    try:
        result = subprocess.run(
            ["git", "diff", "--no-color", "%s...HEAD" % base],
            capture_output=True,
            text=True,
        )
    except (OSError, subprocess.SubprocessError):
        return ""
    if result.returncode != 0:
        # Fall back to a plain diff against the ref (e.g. uncommitted work).
        try:
            result = subprocess.run(
                ["git", "diff", "--no-color", base],
                capture_output=True,
                text=True,
            )
        except (OSError, subprocess.SubprocessError):
            return ""
    return result.stdout


def build_packet(args, caller_kind, reviewer):
    # type: (argparse.Namespace, str, str) -> dict
    """Assemble the review packet for the requested mode (pre-redaction)."""
    packet = {
        "mode": args.mode,
        "caller_kind": caller_kind,
        "reviewer": reviewer,
        "task": _read_text(args.task),
        "plan": _read_text(args.plan),
    }  # type: Dict[str, object]
    if args.mode == "task-readiness":
        packet["task_card"] = _read_text(getattr(args, "task_card", None))
    else:
        packet["base_ref"] = args.base or ""
        packet["diff"] = read_diff(args.base)
        packet["verification_log"] = _read_text(
            getattr(args, "verification_log", None)
        )
    return packet


def build_prompt(mode, packet):
    # type: (str, dict) -> str
    """Render the review prompt sent to the peer adapter."""
    parts = [REVIEW_INSTRUCTIONS, "", "MODE: %s" % mode, ""]
    for section in ("task", "plan", "task_card", "verification_log", "diff"):
        if section in packet and packet[section]:
            parts.append("=== %s ===" % section.upper())
            parts.append(str(packet[section]))
            parts.append("")
    return "\n".join(parts)


def build_local_fallback_system_prompt():
    # type: () -> str
    return (
        "You are an independent local backup peer reviewer. The primary reviewer "
        "was already attempted and blocked. Do not modify any files. "
        + REVIEW_INSTRUCTIONS
    )


# --------------------------------------------------------------------------- #
# Reviewer adapters — command construction kept in isolated functions so tests
# can assert argv without invoking a real CLI.
# --------------------------------------------------------------------------- #
def claude_command(prompt):
    # type: (str) -> List[str]
    """Claude Code in non-mutating print mode requesting a JSON verdict."""
    return ["claude", "-p", prompt, "--output-format", "json"]


def codex_exec_command(prompt):
    # type: (str) -> List[str]
    """Codex readiness review: `codex exec` with read-only sandboxing."""
    return ["codex", "exec", "--sandbox", "read-only", prompt]


def codex_review_command(base, prompt):
    # type: (Optional[str], str) -> List[str]
    """Codex post-code review against a base ref with custom instructions."""
    cmd = ["codex", "review"]
    if base:
        cmd += ["--base", base]
    cmd += ["--instructions", prompt]
    return cmd


def _run(cmd, timeout):
    # type: (List[str], int) -> str
    try:
        result = subprocess.run(
            cmd, capture_output=True, text=True, timeout=timeout
        )
    except FileNotFoundError:
        raise ReviewerUnavailable("reviewer CLI not found: %s" % cmd[0])
    except subprocess.TimeoutExpired:
        raise ReviewerTimeout("reviewer timed out after %ss" % timeout)
    if result.returncode != 0:
        raise ReviewerUnavailable(
            "reviewer '%s' exited %d: %s"
            % (cmd[0], result.returncode, (result.stderr or "").strip())
        )
    return result.stdout


def invoke_reviewer(reviewer, mode, packet, base, timeout):
    # type: (str, str, dict, Optional[str], int) -> str
    """Dispatch to the correct adapter and return raw reviewer stdout."""
    prompt = build_prompt(mode, packet)
    if reviewer == "claude":
        return _run(claude_command(prompt), timeout)
    if reviewer == "codex":
        if mode == "task-readiness":
            return _run(codex_exec_command(prompt), timeout)
        return _run(codex_review_command(base, prompt), timeout)
    raise ReviewerUnavailable("unknown reviewer: %s" % reviewer)


def invoke_local_fallback_reviewer(mode, packet, timeout):
    # type: (str, dict, int) -> str
    host = _env_first("FENIX_OLLAMA_HOST", "OLLAMA_HOST", default=gemma_local.DEFAULT_HOST)
    model = _env_first(
        "FENIX_REVIEW_MODEL",
        "FENIX_LOW_RRI_MODEL",
        default=gemma_local.DEFAULT_MODEL,
    )
    idle_timeout = int(
        _env_first(
            "FENIX_REVIEW_IDLE_TIMEOUT_SECONDS",
            "FENIX_LOW_RRI_IDLE_TIMEOUT_SECONDS",
            default=str(gemma_local.DEFAULT_IDLE_TIMEOUT_SECONDS),
        )
    )
    max_wall = int(
        _env_first(
            "FENIX_REVIEW_MAX_WALL_SECONDS",
            "FENIX_LOW_RRI_MAX_WALL_SECONDS",
            default=str(gemma_local.DEFAULT_MAX_WALL_SECONDS),
        )
    )
    num_ctx = int(
        _env_first(
            "FENIX_REVIEW_NUM_CTX",
            "FENIX_LOW_RRI_NUM_CTX",
            default=str(gemma_local.DEFAULT_NUM_CTX),
        )
    )
    num_predict = int(
        _env_first(
            "FENIX_REVIEW_NUM_PREDICT",
            "FENIX_LOW_RRI_NUM_PREDICT",
            default=str(gemma_local.DEFAULT_NUM_PREDICT),
        )
    )
    temperature = float(
        _env_first(
            "FENIX_REVIEW_TEMPERATURE",
            "FENIX_LOW_RRI_TEMPERATURE",
            default=str(gemma_local.DEFAULT_TEMPERATURE),
        )
    )
    think = gemma_local.bool_from_env(
        "FENIX_REVIEW_THINK",
        gemma_local.bool_from_env("FENIX_LOW_RRI_THINK", gemma_local.DEFAULT_THINK),
    )

    try:
        gemma_local.ensure_model_available(host, model, idle_timeout)
        payload = gemma_local.build_chat_payload(
            model=model,
            system_prompt=build_local_fallback_system_prompt(),
            packet=build_prompt(mode, packet),
            num_ctx=num_ctx,
            num_predict=num_predict,
            temperature=temperature,
            think=think,
        )
        result = gemma_local.stream_chat(
            gemma_local.endpoint(host, "/api/chat"),
            payload,
            idle_timeout=idle_timeout,
            max_wall=max_wall,
            progress_label="peer-review-fallback",
        )
    except gemma_local.GemmaIdleTimeout as err:
        raise ReviewerTimeout(str(err))
    except gemma_local.GemmaWallTimeout as err:
        raise ReviewerTimeout(str(err))
    except RuntimeError as err:
        raise ReviewerUnavailable(str(err))
    return gemma_local.stream_result_content(result)


# --------------------------------------------------------------------------- #
# Verdict parsing
# --------------------------------------------------------------------------- #
def _load_json_relaxed(raw):
    # type: (str) -> Optional[object]
    if raw is None:
        return None
    try:
        return json.loads(raw)
    except (ValueError, TypeError):
        pass
    start = raw.find("{")
    end = raw.rfind("}")
    if start != -1 and end != -1 and end > start:
        try:
            return json.loads(raw[start : end + 1])
        except ValueError:
            return None
    return None


def parse_verdict(raw):
    # type: (str) -> dict
    """Parse a strict JSON verdict; raise InvalidVerdict on any deviation."""
    obj = _load_json_relaxed(raw)
    # Unwrap a `claude -p --output-format json` envelope: {"result": "..."}.
    if (
        isinstance(obj, dict)
        and "status" not in obj
        and isinstance(obj.get("result"), str)
    ):
        obj = _load_json_relaxed(obj["result"])
    if not isinstance(obj, dict):
        raise InvalidVerdict("verdict is not a JSON object")
    status = obj.get("status")
    if status not in VALID_STATUSES:
        raise InvalidVerdict("invalid or missing status: %r" % (status,))
    findings = obj.get("findings", [])
    if not isinstance(findings, list):
        findings = [str(findings)]
    return {
        "status": status,
        "summary": str(obj.get("summary", "")),
        "findings": [str(f) for f in findings],
    }


def blocked_verdict(reason, detail):
    # type: (str, str) -> dict
    """Build a fail-closed verdict for an unreachable/invalid review."""
    return {
        "status": "blocked",
        "summary": "peer review blocked: %s" % reason,
        "findings": [detail],
        "reason": reason,
    }


def should_attempt_fallback(verdict):
    # type: (dict) -> bool
    return verdict.get("status") == "blocked" and verdict.get("reason") in (
        "reviewer_timeout",
        "reviewer_unavailable",
    )


# --------------------------------------------------------------------------- #
# Artifact + reporting
# --------------------------------------------------------------------------- #
def _timestamp():
    # type: () -> str
    return datetime.datetime.now(datetime.timezone.utc).strftime(
        "%Y%m%dT%H%M%SZ"
    )


def write_artifact(out_dir, mode, caller_kind, reviewer, packet, verdict, attempts=None):
    # type: (str, str, str, str, dict, dict, Optional[list]) -> str
    """Write a redacted JSON review artifact and return its path."""
    os.makedirs(out_dir, exist_ok=True)
    record = {
        "mode": mode,
        "caller_kind": caller_kind,
        "reviewer": reviewer,
        "timestamp": _timestamp(),
        "verdict": verdict,
        "packet": redact_packet(packet),
    }
    if attempts is not None:
        record["attempts"] = attempts
    name = "%s_%s_by_%s_%s.json" % (
        mode,
        caller_kind,
        reviewer,
        record["timestamp"],
    )
    path = os.path.join(out_dir, name)
    with open(path, "w", encoding="utf-8") as fh:
        json.dump(record, fh, indent=2, sort_keys=True)
        fh.write("\n")
    return path


def print_summary(artifact, reviewer, verdict):
    # type: (str, str, dict) -> None
    status = verdict.get("status", "unknown")
    label = "PASS" if status == "pass" else "BLOCKED"
    print("Peer review: %s -> %s" % (reviewer, label))
    print("Status: %s" % status)
    if verdict.get("summary"):
        print("Summary: %s" % verdict["summary"])
    print("Artifact: %s" % artifact)


def _attempt_record(role, reviewer, verdict):
    # type: (str, str, dict) -> dict
    return {
        "role": role,
        "reviewer": reviewer,
        "verdict": verdict,
    }


# --------------------------------------------------------------------------- #
# CLI
# --------------------------------------------------------------------------- #
def _add_common(sub):
    # type: (argparse.ArgumentParser) -> None
    sub.add_argument("--caller-kind", default="unknown")
    sub.add_argument("--caller-provider", default=None)
    sub.add_argument("--task", required=True)
    sub.add_argument("--plan", required=True)
    sub.add_argument("--out-dir", default=DEFAULT_OUT_DIR)
    sub.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT)


def parse_args(argv):
    # type: (List[str]) -> argparse.Namespace
    parser = argparse.ArgumentParser(
        description="Provider-aware peer workflow review gate (Phase F)."
    )
    subparsers = parser.add_subparsers(dest="mode")
    subparsers.required = True

    readiness = subparsers.add_parser(
        "task-readiness", help="Peer-review a task before task-card presentation."
    )
    _add_common(readiness)
    readiness.add_argument("--task-card", default=None)

    postcode = subparsers.add_parser(
        "post-code-review", help="Peer-review a diff before task closure."
    )
    _add_common(postcode)
    postcode.add_argument("--base", required=True)
    postcode.add_argument("--verification-log", default=None)

    return parser.parse_args(argv)


def main(argv=None):
    # type: (Optional[List[str]]) -> int
    args = parse_args(sys.argv[1:] if argv is None else argv)
    caller_kind = normalize_caller_kind(args.caller_kind, args.caller_provider)
    primary_reviewer = resolve_reviewer(caller_kind)
    packet = build_packet(args, caller_kind, primary_reviewer)
    base = getattr(args, "base", None)
    attempts = []
    reviewer = primary_reviewer

    try:
        raw = invoke_reviewer(primary_reviewer, args.mode, packet, base, args.timeout)
        verdict = parse_verdict(raw)
    except ReviewerError as err:
        verdict = blocked_verdict(err.reason, str(err))
    attempts.append(_attempt_record("primary", primary_reviewer, verdict))

    if should_attempt_fallback(verdict):
        reviewer = resolve_fallback_reviewer()
        try:
            raw = invoke_local_fallback_reviewer(args.mode, packet, args.timeout)
            verdict = parse_verdict(raw)
        except ReviewerError as err:
            verdict = blocked_verdict(err.reason, str(err))
        attempts.append(_attempt_record("fallback", reviewer, verdict))

    artifact = write_artifact(
        args.out_dir, args.mode, caller_kind, reviewer, packet, verdict, attempts
    )
    print_summary(artifact, reviewer, verdict)
    return 0 if verdict.get("status") == "pass" else 1


if __name__ == "__main__":
    sys.exit(main())
