"""W6.3 candidate: immutable context resolution; no inference or approval.

The frozen input map is supplied by a trusted supervisor, never model output.
The returned receipt is constructed only: it is not a delivery attestation.
"""
import hashlib
import subprocess

import workflow_exchange as wire


class ContextBlocked(wire.InvalidExchange):
    pass


def _check(condition, reason):
    if not condition:
        raise ContextBlocked(reason)


def manifest_digest(packet, *, repository, absent_paths=()):
    refs = wire.packet(packet, repository=repository)
    return _manifest(refs, absent_paths, packet['allowed_paths'])


def _manifest(refs, absent_paths, allowed_paths):
    entries = {}
    for ref in refs.values():
        path = ref['path']
        item = {k: ref[k] for k in ('path', 'sha256', 'byte_length', 'source_kind')}
        _check(path not in entries or entries[path] == item, 'conflicting snapshot path')
        entries[path] = item
    for path in absent_paths:
        wire.path(path)
        _check(path in allowed_paths and path not in entries, 'invalid absent target')
        entries[path] = {'path': path, 'source_kind': 'absent', 'sha256': None, 'byte_length': 0}
    return wire.digest([entries[k] for k in sorted(entries)])


def read_git_blob(root, commit, path, *, max_bytes):
    """Read an exact regular blob, never a symlink, from a pinned Git tree."""
    wire.hex_digest(commit, 40)
    wire.path(path)
    _check(type(max_bytes) is int and max_bytes > 0, 'invalid byte budget')
    def git(*args):
        try:
            return subprocess.run(['git', '--literal-pathspecs', *args], cwd=root,
                                  capture_output=True, check=True, timeout=15).stdout
        except (OSError, subprocess.SubprocessError) as exc:
            raise ContextBlocked('git object unavailable') from exc
    _check(git('cat-file', '-t', commit).strip() == b'commit', 'source is not commit')
    result = git('ls-tree', '-z', commit, '--', path)
    rows = result.split(b'\x00')
    _check(len(rows) == 2 and rows[-1] == b'', 'missing or ambiguous git path')
    try:
        info, name = rows[0].split(b'\t', 1)
        mode, kind, oid = info.split()
    except ValueError as exc:
        raise ContextBlocked('invalid git tree entry') from exc
    _check(name == path.encode('utf-8') and kind == b'blob' and mode in (b'100644', b'100755'), 'nonregular git path')
    oid = oid.decode('ascii')
    wire.hex_digest(oid, 40)
    size = int(git('cat-file', '-s', oid))
    _check(0 <= size <= max_bytes, 'blob exceeds budget')
    data = git('cat-file', 'blob', oid)
    _check(len(data) == size, 'blob size mismatch')
    return data


def prepare(packet, *, repository, git_reader, frozen_bytes, absent_paths=(),
            external=False, authorized_external=False, max_bytes=1048576,
            receiver_known=None):
    """Resolve a full capsule, returning exact bytes and a constructed receipt.

    Optional receiver_known is a trusted receiver manifest of section IDs and
    digests. It is not inferred from model output. W6.4 must establish delivery.
    The caller must supply independently authorized paths and external policy;
    packet classification alone is never authorization for export.
    """
    refs = wire.packet(packet, repository=repository)
    _check(type(external) is bool and type(authorized_external) is bool, 'invalid authority flags')
    _check(not external or authorized_external, 'external export not authorized')
    _check(type(max_bytes) is int and max_bytes > 0, 'invalid byte budget')
    _check(_manifest(refs, absent_paths, packet['allowed_paths']) == packet['snapshot_digest'], 'snapshot mismatch')
    if packet['stage'] == 'delegation':
        source_paths = {r['path'] for r in refs.values() if r['role'] == 'source' and r['required']}
        _check(set(packet['allowed_paths']) <= source_paths | set(absent_paths), 'uncovered delegation target')
    if external:
        _check(all(r['classification'] == 'cloud_permitted' for r in refs.values()), 'local-only context')
    _check(sum(r['byte_length'] for r in refs.values()) <= max_bytes, 'capsule exceeds budget')
    data, actual = {}, []
    for key, ref in refs.items():
        try:
            raw = (git_reader(packet['source_commit'], ref['path'], max_bytes=ref['byte_length'] or 1)
                   if ref['source_kind'] == 'git_blob' else frozen_bytes[ref['path']])
        except (KeyError, OSError) as exc:
            raise ContextBlocked('context unavailable') from exc
        _check(type(raw) is bytes and len(raw) == ref['byte_length'], 'context size mismatch')
        _check(hashlib.sha256(raw).hexdigest() == ref['sha256'], 'context hash mismatch')
        data[key] = raw
        actual.append(dict(ref))
    known = {} if receiver_known is None else receiver_known
    _check(type(known) is dict, "invalid receiver manifest")
    for key, sha in known.items():
        wire.ident(key); wire.hex_digest(sha)
    retained = {k for k, ref in refs.items() if known.get(k) == ref["sha256"]}
    mode = "resolved_refs" if retained == set(refs) else ("mixed" if retained else "full_capsule")
    receipt = {k: packet[k] for k in wire.IDENTITY}
    receipt.update(schema='fenix.context-receipt.v1', delivery_mode=mode,
                   delivery_state='constructed', content_refs=actual,
                   delivered_sections=[{'id': r['section_id'], 'sha256': r['sha256']} for r in actual],
                   omitted_sections=[])
    receipt['receipt_digest'] = wire.digest(receipt, 'receipt_digest')
    wire.receipt(receipt, packet, repository=repository)
    return {k: v for k, v in data.items() if k not in retained}, receipt
