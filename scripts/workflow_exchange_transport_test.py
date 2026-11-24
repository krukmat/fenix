import copy
import hashlib
import unittest

import workflow_context as c
import workflow_exchange as w
import workflow_exchange_transport as t
from workflow_context_test import prepared_fixture
from workflow_exchange_test import REPO, seal


def prepare(p, data, known=None, **kw):
    return c.prepare(p, repository=REPO, git_reader=None, frozen_bytes=data, receiver_known=known, **kw)


class Transport(unittest.TestCase):
    def setUp(self):
        self.p, self.data = prepared_fixture()
        self.full, self.receipt = prepare(self.p, self.data)
        self.raw = t.encode(self.p, self.receipt, self.full, repository=REPO)

    def test_full_roundtrip_and_ack(self):
        content, ack = t.receive(self.raw, self.p, repository=REPO)
        self.assertEqual(content, self.full)
        self.assertTrue(t.verify_ack(self.raw, self.p, self.receipt, ack, repository=REPO))
        self.assertEqual(self.receipt['delivery_state'], 'constructed')
        self.assertNotIn('delivery_state', ack)

    def test_delta_and_fallback(self):
        known = {k: hashlib.sha256(v).hexdigest() for k,v in self.full.items() if k != 'plan'}
        delta, receipt = prepare(self.p, self.data, known)
        raw = t.encode(self.p, receipt, delta, repository=REPO)
        with self.assertRaises(w.InvalidExchange): t.receive(raw, self.p, repository=REPO)
        content, ack = t.receive(raw, self.p, repository=REPO, cache=self.full)
        self.assertEqual(content, self.full); self.assertTrue(t.verify_ack(raw,self.p,receipt,ack,repository=REPO))
        self.assertEqual(t.receive(self.raw,self.p,repository=REPO)[0],self.full)

    def test_all_cached(self):
        known = {k:hashlib.sha256(v).hexdigest() for k,v in self.full.items()}
        delta, receipt = prepare(self.p,self.data,known)
        self.assertEqual(delta,{})
        raw = t.encode(self.p,receipt,delta,repository=REPO)
        self.assertEqual(t.receive(raw,self.p,repository=REPO,cache=self.full)[0],self.full)

    def test_corrupt_cache_and_chunk(self):
        known = {'task':hashlib.sha256(self.full['task']).hexdigest()}
        delta,receipt=prepare(self.p,self.data,known); raw=t.encode(self.p,receipt,delta,repository=REPO)
        with self.assertRaises(w.InvalidExchange):t.receive(raw,self.p,repository=REPO,cache={'task':b'xxxx'})
        altered=w.decode(self.raw,max_bytes=4194304); altered['chunks']['task']='eHh4eA=='
        with self.assertRaises(w.InvalidExchange):t.receive(w.canonical(altered).encode(),self.p,repository=REPO)

    def test_identity_and_ack_replay(self):
        _,ack=t.receive(self.raw,self.p,repository=REPO)
        p=copy.deepcopy(self.p);p['attempt_id']='new';seal(p,'input_digest')
        with self.assertRaises(w.InvalidExchange):t.receive(self.raw,p,repository=REPO)
        ack['transfer_digest']='a'*64
        with self.assertRaises(w.InvalidExchange):t.verify_ack(self.raw,self.p,self.receipt,ack,repository=REPO)

    def test_unknown_and_duplicate_fields(self):
        altered=w.decode(self.raw,max_bytes=4194304); altered['extra']=True
        for raw in [w.canonical(altered).encode(),b'{"schema":1,"schema":2}']:
            with self.assertRaises(w.InvalidExchange):t.receive(raw,self.p,repository=REPO)

    def test_external_policy_both_boundaries(self):
        with self.assertRaises(w.InvalidExchange):t.encode(self.p,self.receipt,self.full,repository=REPO,external=True)
        with self.assertRaises(w.InvalidExchange):t.receive(self.raw,self.p,repository=REPO,external=True)
        p=copy.deepcopy(self.p);p['context_refs'][0]['classification']='local_only';seal(p,'input_digest')
        delta,receipt=prepare(p,self.data)
        raw=t.encode(p,receipt,delta,repository=REPO)
        with self.assertRaises(w.InvalidExchange):t.receive(raw,p,repository=REPO,external=True,authorized_external=True)
        with self.assertRaises(w.InvalidExchange):t.encode(p,receipt,delta,repository=REPO,external=True,authorized_external=True)

    def test_budgets(self):
        for kw in [{'max_wire_bytes':1},{'max_context_bytes':1}]:
            with self.assertRaises(w.InvalidExchange):t.encode(self.p,self.receipt,self.full,repository=REPO,**kw)
            with self.assertRaises(w.InvalidExchange):t.receive(self.raw,self.p,repository=REPO,**kw)

    def test_wrong_delta_mode(self):
        with self.assertRaises(w.InvalidExchange):t.encode(self.p,self.receipt,{},repository=REPO)
        altered=w.decode(self.raw,max_bytes=4194304);del altered['chunks']['task']
        with self.assertRaises(w.InvalidExchange):t.receive(w.canonical(altered).encode(),self.p,repository=REPO,cache=self.full)

    def test_transport_byte_savings_not_token_savings(self):
        p=copy.deepcopy(self.p);data=dict(self.data);r=p['context_refs'][0];data[r['path']]=b'x'*10000
        r['byte_length']=10000;r['sha256']=hashlib.sha256(data[r['path']]).hexdigest();seal(p,'input_digest')
        p['snapshot_digest']=c.manifest_digest(p,repository=REPO);seal(p,'input_digest')
        full,receipt=prepare(p,data);base=t.encode(p,receipt,full,repository=REPO)
        delta,receipt=prepare(p,data,{'task':r['sha256']});candidate=t.encode(p,receipt,delta,repository=REPO)
        self.assertLess(len(candidate),len(base));self.assertEqual(t.receive(candidate,p,repository=REPO,cache=full)[0],full)

if __name__=='__main__':unittest.main()
