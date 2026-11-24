from pathlib import Path
import unittest

import workflow_prompt_pilot as pilot


class Pilot(unittest.TestCase):
    def read(self, commit, path):
        if path == pilot.NATIVE:
            self.assertEqual(commit,pilot.CANDIDATE)
            return Path(__file__).with_name('peer-workflow-review.py').read_bytes()
        if path == pilot.PLAN:
            return b'current plan' if commit == pilot.CANDIDATE else b'current plan plus historical detail'
        self.assertEqual(commit,pilot.CANDIDATE)
        if path.startswith('docs/tasks/'):
            return path.encode()
        return ('fixture evidence '+path).encode()

    def test_pinned_arms_and_shared_prefix(self):
        r=pilot.report(self.read)
        self.assertEqual(r['case_count'],9)
        self.assertFalse(r['inference_invoked'])
        self.assertGreater(r['common_prefix_bytes']['prefix'],r['common_prefix_bytes']['compact'])
        for case in r['cases']:
            a,b,c=(case['arms'][k] for k in ('historical','compact','prefix'))
            self.assertGreater(a['native_input_bytes'],b['native_input_bytes'])
            self.assertEqual(b['native_input_bytes'],c['native_input_bytes'])
            self.assertEqual(b['section_sha256'],c['section_sha256'])
            self.assertIsNone(c['provider_input_tokens'])
            self.assertNotIn('prompt',c)

    def test_missing_evidence_fails(self):
        def missing(commit,path):
            if path == pilot.EVIDENCE[0][1]:raise KeyError(path)
            return self.read(commit,path)
        with self.assertRaises(KeyError):pilot.report(missing)

    def test_bad_source_fails(self):
        with self.assertRaises(ValueError):pilot.load_renderer('def build_prompt(a,b): pass')

    def test_raw_not_exported(self):
        import json
        self.assertNotIn('fixture evidence',json.dumps(pilot.report(self.read)))

    def test_immutable_bytes_required(self):
        with self.assertRaises(ValueError):pilot.report(lambda *a:'not bytes')

if __name__ == '__main__':unittest.main()
