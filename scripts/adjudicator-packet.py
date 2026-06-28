"""
adjudicator-packet.py — D14 adjudication gate for Fenix code-review workflow.

Decides when context-isolated adjudication is required and builds the isolation
packet passed to the adjudicator subagent.

Policy reference: docs/policies/HITL_AUTONOMY_POLICY.md

Pure logic module — no IO, no network, no file writes.
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

# Band labels must match rri.py resolve_band() output exactly.
BANDS_MED_HIGH_OR_ABOVE = {"Med-high", "Complex", "High", "Very high", "Excessive"}

# Allowlist enforced by _assert_packet_isolation (D14 isolation invariant).
ALLOWED_PACKET_SECTIONS = {"diff", "criteria", "reconciled_findings"}

DISPOSITION_DIVERGENCE_VALUES = {"none", "partial", "full"}


def should_adjudicate(
    aggregate: Dict[str, Any],
    band: str,
    gemma_blocked: bool = False,
) -> bool:
    """Return True if the D14 isolation adjudication gate should fire.

    Triggers:
    - Gemma pass was blocked/unavailable.
    - RRI band is Med-high or above.
    - Any consensus finding has severity 'blocking' or 'major'.
    - Inter-pass disagreement detected (severity or location inconsistency > 0).
    """
    if gemma_blocked:
        return True
    if band in BANDS_MED_HIGH_OR_ABOVE:
        return True
    findings: List[Dict[str, Any]] = aggregate.get("findings", [])
    if any(f.get("severity") in {"blocking", "major"} for f in findings):
        return True
    recon: Dict[str, Any] = aggregate.get("reconciliation", {})
    if recon.get("severity_inconsistent", 0) > 0:
        return True
    if recon.get("location_inconsistent", 0) > 0:
        return True
    return False


def build_adjudicator_packet(
    diff: str,
    criteria: Any,
    reconciled_findings: List[Dict[str, Any]],
    disposition_divergence: Optional[str] = None,
) -> Dict[str, Any]:
    """Build and return the isolation packet for the adjudicator subagent.

    Raises RuntimeError if any key outside ALLOWED_PACKET_SECTIONS is present
    (D14 isolation invariant).

    disposition_divergence: optional audit field — one of 'none', 'partial',
    'full', or None.
    """
    if disposition_divergence is not None and disposition_divergence not in DISPOSITION_DIVERGENCE_VALUES:
        raise ValueError(
            f"disposition_divergence must be one of {DISPOSITION_DIVERGENCE_VALUES} or None, "
            f"got {disposition_divergence!r}"
        )
    packet: Dict[str, Any] = {
        "diff": diff,
        "criteria": criteria,
        "reconciled_findings": reconciled_findings,
    }
    _assert_packet_isolation(packet)
    return packet


def _assert_packet_isolation(packet: Dict[str, Any]) -> None:
    """Raise RuntimeError if packet contains keys outside ALLOWED_PACKET_SECTIONS."""
    unknown = set(packet.keys()) - ALLOWED_PACKET_SECTIONS
    if unknown:
        raise RuntimeError(f"D14 isolation violation: unexpected packet sections {unknown}")
