import contextlib
import io
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import workflow_shadow as s


class Shadow(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name) / 'logs' / 'shadow'
        self.env = patch.dict(os.environ, {'FENIX_W6_SHADOW':'1','FENIX_W6_RUN_ID':'run-1'}); self.env.start(); self.addCleanup(self.env.stop)
        self.log = patch.object(s,'_LOG_DIR',self.dir); self.log.start(); self.addCleanup(self.log.stop)
    def rows(self):
        return [json.loads(p.read_text()) for p in self.dir.glob('*.json')]
    def emit(self, stage='task-readiness', role='primary', provider='codex', status='needs_changes'):
        with s.observe(stage,role,provider,task_id='FENIX-W6'):
            s.note('explicit_prompt',prompts=('secret fixture',)); s.outcome(status)
    def test_default_off(self):
        with patch.dict(os.environ,{'FENIX_W6_SHADOW':'0'}):
            self.emit()
        self.assertFalse(self.dir.exists())
    def test_dry_run(self):
        with s.observe('delegation','developer','ollama',task_id='FENIX-W6',active=False): s.outcome('patch')
        self.assertFalse(self.dir.exists())
    def test_scope_and_no_raw(self):
        self.emit(); e=self.rows()[0]; self.assertEqual(e['prepared_prompt_bytes'],14)
        self.assertEqual(e['delivery_state'],'unknown'); self.assertEqual(e['native_status'],'needs_changes')
        self.assertNotIn('secret fixture',json.dumps(e)); self.assertIsNone(e['input_tokens'])
        self.assertEqual(next(self.dir.glob('*.json')).stat().st_mode & 0o777,0o600)
    def test_shared_run_distinct_attempts(self):
        self.emit(); self.emit(role='fallback',provider='local-gemma'); self.emit('delegation','developer','ollama','patch')
        r=s.aggregate(self.dir); self.assertEqual(r['observed_attempts'],3); self.assertEqual(r['observed_runs'],1)
        self.assertEqual(r['fallback_attempts'],1); self.assertIsNone(r['fallback_rate']); self.assertIsNone(r['accepted_tasks'])
    def test_timeout_and_exception_passthrough(self):
        class GemmaIdleTimeout(RuntimeError): pass
        with self.assertRaises(GemmaIdleTimeout):
            with s.observe('delegation','developer','ollama',task_id='FENIX-W6'): raise GemmaIdleTimeout('secret detail')
        e=self.rows()[0]; self.assertEqual(e['error_kind'],'timeout'); self.assertEqual(e['native_status'],'error'); self.assertNotIn('secret detail',json.dumps(e))
    def test_missing_run_no_record(self):
        with patch.dict(os.environ,{'FENIX_W6_RUN_ID':''}),contextlib.redirect_stderr(io.StringIO()): self.emit()
        self.assertFalse(self.dir.exists())
    def test_write_failure_does_not_change_result(self):
        with patch.object(s,'_write',side_effect=OSError('private')),contextlib.redirect_stderr(io.StringIO()) as stderr:
            self.emit(status='pass')
        self.assertIn('native behavior unchanged',stderr.getvalue()); self.assertNotIn('private',stderr.getvalue())
    def test_usage_and_native_base(self):
        with s.observe('post-code-review','primary','codex',task_id='FENIX-W6'):
            s.note('native_base_only'); s.outcome('pass')
        with s.observe('delegation','developer','ollama',task_id='FENIX-W6'):
            s.note_payload('ollama_packet',{'messages':[{'content':'abc'},{'content':'é'}]},model='gemma:test')
            s.usage(lambda:SimpleNamespace(prompt_tokens=20,response_tokens=4)); s.outcome('patch')
        r=s.aggregate(self.dir); self.assertEqual(r['by_scope']['native_base_only'],1)
        self.assertEqual(r['by_provider']['ollama']['input_tokens']['observed_total'],20)
        self.assertEqual(r['by_provider']['ollama']['prepared_prompt_bytes']['observed_total'],5)
        self.assertIsNone(r['by_provider']['codex']['input_tokens']['observed_total'])
    def test_duplicates_conflicts(self):
        self.emit(); e=self.rows()[0]
        (self.dir/'duplicate.json').write_text(json.dumps(e)); self.assertEqual(s.aggregate(self.dir)['duplicate_records'],1)
        e['elapsed_ms']+=1; (self.dir/'conflict.json').write_text(json.dumps(e))
        r=s.aggregate(self.dir); self.assertEqual(r['conflicting_runs'],1); self.assertEqual(r['observed_attempts'],0)
    def test_missing_and_malformed(self):
        self.assertEqual(s.aggregate(self.dir)['observed_attempts'],0); self.emit(); e=self.rows()[0]
        (self.dir/'bad.json').write_text('{"x":1,"x":2}')
        (self.dir/'forged.json').write_text(json.dumps(dict(e,delivery_state='delivered')))
        self.assertEqual(s.aggregate(self.dir)['invalid_records'],2)
    def test_task_id(self):
        self.assertEqual(s.task_id_from_text('---\nid: FENIX-W6\n---\n# Task'),'FENIX-W6')
        self.assertIsNone(s.task_id_from_text('# Task\nid: FENIX-W6'))
    def test_nested_context(self):
        with s.observe('task-readiness','primary','codex',task_id='FENIX-W6'):
            with s.observe('task-readiness','fallback','local-gemma',task_id='FENIX-W6'): s.outcome('blocked')
            s.outcome('needs_changes')
        self.assertEqual({e['role']:e['native_status'] for e in self.rows()},{'primary':'needs_changes','fallback':'blocked'})

if __name__ == '__main__': unittest.main()
