"""W6.5 advisory paired comparison. Does not authorize delegation."""
from math import sqrt
from statistics import median

import workflow_exchange as w


def wilson(successes, count):
    if not count:
        return None
    p = successes / count; z = 1.959963984540054
    d = 1 + z*z/count; center = (p + z*z/(2*count))/d
    half = z*sqrt(p*(1-p)/count + z*z/(4*count*count))/d
    return [max(0.0, center-half), min(1.0, center+half)]


def assess(pairs, *, rri, min_pairs=5):
    """Use supervisor-verified paired observations; no inference-based acceptance.

    Matching case/source/criteria and execution profile is an upstream obligation.
    Even CANDIDATE_PREFERRED is advisory; current RRI policy remains authoritative.
    """
    w.integer(rri); w.require(rri <= 100, 'invalid RRI')
    w.integer(min_pairs); w.require(min_pairs > 0, 'invalid minimum sample')
    w.array(pairs)
    if rri > 25:
        return {'status': 'UNKNOWN', 'reason': 'active_policy_forbids_local_delegation', 'authorization': False}
    seen = set(); valid = []
    for pair in pairs:
        w.obj(pair, {'case_id', 'matched', 'baseline', 'candidate'})
        w.ident(pair['case_id']); w.require(pair['case_id'] not in seen, 'duplicate case'); seen.add(pair['case_id'])
        w.require(type(pair['matched']) is bool, 'invalid matching flag')
        for arm in ('baseline', 'candidate'):
            v = pair[arm]
            w.obj(v, {'verified', 'accepted', 'critical_miss', 'elapsed_ms', 'input_tokens'})
            for k in ('verified', 'accepted', 'critical_miss'):
                w.require(type(v[k]) is bool, 'invalid observation flag')
            for k in ('elapsed_ms', 'input_tokens'):
                if v[k] is not None: w.integer(v[k])
            w.require(not (v['accepted'] and v['critical_miss']), 'invalid acceptance observation')
        if pair['matched'] and all(pair[a]['verified'] for a in ('baseline', 'candidate')):
            valid.append(pair)
    n = len(valid)
    report = {'status': 'UNKNOWN', 'matched_verified_pairs': n, 'total_pairs': len(pairs), 'authorization': False}
    if any(p['candidate']['critical_miss'] for p in valid):
        return dict(report, status='INCUMBENT_PREFERRED', reason='observed_candidate_critical_miss')
    if n < min_pairs:
        return dict(report, reason='insufficient_matched_verified_pairs')
    b = sum(p['baseline']['accepted'] for p in valid); c = sum(p['candidate']['accepted'] for p in valid)
    report.update(baseline_acceptance=b/n, candidate_acceptance=c/n, baseline_interval=wilson(b,n), candidate_interval=wilson(c,n))
    if any(p['baseline']['accepted'] and not p['candidate']['accepted'] for p in valid):
        return dict(report, status='INCUMBENT_PREFERRED', reason='observed_acceptance_regression')
    if any(p[a][k] is None for p in valid for a in ('baseline','candidate') for k in ('elapsed_ms','input_tokens')):
        return dict(report, reason='incomplete_efficiency_measurements')
    time = median(p['candidate']['elapsed_ms']-p['baseline']['elapsed_ms'] for p in valid)
    tokens = median(p['candidate']['input_tokens']-p['baseline']['input_tokens'] for p in valid)
    report.update(paired_median_elapsed_delta_ms=time, paired_median_input_token_delta=tokens)
    if c > 0 and c >= b and time <= 0 and tokens <= 0 and (time < 0 or tokens < 0):
        return dict(report, status='CANDIDATE_PREFERRED', reason='observed_noninferior_acceptance_and_efficiency')
    if time >= 0 and tokens >= 0 and (time > 0 or tokens > 0) and c <= b:
        return dict(report, status='INCUMBENT_PREFERRED', reason='observed_efficiency_regression')
    return dict(report, status='INCONCLUSIVE', reason='mixed_or_equal_observations')
