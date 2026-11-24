"""W6 candidate: rearrange native readiness sections for prefix-cache experiments.

No inference, omission, approval, or change to the default native gate. A caller
must authorize every section before handing a prepared prompt to a provider.
"""
import hashlib
from typing import Callable

NATIVE_ORDER = ('task', 'plan', 'task_card')
PREFIX_ORDER = ('plan', 'task_card', 'task')


def _bytes(value):
    if not isinstance(value, str):
        raise ValueError('expected native prompt text')
    return value.encode('utf-8')


def prepare(packet: dict, native_renderer: Callable, *, layout='native') -> tuple[str, dict]:
    """Use the native renderer, not a second set of reviewer instructions.

    Only readiness is eligible. Keep entire task/plan/card blocks, including
    markers. Verify the native renderer's composition before reordering. No
    assumptions about provider caching, token counts, or actual delivery.
    """
    if type(packet) is not dict or packet.get('mode') != 'task-readiness':
        raise ValueError('only task-readiness is eligible')
    if layout not in ('native', 'stable_prefix'):
        raise ValueError('unknown layout')
    if any(k in packet for k in ('diff', 'verification_log', 'base_ref')):
        raise ValueError('post-code evidence is not eligible')
    if not callable(native_renderer):
        raise ValueError('native renderer required')
    for key in NATIVE_ORDER:
        if key in packet and type(packet[key]) is not str:
            raise ValueError('invalid readiness section')
    if not packet.get('task', '').strip() or not packet.get('plan', '').strip():
        raise ValueError('task and plan are required')
    mode = 'task-readiness'
    prefix = native_renderer(mode, {})
    _bytes(prefix)
    blocks = {}
    for key in NATIVE_ORDER:
        if not packet.get(key):
            continue
        part = native_renderer(mode, {key: packet[key]})
        if not part.startswith(prefix) or part == prefix:
            raise ValueError('native renderer structure changed')
        blocks[key] = part[len(prefix):]
    baseline = native_renderer(mode, packet)
    if baseline != prefix + ''.join(blocks[k] for k in NATIVE_ORDER if k in blocks):
        raise ValueError('native renderer is not section-composable')
    order = NATIVE_ORDER if layout == 'native' else PREFIX_ORDER
    prompt = prefix + ''.join(blocks[k] for k in order if k in blocks)
    if len(_bytes(prompt)) != len(_bytes(baseline)):
        raise ValueError('layout changed content length')
    return prompt, {
        'schema': 'fenix.prompt-layout-candidate.v1',
        'layout': layout,
        'section_order': [k for k in order if k in blocks],
        'section_sha256': {k: hashlib.sha256(_bytes(v)).hexdigest() for k, v in blocks.items()},
        'prompt_sha256': hashlib.sha256(_bytes(prompt)).hexdigest(),
        'native_input_bytes': len(_bytes(prompt)),
        'provider_input_tokens': None,
        'provider_cached_tokens': None,
        'delivery_state': 'unknown',
        'inference_invoked': False,
    }


def common_prefix_bytes(prompts):
    """Byte-level shared prefix, not a prediction of tokenizer cache hits."""
    if type(prompts) is not list or len(prompts) < 2:
        raise ValueError('at least two prompts are required')
    raw = [_bytes(p) for p in prompts]
    limit = min(map(len, raw))
    first = raw[0]
    for i in range(limit):
        if any(p[i] != first[i] for p in raw[1:]):
            return i
    return limit
