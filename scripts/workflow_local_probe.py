#!/usr/bin/env python3
"""Read-only loopback Ollama timing probe for the FINAL Mac-only workflow block.

No model installation, code changes, log upload, prompt export, CI or reviewer
approval. Benchmark outputs are microbenchmarks, not verified code-task results.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
import time
import urllib.parse
import urllib.request
from pathlib import Path


def validate_host(raw: str) -> str:
    url = urllib.parse.urlsplit(raw)
    if url.scheme != "http" or url.hostname not in {"localhost", "127.0.0.1", "::1"}:
        raise ValueError("Ollama host must be plain HTTP loopback")
    if url.username or url.password or url.path not in ("", "/") or url.query or url.fragment:
        raise ValueError("Ollama host must be only a loopback origin")
    return raw.rstrip("/")


def get_installed_models(host: str, timeout: int = 10) -> set[str]:
    req = urllib.request.Request(host + "/api/tags", method="GET")
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        data = json.load(resp)
    if not isinstance(data, dict) or not isinstance(data.get("models"), list):
        raise ValueError("invalid /api/tags payload")
    return {
        item["name"]
        for item in data["models"]
        if isinstance(item, dict) and isinstance(item.get("name"), str)
    }


def memory_snapshot() -> dict | None:
    """Best-effort macOS host VM counters; NOT per-Ollama RSS."""
    if sys.platform != "darwin":
        return None
    try:
        p = subprocess.run(["vm_stat"], capture_output=True, text=True, timeout=3, check=True)
    except (OSError, subprocess.SubprocessError):
        return None
    size = re.search(r"page size of (\d+) bytes", p.stdout)
    page_bytes = int(size.group(1)) if size else 16384
    output = {"page_size_bytes": page_bytes}
    for key, label in (("free", "Pages free"), ("active", "Pages active"),
                       ("wired", "Pages wired down"), ("compressed", "Pages occupied by compressor")):
        match = re.search(r"^" + re.escape(label) + r":\s*(\d+)", p.stdout, re.M)
        if match:
            output[key + "_bytes"] = int(match.group(1)) * page_bytes
    return output


def generate_metrics(host: str, model: str, prompt: str, max_tokens: int, timeout: int) -> dict:
    body = json.dumps({
        "model": model, "prompt": prompt, "stream": True, "keep_alive": "3m",
        "options": {"num_predict": max_tokens, "temperature": 0},
    }).encode("utf-8")
    req = urllib.request.Request(host + "/api/generate", data=body,
                                 headers={"Content-Type": "application/json"}, method="POST")
    start = time.perf_counter()
    first = None
    last = None
    with urllib.request.urlopen(req, timeout=timeout) as response:
        for line in response:
            if not line.strip():
                continue
            try:
                event = json.loads(line)
            except (ValueError, TypeError):
                continue
            if not isinstance(event, dict):
                continue
            if event.get("response") and first is None:
                first = round(time.perf_counter() - start, 3)
            if event.get("done"):
                last = event
                break
    if last is None:
        raise RuntimeError("Ollama stream ended without done event")
    wall = round(time.perf_counter() - start, 3)
    metrics = {"status": "complete", "wall_seconds": wall, "first_token_seconds": first}
    for key in ("total_duration", "load_duration", "prompt_eval_duration", "eval_duration"):
        value = last.get(key)
        metrics[key + "_seconds"] = round(value / 1e9, 6) if type(value) is int and value >= 0 else None
    for key in ("prompt_eval_count", "eval_count"):
        value = last.get(key)
        metrics[key] = value if type(value) is int and value >= 0 else None
    rate = last.get("eval_count")
    duration = last.get("eval_duration")
    metrics["generated_tokens_per_second"] = (
        round(rate * 1e9 / duration, 2)
        if type(rate) is int and type(duration) is int and duration > 0 else None
    )
    return metrics


def run_probe(*, host: str, models: list[str], prompt: str, runs: int, timeout: int, max_tokens: int) -> dict:
    host = validate_host(host)
    if not 1 <= runs <= 10 or not 1 <= max_tokens <= 1024 or not 1 <= timeout <= 600:
        raise ValueError("runs, timeout or max_tokens out of bounded range")
    if not prompt.strip():
        raise ValueError("prompt is empty")
    installed = get_installed_models(host, timeout=min(timeout, 10))
    report = {
        "schema": "fenix-local-workflow-probe-v1",
        "scope": "local-model-call-only-not-code-task-certification",
        "prompt_sha256": hashlib.sha256(prompt.encode("utf-8")).hexdigest(),
        "runs_requested": runs, "memory_before": memory_snapshot(), "models": [],
    }
    for model in models:
        entry = {"model": model, "available": model in installed, "measurements": []}
        if model in installed:
            for i in range(runs):
                try:
                    measurement = generate_metrics(host, model, prompt, max_tokens, timeout)
                except (OSError, RuntimeError, ValueError, TimeoutError):
                    measurement = {"status": "error", "error_class": "model_call_failed"}
                entry["measurements"].append({"run": i + 1, **measurement})
        report["models"].append(entry)
    report["memory_after"] = memory_snapshot()
    return report


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--host", default="http://127.0.0.1:11434")
    p.add_argument("--models", nargs="+", required=True)
    p.add_argument("--prompt-file", type=Path, required=True)
    p.add_argument("--runs", type=int, default=3)
    p.add_argument("--timeout", type=int, default=120)
    p.add_argument("--max-tokens", type=int, default=96)
    p.add_argument("--output", type=Path, required=True)
    p.add_argument("--dry-run", action="store_true")
    args = p.parse_args(argv)
    validate_host(args.host)
    prompt = args.prompt_file.read_text(encoding="utf-8")
    if not prompt.strip():
        p.error("empty prompt")
    if args.dry_run:
        report = {"status": "dry_run", "models": args.models, "runs": args.runs,
                  "prompt_sha256": hashlib.sha256(prompt.encode("utf-8")).hexdigest()}
    else:
        report = run_probe(host=args.host, models=args.models, prompt=prompt,
                           runs=args.runs, timeout=args.timeout, max_tokens=args.max_tokens)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
