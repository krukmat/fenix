#!/usr/bin/env python3
"""Delegate a Low-RRI task packet to local Ollama/Gemma.

The script intentionally keeps Gemma sandboxed: it receives a packet and returns
tagged text with complete file contents. The orchestrating agent still validates,
builds the diff, applies it, reviews it, and verifies any patch.

Transport: streaming (`stream: true`).  Each NDJSON chunk resets the idle
timer (`--idle-timeout`, default 60 s).  A separate `--max-wall` cap (default
900 s) guards against runaway generation.  This lets the agent invoke the script
in the background (no 120 s Bash foreground limit) and be notified when it
completes naturally — rather than racing a blind wall-clock timeout against
Gemma's generation speed.

Invoke from an agent:
    scripts/delegate-low-rri.py packet.md --out result.json
The agent watches for completion (exit 0 -> read result.json) or timeout (124).

Environment variables:
  FENIX_OLLAMA_HOST                  Ollama base URL
  FENIX_LOW_RRI_MODEL                Model for delegation
  FENIX_LOW_RRI_IDLE_TIMEOUT_SECONDS Idle timeout in seconds
  FENIX_LOW_RRI_MAX_WALL_SECONDS     Wall-clock cap in seconds
  FENIX_LOW_RRI_NUM_CTX              Context window size
  FENIX_LOW_RRI_NUM_PREDICT          Max tokens to generate
  FENIX_LOW_RRI_TEMPERATURE          Sampling temperature
  FENIX_LOW_RRI_THINK                Enable thinking mode (truthy value)
"""

import argparse
import datetime
import fnmatch
import json
import os
import subprocess
import sys
import tempfile
import time
from urllib.error import URLError

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import gemma_local


DEFAULT_HOST = gemma_local.DEFAULT_HOST
DEFAULT_MODEL = gemma_local.DEFAULT_MODEL
DEFAULT_IDLE_TIMEOUT_SECONDS = gemma_local.DEFAULT_IDLE_TIMEOUT_SECONDS
DEFAULT_MAX_WALL_SECONDS = gemma_local.DEFAULT_MAX_WALL_SECONDS
DEFAULT_NUM_CTX = gemma_local.DEFAULT_NUM_CTX
DEFAULT_NUM_PREDICT = gemma_local.DEFAULT_NUM_PREDICT
DEFAULT_TEMPERATURE = gemma_local.DEFAULT_TEMPERATURE
DEFAULT_THINK = gemma_local.DEFAULT_THINK
DelegationIdleTimeout = gemma_local.GemmaIdleTimeout
DelegationWallTimeout = gemma_local.GemmaWallTimeout

STATUS_VALUES = {"PATCH": "patch", "NO_PATCH": "no_patch", "BLOCKED": "blocked"}
ACTION_VALUES = {"create", "modify", "delete"}
FILE_START_MARKER = "=== FILE START ==="
FILE_END_MARKER = "=== FILE END ==="
CONTENT_MARKER = "--- CONTENT ---"
REPLACEMENT_START_MARKER = "=== REPLACEMENT START ==="
REPLACEMENT_END_MARKER = "=== REPLACEMENT END ==="
AFTER_MARKER = "--- AFTER ---"


