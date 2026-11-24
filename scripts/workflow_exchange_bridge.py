"""Offline equivalence check at a native provider input boundary.

The supervisor supplies an already authorized packet, cache, native renderer
and the exact legacy input bytes. No provider invocation or approval occurs.
"""
import hashlib

import workflow_exchange as wire
import workflow_exchange_transport as transport


def prepare_native_input(raw, packet, *, repository, provider, render,
                         baseline_input, cache=None, authorized_external=False,
                         ollama_external=False, max_wire_bytes=4194304,
                         max_context_bytes=1048576, max_native_bytes=1048576):
    """Return equivalent native input bytes and nonauthoritative measurements.

    render is trusted code, not a model-generated function. Its only input is
    the reconstructed immutable section bytes. A native prompt that differs
    from the baseline is a different experiment, not an equivalent transport.
    Provider boundary delivery and actual usage must be observed separately.
    """
    wire.packet(packet, repository=repository)
    wire.enum(provider, {'claude', 'codex', 'ollama'})
    stage = packet['stage']
    wire.require((provider == 'ollama' and stage == 'delegation') or
                 (provider == 'claude' and stage in {'task_readiness', 'post_code_review'}) or
                 (provider == 'codex' and stage == 'task_readiness'),
                 'native stage unsupported for explicit context')
    wire.require(type(ollama_external) is bool, 'invalid Ollama boundary')
    external = provider in {'claude', 'codex'} or ollama_external
    wire.require(type(max_native_bytes) is int and 0 < max_native_bytes <= 64 * 1024 * 1024,
                 'invalid native budget')
    wire.require(type(baseline_input) is bytes and 0 < len(baseline_input) <= max_native_bytes,
                 'invalid legacy input')
    wire.require(callable(render), 'trusted renderer required')
    content, ack = transport.receive(raw, packet, repository=repository, cache=cache,
                                     external=external, authorized_external=authorized_external,
                                     max_wire_bytes=max_wire_bytes, max_context_bytes=max_context_bytes)
    candidate = render(dict(content))
    wire.require(type(candidate) is bytes and 0 < len(candidate) <= max_native_bytes,
                 'invalid native candidate')
    wire.require(candidate == baseline_input, 'native input differs from baseline')
    return candidate, dict(schema='fenix.native-input-equivalence.v1', provider=provider,
                           stage=stage, packet_digest=packet['input_digest'],
                           reconstruction=ack, transfer_bytes=len(raw),
                           native_input_bytes=len(candidate),
                           native_input_digest=hashlib.sha256(candidate).hexdigest(),
                           delivery_state='unknown', input_tokens=None,
                           inference_invoked=False)
