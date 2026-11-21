"""Conservative, opt-in model routing for Fénix development tasks.

This module never runs models, approves work, applies patches or weakens peer
review. Its output is only a candidate model ID for an already low-RRI task.
"""
from __future__ import annotations

import os
from typing import Mapping, Sequence

ROLES = ("fix", "complex_code", "documentation", "analysis")
ALLOWED_PATHS = ("internal/", "mobile/", "bff/", "scripts/", "docs/", "features/", "cmd/", "pkg/")
PROTECTED_PATHS = (
    "internal/domain/auth/", "internal/domain/audit/", "internal/domain/policy/",
    "internal/infra/auth/", "docs/policies/", "docs/decisions/",
)
ROLE_ENV = {
    "fix": "FENIX_ROLE_FIX_MODEL",
    "complex_code": "FENIX_ROLE_COMPLEX_CODE_MODEL",
    "documentation": "FENIX_ROLE_DOCUMENTATION_MODEL",
    "analysis": "FENIX_ROLE_ANALYSIS_MODEL",
}


class RoutingBlocked(ValueError):
    """Fail closed: unsuitable task, missing policy evidence, or missing tag."""


def _safe_path(path: str) -> bool:
    if not isinstance(path, str) or not path or path.startswith("/") or "\\" in path:
        return False
    parts = path.split("/")
    if any(part in ("", ".", "..") for part in parts):
        return False
    if any(char in path for char in ("*", "?", "[", "]")):
        return False
    return any(path.startswith(prefix) for prefix in ALLOWED_PATHS)


def configured_low_rri_model(
    *, role: str, rri: int, paths: Sequence[str], task_id: str,
    environment: Mapping[str, str] | None = None,
) -> str:
    """Select a candidate; caller MUST verify its exact tag in Ollama /api/tags.

    No moderate-band local delegation: even approved RRI>=26 is rejected.
    No implicit fallback to an arbitrary installed model.
    """
    env = os.environ if environment is None else environment
    if not isinstance(task_id, str) or not task_id.strip():
        raise RoutingBlocked("task ID is required for model attribution")
    if type(rri) is not int or not (0 <= rri <= 25):
        raise RoutingBlocked("local routing is permitted only for RRI 0-25")
    if role not in ROLE_ENV:
        raise RoutingBlocked("unknown role")
    if not paths or not all(_safe_path(path) for path in paths):
        raise RoutingBlocked("unsafe or missing bounded paths")
    if any(path.startswith(prefix) for path in paths for prefix in PROTECTED_PATHS):
        raise RoutingBlocked("protected boundary: delegate to cloud supervisor")
    model = env.get(ROLE_ENV[role], "").strip()
    if not model:
        raise RoutingBlocked("role model is not configured; no silent fallback")
    return model


def advisory_precheck(raw: object) -> dict:
    """Validate an optional Nimble verdict. Never approve a review gate."""
    if not isinstance(raw, dict):
        return {"status": "ESCALATE", "valid": False, "peer_review_required": True}
    status, reason, evidence = (raw.get("status"), raw.get("reason"), raw.get("evidence_refs"))
    valid = (
        status in ("PASS", "FINDINGS", "ESCALATE")
        and isinstance(reason, str) and bool(reason.strip())
        and isinstance(evidence, list)
        and all(isinstance(item, str) for item in evidence)
    )
    return {
        "status": status if valid else "ESCALATE",
        "valid": valid,
        "peer_review_required": True,
        "reason": reason.strip() if valid else "invalid_precheck_contract",
        "evidence_refs": evidence if valid else [],
    }
