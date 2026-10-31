#!/usr/bin/env python3
# OPTIONAL FUTURE B6 functional workflow probe. A real workflow is NOT required by B1-B5.
# This script is not a foundation readiness gate.
"""INT B1.T1: real Fenix workflow diagnostic. NEVER E2E/S2S certification.

Runs a genuine Fenix workflow in a disposable, operator-verified read-only
environment. Requires an existing non-production USER test JWT because the
current Fenix endpoint has no service principal; do not use this as the
DubBridge integration credential.
"""
import argparse
import ipaddress
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request

MAX_RESPONSE = 1024 * 1024


def request(url, token, method="GET", body=None):
    data = json.dumps(body, separators=(",", ":")).encode() if body is not None else None
    req = urllib.request.Request(
        url,
        data=data,
        method=method,
        headers={"Authorization": "Bearer " + token, "Content-Type": "application/json"},
    )
    # Prevent proxy-environment interception of a disposable bearer token.
    # This diagnostic only targets loopback and must connect directly.
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    try:
        with opener.open(req, timeout=35) as resp:
            raw = resp.read(MAX_RESPONSE + 1)
            if len(raw) > MAX_RESPONSE:
                raise RuntimeError("Fenix response exceeds diagnostic limit")
            if resp.headers.get_content_type() != "application/json":
                raise RuntimeError("unexpected provider content type")
            return resp.status, json.loads(raw)
    except urllib.error.HTTPError as exc:
        # Never print response bodies: they may contain private agent inputs.
        raise RuntimeError("Fenix returned HTTP " + str(exc.code)) from None
    except urllib.error.URLError:
        raise RuntimeError("Fenix endpoint unavailable") from None


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise RuntimeError("Fenix redirected an authenticated request")


def local_base_url(raw):
    parsed = urllib.parse.urlsplit(raw)
    if parsed.scheme not in {"http", "https"} or parsed.username or parsed.password:
        raise ValueError("invalid base URL")
    if parsed.path not in {"", "/"} or parsed.query or parsed.fragment:
        raise ValueError("base URL must contain only scheme, host and optional port")
    try:
        host = parsed.hostname
        if not host or not ipaddress.ip_address(host).is_loopback:
            raise ValueError("only loopback URLs allowed")
    except ValueError:
        if parsed.hostname != "localhost":
            raise ValueError("only loopback URLs allowed") from None
    return raw.rstrip("/")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://127.0.0.1:8080")
    parser.add_argument("--workflow-id", required=True)
    parser.add_argument("--ack-disposable-readonly", action="store_true")
    parser.add_argument("--token-env", default="FENIX_B1_DISPOSABLE_TEST_JWT")
    args = parser.parse_args(argv)
    if not args.ack_disposable_readonly:
        parser.error("requires --ack-disposable-readonly after inspecting the workflow's tools")
    try:
        base = local_base_url(args.base_url)
    except ValueError as exc:
        parser.error(str(exc))
    if not args.workflow_id or not all(c.isalnum() or c in "-_" for c in args.workflow_id) or len(args.workflow_id) > 128:
        parser.error("invalid workflow ID")
    token = os.environ.get(args.token_env, "")
    if not token or any(c.isspace() for c in token) or len(token) > 8192:
        parser.error("missing/unsafe disposable test JWT in named environment variable")
    root = base + "/api/v1"
    # Actual Fenix handler calls DSLRunner.Run inline and returns HTTP 200.
    status, doc = request(root + "/workflows/" + args.workflow_id + "/execute",
                          token, "POST", {"trigger_context": {"int_b1_diagnostic": True}, "inputs": {}})
    if status != 200 or not isinstance(doc, dict):
        raise RuntimeError("unexpected workflow execution HTTP/JSON contract")
    data = doc.get("data")
    if not isinstance(data, dict) or not isinstance(data.get("run"), dict):
        raise RuntimeError("invalid workflow response shape")
    run = data["run"]
    run_id = run.get("id")
    if not isinstance(run_id, str) or not run_id or len(run_id) > 128:
        raise RuntimeError("workflow response lacks a valid persisted run ID")
    follow_status, follow = request(root + "/agents/runs/" +
                                    urllib.parse.quote(run_id, safe=""), token)
    if not isinstance(follow, dict) or not isinstance(follow.get("data"), dict):
        raise RuntimeError("invalid persisted run response shape")
    observed = follow["data"]
    if follow_status != 200 or observed.get("id") != run_id:
        raise RuntimeError("persisted run mismatch")
    # Do NOT print raw output, inputs, tools, reasoning, credential or user info.
    report = {
        "kind": "FENIX_B1_REAL_WORKFLOW_PROBE",
        "evidence": "real-local-user-jwt-diagnostic-NOT-S2S-NOT-E2E",
        "workflow_http": status,
        "run_read_http": follow_status,
        "persisted_run_matches": True,
        "runtime_status": observed.get("runtime_status"),
        "public_status": observed.get("status"),
        "trace_present": isinstance(observed.get("traceId"), str) and bool(observed.get("traceId")),
        "completed_at_present": bool(observed.get("completedAt")),
    }
    print(json.dumps(report, sort_keys=True))
    if not report["trace_present"]:
        raise RuntimeError("run does not expose a trace ID")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (RuntimeError, ValueError) as exc:
        print("FENIX_B1_PROBE=BLOCKED reason=" + str(exc), file=sys.stderr)
        sys.exit(1)
