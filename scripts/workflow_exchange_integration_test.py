import json
import unittest

import workflow_context as context
import workflow_exchange as wire
import workflow_exchange_adapter as adapter
import workflow_exchange_metrics as metrics
import workflow_suitability as dss
from workflow_context_test import prepared_fixture
from workflow_exchange_test import REPO


def event():
    p, data = prepared_fixture()
    capsule, prepared = context.prepare(p, repository=REPO, git_reader=None, frozen_bytes=data)
    observed = {s['id']: s['sha256'] for s in prepared['delivered_sections']}
    receipt = adapter.delivered_receipt(p, prepared, repository=REPO, observed_sections=observed)
    raw = json.dumps({'mode':'task-readiness','reviewer':'codex','verdict':{'status':'needs_changes','summary':'fixture'}}).encode()
    result = adapter.envelope(p, receipt, repository=REPO, raw=raw, artifact_path='logs/review.json', duration_ms=10)
    return metrics.attempt_event(p, receipt, result, repository=REPO, arm='candidate', provider='openai', model='fixture', channel='cloud', role='primary', start_ms=0,end_ms=10)


def pair(i, **changes):
    b = dict(verified=True, accepted=True, critical_miss=False, elapsed_ms=100, input_tokens=100)
    c = dict(b, elapsed_ms=90, input_tokens=90); c.update(changes)
    return dict(case_id='case-'+str(i), matched=True, baseline=b, candidate=c)


class Integration(unittest.TestCase):
    def test_pipeline_no_acceptance(self):
        e=event(); r=metrics.aggregate([e]); c=r['arms']['candidate']
        self.assertEqual(c['attempts'],1); self.assertEqual(c['status_counts'],{'REVIEW_REPORTED':1})
        self.assertIsNone(c['accepted_tasks']); self.assertIsNone(c['tokens_per_accepted_task'])
        self.assertIsNone(c['usage']['cloud']['input_tokens']['observed_total'])
    def test_duplicate_and_conflicting(self):
        e=event(); r=metrics.aggregate([e,e]); self.assertEqual(r['duplicate_records'],1)
        r=metrics.aggregate([e,dict(e,end_ms=11)]); self.assertEqual(r['conflicting_runs'],1)
        self.assertEqual(r['arms']['candidate']['attempts'],0)
    def test_invalid_event(self):
        e=event(); e['status']='ACCEPTED'; r=metrics.aggregate([e]); self.assertEqual(r['invalid_records'],1)
    def test_token_coverage(self):
        e=event(); e['usage']=dict(e['usage'],measurement='provider_reported',input_tokens=100)
        r=metrics.aggregate([e]); self.assertEqual(r['arms']['candidate']['usage']['cloud']['input_tokens'],{'observed_total':100,'covered':1,'attempts':1})
    def test_dss_advisory_only(self):
        r=dss.assess([pair(i) for i in range(5)],rri=25); self.assertEqual(r['status'],'CANDIDATE_PREFERRED'); self.assertFalse(r['authorization'])
        self.assertEqual(dss.assess([pair(i) for i in range(5)],rri=26)['status'],'UNKNOWN')
    def test_dss_unknown_and_regression(self):
        self.assertEqual(dss.assess([pair(1)],rri=10)['status'],'UNKNOWN')
        self.assertEqual(dss.assess([pair(i,input_tokens=None) for i in range(5)],rri=10)['status'],'UNKNOWN')
        self.assertEqual(dss.assess([pair(1,accepted=False,critical_miss=True)],rri=10)['status'],'INCUMBENT_PREFERRED')
    def test_dss_duplicate_rejected(self):
        with self.assertRaises(wire.InvalidExchange): dss.assess([pair(1),pair(1)],rri=10)

if __name__ == '__main__': unittest.main()
