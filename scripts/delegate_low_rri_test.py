#!/usr/bin/env python3
"""Unit tests for delegate-low-rri.py (fenix port).

Run: python3 scripts/delegate_low_rri_test.py
     python3 -m unittest scripts/delegate_low_rri_test.py
"""

import importlib.util
import io
import json
import os
import socket
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import MagicMock, patch

_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "delegate-low-rri.py")
_spec = importlib.util.spec_from_file_location("delegate_low_rri", _SCRIPT)
_mod = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_mod)


def _valid_payload(**overrides):
    base = {
        "status": "patch",
        "summary": "ok",
        "files": [
            {"path": "scripts/x.py", "action": "create", "contents": "x = 1\n"},
        ],
        "test_commands": ["echo ok"],
        "risk_notes": [],
    }
    base.update(overrides)
    return base


def _tagged_response(status="PATCH", summary="ok", tests=None, risks=None, files=None):
    tests = tests or []
    risks = risks or []
    files = files or [
        {"path": "scripts/x.py", "action": "create", "contents": "x = 1\n"},
    ]
    lines = [
        "STATUS: %s" % status,
        "SUMMARY: %s" % summary,
    ]
    for cmd in tests:
        lines.append("TEST: %s" % cmd)
    for risk in risks:
        lines.append("RISK: %s" % risk)
    for entry in files:
        lines.extend([
            "=== FILE START ===",
            "PATH: %s" % entry["path"],
            "ACTION: %s" % entry["action"],
            "--- CONTENT ---",
        ])
        if entry["contents"]:
            lines.extend(entry["contents"].splitlines())
        lines.append("=== FILE END ===")
    return "\n".join(lines)


def _ndjson_lines(*chunks, done_last=True):
    lines = []
    for text in chunks:
        lines.append(json.dumps({"message": {"content": text}, "done": False}).encode())
    if done_last:
        lines.append(json.dumps({"message": {"content": ""}, "done": True}).encode())
    return lines


