"""Test the actual native hook functions with mocked provider/IO boundaries.

AST extraction avoids importing CLI dependencies or invoking any model.
This is not a replacement for full native repository regression tests.
"""
import argparse
import ast
import contextlib
import datetime
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import time
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

import workflow_shadow as s


def load_functions(filename, names, bindings):
    path = Path(__file__).with_name(filename)
    tree = ast.parse(path.read_text(encoding='utf-8'))
    nodes = [n for n in tree.body if isinstance(n, ast.FunctionDef) and n.name in names]
    assert {n.name for n in nodes} == set(names)
    ns = dict(bindings, workflow_shadow=s, sys=sys, json=json, datetime=datetime, time=time,
              argparse=argparse, Path=Path, __doc__='fixture')
    exec(compile(ast.Module(body=nodes, type_ignores=[]), str(path), 'exec'), ns)
    return ns


class ReviewerError(RuntimeError):
    reason = 'reviewer_timeout'


class GemmaIdleTimeout(RuntimeError):
    pass


class Hooks(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name) / 'logs' / 'shadow'
        env = patch.dict(os.environ, {'FENIX_W6_SHADOW':'1','FENIX_W6_RUN_ID':'run-1'}); env.start(); self.addCleanup(env.stop)
        log = patch.object(s,'_LOG_DIR',self.dir); log.start(); self.addCleanup(log.stop)
    def rows(self):
        return [json.loads(p.read_text()) for p in self.dir.glob('*.json')]
    def peer(self, mode='task-readiness', result='needs_changes', failure=False):
        args=SimpleNamespace(mode=mode,caller_kind='claude-code',caller_provider=None,base='main',timeout=1,out_dir='unused',criticality='standard')
        run=Mock(side_effect=ReviewerError('timeout') if failure else None, return_value=result)
        fallback=Mock(return_value='pass'); artifact=Mock(return_value='unused')
        return load_functions('peer-workflow-review.py', {'main','invoke_reviewer'}, dict(
            parse_args=lambda argv:args, normalize_caller_kind=lambda *a:'claude-code',resolve_reviewer=lambda *a:'codex',
            build_packet=lambda *a:{'task':'---\nid: FENIX-W6\n---\n'},build_prompt=lambda *a:'fixture prompt',
            resolve_reviewer_executable=lambda *a:'codex',codex_exec_command=lambda *a,**k:['codex','exec'],
            codex_review_command=lambda *a,**k:['codex','review'],parse_codex_review_output=lambda x:x,_run=run,
            parse_verdict=lambda x:{'status':x},ReviewerError=ReviewerError,
            blocked_verdict=lambda reason,detail:{'status':'blocked','reason':reason},
            should_attempt_fallback=lambda v:v.get('reason')=='reviewer_timeout',resolve_fallback_reviewer=lambda:'local-gemma',
            invoke_local_fallback_reviewer=fallback,_attempt_record=lambda role,reviewer,verdict:dict(role=role,reviewer=reviewer,verdict=verdict),
            write_artifact=artifact,print_summary=Mock()))
    def test_peer_default_and_shadow_same_nonpass(self):
        with patch.dict(os.environ,{'FENIX_W6_SHADOW':'0'}):
            off=self.peer(); self.assertEqual(off['main']([]),1)
        self.assertFalse(self.dir.exists())
        on=self.peer(); self.assertEqual(on['main']([]),1)
        self.assertEqual(off['write_artifact'].call_args,on['write_artifact'].call_args)
        self.assertEqual(on['_run'].call_count,1); self.assertEqual(on['invoke_local_fallback_reviewer'].call_count,0)
        self.assertEqual(self.rows()[0]['scope'],'explicit_prompt')
    def test_peer_native_base_not_context_delivered(self):
        n=self.peer(mode='post-code-review',result='pass'); self.assertEqual(n['main']([]),0)
        e=self.rows()[0]; self.assertEqual(e['scope'],'native_base_only'); self.assertIsNone(e['prepared_prompt_bytes']); self.assertEqual(e['delivery_state'],'unknown')
    def test_peer_timeout_fallback_single_calls(self):
        n=self.peer(failure=True); self.assertEqual(n['main']([]),0)
        self.assertEqual(n['_run'].call_count,1); self.assertEqual(n['invoke_local_fallback_reviewer'].call_count,1)
        by_role={e['role']:e for e in self.rows()}; self.assertEqual(by_role['primary']['error_kind'],'timeout'); self.assertEqual(by_role['fallback']['native_status'],'pass')
    def delegate(self, dry=False, error=False):
        args=SimpleNamespace(task_id='FENIX-W6',dry_run=dry,role_kind=None,packet='fixture',apply=False,allow_path=[],mode='full-files',
                             model='gemma:test',num_ctx=1,num_predict=1,temperature=0,think=False,host='unused',idle_timeout=1,max_wall=1,out=None,attempt=1)
        stream=Mock(side_effect=GemmaIdleTimeout('timeout') if error else None, return_value='response')
        gemma=SimpleNamespace(stream_result_content=lambda x:x,stream_result_usage=lambda x:SimpleNamespace(prompt_tokens=4,response_tokens=2),
                              append_audit_log=Mock(),estimate_payload_tokens=lambda x:5)
        return load_functions('delegate-low-rri.py',{'main','_main'},dict(parse_args=lambda:args,read_packet=lambda p:'fixture',
            build_payload=lambda *a:{'messages':[{'content':'system'},{'content':'user'}]},ensure_model_available=Mock(),
            stream_chat=stream,endpoint=lambda *a:'unused',gemma_local=gemma,parse_stream_content=lambda x:{'status':'no_patch'}))
    def test_delegate_default_and_shadow_same_output(self):
        with patch.dict(os.environ,{'FENIX_W6_SHADOW':'0'}), contextlib.redirect_stdout(io.StringIO()) as off:
            a=self.delegate(); self.assertEqual(a['main'](),0)
        self.assertFalse(self.dir.exists())
        with contextlib.redirect_stdout(io.StringIO()) as on:
            b=self.delegate(); self.assertEqual(b['main'](),0)
        self.assertEqual(off.getvalue(),on.getvalue()); self.assertEqual(b['stream_chat'].call_count,1)
        self.assertEqual(b['gemma_local'].append_audit_log.call_count,1); e=self.rows()[0]
        self.assertEqual((e['input_tokens'],e['output_tokens'],e['native_status']),(4,2,'no_patch'))
    def test_delegate_timeout_preserves_exception(self):
        n=self.delegate(error=True)
        with self.assertRaises(GemmaIdleTimeout): n['main']()
        self.assertEqual(n['stream_chat'].call_count,1); self.assertEqual(self.rows()[0]['error_kind'],'timeout')
    def test_delegate_dry_run_no_observation_or_call(self):
        n=self.delegate(dry=True)
        with contextlib.redirect_stdout(io.StringIO()): self.assertEqual(n['main'](),0)
        self.assertEqual(n['stream_chat'].call_count,0); self.assertFalse(self.dir.exists())
    def test_aggregate_cli_opt_in_only(self):
        n=load_functions('workflow_audit_aggregate.py',{'main'},dict(aggregate=lambda *a:{'schema':'legacy','value':1}))
        with contextlib.redirect_stdout(io.StringIO()) as off: self.assertEqual(n['main']([]),0)
        with contextlib.redirect_stdout(io.StringIO()) as on: self.assertEqual(n['main'](['--w6-shadow-dir',str(self.dir)]),0)
        legacy=json.loads(off.getvalue()); observed=json.loads(on.getvalue()); shadow=observed.pop('w6_shadow')
        self.assertEqual(legacy,observed); self.assertEqual(shadow['observed_attempts'],0); self.assertIsNone(shadow['accepted_tasks'])

if __name__ == '__main__': unittest.main()
