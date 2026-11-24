import hashlib
import unittest

import workflow_exchange as wire
import workflow_exchange_transport as transport
import workflow_exchange_bridge as bridge

REPO = 'krukmat/fenix'


def fixture(stage='task_readiness', *, local_only=False):
    data = {'task': b'Task contents\n' * 1000, 'plan': b'Plan', 'policy': b'Policy', 'rri': b'RRI'}
    if stage == 'post_code_review':
        data.update(diff=b'Diff', verification=b'QA')
    refs = {k: dict(section_id=k, role=k, path='docs/'+k+'.md', sha256=hashlib.sha256(v).hexdigest(),
                    byte_length=len(v), source_kind='frozen_snapshot', required=True,
                    classification='local_only' if local_only and k == 'task' else 'cloud_permitted')
            for k, v in data.items()}
    p = dict(schema='fenix.task-packet.v1', repository=REPO, source_commit='a'*40,
             snapshot_digest=wire.digest([dict(path=r['path'], sha256=r['sha256'], byte_length=r['byte_length'],
                                              source_kind=r['source_kind']) for _,r in sorted(refs.items())]),
             task_id='W6', run_id='run', attempt_id='attempt', stage=stage,
             caller=dict(provider='openai',model='fixture',channel='local_cli'), objective='Compare',
             acceptance_criteria=[dict(id='C1',text='Compare')], allowed_paths=[], policy_refs=[refs['policy']],
             rri_evidence_ref=refs['rri'], context_refs=[r for k,r in refs.items() if k not in {'policy','rri'}])
    p['input_digest']=wire.digest(p,'input_digest')
    def transfer(cached):
        r=dict({k:p[k] for k in wire.IDENTITY},schema='fenix.context-receipt.v1',
               delivery_mode='mixed' if cached else 'full_capsule', delivery_state='constructed',
               content_refs=list(refs.values()),delivered_sections=[dict(id=k,sha256=v['sha256']) for k,v in refs.items()],
               omitted_sections=[])
        r['receipt_digest']=wire.digest(r,'receipt_digest')
        return transport.encode(p,r,{k:v for k,v in data.items() if k not in cached},repository=REPO)
    return p, data, transfer(set()), transfer({'task'})


def render(content):
    return b'\n'.join(k.encode()+b':\n'+v for k,v in sorted(content.items()))


class Bridge(unittest.TestCase):
    def prepare(self, raw, p, data, provider='claude', **kw):
        args=dict(repository=REPO,provider=provider,render=render,baseline_input=render(data),
                  authorized_external=True)
        args.update(kw)
        return bridge.prepare_native_input(raw,p,**args)

    def test_full_and_delta_equivalence(self):
        p,data,full,delta=fixture(); a,m=self.prepare(full,p,data); b,n=self.prepare(delta,p,data,cache=data)
        self.assertEqual(a,b); self.assertLess(n['transfer_bytes'],m['transfer_bytes'])
        self.assertEqual(n['native_input_bytes'],m['native_input_bytes']); self.assertEqual(n['native_input_digest'],m['native_input_digest'])
        self.assertIsNone(n['input_tokens']); self.assertFalse(n['inference_invoked']); self.assertEqual(n['delivery_state'],'unknown')

    def test_codex_readiness(self):
        p,data,full,_=fixture(); self.assertEqual(self.prepare(full,p,data,'codex')[0],render(data))

    def test_claude_post_code(self):
        p,data,full,_=fixture('post_code_review'); self.assertEqual(self.prepare(full,p,data)[0],render(data))

    def test_codex_post_code_rejected(self):
        p,data,full,_=fixture('post_code_review')
        with self.assertRaises(wire.InvalidExchange): self.prepare(full,p,data,'codex')

    def test_ollama_delegation(self):
        p,data,full,_=fixture('delegation',local_only=True)
        self.assertEqual(self.prepare(full,p,data,'ollama',authorized_external=False)[0],render(data))
        with self.assertRaises(wire.InvalidExchange): self.prepare(full,p,data,'ollama',ollama_external=True)

    def test_no_external_permission(self):
        p,data,full,_=fixture()
        with self.assertRaises(wire.InvalidExchange): self.prepare(full,p,data,authorized_external=False)
        p,data,full,_=fixture(local_only=True)
        with self.assertRaises(wire.InvalidExchange): self.prepare(full,p,data)

    def test_cache_miss_and_corruption(self):
        p,data,_,delta=fixture()
        for cache in ({},{'task':b'x'*len(data['task'])}):
            with self.subTest(cache=bool(cache)),self.assertRaises(wire.InvalidExchange): self.prepare(delta,p,data,cache=cache)

    def test_native_renderer_mismatch(self):
        p,data,full,_=fixture()
        for renderer in (lambda c:b'other',lambda c:'not bytes'):
            with self.assertRaises(wire.InvalidExchange): self.prepare(full,p,data,render=renderer)

    def test_native_budget(self):
        p,data,full,_=fixture()
        with self.assertRaises(wire.InvalidExchange): self.prepare(full,p,data,max_native_bytes=1)


if __name__=='__main__': unittest.main()
