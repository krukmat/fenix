"""Shared local Gemma/Ollama helpers for fenix agent roles.

Role-specific scripts own their prompt contract and parser. This module owns the
transport, common generation options, timeout behavior, packet IO, and atomic
result writing so developer and reviewer roles do not drift.

Environment variables (all optional):
  FENIX_OLLAMA_HOST                  Ollama base URL (default: http://localhost:11434)
  FENIX_LOW_RRI_MODEL                Model for delegation (default: gemma4:26b-a4b-it-qat)
  FENIX_LOW_RRI_IDLE_TIMEOUT_SECONDS Seconds without a token before idle timeout (default: 60)
  FENIX_LOW_RRI_MAX_WALL_SECONDS     Total wall-clock cap (default: 900)
"""

import datetime
import json
import os
import pathlib
import re
import socket
import sys
import time
import urllib.request
from dataclasses import dataclass
from typing import Optional


DEFAULT_HOST = os.environ.get("FENIX_OLLAMA_HOST", "http://localhost:11434")
DEFAULT_MODEL = os.environ.get("FENIX_LOW_RRI_MODEL", "gemma4:26b-a4b-it-qat")
DEFAULT_IDLE_TIMEOUT_SECONDS = int(
    os.environ.get("FENIX_LOW_RRI_IDLE_TIMEOUT_SECONDS", "60")
)
DEFAULT_MAX_WALL_SECONDS = int(
    os.environ.get("FENIX_LOW_RRI_MAX_WALL_SECONDS", "900")
)
DEFAULT_NUM_CTX = 16384
DEFAULT_NUM_PREDICT = 4096
DEFAULT_TEMPERATURE = 0.1
DEFAULT_THINK = False

TRUTHY_ENV_VALUES = {"1", "true", "TRUE", "yes", "YES", "on", "ON"}


@dataclass(frozen=True)
class StreamUsage:
    response_tokens: Optional[int] = None
    prompt_tokens: Optional[int] = None
    done_reason: Optional[str] = None


@dataclass(frozen=True)
class StreamChatResult:
    content: str
    usage: StreamUsage


class GemmaIdleTimeout(RuntimeError):
    def __init__(self, idle):
        super().__init__(f"Gemma idle timeout after {idle}s without a token")
        self.exit_code = 124


class GemmaWallTimeout(RuntimeError):
    def __init__(self, wall):
        super().__init__(f"Gemma wall timeout after {wall}s total")
        self.exit_code = 124


def bool_from_env(name, default=False):
    value = os.environ.get(name)
    if value is None:
        return default
    return value in TRUTHY_ENV_VALUES


def read_packet(path):
    if not path or path == "-":
        return sys.stdin.read()
    with open(path, encoding="utf-8") as handle:
        return handle.read()


def endpoint(host, path):
    if host and "://" not in host:
        host = "http://" + host
    return host.rstrip("/") + path


def get_json(url, timeout):
    request = urllib.request.Request(url, method="GET")
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            return json.loads(response.read().decode("utf-8"))
    except (TimeoutError, socket.timeout) as exc:
        raise GemmaIdleTimeout(timeout) from exc


def ensure_model_available(host, model, timeout):
    tags = get_json(endpoint(host, "/api/tags"), timeout)
    installed = {item.get("name") for item in tags.get("models", [])}
    if model not in installed:
        available = ", ".join(sorted(name for name in installed if name)) or "<none>"
        raise RuntimeError(
            "local Ollama model %r is not installed; available: %s" % (model, available)
        )


def build_chat_payload(
    *,
    model,
    system_prompt,
    packet,
    num_ctx,
    num_predict,
    temperature,
    think,
    keep_alive="10m",
):
    return {
        "model": model,
        "stream": True,
        "think": think,
        "keep_alive": keep_alive,
        "options": {
            "temperature": temperature,
            "num_predict": num_predict,
            "num_ctx": num_ctx,
        },
        "messages": [
            {
                "role": "system",
                "content": system_prompt,
            },
            {
                "role": "user",
                "content": packet,
            },
        ],
    }


def _coerce_usage_int(value):
    if isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    return None


