import json
import unittest

import workflow_exchange as w
import workflow_exchange_adapter as a
from workflow_exchange_test import REPO, fixture, seal


def native(status='pass', mode='task-readiness'):
    return json.dumps(dict(mode=mode, reviewer='codex', verdict=dict(status=status, summary='fixture', findings=[]))).encode()


class Adapter(unittest.TestCase):
    def test_delivery_explicit(self):
        p, d, _ = fixture(); d['delivery_state']='constructed'; seal(d, 'receipt_digest')
        seen = {s['id']: s['sha256'] for s in d['delivered_sections']}
        with self.assertRaises(w.InvalidExchange): a.delivered_receipt(p,d,repository=REPO,observed_sections={})
        final = a.delivered_receipt(p,d,repository=REPO,observed_sections=seen)
        self.assertEqual(final['delivery_state'],'delivered'); self.assertEqual(d['delivery_state'],'constructed')

    def test_peer_pass_is_reported_not_accepted(self):
        p,d,_=fixture(); r=a.envelope(p,d,repository=REPO,raw=native(),artifact_path='logs/review.json',duration_ms=4)
        self.assertEqual(r['status'],'REVIEW_REPORTED'); self.assertNotIn('verdict',r)
        self.assertEqual(r['evidence_refs'][0]['classification'],'local_only')
        self.assertIsNone(r['usage']['input_tokens'])

    def test_needs_changes_not_accepted(self):
        p,d,_=fixture(); r=a.envelope(p,d,repository=REPO,raw=native('needs_changes'),artifact_path='logs/review.json',duration_ms=4)
        self.assertEqual(r['status'],'REVIEW_REPORTED')

    def test_no_receipt_no_normal_review(self):
        p,d,_=fixture()
        with self.assertRaises(w.InvalidExchange): a.envelope(p,None,repository=REPO,raw=native(),artifact_path='logs/review.json',duration_ms=4)

    def test_blocked_before_delivery(self):
        p,d,_=fixture(); r=a.envelope(p,None,repository=REPO,failure=dict(status='TIMEOUT',error_code='CLI_TIMEOUT'),duration_ms=4)
        self.assertEqual(r['status'],'TIMEOUT'); self.assertIsNone(r['receipt_digest'])

    def test_native_wrong_stage(self):
        p,d,_=fixture()
        with self.assertRaises(w.InvalidExchange): a.envelope(p,d,repository=REPO,raw=native(mode='post-code-review'),artifact_path='logs/review.json',duration_ms=4)

    def test_invalid_verdict(self):
        p,d,_=fixture()
        with self.assertRaises(w.InvalidExchange): a.envelope(p,d,repository=REPO,raw=native('ACCEPTED'),artifact_path='logs/review.json',duration_ms=4)

    def test_delegate_patch(self):
        p,d,_=fixture(); p['stage']='delegation'; seal(p,'input_digest'); d.update(stage='delegation',input_digest=p['input_digest']); seal(d,'receipt_digest')
        raw=b'{"status":"patch","files":[{"path":"scripts/a.py","action":"modify","contents":"fixture"}]}'
        r=a.envelope(p,d,repository=REPO,raw=raw,artifact_path='logs/result.json',duration_ms=4)
        self.assertEqual(r['status'],'PROPOSED'); self.assertNotIn('files',r)

    def test_incomplete_native_patch(self):
        p,d,_=fixture(); p['stage']='delegation'; seal(p,'input_digest'); d.update(stage='delegation',input_digest=p['input_digest']); seal(d,'receipt_digest')
        with self.assertRaises(w.InvalidExchange): a.envelope(p,d,repository=REPO,raw=b'{"status":"patch"}',artifact_path='logs/result.json',duration_ms=4)


if __name__ == '__main__': unittest.main()
