import ast
from pathlib import Path
import unittest

import workflow_prompt_layout as layout


def native(mode, packet):
    parts = ['Review instructions.', '', 'MODE: ' + mode, '', 'Criticality instructions.', '']
    for key in ('task', 'plan', 'task_card', 'verification_log', 'diff'):
        if packet.get(key):
            parts.extend(['=== %s ===' % key.upper(), str(packet[key]), ''])
    return '\n'.join(parts)


class Layout(unittest.TestCase):
    def packet(self, task='task α'):
        return dict(mode='task-readiness', task=task, plan='governing plan', task_card='common evidence')

    def test_native_exact(self):
        p=self.packet(); text, meta=layout.prepare(p,native)
        self.assertEqual(text,native('task-readiness',p))
        self.assertIsNone(meta['provider_input_tokens'])
        self.assertFalse(meta['inference_invoked'])

    def test_prefix_preserves_all_sections(self):
        p=self.packet(); a, am=layout.prepare(p,native); b,bm=layout.prepare(p,native,layout='stable_prefix')
        self.assertNotEqual(a,b);self.assertEqual(am['section_sha256'],bm['section_sha256'])
        self.assertEqual(am['native_input_bytes'],bm['native_input_bytes'])
        self.assertEqual(bm['section_order'],['plan','task_card','task'])
        self.assertEqual(bm['delivery_state'],'unknown')

    def test_common_prefix_increases(self):
        p=[self.packet('alpha'), self.packet('beta')]
        a=[layout.prepare(v,native)[0] for v in p]
        b=[layout.prepare(v,native,layout='stable_prefix')[0] for v in p]
        self.assertGreater(layout.common_prefix_bytes(b),layout.common_prefix_bytes(a))

    def test_no_card(self):
        p=self.packet();del p['task_card']
        _, m=layout.prepare(p,native,layout='stable_prefix')
        self.assertEqual(m['section_order'],['plan','task'])

    def test_no_task_or_plan(self):
        for key in ('task','plan'):
            p=self.packet();p[key]=' '
            with self.assertRaises(ValueError):layout.prepare(p,native)

    def test_post_code_excluded(self):
        for p in [dict(self.packet(),mode='post-code-review'),dict(self.packet(),diff='changes'),dict(self.packet(),verification_log='qa')]:
            with self.assertRaises(ValueError):layout.prepare(p,native)

    def test_unknown_layout(self):
        with self.assertRaises(ValueError):layout.prepare(self.packet(),native,layout='other')

    def test_changed_renderer_fails(self):
        def changed(mode,p):return native(mode,p)+('tail' if p.get('task') and p.get('plan') else '')
        with self.assertRaises(ValueError):layout.prepare(self.packet(),changed)

    def test_embedded_section_markers(self):
        p=self.packet('task\n=== PLAN ===\ntext');a,am=layout.prepare(p,native);b,bm=layout.prepare(p,native,layout='stable_prefix')
        self.assertEqual(am['section_sha256'],bm['section_sha256']);self.assertIn(p['task'],b)

    def test_bad_section_types(self):
        with self.assertRaises(ValueError):layout.prepare(dict(self.packet(),task_card=['evidence']),native)

    def test_actual_native_renderer(self):
        source = Path(__file__).with_name('peer-workflow-review.py').read_text(encoding='utf-8')
        tree = ast.parse(source)
        required = {'REVIEW_INSTRUCTIONS', 'TASK_READINESS_CRITICALITY_INSTRUCTIONS'}
        nodes = [n for n in tree.body if (isinstance(n, ast.Assign) and any(isinstance(t, ast.Name) and t.id in required for t in n.targets)) or (isinstance(n, ast.FunctionDef) and n.name == 'build_prompt')]
        self.assertEqual(len(nodes),3)
        ns = {};exec(compile(ast.Module(body=nodes,type_ignores=[]),'native_renderer','exec'),ns)
        renderer=ns['build_prompt'];p=self.packet()
        a,am=layout.prepare(p,renderer);b,bm=layout.prepare(p,renderer,layout='stable_prefix')
        self.assertEqual(a,renderer('task-readiness',p));self.assertEqual(am['section_sha256'],bm['section_sha256'])
        self.assertIn(ns['REVIEW_INSTRUCTIONS'],b);self.assertIn(ns['TASK_READINESS_CRITICALITY_INSTRUCTIONS'],b)

    def test_prefix_metric(self):
        self.assertEqual(layout.common_prefix_bytes(['éa','éb']),2)
        with self.assertRaises(ValueError):layout.common_prefix_bytes(['only one'])

if __name__=='__main__':unittest.main()
