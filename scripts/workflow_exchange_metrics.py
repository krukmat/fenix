"""W6.6 candidate. Metadata only; no gate, provider call or approval decision."""
from collections import Counter, defaultdict

import workflow_exchange as w

EVENT = 'fenix.attempt-event.v1'
FIELDS = set('schema task_id run_id attempt_id stage repository source_commit snapshot_digest input_digest receipt_digest arm provider model channel role status start_ms end_ms usage'.split())


def attempt_event(packet, receipt, result, *, repository, arm, provider, model,
                  channel, role, start_ms, end_ms):
    """Project a validated envelope and trusted adapter timing onto metadata."""
    w.result(result, packet, receipt, repository=repository)
    event = {k: result[k] for k in ('task_id', 'run_id', 'attempt_id', 'stage',
             'repository', 'source_commit', 'snapshot_digest', 'input_digest',
             'receipt_digest', 'status', 'usage')}
    event.update(schema=EVENT, arm=arm, provider=provider, model=model,
                 channel=channel, role=role, start_ms=start_ms, end_ms=end_ms)
    validate_event(event)
    w.require(end_ms - start_ms == result['duration_ms'], 'duration mismatch')
    return event


def validate_event(event):
    w.obj(event, FIELDS)
    w.require(event['schema'] == EVENT, 'unsupported event schema')
    for k in ('task_id', 'run_id', 'attempt_id', 'provider', 'model', 'repository'):
        w.ident(event[k])
    w.hex_digest(event['source_commit'], 40)
    for k in ('snapshot_digest', 'input_digest'):
        w.hex_digest(event[k])
    if event['receipt_digest'] is not None:
        w.hex_digest(event['receipt_digest'])
    w.enum(event['stage'], w.STAGES); w.enum(event['status'], w.NORMAL | w.FAILURES)
    w.enum(event['arm'], {'baseline', 'candidate'})
    w.enum(event['channel'], {'cloud', 'local'})
    w.enum(event['role'], {'developer', 'primary', 'fallback', 'advisory'})
    w.require((event['stage'] == 'delegation') == (event['role'] == 'developer'), 'role/stage mismatch')
    w.require(event['status'] not in w.NORMAL or event['receipt_digest'] is not None, 'missing receipt')
    w.integer(event['start_ms']); w.integer(event['end_ms'])
    w.require(event['end_ms'] >= event['start_ms'], 'negative duration')
    usage = event['usage']
    w.obj(usage, 'input_tokens output_tokens cached_input_tokens measurement estimated_input_tokens'.split())
    w.enum(usage['measurement'], {'provider_reported', 'estimated', 'unavailable'})
    for k in ('input_tokens', 'output_tokens', 'cached_input_tokens', 'estimated_input_tokens'):
        if usage[k] is not None:
            w.integer(usage[k])
    if usage['measurement'] != 'provider_reported':
        w.require(all(usage[k] is None for k in ('input_tokens', 'output_tokens', 'cached_input_tokens')), 'unmeasured tokens')
    if usage['measurement'] == 'unavailable':
        w.require(usage['estimated_input_tokens'] is None, 'unexpected estimate')
    if usage['measurement'] == 'estimated':
        w.require(usage['estimated_input_tokens'] is not None, 'missing estimate')
    if usage['measurement'] == 'provider_reported':
        w.require(any(usage[k] is not None for k in ('input_tokens', 'output_tokens', 'cached_input_tokens')), 'missing measured tokens')


def aggregate(events):
    """Aggregate valid observed attempts; no inferred completion or acceptance.

    start/end offsets must share a supervisor's monotonic clock within each run.
    Cross-machine durations, verified closure and fallback-history completeness
    require a separate trusted supervisor record before E2E metrics are claimed.
    """
    by_key, bad_runs = {}, set()
    invalid = duplicates = 0
    for e in events:
        try:
            validate_event(e)
        except w.InvalidExchange:
            invalid += 1
            # A malformed event cannot safely supply identity.
            continue
        run = (e['repository'], e['task_id'], e['run_id'])
        key = run + (e['attempt_id'], e['stage'])
        if key in by_key:
            if w.digest(by_key[key]) == w.digest(e):
                duplicates += 1
            else:
                bad_runs.add(run)
            continue
        by_key[key] = e
    groups = defaultdict(list)
    for key, e in by_key.items():
        groups[key[:3]].append(e)
    for key, rows in groups.items():
        if len({(e['arm'], e['source_commit'], e['snapshot_digest']) for e in rows}) != 1:
            bad_runs.add(key)
    usable = [e for k, rows in groups.items() if k not in bad_runs for e in rows]
    def usage(rows, field):
        values = [e['usage'][field] for e in rows if e['usage'][field] is not None]
        return {'observed_total': sum(values) if values else None, 'covered': len(values), 'attempts': len(rows)}
    out = {}
    for arm in ('baseline', 'candidate'):
        rows = [e for e in usable if e['arm'] == arm]
        by_channel = {}
        for channel in ('cloud', 'local'):
            scope = [e for e in rows if e['channel'] == channel]
            by_channel[channel] = {k: usage(scope, k) for k in ('input_tokens', 'output_tokens', 'estimated_input_tokens')}
        out[arm] = {'attempts': len(rows), 'status_counts': dict(Counter(e['status'] for e in rows)),
                    'usage': by_channel, 'accepted_tasks': None, 'tokens_per_accepted_task': None,
                    'fallback_attempts': sum(e['role'] == 'fallback' for e in rows), 'fallback_rate': None,
                    'end_to_end_latency_ms': None}
    return {'schema': 'fenix.attempt-metrics.v1', 'invalid_records': invalid, 'duplicate_records': duplicates,
            'conflicting_runs': len(bad_runs), 'observed_runs': len(groups), 'arms': out,
            'limitations': ['No verified supervisor closure; acceptance and fallback rate unknown.',
                            'Observed input and estimated tokens are separate; no missing usage imputed.']}
