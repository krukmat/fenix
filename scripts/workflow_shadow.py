"""Opt-in native workflow observations, never a receipt or a verdict authority.

One record per observed native attempt. No additional provider calls. The caller
sets a unique FENIX_W6_RUN_ID across commands of the same task/run. Legacy input
has no validated snapshot or delivery attestation, so those metrics stay unknown.
"""
from collections import Counter, defaultdict
from contextlib import contextmanager
from contextvars import ContextVar
import json
import os
from pathlib import Path
import re
import sys
import time
import uuid

import workflow_exchange as wire

SCHEMA = 'fenix.native-shadow-attempt.v1'
FIELDS = set('schema task_id run_id attempt_id stage role provider model scope prepared_prompt_bytes input_tokens output_tokens native_status error_kind elapsed_ms delivery_state'.split())
STAGES = {'delegation', 'task-readiness', 'post-code-review'}
PROVIDERS = {'claude', 'codex', 'local-gemma', 'local-qwen', 'ollama'}
SCOPES = {'unknown', 'explicit_prompt', 'native_base_only', 'local_compacted_prompt', 'ollama_packet'}
STATUSES = {'unknown', 'pass', 'needs_changes', 'out_of_scope', 'blocked', 'patch', 'no_patch', 'error'}
ERRORS = {'none', 'timeout', 'unavailable', 'invalid', 'other'}
_CURRENT = ContextVar('fenix_w6_shadow', default=None)
_LOG_DIR = Path(__file__).resolve().parent.parent / 'logs' / 'workflow-shadow'


def enabled():
    return os.environ.get('FENIX_W6_SHADOW') == '1'


def task_id_from_text(value):
    if not enabled() or not isinstance(value, str) or not value.startswith('---\n'):
        return None
    front = value.split('---\n', 2)
    if len(front) != 3:
        return None
    found = re.findall(r'^id:\s*[\"\']?([A-Za-z0-9._-]{1,128})[\"\']?\s*$', front[1], re.M)
    return found[0] if len(found) == 1 else None


def _warn():
    try:
        print('[w6-shadow] observation unavailable; native behavior unchanged', file=sys.stderr)
    except Exception:
        pass


def _safe_id(value):
    return type(value) is str and re.fullmatch(r'[A-Za-z0-9._:-]{1,128}', value) is not None


def _number(value):
    return value if type(value) is int and 0 <= value <= wire.MAX_INT else None


def _error(value):
    if value in ('reviewer_timeout', 'timeout'):
        return 'timeout'
    if value in ('reviewer_unavailable', 'unavailable'):
        return 'unavailable'
    if value in ('invalid_verdict', 'invalid'):
        return 'invalid'
    return 'other'


def validate(event):
    wire.obj(event, FIELDS)
    wire.require(event['schema'] == SCHEMA, 'unknown shadow version')
    for key in ('task_id', 'run_id', 'attempt_id'):
        wire.require(_safe_id(event[key]), 'invalid correlation id')
    wire.enum(event['stage'], STAGES); wire.enum(event['provider'], PROVIDERS)
    wire.enum(event['role'], {'developer', 'primary', 'fallback', 'advisory'})
    wire.require((event['stage'] == 'delegation') == (event['role'] == 'developer'), 'stage/role mismatch')
    wire.require((event['role'] == 'developer') == (event['provider'] == 'ollama'), 'provider/role mismatch')
    wire.enum(event['scope'], SCOPES); wire.enum(event['native_status'], STATUSES)
    wire.enum(event['error_kind'], ERRORS)
    wire.require(event['delivery_state'] == 'unknown', 'legacy observation cannot attest delivery')
    if event['model'] is not None:
        wire.ident(event['model'])
    for key in ('prepared_prompt_bytes', 'input_tokens', 'output_tokens'):
        if event[key] is not None:
            wire.integer(event[key])
    wire.integer(event['elapsed_ms'])
    return event


def _write(event):
    validate(event)
    directory = _LOG_DIR
    if directory.is_symlink() or directory.parent.is_symlink():
        raise OSError('unsafe shadow directory')
    directory.mkdir(parents=True, exist_ok=True)
    path = directory / (event['attempt_id'] + '.json')
    # Unique files, exclusive creation, private permissions; never overwrite.
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, 'O_NOFOLLOW', 0), 0o600)
    with os.fdopen(fd, 'w', encoding='utf-8') as out:
        out.write(json.dumps(event, sort_keys=True, separators=(',', ':')) + '\n')


