"""W6 candidate: pure wire validation, not authorization or delivery proof."""
import hashlib
import json
import re

MAX_INT = 2**53 - 1
COMMON = set('schema repository source_commit snapshot_digest task_id run_id attempt_id stage input_digest'.split())
IDENTITY = COMMON - {'schema'}
STAGES = {'delegation', 'task_readiness', 'post_code_review'}
ROLES = set('task plan policy rri source diff verification native_output'.split())
NORMAL = {'PROPOSED', 'NO_PATCH', 'REVIEW_REPORTED'}
FAILURES = {'BLOCKED', 'INVALID', 'TIMEOUT'}

class InvalidExchange(ValueError):
    pass

def require(condition, message):
    if not condition:
        raise InvalidExchange(message)

def obj(value, fields):
    require(type(value) is dict and set(value) == set(fields), 'unexpected or missing fields')

def text(value):
    require(type(value) is str and bool(value.strip()), 'expected nonempty string')
    require(not any(0xD800 <= ord(c) <= 0xDFFF for c in value), 'invalid Unicode')

def ident(value):
    text(value)
    require(re.fullmatch(r'[A-Za-z0-9._:/-]{1,128}', value) is not None, 'invalid identifier')

def integer(value):
    require(type(value) is int and 0 <= value <= MAX_INT, 'invalid unsigned integer')

def hex_digest(value, size=64):
    require(type(value) is str and re.fullmatch('[0-9a-f]{%d}' % size, value) is not None, 'invalid digest')

def enum(value, values):
    require(type(value) is str and value in values, 'invalid enum')

def array(value):
    require(type(value) is list, 'expected array')

def path(value):
    text(value)
    require(not any(c in value for c in '\\:*?[]\x00') and not any(ord(c) < 32 or ord(c) == 127 for c in value) and all(p not in ('', '.', '..') for p in value.split('/')), 'unsafe relative path')

def canonical(value):
    if value is None or type(value) is bool:
        return json.dumps(value)
    if type(value) is int:
        require(abs(value) <= MAX_INT, 'unsafe integer')
        return str(value)
    if type(value) is str:
        require(not any(0xD800 <= ord(c) <= 0xDFFF for c in value), 'invalid Unicode')
        return json.dumps(value, ensure_ascii=False)
    if type(value) is list:
        return '[' + ','.join(canonical(v) for v in value) + ']'
    if type(value) is dict:
        for k in value:
            require(type(k) is str, 'invalid key')
            canonical(k)
        keys = sorted(value, key=lambda k: k.encode('utf-16-be'))
        return '{' + ','.join(canonical(k) + ':' + canonical(value[k]) for k in keys) + '}'
    raise InvalidExchange('unsupported JSON value')

def digest(value, exclude=None):
    if exclude is not None:
        require(type(value) is dict, 'digest object required')
        value = {k: v for k, v in value.items() if k != exclude}
    return hashlib.sha256(canonical(value).encode('utf-8')).hexdigest()

def decode(raw, *, max_bytes):
    require(type(raw) is bytes and type(max_bytes) is int and max_bytes > 0 and len(raw) <= max_bytes, 'invalid input size')
    def unique(pairs):
        out = {}
        for k, v in pairs:
            require(k not in out, 'duplicate key')
            out[k] = v
        return out
    try:
        value = json.loads(raw.decode('utf-8'), object_pairs_hook=unique)
        canonical(value)
        return value
    except (UnicodeError, ValueError, RecursionError) as exc:
        raise InvalidExchange('invalid JSON: ' + str(exc)) from exc

def common(value, schema, extra, repository):
    obj(value, COMMON | set(extra))
    require(value['schema'] == 'fenix.' + schema + '.v1', 'unsupported schema')
    require(type(repository) is str and value['repository'] == repository and re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repository) is not None, 'repository mismatch')
    hex_digest(value['source_commit'], 40)
    for key in ('snapshot_digest', 'input_digest'):
        hex_digest(value[key])
    for key in ('task_id', 'run_id', 'attempt_id'):
        ident(value[key])
    enum(value['stage'], STAGES)

def ref(value):
    obj(value, 'section_id role path sha256 byte_length source_kind required classification'.split())
    ident(value['section_id']); enum(value['role'], ROLES); path(value['path'])
    hex_digest(value['sha256']); integer(value['byte_length'])
    enum(value['source_kind'], {'git_blob', 'frozen_snapshot'})
    enum(value['classification'], {'local_only', 'cloud_permitted'})
    require(type(value['required']) is bool, 'invalid required flag')

def refs(values):
    array(values); result = {}
    for value in values:
        ref(value); key = value['section_id']
        require(key not in result, 'duplicate section'); result[key] = value
    return result