def parse_args():
    parser = argparse.ArgumentParser(
        description="Send a Low-RRI delegation packet to local Ollama/Gemma.",
    )
    parser.add_argument(
        "packet",
        nargs="?",
        help="Packet file to send. Reads stdin when omitted or set to '-'.",
    )
    parser.add_argument(
        "--host",
        default=os.environ.get("FENIX_OLLAMA_HOST", DEFAULT_HOST),
        help="Ollama host; defaults to FENIX_OLLAMA_HOST or %s." % DEFAULT_HOST,
    )
    parser.add_argument(
        "--model",
        default=os.environ.get("FENIX_LOW_RRI_MODEL", DEFAULT_MODEL),
        help=(
            "Local model name; defaults to FENIX_LOW_RRI_MODEL or %s." % DEFAULT_MODEL
        ),
    )
    parser.add_argument(
        "--idle-timeout",
        type=int,
        dest="idle_timeout",
        default=int(
            os.environ.get(
                "FENIX_LOW_RRI_IDLE_TIMEOUT_SECONDS",
                str(DEFAULT_IDLE_TIMEOUT_SECONDS),
            )
        ),
        help=(
            "Seconds without a new token before treating Gemma as stalled "
            "(exit 124); default %d." % DEFAULT_IDLE_TIMEOUT_SECONDS
        ),
    )
    parser.add_argument(
        "--max-wall",
        type=int,
        dest="max_wall",
        default=int(
            os.environ.get(
                "FENIX_LOW_RRI_MAX_WALL_SECONDS",
                str(DEFAULT_MAX_WALL_SECONDS),
            )
        ),
        help="Hard wall-time cap in seconds (exit 124); default %d." % DEFAULT_MAX_WALL_SECONDS,
    )
    parser.add_argument(
        "--num-ctx",
        type=int,
        dest="num_ctx",
        default=int(
            os.environ.get("FENIX_LOW_RRI_NUM_CTX", str(DEFAULT_NUM_CTX))
        ),
        help=(
            "Context window size for Ollama; default %d. "
            "Raise if the packet + tagged contract is truncated." % DEFAULT_NUM_CTX
        ),
    )
    parser.add_argument(
        "--num-predict",
        type=int,
        dest="num_predict",
        default=int(
            os.environ.get("FENIX_LOW_RRI_NUM_PREDICT", str(DEFAULT_NUM_PREDICT))
        ),
        help=(
            "Max tokens Ollama may generate; default %d. "
            "Raise when the response is truncated before the final FILE END marker."
            % DEFAULT_NUM_PREDICT
        ),
    )
    parser.add_argument(
        "--temperature",
        type=float,
        default=float(
            os.environ.get("FENIX_LOW_RRI_TEMPERATURE", str(DEFAULT_TEMPERATURE))
        ),
        help=(
            "Sampling temperature for Ollama; default %s. Raise slightly for harder "
            "local delegation tasks when the tagged contract remains stable."
            % DEFAULT_TEMPERATURE
        ),
    )
    parser.add_argument(
        "--think",
        action="store_true",
        default=gemma_local.bool_from_env("FENIX_LOW_RRI_THINK", DEFAULT_THINK),
        help=(
            "Enable Ollama thinking mode for models that support it. Defaults to "
            "off unless FENIX_LOW_RRI_THINK is truthy."
        ),
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
        help=(
            "Write the validated result JSON atomically to FILE instead of stdout. "
            "Recommended for agent invocations: the agent polls/reads FILE on exit 0."
        ),
    )
    parser.add_argument(
        "--allow-path",
        action="append",
        default=[],
        metavar="PATH",
        help=(
            "Repo-relative in-scope path prefix or glob; repeatable. Any file "
            "Gemma returns outside this set is rejected. Required for --apply."
        ),
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help=(
            "Build a unified diff from the returned file contents (via git) and "
            "apply it with `git apply`. Requires --allow-path. The diff is "
            "constructed deterministically by this script, never by the model."
        ),
    )
    parser.add_argument(
        "--repo-root",
        default=".",
        metavar="DIR",
        help="Repository root for scope checks and git apply; default cwd.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print the Ollama request payload without sending it.",
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
    parser.add_argument(
        "--mode",
        choices=["full-file", "before-after"],
        default="full-file",
        dest="mode",
        help=(
            "Delegation mode. 'full-file' (default): Gemma emits the complete file. "
            "'before-after': Gemma emits only the replacement block; the wrapper "
            "performs a literal find-and-replace on the target file. Use for files "
            "over ~400 lines. Requires --target-path and --before-file."
        ),
    )
    parser.add_argument(
        "--target-path",
        default=None,
        dest="target_path",
        metavar="PATH",
        help="Repo-relative path of the file to modify. Required for --mode before-after.",
    )
    parser.add_argument(
        "--before-file",
        default=None,
        dest="before_file",
        metavar="FILE",
        help=(
            "File containing the exact block to replace, copied verbatim from the "
            "current target file. Required for --mode before-after."
        ),
    )
    return parser.parse_args()


def read_packet(path):
    return gemma_local.read_packet(path)


def endpoint(host, path):
    return gemma_local.endpoint(host, path)


def get_json(url, timeout):
    try:
        return gemma_local.get_json(url, timeout)
    except gemma_local.GemmaIdleTimeout as exc:
        raise DelegationIdleTimeout(timeout) from exc


def ensure_model_available(host, model, timeout):
    return gemma_local.ensure_model_available(host, model, timeout)


def build_replacement_payload(model, packet, num_ctx, num_predict, temperature, think):
    system_prompt = (
        "You are a sandboxed local delegation model for fenix Low-RRI tasks.\n"
        "Return ONLY tagged text in this exact shape:\n"
        "STATUS: PATCH\n"
        "SUMMARY: short summary\n"
        "TEST: optional verification command\n"
        "RISK: optional risk note\n"
        "=== REPLACEMENT START ===\n"
        "PATH: relative/path.ext\n"
        "--- AFTER ---\n"
        "<replacement block only - not the full file>\n"
        "=== REPLACEMENT END ===\n"
        "Rules: use exactly one STATUS value: PATCH, NO_PATCH, or "
        "BLOCKED. Do not output the pipe-separated list. "
        "no JSON, no markdown fences, no unified diff, no "
        "explanations, no extra text outside these sections. "
        "Emit only the lines that replace the BEFORE block. "
        "For a deletion, emit nothing between --- AFTER --- and "
        "=== REPLACEMENT END ===."
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


def build_payload(model, packet, num_ctx, num_predict, temperature, think):
    system_prompt = (
        "You are a sandboxed local delegation model for fenix Low-RRI tasks.\n"
        "Return ONLY tagged text in this exact shape:\n"
        "STATUS: PATCH\n"
        "SUMMARY: short summary\n"
        "TEST: optional verification command\n"
        "RISK: optional risk note\n"
        "=== FILE START ===\n"
        "PATH: relative/path.ext\n"
        "ACTION: create|modify|delete\n"
        "--- CONTENT ---\n"
        "<COMPLETE final file contents>\n"
        "=== FILE END ===\n"
        "Rules: use exactly one STATUS value: PATCH, NO_PATCH, or "
        "BLOCKED. Do not output the pipe-separated list. "
        "no JSON, no markdown fences, no unified diff, no "
        "explanations, no extra text outside these sections. For "
        "ACTION delete, emit empty content."
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


def validate_delegation_payload(payload):
    required = ["status", "summary", "files", "test_commands", "risk_notes"]
    missing = [key for key in required if key not in payload]
    if missing:
        raise RuntimeError("Gemma response is missing required keys: %s" % missing)
    if payload["status"] not in {"patch", "no_patch", "blocked"}:
        raise RuntimeError("Gemma response has invalid status: %r" % payload["status"])
    if not isinstance(payload["files"], list):
        raise RuntimeError("Gemma response files must be an array")
    for i, entry in enumerate(payload["files"]):
        if not isinstance(entry, dict):
            raise RuntimeError("Gemma response files[%d] is not an object" % i)
        for key in ("path", "action", "contents"):
            if key not in entry:
                raise RuntimeError("Gemma response files[%d] missing %r" % (i, key))
        if entry["action"] not in {"create", "modify", "delete"}:
            raise RuntimeError(
                "Gemma response files[%d] has invalid action: %r" % (i, entry["action"])
            )
        if not isinstance(entry["path"], str) or not entry["path"].strip():
            raise RuntimeError(
                "Gemma response files[%d] path must be a non-empty string" % i
            )
        if not isinstance(entry["contents"], str):
            raise RuntimeError(
                "Gemma response files[%d] contents must be a string" % i
            )
    if payload["status"] == "patch" and not payload["files"]:
        raise RuntimeError("Gemma response status is 'patch' but files is empty")
    if not isinstance(payload["test_commands"], list):
        raise RuntimeError("Gemma response test_commands must be an array")
    if not isinstance(payload["risk_notes"], list):
        raise RuntimeError("Gemma response risk_notes must be an array")


def stream_chat(url, payload, idle_timeout, max_wall):
    try:
        return gemma_local.stream_chat(
            url,
            payload,
            idle_timeout=idle_timeout,
            max_wall=max_wall,
            progress_label="delegate",
        )
    except gemma_local.GemmaIdleTimeout as exc:
        raise DelegationIdleTimeout(idle_timeout) from exc
    except gemma_local.GemmaWallTimeout as exc:
        raise DelegationWallTimeout(max_wall) from exc


def parse_stream_content(content):
    content = gemma_local.normalize_tagged_content(content, "tagged")
    payload = parse_tagged_response(content)
    validate_delegation_payload(payload)
    return payload


def parse_replacement_response(content):
    """Parse a before-after mode response into a validated payload dict."""
    content = gemma_local.normalize_tagged_content(content, "replacement")

    lines = content.split("\n")
    status = None
    summary = None
    test_commands = []
    risk_notes = []
    path = None
    after_block = None
    replacement_block_count = 0
    idx = 0

    def skip_blank(i):
        while i < len(lines) and not lines[i].strip():
            i += 1
        return i

    idx = skip_blank(idx)
    while idx < len(lines):
        line = lines[idx]

        if line == REPLACEMENT_START_MARKER:
            replacement_block_count += 1
            if replacement_block_count > 1:
                raise RuntimeError(
                    "invalid replacement response: multiple replacement blocks"
                )
            path, after_block, idx = _parse_replacement_block(lines, idx)
            idx = skip_blank(idx)
            continue

        if line.startswith("STATUS: "):
            if status is not None:
                raise RuntimeError(
                    "invalid replacement response: duplicate STATUS header"
                )
            raw = line[len("STATUS: "):].strip()
            if raw not in STATUS_VALUES:
                raise RuntimeError(
                    "invalid replacement response: unknown STATUS %r" % raw
                )
            status = STATUS_VALUES[raw]
        elif line.startswith("SUMMARY: "):
            if summary is not None:
                raise RuntimeError(
                    "invalid replacement response: duplicate SUMMARY header"
                )
            summary = line[len("SUMMARY: "):].strip()
        elif line.startswith("TEST: "):
            test_commands.append(line[len("TEST: "):].strip())
        elif line.startswith("RISK: "):
            risk_notes.append(line[len("RISK: "):].strip())
        else:
            raise RuntimeError(
                "invalid replacement response: unexpected text outside sections: %r"
                % line
            )
        idx += 1
        idx = skip_blank(idx)

    if status is None:
        raise RuntimeError("invalid replacement response: missing STATUS header")
    if summary is None:
        raise RuntimeError("invalid replacement response: missing SUMMARY header")

    return {
        "status": status,
        "summary": summary,
        "test_commands": [c for c in test_commands if c],
        "risk_notes": [n for n in risk_notes if n],
        "path": path,
        "after_block": after_block,
    }


def _parse_replacement_block(lines, start_idx):
    idx = start_idx
    if lines[idx] != REPLACEMENT_START_MARKER:
        raise RuntimeError(
            "invalid replacement response: missing replacement start marker"
        )
    idx += 1

    while idx < len(lines) and not lines[idx].strip():
        idx += 1
    if idx >= len(lines):
        raise RuntimeError("invalid replacement response: missing PATH line")
    path_line = lines[idx]
    idx += 1

    path = _parse_header_value(path_line, "PATH")

    while idx < len(lines) and not lines[idx].strip():
        idx += 1
    if idx >= len(lines) or lines[idx] != AFTER_MARKER:
        raise RuntimeError(
            "invalid replacement response: missing --- AFTER --- marker"
        )
    idx += 1

    after_lines = []
    while idx < len(lines) and lines[idx] != REPLACEMENT_END_MARKER:
        after_lines.append(lines[idx])
        idx += 1

    if idx >= len(lines):
        raise RuntimeError(
            "invalid replacement response: missing replacement end marker "
            "(response may be truncated)"
        )
    idx += 1  # consume REPLACEMENT_END_MARKER

    after_block = "\n".join(after_lines)
    return path, after_block, idx


def validate_replacement_payload(payload, allow_paths, target_path):
    status = payload["status"]
    path = payload["path"]

    if status == "patch":
        if path is None:
            raise RuntimeError(
                "invalid replacement response: STATUS PATCH requires a replacement block"
            )
        norm_path = _normalize(path)
        norm_target = _normalize(target_path)
        if norm_path != norm_target:
            raise RuntimeError(
                "replacement response PATH %r does not match --target-path %r"
                % (path, target_path)
            )
        if allow_paths:
            norm_allowed = [_normalize(a) for a in allow_paths]
            ok = any(
                norm_path == a
                or norm_path.startswith(a.rstrip("/") + "/")
                or fnmatch.fnmatch(norm_path, a)
                for a in norm_allowed
            )
            if not ok:
                raise RuntimeError(
                    "replacement response PATH %r is outside --allow-path %s"
                    % (path, allow_paths)
                )


def parse_tagged_response(content):
    lines = content.split("\n")
    idx = 0
    status = None
    summary = None
    test_commands = []
    risk_notes = []
    files = []
    seen_paths = set()

    def skip_blank(i):
        while i < len(lines) and not lines[i].strip():
            i += 1
        return i

    idx = skip_blank(idx)
    while idx < len(lines):
        line = lines[idx]
        if line == FILE_START_MARKER:
            block, idx = parse_file_block(lines, idx)
            path = block["path"]
            if path in seen_paths:
                raise RuntimeError(
                    "invalid tagged response: duplicate path %r" % path
                )
            seen_paths.add(path)
            files.append(block)
            idx = skip_blank(idx)
            continue
        if line.startswith("STATUS: "):
            if status is not None:
                raise RuntimeError("invalid tagged response: duplicate STATUS header")
            raw_status = line[len("STATUS: "):].strip()
            if raw_status not in STATUS_VALUES:
                raise RuntimeError(
                    "invalid tagged response: unknown STATUS %r" % raw_status
                )
            status = STATUS_VALUES[raw_status]
        elif line.startswith("SUMMARY: "):
            if summary is not None:
                raise RuntimeError("invalid tagged response: duplicate SUMMARY header")
            summary = line[len("SUMMARY: "):].strip()
        elif line.startswith("TEST: "):
            test_commands.append(line[len("TEST: "):].strip())
        elif line.startswith("RISK: "):
            risk_notes.append(line[len("RISK: "):].strip())
        else:
            raise RuntimeError(
                "invalid tagged response: unexpected text outside sections: %r" % line
            )
        idx += 1
        idx = skip_blank(idx)

    if status is None:
        raise RuntimeError("invalid tagged response: missing STATUS header")
    if summary is None:
        raise RuntimeError("invalid tagged response: missing SUMMARY header")
    return {
        "status": status,
        "summary": summary,
        "files": files,
        "test_commands": [cmd for cmd in test_commands if cmd],
        "risk_notes": [note for note in risk_notes if note],
    }


def parse_file_block(lines, start_idx):
    idx = start_idx
    if lines[idx] != FILE_START_MARKER:
        raise RuntimeError("invalid tagged response: missing file start marker")
    idx += 1
    path_line, idx = _next_nonempty_line(lines, idx, "PATH")
    action_line, idx = _next_nonempty_line(lines, idx, "ACTION")
    content_line, idx = _next_nonempty_line(lines, idx, CONTENT_MARKER)

    path = _parse_header_value(path_line, "PATH")
    action = _parse_header_value(action_line, "ACTION")
    if action not in ACTION_VALUES:
        raise RuntimeError("invalid tagged response: unknown ACTION %r" % action)
    if content_line != CONTENT_MARKER:
        raise RuntimeError("invalid tagged response: missing content marker")

    content_lines = []
    while idx < len(lines) and lines[idx] != FILE_END_MARKER:
        content_lines.append(lines[idx])
        idx += 1
    if idx >= len(lines):
        raise RuntimeError("invalid tagged response: missing file end marker")
    idx += 1
    contents = "\n".join(content_lines)
    if action == "delete" and contents:
        raise RuntimeError(
            "invalid tagged response: delete action requires empty content"
        )
    return {
        "path": path,
        "action": action,
        "contents": contents,
    }, idx


def _next_nonempty_line(lines, idx, label):
    return gemma_local.next_nonempty_line(
        lines,
        idx,
        label,
        "invalid tagged response",
    )


def _parse_header_value(line, label):
    return gemma_local.parse_header_value(line, label, "invalid tagged response")


def write_result(delegation, out_path):
    return gemma_local.write_result(delegation, out_path)


def _normalize(path):
    """Collapse a repo-relative path, rejecting absolute or parent-escaping paths."""
    norm = os.path.normpath(path)
    if os.path.isabs(norm) or norm.startswith(".."):
        raise RuntimeError("path escapes the repository: %r" % path)
    return norm


def enforce_scope(files, allowed):
    """Reject any file whose path is not within an allowed glob/prefix."""
    if not allowed:
        raise RuntimeError(
            "no --allow-path given; Low-RRI delegation requires an explicit "
            "in-scope path set before any file is written"
        )
    norm_allowed = [_normalize(a) for a in allowed]
    for entry in files:
        path = _normalize(entry["path"])
        ok = any(
            path == a
            or path.startswith(a.rstrip("/") + "/")
            or fnmatch.fnmatch(path, a)
            for a in norm_allowed
        )
        if not ok:
            raise RuntimeError(
                "out-of-scope path %r; allowed: %s" % (entry["path"], allowed)
            )


def validate_file_actions(files, repo_root):
    for entry in files:
        rel = _normalize(entry["path"])
        target = os.path.join(repo_root, rel)
        action = entry["action"]
        exists = os.path.exists(target)
        if action == "create" and exists:
            raise RuntimeError(
                "invalid file action: create targets existing path %r" % entry["path"]
            )
        if action == "modify" and not exists:
            raise RuntimeError(
                "invalid file action: modify targets missing path %r" % entry["path"]
            )
        if action == "delete" and not exists:
            raise RuntimeError(
                "invalid file action: delete targets missing path %r" % entry["path"]
            )


def build_diff(files, repo_root):
    """Build a unified diff from full file contents using git diff --no-index.

    git owns all hunk framing so the result always applies cleanly — sidestepping
    the model's diff-formatting weakness entirely.
    """
    parts = []
    for entry in files:
        rel = _normalize(entry["path"])
        target = os.path.join(repo_root, rel)
        action = entry["action"]

        if action == "delete":
            old = target if os.path.exists(target) else os.devnull
            new = os.devnull
            cleanup = None
        else:
            old = target if os.path.exists(target) else os.devnull
            tmp = tempfile.NamedTemporaryFile(
                mode="w", encoding="utf-8", suffix=".new", delete=False
            )
            cleanup = tmp.name
            try:
                tmp.write(entry["contents"])
                tmp.close()
            except Exception:
                os.unlink(cleanup)
                cleanup = None
                raise
            new = tmp.name

        try:
            out = subprocess.run(
                [
                    "git", "diff", "--no-index", "--no-color",
                    ("--src-prefix=a/%s/" % os.path.dirname(rel)).replace("//", "/"),
                    old, new,
                ],
                capture_output=True, text=True, cwd=repo_root,
            )
        finally:
            if cleanup:
                os.unlink(cleanup)

        diff = _relabel_diff(out.stdout, old, new, rel, action)
        if diff:
            parts.append(diff)
    return "\n".join(parts)


def _relabel_diff(diff_text, old, new, rel, action):
    if not diff_text.strip():
        return ""
    lines = diff_text.splitlines()
    out = []
    for line in lines:
        if line.startswith("diff --git"):
            out.append("diff --git a/%s b/%s" % (rel, rel))
        elif line.startswith("--- "):
            out.append("--- /dev/null" if old == os.devnull else "--- a/%s" % rel)
        elif line.startswith("+++ "):
            out.append("+++ /dev/null" if new == os.devnull else "+++ b/%s" % rel)
        elif line.startswith("old mode ") or line.startswith("new mode "):
            pass  # tempfile permission artifact — strip it
        else:
            out.append(line)
    return "\n".join(out) + "\n"


def apply_diff(diff_text, repo_root):
    """Validate then apply a unified diff with git. Raises on a check failure."""
    if not diff_text.strip():
        return "empty diff; nothing to apply"
    with tempfile.NamedTemporaryFile(
        mode="w", encoding="utf-8", suffix=".patch", delete=False
    ) as f:
        f.write(diff_text)
        patch = f.name
    try:
        check = subprocess.run(
            ["git", "apply", "--check", patch],
            capture_output=True, text=True, cwd=repo_root,
        )
        if check.returncode != 0:
            raise RuntimeError("git apply --check failed: %s" % check.stderr.strip())
        applied = subprocess.run(
            ["git", "apply", patch],
            capture_output=True, text=True, cwd=repo_root,
        )
        if applied.returncode != 0:
            raise RuntimeError("git apply failed: %s" % applied.stderr.strip())
    finally:
        os.unlink(patch)
    return "applied"


def apply_before_after(target_path, before_file, after_block, allow_paths, repo_root, do_apply):
    """Perform a literal find-and-replace on target_path using before_file and after_block."""
    abs_target = os.path.join(repo_root, _normalize(target_path))
    if not os.path.exists(abs_target):
        raise RuntimeError(
            "before-after target file not found: %r" % target_path
        )
    with open(abs_target, encoding="utf-8") as f:
        original = f.read()
    with open(before_file, encoding="utf-8") as f:
        before = f.read()

    def _strip_trailing(text):
        return "\n".join(line.rstrip() for line in text.splitlines())

    original_stripped = _strip_trailing(original)
    before_stripped = _strip_trailing(before)

    count = original_stripped.count(before_stripped)
    if count == 0:
        raise RuntimeError(
            "before-after: BEFORE block not found in target file; "
            "ensure the block is copied verbatim from the current file"
        )
    if count > 1:
        raise RuntimeError(
            "before-after: BEFORE block found %d times in target file; "
            "must match exactly once to avoid editing the wrong occurrence" % count
        )

    after_stripped = _strip_trailing(after_block)
    final_contents = original_stripped.replace(before_stripped, after_stripped, 1)
    if original.endswith("\n") and not final_contents.endswith("\n"):
        final_contents += "\n"

    files = [{"path": target_path, "action": "modify", "contents": final_contents}]
    enforce_scope(files, allow_paths)
    validate_file_actions(files, repo_root)
    diff = build_diff(files, repo_root)
    result = {"unified_diff": diff}
    print(
        "[delegate] before-after: built diff: %d lines" % len(diff.splitlines()),
        file=sys.stderr,
    )
    if do_apply:
        outcome = apply_diff(diff, repo_root)
        result["apply_result"] = outcome
        print("[delegate] %s" % outcome, file=sys.stderr)
    return result


def main():
    args = parse_args()
    packet = read_packet(args.packet).strip()
    if not packet:
        raise RuntimeError("delegation packet is empty")

    if args.apply and not args.allow_path:
        raise RuntimeError("--apply requires at least one --allow-path")

    if args.mode == "before-after":
        if not args.target_path:
            raise RuntimeError("--mode before-after requires --target-path")
        if not args.before_file:
            raise RuntimeError("--mode before-after requires --before-file")

    if args.mode == "before-after":
        payload = build_replacement_payload(
            args.model,
            packet,
            args.num_ctx,
            args.num_predict,
            args.temperature,
            args.think,
        )
    else:
        payload = build_payload(
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

    ensure_model_available(args.host, args.model, args.idle_timeout)
    wall_start = time.monotonic()
    stream_result = stream_chat(
        endpoint(args.host, "/api/chat"),
        payload,
        idle_timeout=args.idle_timeout,
        max_wall=args.max_wall,
    )
    content = gemma_local.stream_result_content(stream_result)
    usage = gemma_local.stream_result_usage(stream_result)

    if args.mode == "before-after":
        delegation = parse_replacement_response(content)
        validate_replacement_payload(delegation, args.allow_path, args.target_path)
        if delegation["status"] == "patch":
            patch_result = apply_before_after(
                target_path=args.target_path,
                before_file=args.before_file,
                after_block=delegation["after_block"] or "",
                allow_paths=args.allow_path,
                repo_root=args.repo_root,
                do_apply=args.apply,
            )
            delegation.update(patch_result)
    else:
        delegation = parse_stream_content(content)
        if args.allow_path and delegation["status"] == "patch":
            enforce_scope(delegation["files"], args.allow_path)
            validate_file_actions(delegation["files"], args.repo_root)
            diff = build_diff(delegation["files"], args.repo_root)
            delegation["unified_diff"] = diff
            print(
                "[delegate] built diff: %d lines from %d file(s)"
                % (len(diff.splitlines()), len(delegation["files"])),
                file=sys.stderr,
            )
            if args.apply:
                outcome = apply_diff(diff, args.repo_root)
                delegation["apply_result"] = outcome
                print("[delegate] %s" % outcome, file=sys.stderr)

    result_json = json.dumps(delegation, indent=2, sort_keys=True)
    if args.out:
        write_result(delegation, args.out)
        print("[delegate] result written to %s" % args.out, file=sys.stderr)
    else:
        print(result_json)

    unified_diff = delegation.get("unified_diff", "")
    diff_lines = unified_diff.splitlines()
    diff_added = sum(
        1 for ln in diff_lines if ln.startswith("+") and not ln.startswith("+++")
    )
    diff_removed = sum(
        1 for ln in diff_lines if ln.startswith("-") and not ln.startswith("---")
    )

    gemma_local.append_audit_log({
        "ts": datetime.datetime.utcnow().isoformat() + "Z",
        "role": "developer",
        "outcome": delegation["status"].upper(),
        "done_reason": "policy" if delegation["status"] == "blocked" else "stop",
        "mode": args.mode,
        "elapsed_s": round(time.monotonic() - wall_start, 3),
        "escalated": delegation["status"] == "blocked",
        "system_prompt": system_prompt,
        "user_prompt": user_prompt,
        "task_id": args.task_id,
        "rri": None,
        "band": None,
        "attempt": args.attempt,
        "disposition": None,
        "diff_added": diff_added,
        "diff_removed": diff_removed,
        "scope_violations": 0,
        "apply_result": delegation.get("apply_result", "skipped"),
        "verify_ok": None,
        "file_lines": None,
        "file_tokens_est": None,
        "packet_tokens_est": gemma_local.estimate_payload_tokens(payload),
        "response_tokens": usage.response_tokens,
    })

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (DelegationIdleTimeout, DelegationWallTimeout) as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(124)
    except URLError as exc:
        print("Gemma/Ollama request failed: %s" % exc, file=sys.stderr)
        raise SystemExit(2)
    except RuntimeError as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)