def estimate_text_tokens(text):
    """Deterministic local token estimate for comparative telemetry only.

    Not a billing-grade substitute for model/runtime-reported token counts.
    """
    if not text:
        return 0
    return max(1, (len(text.encode("utf-8")) + 3) // 4)


def estimate_payload_tokens(payload):
    serialized = json.dumps(
        payload,
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    )
    return estimate_text_tokens(serialized)


def stream_result_content(result):
    if isinstance(result, StreamChatResult):
        return result.content
    return result


def stream_result_usage(result):
    if isinstance(result, StreamChatResult):
        return result.usage
    return StreamUsage()


def sum_measured_tokens(values):
    values = list(values)
    if not values:
        return None
    if any(value is None for value in values):
        return None
    return sum(values)


def stream_chat(url, payload, idle_timeout, max_wall, progress_label="delegate"):
    """POST to /api/chat with stream:true, return content plus usage metadata."""
    data = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        url,
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    wall_start = time.monotonic()
    content_parts = []
    tokens_received = 0
    usage = StreamUsage()

    try:
        with urllib.request.urlopen(request, timeout=idle_timeout) as response:
            while True:
                elapsed = time.monotonic() - wall_start
                if elapsed > max_wall:
                    raise GemmaWallTimeout(max_wall)

                try:
                    line = response.readline()
                except (TimeoutError, socket.timeout) as exc:
                    raise GemmaIdleTimeout(idle_timeout) from exc

                if not line:
                    break

                try:
                    chunk = json.loads(line.decode("utf-8"))
                except json.JSONDecodeError:
                    continue

                msg = chunk.get("message", {})
                fragment = msg.get("content", "")
                if fragment:
                    content_parts.append(fragment)
                    tokens_received += 1
                    print(
                        "[%s] tokens: %d elapsed: %.0fs" % (
                            progress_label,
                            tokens_received,
                            time.monotonic() - wall_start,
                        ),
                        file=sys.stderr,
                    )

                if chunk.get("done"):
                    usage = StreamUsage(
                        response_tokens=_coerce_usage_int(chunk.get("eval_count")),
                        prompt_tokens=_coerce_usage_int(chunk.get("prompt_eval_count")),
                        done_reason=chunk.get("done_reason"),
                    )
                    if chunk.get("done_reason") == "length":
                        raise RuntimeError(
                            "response cut by token limit; output may be truncated"
                        )
                    break

    except (TimeoutError, socket.timeout) as exc:
        raise GemmaIdleTimeout(idle_timeout) from exc

    return StreamChatResult(content="".join(content_parts), usage=usage)


def normalize_tagged_content(content, label):
    if not isinstance(content, str):
        raise RuntimeError("%s stream produced no content string" % label)
    content = content.replace("\r\n", "\n").replace("\r", "\n").strip()
    if not content:
        raise RuntimeError("invalid %s response: empty content" % label)
    return content


def next_nonempty_line(lines, idx, label, error_prefix):
    while idx < len(lines) and not lines[idx].strip():
        idx += 1
    if idx >= len(lines):
        raise RuntimeError("%s: missing %s line" % (error_prefix, label))
    return lines[idx], idx + 1


def parse_header_value(line, label, error_prefix):
    prefix = label + ": "
    if not line.startswith(prefix):
        raise RuntimeError("%s: expected %r" % (error_prefix, prefix))
    value = line[len(prefix):].strip()
    if not value:
        raise RuntimeError("%s: empty %s value" % (error_prefix, label))
    return value


_SECRET_PATTERN = re.compile(
    r'(api[_\-]?key|token|password|secret|credential)[^\s]*\s*[=:]\s*\S+',
    re.IGNORECASE,
)


def _redact(value):
    return _SECRET_PATTERN.sub(r'\1=***REDACTED***', value)


def append_audit_log(record, *, now=None):
    ts = now or datetime.datetime.utcnow()
    log_dir = pathlib.Path("logs/gemma-audit")
    log_dir.mkdir(parents=True, exist_ok=True)
    log_path = log_dir / ts.strftime("%Y-%m.jsonl")
    safe = {k: (_redact(v) if isinstance(v, str) else v) for k, v in record.items()}
    with open(log_path, "a", encoding="utf-8") as f:
        f.write(json.dumps(safe, sort_keys=True) + "\n")


def write_result(delegation, out_path):
    """Write JSON atomically: write to a temp file then rename."""
    tmp = out_path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(delegation, f, indent=2, sort_keys=True)
        f.write("\n")
    os.replace(tmp, out_path)