def packet(value, *, repository):
    common(value, 'task-packet', 'caller objective acceptance_criteria allowed_paths policy_refs rri_evidence_ref context_refs'.split(), repository)
    obj(value['caller'], {'provider', 'model', 'channel'})
    for k in ('provider', 'model'): ident(value['caller'][k])
    enum(value['caller']['channel'], {'local_cli', 'remote_handoff', 'local_inference'})
    text(value['objective']); array(value['acceptance_criteria']); array(value['allowed_paths'])
    criteria = set()
    for criterion in value['acceptance_criteria']:
        obj(criterion, {'id', 'text'}); ident(criterion['id']); text(criterion['text'])
        require(criterion['id'] not in criteria, 'duplicate criterion'); criteria.add(criterion['id'])
    require(bool(criteria), 'missing criteria')
    for p in value['allowed_paths']: path(p)
    require(len(set(value['allowed_paths'])) == len(value['allowed_paths']), 'duplicate path')
    policy = refs(value['policy_refs']); context = refs(value['context_refs']); rri = value['rri_evidence_ref']; ref(rri)
    require(bool(policy) and all(r['role'] == 'policy' and r['required'] for r in policy.values()), 'missing policy')
    require(rri['role'] == 'rri' and rri['required'], 'missing RRI')
    required_roles = {'task', 'plan'} | ({'diff', 'verification'} if value['stage'] == 'post_code_review' else set())
    require(required_roles <= {r['role'] for r in context.values() if r['required']}, 'missing mandatory evidence')
    combined = refs(list(policy.values()) + [rri] + list(context.values()))
    require(digest(value, 'input_digest') == value['input_digest'], 'packet digest mismatch')
    return combined

def bind(value, source):
    require(all(value[k] == source[k] for k in IDENTITY), 'identity mismatch')

def receipt(value, source, *, repository):
    expected = packet(source, repository=repository)
    common(value, 'context-receipt', 'delivery_mode delivery_state content_refs delivered_sections omitted_sections receipt_digest'.split(), repository)
    bind(value, source); hex_digest(value['receipt_digest'])
    enum(value['delivery_mode'], {'full_capsule', 'resolved_refs', 'mixed'})
    enum(value['delivery_state'], {'constructed', 'delivered', 'receiver_confirmed'})
    actual = refs(value['content_refs']); array(value['delivered_sections']); array(value['omitted_sections']); seen = set()
    for delivered in value['delivered_sections']:
        obj(delivered, {'id', 'sha256'}); ident(delivered['id']); hex_digest(delivered['sha256']); k = delivered['id']
        require(k not in seen and k in expected and k in actual and actual[k] == expected[k], 'invalid delivered section')
        require(delivered['sha256'] == actual[k]['sha256'], 'delivery digest mismatch'); seen.add(k)
    require(set(actual) == seen, 'unaccounted delivered content')
    for omitted in value['omitted_sections']:
        obj(omitted, {'id', 'reason'}); ident(omitted['id']); text(omitted['reason']); k = omitted['id']
        require(k not in seen and k in expected, 'invalid omitted section'); seen.add(k)
    require(set(expected) == seen, 'unaccounted sections')
    require(digest(value, 'receipt_digest') == value['receipt_digest'], 'receipt digest mismatch')
    return not any(expected[r['id']]['required'] for r in value['omitted_sections'])

def result(value, source, delivery, *, repository):
    packet(source, repository=repository)
    common(value, 'result-envelope', 'receipt_digest status evidence_refs error_code usage duration_ms'.split(), repository)
    bind(value, source); enum(value['status'], NORMAL | FAILURES); integer(value['duration_ms'])
    evidence = refs(value['evidence_refs']); usage = value['usage']
    obj(usage, 'input_tokens output_tokens cached_input_tokens measurement estimated_input_tokens'.split())
    enum(usage['measurement'], {'provider_reported', 'estimated', 'unavailable'})
    for k in ('input_tokens', 'output_tokens', 'cached_input_tokens', 'estimated_input_tokens'):
        if usage[k] is not None: integer(usage[k])
    if usage['measurement'] != 'provider_reported':
        require(all(usage[k] is None for k in ('input_tokens', 'output_tokens', 'cached_input_tokens')), 'unmeasured tokens')
    if usage['measurement'] == 'unavailable': require(usage['estimated_input_tokens'] is None, 'unexpected estimate')
    if usage['measurement'] == 'estimated': require(usage['estimated_input_tokens'] is not None, 'missing estimate')
    if usage['measurement'] == 'provider_reported': require(any(usage[k] is not None for k in ('input_tokens', 'output_tokens', 'cached_input_tokens')), 'missing measured tokens')
    complete = False
    if delivery is None:
        require(value['receipt_digest'] is None, 'missing receipt')
    else:
        complete = receipt(delivery, source, repository=repository)
        require(delivery['delivery_state'] != 'constructed' and value['receipt_digest'] == delivery['receipt_digest'], 'delivery not established')
    if value['status'] in NORMAL:
        require(delivery is not None and complete and value['error_code'] is None and any(r['role'] == 'native_output' for r in evidence.values()), 'missing normal result evidence')
        require((value['stage'] == 'delegation') == (value['status'] in {'PROPOSED', 'NO_PATCH'}), 'status/stage mismatch')
    else:
        ident(value['error_code'])
    # No acceptance result: actual authorization and verified delivery belong to trusted callers.