@contextmanager
def observe(stage, role, provider, *, task_id, active=True):
    """Best-effort telemetry only. Preserve exceptions and native exit semantics."""
    if not active or not enabled():
        yield None
        return
    try:
        run = os.environ.get('FENIX_W6_RUN_ID')
        if not (_safe_id(run) and _safe_id(task_id)):
            raise ValueError('missing trusted correlation')
        event = dict(schema=SCHEMA, task_id=task_id, run_id=run,
                     attempt_id=uuid.uuid4().hex, stage=stage, role=role,
                     provider=provider, model=None, scope='unknown',
                     prepared_prompt_bytes=None, input_tokens=None, output_tokens=None,
                     native_status='unknown', error_kind='none', elapsed_ms=0,
                     delivery_state='unknown')
        validate(event)
        started = time.monotonic_ns()
        token = _CURRENT.set(event)
    except Exception:
        _warn()
        yield None
        return
    try:
        yield event
    except BaseException as exc:
        event['native_status'] = 'error'
        name = type(exc).__name__
        event['error_kind'] = ('timeout' if 'Timeout' in name else
                               'unavailable' if name in ('URLError', 'ReviewerUnavailable') else
                               'invalid' if name == 'InvalidVerdict' else 'other')
        raise
    finally:
        _CURRENT.reset(token)
        try:
            event['elapsed_ms'] = max(0, (time.monotonic_ns() - started) // 1000000)
            _write(event)
        except Exception:
            _warn()


def note(scope, *, model=None, prompts=()):
    """Prepared bytes at the native adapter boundary, not provider input tokens."""
    event = _CURRENT.get()
    if event is None:
        return
    try:
        wire.enum(scope, SCOPES)
        event['scope'] = scope
        if model is not None:
            wire.ident(model)
            event['model'] = model
        if prompts:
            event['prepared_prompt_bytes'] = sum(len(s.encode('utf-8')) for s in prompts)
    except Exception:
        _warn()


def note_payload(scope, payload, *, model=None):
    if _CURRENT.get() is None:
        return
    try:
        note(scope, model=model, prompts=tuple(m['content'] for m in payload['messages']))
    except Exception:
        _warn()


def usage(read_usage):
    """Only call the existing usage extractor when shadow observation is enabled."""
    event = _CURRENT.get()
    if event is None:
        return
    try:
        counts = read_usage()
        event['input_tokens'] = _number(counts.prompt_tokens)
        event['output_tokens'] = _number(counts.response_tokens)
    except Exception:
        _warn()


def outcome(value):
    event = _CURRENT.get()
    if event is None:
        return
    try:
        status = value.get('status') if isinstance(value, dict) else value
        wire.enum(status, STATUSES)
        event['native_status'] = status
        reason = value.get('reason') if isinstance(value, dict) else None
        if status == 'blocked' and reason:
            event['error_kind'] = _error(reason)
    except Exception:
        _warn()


def failure(kind):
    event = _CURRENT.get()
    if event is not None:
        event['native_status'] = 'blocked'
        event['error_kind'] = _error(kind)


def aggregate(directory):
    """Read shadow observations separately from strict W6 exchange metrics."""
    unique, conflicted, invalid, duplicates = {}, set(), 0, 0
    directory = Path(directory)
    if directory.is_dir() and not directory.is_symlink():
        for path in sorted(directory.glob('*.json')):
            try:
                if path.is_symlink() or not path.is_file():
                    raise ValueError('unsafe observation file')
                with path.open('rb') as f:
                    raw = f.read(65537)
                e = validate(wire.decode(raw, max_bytes=65536))
            except (OSError, ValueError):
                invalid += 1
                continue
            run = (e['task_id'], e['run_id'])
            key = run + (e['attempt_id'],)
            if key in unique:
                if unique[key] == e:
                    duplicates += 1
                else:
                    conflicted.add(run)
            else:
                unique[key] = e
    rows = [v for k, v in unique.items() if k[:2] not in conflicted]
    groups = defaultdict(list)
    for e in rows:
        groups[e['provider']].append(e)
    def metric(records, field):
        values = [e[field] for e in records if e[field] is not None]
        return dict(observed_total=sum(values) if values else None, covered=len(values), attempts=len(records))
    return dict(schema='fenix.native-shadow-summary.v1', observed_attempts=len(rows),
                observed_runs=len({(e['task_id'], e['run_id']) for e in rows}),
                duplicate_records=duplicates, conflicting_runs=len(conflicted), invalid_records=invalid,
                by_scope=dict(Counter(e['scope'] for e in rows)),
                by_status=dict(Counter(e['native_status'] for e in rows)),
                by_provider={p: {k: metric(records, k) for k in ('input_tokens','output_tokens','prepared_prompt_bytes','elapsed_ms')} for p, records in sorted(groups.items())},
                fallback_attempts=sum(e['role'] == 'fallback' for e in rows),
                accepted_tasks=None, fallback_rate=None, tokens_per_accepted_task=None,
                end_to_end_latency_ms=None, delivery_coverage=None,
                limitations=['Native observations only; snapshot, delivery and verified closure unavailable.',
                             'Prepared bytes are not measured provider tokens. Attempt times are not end-to-end latency.'])