class ValidatePayload(unittest.TestCase):
    def test_valid_passes(self):
        _mod.validate_delegation_payload(_valid_payload())

    def test_missing_key_raises(self):
        p = _valid_payload()
        del p["files"]
        with self.assertRaises(RuntimeError) as ctx:
            _mod.validate_delegation_payload(p)
        self.assertIn("files", str(ctx.exception))

    def test_invalid_status_raises(self):
        with self.assertRaises(RuntimeError):
            _mod.validate_delegation_payload(_valid_payload(status="unknown"))

    def test_valid_statuses(self):
        _mod.validate_delegation_payload(_valid_payload(status="patch"))
        _mod.validate_delegation_payload(_valid_payload(status="no_patch", files=[]))
        _mod.validate_delegation_payload(_valid_payload(status="blocked", files=[]))

    def test_test_commands_must_be_list(self):
        with self.assertRaises(RuntimeError):
            _mod.validate_delegation_payload(_valid_payload(test_commands="echo ok"))

    def test_risk_notes_must_be_list(self):
        with self.assertRaises(RuntimeError):
            _mod.validate_delegation_payload(_valid_payload(risk_notes="none"))

    def test_files_must_be_list(self):
        with self.assertRaises(RuntimeError):
            _mod.validate_delegation_payload(_valid_payload(files={"path": "x"}))

    def test_file_entry_missing_field(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.validate_delegation_payload(
                _valid_payload(files=[{"path": "a.py", "action": "create"}]))
        self.assertIn("contents", str(ctx.exception))

    def test_file_invalid_action(self):
        with self.assertRaises(RuntimeError):
            _mod.validate_delegation_payload(_valid_payload(
                files=[{"path": "a.py", "action": "rename", "contents": ""}]))

    def test_patch_status_requires_files(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.validate_delegation_payload(_valid_payload(status="patch", files=[]))
        self.assertIn("empty", str(ctx.exception))

    def test_empty_path_rejected(self):
        with self.assertRaises(RuntimeError):
            _mod.validate_delegation_payload(_valid_payload(
                files=[{"path": "  ", "action": "create", "contents": "x"}]))


class ParseStreamContent(unittest.TestCase):
    def test_single_file_tagged_response(self):
        content = _tagged_response()
        result = _mod.parse_stream_content(content)
        self.assertEqual(result["status"], "patch")
        self.assertEqual(result["files"][0]["path"], "scripts/x.py")

    def test_multiple_file_blocks_parse(self):
        content = _tagged_response(files=[
            {"path": "scripts/a.py", "action": "create", "contents": "a = 1\n"},
            {"path": "scripts/b.py", "action": "modify", "contents": "b = 2\n"},
        ])
        result = _mod.parse_stream_content(content)
        self.assertEqual(len(result["files"]), 2)
        self.assertEqual(result["files"][1]["action"], "modify")

    def test_no_patch_allowed_without_files(self):
        content = "STATUS: NO_PATCH\nSUMMARY: too broad"
        result = _mod.parse_stream_content(content)
        self.assertEqual(result["status"], "no_patch")
        self.assertEqual(result["files"], [])

    def test_blocked_allowed_without_files(self):
        content = "STATUS: BLOCKED\nSUMMARY: waiting"
        result = _mod.parse_stream_content(content)
        self.assertEqual(result["status"], "blocked")

    def test_unexpected_text_raises(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_stream_content("hello\nSTATUS: PATCH\nSUMMARY: nope")
        self.assertIn("unexpected text", str(ctx.exception))

    def test_whitespace_stripped(self):
        content = "\n\n" + _tagged_response() + "\n"
        result = _mod.parse_stream_content(content)
        self.assertEqual(result["status"], "patch")

    def test_missing_path_raises(self):
        content = "\n".join([
            "STATUS: PATCH",
            "SUMMARY: ok",
            "=== FILE START ===",
            "ACTION: create",
            "--- CONTENT ---",
            "x = 1",
            "=== FILE END ===",
        ])
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_stream_content(content)
        self.assertIn("PATH", str(ctx.exception))

    def test_missing_action_raises(self):
        content = "\n".join([
            "STATUS: PATCH",
            "SUMMARY: ok",
            "=== FILE START ===",
            "PATH: scripts/x.py",
            "--- CONTENT ---",
            "x = 1",
            "=== FILE END ===",
        ])
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_stream_content(content)
        self.assertIn("ACTION", str(ctx.exception))

    def test_missing_end_marker_raises(self):
        content = "\n".join([
            "STATUS: PATCH",
            "SUMMARY: ok",
            "=== FILE START ===",
            "PATH: scripts/x.py",
            "ACTION: create",
            "--- CONTENT ---",
            "x = 1",
        ])
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_stream_content(content)
        self.assertIn("file end", str(ctx.exception))

    def test_invalid_action_raises(self):
        content = _tagged_response(files=[
            {"path": "scripts/x.py", "action": "rename", "contents": "x = 1\n"},
        ])
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_stream_content(content)
        self.assertIn("unknown ACTION", str(ctx.exception))

    def test_duplicate_path_raises(self):
        content = _tagged_response(files=[
            {"path": "scripts/x.py", "action": "create", "contents": "x = 1\n"},
            {"path": "scripts/x.py", "action": "modify", "contents": "x = 2\n"},
        ])
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_stream_content(content)
        self.assertIn("duplicate path", str(ctx.exception))

    def test_delete_requires_empty_content(self):
        content = _tagged_response(files=[
            {"path": "scripts/x.py", "action": "delete", "contents": "still here\n"},
        ])
        with self.assertRaises(RuntimeError) as ctx:
            _mod.parse_stream_content(content)
        self.assertIn("delete action requires empty content", str(ctx.exception))


class BuildPayload(unittest.TestCase):
    def _build_payload(self, model="model", packet="text", num_ctx=16384):
        return _mod.build_payload(
            model, packet, num_ctx,
            _mod.DEFAULT_NUM_PREDICT, _mod.DEFAULT_TEMPERATURE, _mod.DEFAULT_THINK,
        )

    def _build_replacement_payload(self, model="model", packet="text", num_ctx=16384):
        return _mod.build_replacement_payload(
            model, packet, num_ctx,
            _mod.DEFAULT_NUM_PREDICT, _mod.DEFAULT_TEMPERATURE, _mod.DEFAULT_THINK,
        )

    def test_stream_true(self):
        p = self._build_payload("gemma4:26b-a4b-it-qat", "packet text")
        self.assertTrue(p["stream"])

    def test_think_false(self):
        p = self._build_payload("gemma4:26b-a4b-it-qat", "packet text")
        self.assertFalse(p["think"])

    def test_custom_think_true(self):
        p = _mod.build_payload(
            "model", "text", 16384,
            _mod.DEFAULT_NUM_PREDICT, _mod.DEFAULT_TEMPERATURE, True,
        )
        self.assertTrue(p["think"])

    def test_temperature_default(self):
        p = self._build_payload()
        self.assertEqual(p["options"]["temperature"], _mod.DEFAULT_TEMPERATURE)

    def test_num_ctx_set(self):
        p = self._build_payload("gemma4:26b-a4b-it-qat", "packet text")
        self.assertEqual(p["options"]["num_ctx"], 16384)

    def test_custom_num_ctx(self):
        p = self._build_payload(num_ctx=8192)
        self.assertEqual(p["options"]["num_ctx"], 8192)

    def test_packet_in_user_message(self):
        p = self._build_payload(packet="MY PACKET")
        user_msg = next(m for m in p["messages"] if m["role"] == "user")
        self.assertIn("MY PACKET", user_msg["content"])

    def test_system_prompt_contains_tagged_contract(self):
        p = self._build_payload()
        sys_msg = next(m for m in p["messages"] if m["role"] == "system")
        self.assertIn("=== FILE START ===", sys_msg["content"])
        self.assertIn("no JSON", sys_msg["content"])
        self.assertIn("STATUS: PATCH\n", sys_msg["content"])
        self.assertNotIn("STATUS: PATCH|NO_PATCH|BLOCKED", sys_msg["content"])
        self.assertIn("Do not output the pipe-separated list", sys_msg["content"])

    def test_system_prompt_references_fenix_not_dubbridge(self):
        p = self._build_payload()
        sys_msg = next(m for m in p["messages"] if m["role"] == "system")
        self.assertIn("fenix", sys_msg["content"])
        self.assertNotIn("DubBridge", sys_msg["content"])

    def test_replacement_prompt_uses_single_status_example(self):
        p = self._build_replacement_payload()
        sys_msg = next(m for m in p["messages"] if m["role"] == "system")
        self.assertIn("=== REPLACEMENT START ===", sys_msg["content"])
        self.assertIn("STATUS: PATCH\n", sys_msg["content"])
        self.assertNotIn("STATUS: PATCH|NO_PATCH|BLOCKED", sys_msg["content"])
        self.assertIn("Do not output the pipe-separated list", sys_msg["content"])


class _FakeResponse:
    def __init__(self, lines, block_after=None, delay_per_line=0):
        self._lines = list(lines)
        self._idx = 0
        self._block_after = block_after
        self._delay = delay_per_line

    def __enter__(self):
        return self

    def __exit__(self, *a):
        pass

    def readline(self):
        if self._block_after is not None and self._idx >= self._block_after:
            raise socket.timeout("simulated stall")
        if self._idx >= len(self._lines):
            return b""
        if self._delay:
            time.sleep(self._delay)
        line = self._lines[self._idx]
        self._idx += 1
        return line + b"\n"


class StreamChatTimeouts(unittest.TestCase):
    def _call(self, lines, idle=60, wall=900, block_after=None, delay=0):
        resp = _FakeResponse(lines, block_after=block_after, delay_per_line=delay)
        with patch("urllib.request.urlopen", return_value=resp):
            return _mod.stream_chat(
                "http://localhost:11434/api/chat",
                {"model": "test", "stream": True},
                idle_timeout=idle,
                max_wall=wall,
            )

    def test_happy_path_assembles_content(self):
        lines = _ndjson_lines("STATUS: PATCH\n", "SUMMARY: ok\n")
        result = self._call(lines)
        self.assertIn("STATUS: PATCH", result.content)

    def test_idle_timeout_on_stall(self):
        lines = _ndjson_lines("partial")
        with self.assertRaises(_mod.DelegationIdleTimeout):
            self._call(lines, idle=1, block_after=1)

    def test_wall_timeout_exceeded(self):
        lines = _ndjson_lines("a", "b", "c")
        with self.assertRaises(_mod.DelegationWallTimeout):
            self._call(lines, wall=0.5, delay=0.4)

    def test_empty_response_returns_empty_string(self):
        result = self._call([])
        self.assertEqual(result.content, "")

    def test_done_flag_stops_iteration(self):
        lines = [
            json.dumps({"message": {"content": "hello"}, "done": True}).encode(),
            json.dumps({"message": {"content": " SHOULD_NOT_APPEAR"}, "done": False}).encode(),
        ]
        result = self._call(lines)
        self.assertIn("hello", result.content)
        self.assertNotIn("SHOULD_NOT_APPEAR", result.content)

    def test_malformed_ndjson_line_skipped(self):
        lines = [
            b"not-json\n",
            json.dumps({"message": {"content": "ok"}, "done": True}).encode(),
        ]
        result = self._call(lines)
        self.assertIn("ok", result.content)


class EnforceScope(unittest.TestCase):
    def _files(self, *paths):
        return [{"path": p, "action": "create", "contents": ""} for p in paths]

    def test_empty_allowed_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.enforce_scope(self._files("scripts/x.py"), [])
        self.assertIn("no --allow-path", str(ctx.exception))

    def test_in_scope_prefix_passes(self):
        _mod.enforce_scope(self._files("scripts/x.py"), ["scripts/"])

    def test_in_scope_exact_passes(self):
        _mod.enforce_scope(self._files("scripts/x.py"), ["scripts/x.py"])

    def test_in_scope_glob_passes(self):
        _mod.enforce_scope(self._files("scripts/x.py"), ["scripts/*.py"])

    def test_out_of_scope_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.enforce_scope(self._files("internal/domain/auth/service.go"), ["scripts/"])
        self.assertIn("out-of-scope", str(ctx.exception))

    def test_parent_escape_rejected(self):
        with self.assertRaises(RuntimeError) as ctx:
            _mod.enforce_scope(self._files("../etc/passwd"), ["scripts/"])
        self.assertIn("escapes", str(ctx.exception))

    def test_absolute_path_rejected(self):
        with self.assertRaises(RuntimeError):
            _mod.enforce_scope(self._files("/etc/passwd"), ["scripts/"])

    def test_prefix_does_not_match_sibling(self):
        with self.assertRaises(RuntimeError):
            _mod.enforce_scope(self._files("scripts_evil/x.py"), ["scripts/"])

    def test_fenix_internal_path_allowed(self):
        _mod.enforce_scope(self._files("internal/domain/copilot/service.go"), ["internal/"])

    def test_fenix_mobile_path_allowed(self):
        _mod.enforce_scope(self._files("mobile/src/screens/Home.tsx"), ["mobile/"])


class ValidateFileActions(unittest.TestCase):
    def test_create_rejects_existing_target(self):
        with tempfile.TemporaryDirectory() as d:
            target = os.path.join(d, "a.txt")
            with open(target, "w") as f:
                f.write("x\n")
            with self.assertRaises(RuntimeError) as ctx:
                _mod.validate_file_actions(
                    [{"path": "a.txt", "action": "create", "contents": "y\n"}], d)
            self.assertIn("create targets existing", str(ctx.exception))

    def test_modify_rejects_missing_target(self):
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaises(RuntimeError) as ctx:
                _mod.validate_file_actions(
                    [{"path": "a.txt", "action": "modify", "contents": "y\n"}], d)
            self.assertIn("modify targets missing", str(ctx.exception))

    def test_delete_rejects_missing_target(self):
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaises(RuntimeError) as ctx:
                _mod.validate_file_actions(
                    [{"path": "a.txt", "action": "delete", "contents": ""}], d)
            self.assertIn("delete targets missing", str(ctx.exception))

    def test_valid_actions_pass(self):
        with tempfile.TemporaryDirectory() as d:
            target = os.path.join(d, "a.txt")
            with open(target, "w") as f:
                f.write("x\n")
            _mod.validate_file_actions(
                [
                    {"path": "a.txt", "action": "modify", "contents": "y\n"},
                    {"path": "b.txt", "action": "create", "contents": "z\n"},
                ],
                d,
            )


class BuildAndApplyDiff(unittest.TestCase):
    def _git_repo(self):
        d = tempfile.mkdtemp()
        subprocess.run(["git", "init", "-q"], cwd=d, check=True)
        subprocess.run(["git", "config", "user.email", "t@t"], cwd=d, check=True)
        subprocess.run(["git", "config", "user.name", "t"], cwd=d, check=True)
        return d

    def test_create_then_apply_multifile(self):
        d = self._git_repo()
        os.makedirs(os.path.join(d, "scripts"))
        files = [
            {"path": "scripts/math_utils.py", "action": "create",
             "contents": "def add(a, b):\n    return a + b\n"},
            {"path": "scripts/math_utils_test.py", "action": "create",
             "contents": "from math_utils import add\nassert add(2, 3) == 5\n"},
        ]
        diff = _mod.build_diff(files, d)
        self.assertIn("scripts/math_utils.py", diff)
        self.assertIn("scripts/math_utils_test.py", diff)
        outcome = _mod.apply_diff(diff, d)
        self.assertEqual(outcome, "applied")
        self.assertTrue(os.path.exists(os.path.join(d, "scripts/math_utils.py")))
        self.assertTrue(os.path.exists(os.path.join(d, "scripts/math_utils_test.py")))

    def test_modify_existing_file(self):
        d = self._git_repo()
        target = os.path.join(d, "a.txt")
        with open(target, "w") as f:
            f.write("line one\nline two\n")
        subprocess.run(["git", "add", "-A"], cwd=d, check=True)
        subprocess.run(["git", "commit", "-qm", "init"], cwd=d, check=True)
        files = [{"path": "a.txt", "action": "modify",
                  "contents": "line one\nline two CHANGED\n"}]
        diff = _mod.build_diff(files, d)
        outcome = _mod.apply_diff(diff, d)
        self.assertEqual(outcome, "applied")
        with open(target) as f:
            self.assertIn("CHANGED", f.read())

    def test_empty_diff_when_no_change(self):
        d = self._git_repo()
        target = os.path.join(d, "a.txt")
        with open(target, "w") as f:
            f.write("same\n")
        files = [{"path": "a.txt", "action": "modify", "contents": "same\n"}]
        diff = _mod.build_diff(files, d)
        self.assertEqual(diff.strip(), "")
        self.assertIn("nothing to apply", _mod.apply_diff(diff, d))

    def test_before_after_allows_shorter_replacement_with_new_lines(self):
        d = self._git_repo()
        target = os.path.join(d, "a.py")
        with open(target, "w") as f:
            f.write("def f():\n    first()\n    second()\n    third()\n")
        before = os.path.join(d, "before.txt")
        with open(before, "w") as f:
            f.write("    first()\n    second()\n    third()\n")

        result = _mod.apply_before_after(
            "a.py", before, "    return 1\n", ["a.py"], d, do_apply=False,
        )

        self.assertIn("+    return 1", result["unified_diff"])
        self.assertIn("-    first()", result["unified_diff"])

    def test_delete_existing_file(self):
        d = self._git_repo()
        target = os.path.join(d, "a.txt")
        with open(target, "w") as f:
            f.write("gone\n")
        subprocess.run(["git", "add", "-A"], cwd=d, check=True)
        subprocess.run(["git", "commit", "-qm", "init"], cwd=d, check=True)
        files = [{"path": "a.txt", "action": "delete", "contents": ""}]
        diff = _mod.build_diff(files, d)
        outcome = _mod.apply_diff(diff, d)
        self.assertEqual(outcome, "applied")
        self.assertFalse(os.path.exists(target))


class WriteResult(unittest.TestCase):
    def test_writes_valid_json(self):
        with tempfile.TemporaryDirectory() as d:
            out = os.path.join(d, "result.json")
            _mod.write_result(_valid_payload(), out)
            with open(out) as f:
                data = json.load(f)
            self.assertEqual(data["status"], "patch")

    def test_atomic_replace(self):
        with tempfile.TemporaryDirectory() as d:
            out = os.path.join(d, "result.json")
            _mod.write_result(_valid_payload(summary="first"), out)
            _mod.write_result(_valid_payload(summary="second"), out)
            with open(out) as f:
                data = json.load(f)
            self.assertEqual(data["summary"], "second")

    def test_no_tmp_file_left_on_success(self):
        with tempfile.TemporaryDirectory() as d:
            out = os.path.join(d, "result.json")
            _mod.write_result(_valid_payload(), out)
            self.assertFalse(os.path.exists(out + ".tmp"))


class CliBehavior(unittest.TestCase):
    def run_cli(self, *args, stdin=None, env=None):
        return subprocess.run(
            [sys.executable, _SCRIPT, *args],
            capture_output=True, text=True,
            input=stdin,
            env=env,
        )

    def test_dry_run_prints_payload_no_network(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".md", delete=False) as f:
            f.write("# test packet\n")
            fname = f.name
        try:
            r = self.run_cli(fname, "--dry-run")
            self.assertEqual(r.returncode, 0, r.stderr)
            data = json.loads(r.stdout)
            self.assertTrue(data["stream"])
            self.assertIn("num_ctx", data["options"])
        finally:
            os.unlink(fname)

    def test_dry_run_stream_true(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".md", delete=False) as f:
            f.write("# test packet\n")
            fname = f.name
        try:
            r = self.run_cli(fname, "--dry-run")
            data = json.loads(r.stdout)
            self.assertTrue(data["stream"])
        finally:
            os.unlink(fname)

    def test_dry_run_num_ctx_default(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".md", delete=False) as f:
            f.write("# p\n")
            fname = f.name
        try:
            r = self.run_cli(fname, "--dry-run")
            data = json.loads(r.stdout)
            self.assertEqual(data["options"]["num_ctx"], _mod.DEFAULT_NUM_CTX)
        finally:
            os.unlink(fname)

    def test_dry_run_custom_num_ctx(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".md", delete=False) as f:
            f.write("# p\n")
            fname = f.name
        try:
            r = self.run_cli(fname, "--dry-run", "--num-ctx", "8192")
            data = json.loads(r.stdout)
            self.assertEqual(data["options"]["num_ctx"], 8192)
        finally:
            os.unlink(fname)

    def test_dry_run_think_enabled(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".md", delete=False) as f:
            f.write("# p\n")
            fname = f.name
        try:
            r = self.run_cli(fname, "--dry-run", "--think")
            data = json.loads(r.stdout)
            self.assertTrue(data["think"])
        finally:
            os.unlink(fname)

    def test_dry_run_no_think_overrides_env(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".md", delete=False) as f:
            f.write("# p\n")
            fname = f.name
        try:
            env = os.environ.copy()
            env["FENIX_LOW_RRI_THINK"] = "1"
            r = self.run_cli(fname, "--dry-run", "--no-think", env=env)
            data = json.loads(r.stdout)
            self.assertFalse(data["think"])
        finally:
            os.unlink(fname)

    def test_empty_packet_exits_1(self):
        r = self.run_cli("-", stdin="   \n  ")
        self.assertEqual(r.returncode, 1)
        self.assertIn("empty", r.stderr)

    def test_missing_packet_file_exits_nonzero(self):
        r = self.run_cli("/tmp/does_not_exist_fenix.md")
        self.assertNotEqual(r.returncode, 0)

    def test_help_exits_0(self):
        r = self.run_cli("--help")
        self.assertEqual(r.returncode, 0)

    def test_env_var_prefix_is_fenix(self):
        """FENIX_* env vars must be the canonical interface — no DUBBRIDGE_* refs."""
        with tempfile.NamedTemporaryFile(mode="w", suffix=".md", delete=False) as f:
            f.write("# p\n")
            fname = f.name
        try:
            env = os.environ.copy()
            env["FENIX_LOW_RRI_MODEL"] = "gemma4:12b-it-qat"
            r = self.run_cli(fname, "--dry-run", env=env)
            data = json.loads(r.stdout)
            self.assertEqual(data["model"], "gemma4:12b-it-qat")
        finally:
            os.unlink(fname)


class TimeoutExitCodes(unittest.TestCase):
    def test_idle_timeout_exit_code_124(self):
        exc = _mod.DelegationIdleTimeout(60)
        self.assertEqual(exc.exit_code, 124)
        self.assertIn("60", str(exc))

    def test_wall_timeout_exit_code_124(self):
        exc = _mod.DelegationWallTimeout(900)
        self.assertEqual(exc.exit_code, 124)
        self.assertIn("900", str(exc))


class AuditEmission(unittest.TestCase):
    def _run(self, stream_response, extra_args=None):
        captured = []
        with tempfile.TemporaryDirectory() as tmp:
            out_path = os.path.join(tmp, "result.json")
            packet_file = os.path.join(tmp, "packet.md")
            with open(packet_file, "w") as f:
                f.write("# test packet\n")

            argv = [_SCRIPT, packet_file, "--out", out_path] + (extra_args or [])
            with patch("sys.argv", argv), \
                 patch.object(_mod, "ensure_model_available"), \
                 patch.object(_mod, "stream_chat", return_value=stream_response), \
                 patch.object(_mod.gemma_local, "append_audit_log",
                              side_effect=lambda r: captured.append(r)):
                _mod.main()
        return captured

    def test_patch_emits_one_record_with_developer_role(self):
        response = _mod.gemma_local.StreamChatResult(
            content=(
                "STATUS: PATCH\nSUMMARY: ok\n=== FILE START ===\n"
                "PATH: scripts/x.py\nACTION: create\n--- CONTENT ---\n"
                "x = 1\n=== FILE END ==="
            ),
            usage=_mod.gemma_local.StreamUsage(response_tokens=13),
        )
        records = self._run(response)
        self.assertEqual(len(records), 1)
        r = records[0]
        self.assertEqual(r["role"], "developer")
        self.assertEqual(r["outcome"], "PATCH")
        self.assertEqual(r["done_reason"], "stop")
        self.assertFalse(r["escalated"])
        self.assertIn("system_prompt", r)
        self.assertIn("user_prompt", r)
        self.assertIsNone(r["task_id"])
        self.assertIsNone(r["attempt"])
        self.assertEqual(r["apply_result"], "skipped")
        self.assertEqual(r["response_tokens"], 13)
        self.assertGreater(r["packet_tokens_est"], 0)

    def test_no_patch_emits_skipped_apply_result(self):
        records = self._run("STATUS: NO_PATCH\nSUMMARY: nothing to do")
        self.assertEqual(len(records), 1)
        r = records[0]
        self.assertEqual(r["outcome"], "NO_PATCH")
        self.assertEqual(r["apply_result"], "skipped")
        self.assertFalse(r["escalated"])
        self.assertIsNone(r["response_tokens"])

    def test_blocked_emits_escalated_true(self):
        records = self._run("STATUS: BLOCKED\nSUMMARY: cannot proceed")
        self.assertEqual(len(records), 1)
        r = records[0]
        self.assertEqual(r["outcome"], "BLOCKED")
        self.assertTrue(r["escalated"])
        self.assertEqual(r["apply_result"], "skipped")

    def test_task_id_and_attempt_passed_through(self):
        records = self._run(
            "STATUS: NO_PATCH\nSUMMARY: ok",
            extra_args=["--task-id", "T2", "--attempt", "2"],
        )
        r = records[0]
        self.assertEqual(r["task_id"], "T2")
        self.assertEqual(r["attempt"], 2)

    def test_diff_added_removed_counted(self):
        response = (
            "STATUS: PATCH\nSUMMARY: ok\n=== FILE START ===\n"
            "PATH: scripts/x.py\nACTION: create\n--- CONTENT ---\n"
            "x = 1\n=== FILE END ==="
        )
        fake_diff = "+++ b/scripts/x.py\n+x = 1\n+y = 2\n--- a/scripts/x.py\n-old = 0\n"
        with tempfile.TemporaryDirectory() as tmp:
            out_path = os.path.join(tmp, "result.json")
            packet_file = os.path.join(tmp, "packet.md")
            with open(packet_file, "w") as f:
                f.write("# test packet\n")
            argv = [_SCRIPT, packet_file, "--out", out_path, "--allow-path", "scripts/"]
            captured = []
            with patch("sys.argv", argv), \
                 patch.object(_mod, "ensure_model_available"), \
                 patch.object(_mod, "stream_chat", return_value=response), \
                 patch.object(_mod, "build_diff", return_value=fake_diff), \
                 patch.object(_mod, "validate_file_actions"), \
                 patch.object(_mod.gemma_local, "append_audit_log",
                              side_effect=lambda r: captured.append(r)):
                _mod.main()
        r = captured[0]
        self.assertEqual(r["diff_added"], 2)
        self.assertEqual(r["diff_removed"], 1)


if __name__ == "__main__":
    unittest.main(verbosity=2)
