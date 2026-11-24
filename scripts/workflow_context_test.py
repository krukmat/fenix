import hashlib
import pathlib
import subprocess
import tempfile
import unittest

import workflow_context as c
import workflow_exchange as w
from workflow_exchange_test import fixture, REPO, seal


def prepared_fixture():
    p, _, _ = fixture()
    refs = p['policy_refs'] + [p['rri_evidence_ref']] + p['context_refs']
    data = {}
    for r in refs:
        b = r['role'].encode(); data[r['path']] = b
        r.update(sha256=hashlib.sha256(b).hexdigest(), byte_length=len(b), source_kind='frozen_snapshot')
    seal(p, 'input_digest')
    p['snapshot_digest'] = c.manifest_digest(p, repository=REPO)
    seal(p, 'input_digest')
    return p, data


class Context(unittest.TestCase):
    def resolve(self, p, data, **kw):
        return c.prepare(p, repository=REPO, git_reader=lambda *a, **k: self.fail('unexpected git read'), frozen_bytes=data, **kw)

    def test_full_capsule_is_not_delivery(self):
        p, data = prepared_fixture(); sections, receipt = self.resolve(p, data)
        self.assertEqual(len(sections), 4)
        self.assertEqual(receipt['delivery_state'], 'constructed')
        self.assertTrue(w.receipt(receipt, p, repository=REPO))

    def test_delta_exact_receiver_hash(self):
        p, data = prepared_fixture(); known = {'task': p['context_refs'][0]['sha256'], 'plan': '0'*64}
        delta, receipt = self.resolve(p, data, receiver_known=known)
        self.assertNotIn('task', delta); self.assertIn('plan', delta)
        self.assertEqual(receipt['delivery_mode'], 'mixed')
        self.assertEqual(receipt['delivery_state'], 'constructed')
        self.assertTrue(w.receipt(receipt, p, repository=REPO))

    def test_all_receiver_refs(self):
        p, data = prepared_fixture(); known = {k: r['sha256'] for k, r in w.packet(p, repository=REPO).items()}
        delta, receipt = self.resolve(p, data, receiver_known=known)
        self.assertEqual(delta, {}); self.assertEqual(receipt['delivery_mode'], 'resolved_refs')

    def test_changed_content(self):
        p, data = prepared_fixture(); data['docs/task.md'] = b'xxxx'
        with self.assertRaises(c.ContextBlocked): self.resolve(p, data)

    def test_missing_content(self):
        p, data = prepared_fixture(); del data['docs/task.md']
        with self.assertRaises(c.ContextBlocked): self.resolve(p, data)

    def test_snapshot_mismatch(self):
        p, data = prepared_fixture(); p['snapshot_digest'] = 'c'*64; seal(p, 'input_digest')
        with self.assertRaises(c.ContextBlocked): self.resolve(p, data)

    def test_budget(self):
        p, data = prepared_fixture()
        with self.assertRaises(c.ContextBlocked): self.resolve(p, data, max_bytes=1)

    def test_external_requires_authority(self):
        p, data = prepared_fixture()
        with self.assertRaises(c.ContextBlocked): self.resolve(p, data, external=True)
        self.resolve(p, data, external=True, authorized_external=True)

    def test_local_only(self):
        p, data = prepared_fixture(); p['context_refs'][0]['classification'] = 'local_only'; seal(p, 'input_digest')
        with self.assertRaises(c.ContextBlocked): self.resolve(p, data, external=True, authorized_external=True)

    def test_absent_delegation_target(self):
        p, data = prepared_fixture(); p['stage'] = 'delegation'; seal(p, 'input_digest')
        with self.assertRaises(c.ContextBlocked): self.resolve(p, data)
        p['snapshot_digest'] = c.manifest_digest(p, repository=REPO, absent_paths=['scripts/a.py']); seal(p, 'input_digest')
        self.resolve(p, data, absent_paths=['scripts/a.py'])

    def test_git_pinned_blob_and_symlink(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            def git(*args): return subprocess.check_output(['git', '-C', tmp, *args]).decode().strip()
            git('init', '-q'); git('config', 'user.email', 'fixture@example.invalid'); git('config', 'user.name', 'fixture')
            (root/'regular').write_bytes(b'original'); (root/'link').symlink_to('regular')
            git('add', '.'); git('commit', '-qm', 'fixture'); sha = git('rev-parse', 'HEAD')
            (root/'regular').write_bytes(b'changed')
            self.assertEqual(c.read_git_blob(root, sha, 'regular', max_bytes=8), b'original')
            for path, budget in [('link', 100), ('regular', 1), ('missing', 100), ('../regular', 100)]:
                with self.subTest(path=path), self.assertRaises(w.InvalidExchange): c.read_git_blob(root, sha, path, max_bytes=budget)


if __name__ == '__main__': unittest.main()
