"""Opt-in W6.4 adapter candidate, not a gate or provider invocation.

Adapts existing native delegate/peer JSON outcomes to wire envelopes only when
trusted input identity and actual delivery are known. A stored legacy packet or
native verdict alone is never proof of delivery. No prompts or patch bodies in
exported envelope; no model calls, subprocesses, acceptance or state changes.
"""
import copy
import hashlib

import workflow_exchange as w


def delivered_receipt(packet, constructed, *, repository, observed_sections):
    """Bind actual observed full/delta context from a trusted adapter boundary.

    Caller must attest content actually available to receiver, not merely the
    planned payload. A successful subprocess invocation alone is insufficient.
    """
    w.receipt(constructed, packet, repository=repository)
    w.require(constructed['delivery_state'] == 'constructed', 'expected constructed receipt')
    expected = {s['id']: s['sha256'] for s in constructed['delivered_sections']}
    w.require(type(observed_sections) is dict and observed_sections == expected, 'delivery coverage mismatch')
    out = copy.deepcopy(constructed); out['delivery_state'] = 'delivered'
    out['receipt_digest'] = w.digest(out, 'receipt_digest')
    w.receipt(out, packet, repository=repository)
    return out


def evidence_ref(raw, path):
    w.require(type(raw) is bytes, 'native artifact bytes required')
    w.path(path)
    return dict(section_id='native_output', role='native_output', path=path,
                sha256=hashlib.sha256(raw).hexdigest(), byte_length=len(raw),
                source_kind='frozen_snapshot', required=True, classification='local_only')


def envelope(packet, delivery, *, repository, raw=None, artifact_path=None,
             duration_ms, failure=None, usage=None, max_bytes=1048576):
    """Normalize actual native JSON or a trusted adapter failure.

    Native artifact paths and bytes are supplied by the caller, not resolved by
    this helper. Native status remains in native artifact; REVIEW_REPORTED is
    not peer PASS. Invalid JSON raises a typed error rather than fabricating
    a trusted result. Caller records failure separately with trusted identity.
    """
    w.packet(packet, repository=repository)
    evidence = []
    if failure is not None:
        w.obj(failure, {'status', 'error_code'}); w.enum(failure['status'], w.FAILURES)
        status, error = failure['status'], failure['error_code']
        w.require(raw is None and artifact_path is None, 'ambiguous terminal input')
    else:
        native = w.decode(raw, max_bytes=max_bytes)
        w.require(type(native) is dict, 'invalid native object')
        evidence = [evidence_ref(raw, artifact_path)]
        error = None
        if packet['stage'] == 'delegation':
            mapping = {'patch': 'PROPOSED', 'no_patch': 'NO_PATCH', 'blocked': 'BLOCKED'}
            n = native.get('status'); w.enum(n, set(mapping)); status = mapping[n]
            if status == 'PROPOSED':
                # Existing delegate parser/scope/QA remains authoritative.
                w.require(bool(native.get('files')) or type(native.get('after_block')) is str, 'missing proposed patch')
            if status == 'BLOCKED': error = 'NATIVE_BLOCKED'
        else:
            expected_mode = {'task_readiness': 'task-readiness', 'post_code_review': 'post-code-review'}[packet['stage']]
            w.require(native.get('mode') == expected_mode, 'native stage mismatch')
            verdict = native.get('verdict'); w.require(type(verdict) is dict, 'missing native verdict')
            n = verdict.get('status'); w.enum(n, {'pass', 'needs_changes', 'blocked', 'fail'})
            w.require(type(native.get('reviewer')) is str and bool(native['reviewer'].strip()), 'missing reviewer')
            status = 'BLOCKED' if n == 'blocked' else 'REVIEW_REPORTED'
            if status == 'BLOCKED': error = 'NATIVE_BLOCKED'
    if usage is None:
        usage = dict(input_tokens=None, output_tokens=None, cached_input_tokens=None,
                     measurement='unavailable', estimated_input_tokens=None)
    out = {k: packet[k] for k in w.IDENTITY}
    out.update(schema='fenix.result-envelope.v1', receipt_digest=delivery['receipt_digest'] if delivery is not None else None,
               status=status, evidence_refs=evidence, error_code=error, usage=usage, duration_ms=duration_ms)
    w.result(out, packet, delivery, repository=repository)
    return out
