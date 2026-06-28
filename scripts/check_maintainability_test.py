#!/usr/bin/env python3
import importlib.util
import os
import sys
import unittest


SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "check-maintainability.py")
SPEC = importlib.util.spec_from_file_location("check_maintainability", SCRIPT)
maint = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = maint
SPEC.loader.exec_module(maint)


def added(path, text, line_no=1):
    return maint.AddedLine(path=path, text=text, hunk=1, line_no=line_no)


class ContextBackgroundRatchetTest(unittest.TestCase):
    def test_flags_new_internal_background_root(self):
        violations = maint.analyze_added_lines([
            added("internal/domain/audit/service.go", "context.Background()", 365),
        ])
        self.assertEqual(len(violations), 1)
        self.assertIn("context.Background() root", violations[0].message)

    def test_allows_server_owned_with_cancel_root(self):
        violations = maint.analyze_added_lines([
            added("internal/server/server.go", "bgCtx, cancel := context.WithCancel(context.Background())", 62),
        ])
        self.assertEqual(violations, [])

    def test_allows_background_context_fallback_assignment(self):
        violations = maint.analyze_added_lines([
            added("internal/api/routes.go", "runtime.BackgroundContext = context.Background()", 513),
        ])
        self.assertEqual(violations, [])

    def test_skips_cmd_entrypoints(self):
        violations = maint.analyze_added_lines([
            added("cmd/fenixlsp/main.go", "context.Background()", 20),
        ])
        self.assertEqual(violations, [])


if __name__ == "__main__":
    unittest.main()
