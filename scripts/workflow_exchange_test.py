import copy
import unittest
import workflow_exchange as w

REPO = 'krukmat/fenix'

def ref(role):
    return dict(section_id=role, role=role, path='docs/'+role+'.md', sha256='a'*64, byte_length=1, source_kind='git_blob', required=True, classification='cloud_permitted')

def seal(p, field):
    p[field] = w.digest(p, field); return p

def fixture():
    p = dict(schema='fenix.task-packet.v1', repository=REPO, source_commit='a'*40, snapshot_digest='b'*64, task_id='W6.2', run_id='run', attempt_id='attempt', stage='task_readiness', caller=dict(provider='openai',model='test',channel='local_cli'), objective='Review', acceptance_criteria=[dict(id='C1',text='Review')], allowed_paths=['scripts/a.py'], policy_refs=[ref('policy')], rri_evidence_ref=ref('rri'), context_refs=[ref('task'),ref('plan')])
    seal(p,'input_digest'); common={k:p[k] for k in w.IDENTITY}; refs=list(w.packet(p,repository=REPO).values())
    d=dict(common,schema='fenix.context-receipt.v1',delivery_mode='full_capsule',delivery_state='delivered',content_refs=refs,delivered_sections=[dict(id=r['section_id'],sha256=r['sha256']) for r in refs],omitted_sections=[]);seal(d,'receipt_digest')
    r=dict(common,schema='fenix.result-envelope.v1',receipt_digest=d['receipt_digest'],status='REVIEW_REPORTED',evidence_refs=[ref('native_output')],error_code=None,usage=dict(input_tokens=None,output_tokens=None,cached_input_tokens=None,measurement='unavailable',estimated_input_tokens=None),duration_ms=1)
    return p,d,r

class Wire(unittest.TestCase):
    def test_valid(self):
        p,d,r=fixture();w.result(r,p,d,repository=REPO)
    def test_canonical(self):
        self.assertEqual(w.digest({'b':2,'a':1}), '43258cff783fe7036d8a43033f830adfc60ec037382473548ac742b888292777')
        self.assertNotEqual(w.digest({'criteria':['A','B']}),w.digest({'criteria':['B','A']}))
        self.assertEqual(w.canonical({'\ue000':1,'😀':2}),'{"😀":2,"\ue000":1}')
    def test_parse_reject(self):
        for raw in [b'{"x":1,"x":2}',b'NaN',b'9007199254740992',b'1.0',b'"\\ud800"']:
            with self.subTest(raw=raw),self.assertRaises(w.InvalidExchange):w.decode(raw,max_bytes=1000)
    def test_size(self):
        with self.assertRaises(w.InvalidExchange):w.decode(b'{}',max_bytes=1)
    def test_required_fields(self):
        for which in range(3):
            for key in fixture()[which]:
                p,d,r=fixture();del [p,d,r][which][key]
                with self.subTest(which=which,key=key),self.assertRaises(w.InvalidExchange):w.result(r,p,d,repository=REPO)
    def test_packet_tampering(self):
        p,d,r=fixture();p['objective']='Changed'
        with self.assertRaises(w.InvalidExchange):w.packet(p,repository=REPO)
    def test_cross_attempt(self):
        p,d,r=fixture();d['attempt_id']='different';seal(d,'receipt_digest')
        with self.assertRaises(w.InvalidExchange):w.receipt(d,p,repository=REPO)
    def test_status_and_receipt(self):
        for status in ['ACCEPTED','PASS','PROPOSED']:
            p,d,r=fixture();r['status']=status
            with self.subTest(status=status),self.assertRaises(w.InvalidExchange):w.result(r,p,d,repository=REPO)
        p,d,r=fixture();r['receipt_digest']=None
        with self.assertRaises(w.InvalidExchange):w.result(r,p,None,repository=REPO)
    def test_failure_before_delivery(self):
        p,d,r=fixture();r.update(status='TIMEOUT',receipt_digest=None,error_code='TIMEOUT',evidence_refs=[]);w.result(r,p,None,repository=REPO)
    def test_required_omitted(self):
        p,d,r=fixture();d['content_refs'].pop();v=d['delivered_sections'].pop();d['omitted_sections']=[dict(id=v['id'],reason='unavailable')];seal(d,'receipt_digest');r['receipt_digest']=d['receipt_digest']
        with self.assertRaises(w.InvalidExchange):w.result(r,p,d,repository=REPO)
    def test_bad_path(self):
        for path in ['../a','/a','a//b','a\\b','a/*']:
            p,d,r=fixture();p['allowed_paths']=[path];seal(p,'input_digest')
            with self.subTest(path=path),self.assertRaises(w.InvalidExchange):w.packet(p,repository=REPO)
    def test_wrong_usage(self):
        p,d,r=fixture();r['usage']['input_tokens']=True
        with self.assertRaises(w.InvalidExchange):w.result(r,p,d,repository=REPO)
    def test_wrong_schema(self):
        p,d,r=fixture();p['schema']='fenix.task-packet.v2';seal(p,'input_digest')
        with self.assertRaises(w.InvalidExchange):w.packet(p,repository=REPO)

if __name__=='__main__':unittest.main()
