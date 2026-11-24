"""W6 candidate transport: offline full/delta capsule and receiver reconstruction.

The supervisor owns packet authorization and cache scope. A reconstruction ack
is NOT a provider receipt, peer verdict, or permission. No network or inference.
"""
import base64
import binascii
import hashlib

import workflow_exchange as w

SCHEMA = 'fenix.context-transfer.v1'
ACK = 'fenix.context-reconstruction.v1'


def _limits(max_wire_bytes, max_context_bytes):
    for value in (max_wire_bytes, max_context_bytes):
        w.require(type(value) is int and 0 < value <= 64 * 1024 * 1024, 'invalid transport budget')


def _authorized(refs, external, authorized_external):
    w.require(type(external) is bool and type(authorized_external) is bool, 'invalid export authority')
    if external:
        w.require(authorized_external and all(r['classification'] == 'cloud_permitted' for r in refs.values()), 'external context not authorized')


def _content(raw, ref):
    w.require(type(raw) is bytes and len(raw) == ref['byte_length'], 'context length mismatch')
    w.require(hashlib.sha256(raw).hexdigest() == ref['sha256'], 'context digest mismatch')


def _scope(packet, prepared, repository, external, authorized_external, max_context_bytes):
    refs = w.packet(packet, repository=repository)
    w.require(w.receipt(prepared, packet, repository=repository), 'required context omitted')
    w.require(prepared['delivery_state'] == 'constructed', 'expected constructed receipt')
    w.require(not prepared['omitted_sections'], 'transport requires complete context')
    _authorized(refs, external, authorized_external)
    w.require(sum(r['byte_length'] for r in refs.values()) <= max_context_bytes, 'context exceeds budget')
    return refs


def encode(packet, prepared, delta, *, repository, external=False,
           authorized_external=False, max_wire_bytes=4194304, max_context_bytes=1048576):
    """Serialize exact delta bytes. Caller must send packet separately to receiver."""
    _limits(max_wire_bytes, max_context_bytes)
    refs = _scope(packet, prepared, repository, external, authorized_external, max_context_bytes)
    w.require(type(delta) is dict and set(delta) <= set(refs), 'unexpected delta section')
    mode = 'full_capsule' if set(delta) == set(refs) else ('mixed' if delta else 'resolved_refs')
    w.require(mode == prepared['delivery_mode'], 'delta mode mismatch')
    chunks = {}
    for key, raw in delta.items():
        _content(raw, refs[key])
        chunks[key] = base64.b64encode(raw).decode('ascii')
    transfer = dict(schema=SCHEMA, packet_digest=packet['input_digest'],
                    receipt=prepared, chunks=chunks)
    raw = w.canonical(transfer).encode('utf-8')
    w.require(len(raw) <= max_wire_bytes, 'wire budget exceeded')
    return raw


def receive(raw, packet, *, repository, cache=None, external=False,
            authorized_external=False, max_wire_bytes=4194304, max_context_bytes=1048576):
    """Reconstruct all sections or fail; a cache miss requires full resend.

    Cache is a supervisor-scoped map of section IDs to actual immutable bytes.
    It is never populated from model claims and is not mutated by this function.
    """
    _limits(max_wire_bytes, max_context_bytes)
    transfer = w.decode(raw, max_bytes=max_wire_bytes)
    w.obj(transfer, {'schema', 'packet_digest', 'receipt', 'chunks'})
    w.require(transfer['schema'] == SCHEMA and transfer['packet_digest'] == packet.get('input_digest'), 'transfer identity mismatch')
    prepared = transfer['receipt']
    refs = _scope(packet, prepared, repository, external, authorized_external, max_context_bytes)
    chunks = transfer['chunks']
    w.require(type(chunks) is dict and set(chunks) <= set(refs), 'unexpected delta section')
    mode = 'full_capsule' if set(chunks) == set(refs) else ('mixed' if chunks else 'resolved_refs')
    w.require(mode == prepared['delivery_mode'], 'delta mode mismatch')
    cache = {} if cache is None else cache
    w.require(type(cache) is dict, 'invalid cache')
    content = {}
    for key, ref in refs.items():
        if key in chunks:
            value = chunks[key]
            w.require(type(value) is str and len(value) == 4 * ((ref['byte_length'] + 2) // 3), 'invalid chunk size')
            try:
                data = base64.b64decode(value, validate=True)
            except (ValueError, binascii.Error) as exc:
                raise w.InvalidExchange('invalid chunk encoding') from exc
            w.require(base64.b64encode(data).decode('ascii') == value, 'noncanonical chunk encoding')
        else:
            w.require(key in cache, 'cache miss; full capsule required')
            data = cache[key]
        _content(data, ref)
        content[key] = data
    ack = dict(schema=ACK, packet_digest=packet['input_digest'], receipt_digest=prepared['receipt_digest'],
               transfer_digest=hashlib.sha256(raw).hexdigest(),
               sections={k: r['sha256'] for k, r in refs.items()})
    return content, ack


def verify_ack(raw, packet, prepared, ack, *, repository):
    """Check acknowledgment binding only; trust/authentication is out of scope."""
    refs = w.packet(packet, repository=repository)
    w.require(w.receipt(prepared, packet, repository=repository) and prepared['delivery_state'] == 'constructed', 'invalid constructed receipt')
    transfer = w.decode(raw, max_bytes=64 * 1024 * 1024)
    w.obj(transfer, {'schema', 'packet_digest', 'receipt', 'chunks'})
    w.require(transfer['schema'] == SCHEMA and transfer['packet_digest'] == packet['input_digest'] and transfer['receipt'] == prepared, 'transfer identity mismatch')
    w.obj(ack, {'schema', 'packet_digest', 'receipt_digest', 'transfer_digest', 'sections'})
    expected = dict(schema=ACK, packet_digest=packet['input_digest'], receipt_digest=prepared['receipt_digest'],
                    transfer_digest=hashlib.sha256(raw).hexdigest(),
                    sections={k: r['sha256'] for k, r in refs.items()})
    w.require(ack == expected, 'reconstruction acknowledgment mismatch')
    return True
