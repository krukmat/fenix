---
doc_type: task
id: W5-FINAL
title: Final integrated validation of Fenix, Mermaid2SF and VEL
status: ready
phase: integration
week: W5
tags: [fenix, w5, validation, mermaid2sf, vel]
blocked_by: [W5-C]
blocks: [W6]
created: 2026-09-22
completed:
---

# Task W5-FINAL — Final integrated validation

**Plan**: [W5 Final Validation](../plans/fenix-integration-w5-final-validation.md)

## Tasks

- **F1** Validate governed Fenix → Mermaid2SF execution and normalized result.
- **F2** Validate governed Fenix → VEL append/checkpoint/verify lifecycle.
- **F3** Validate the M2SF/VEL independent-participation matrix.
- **F4** Validate failure, recovery and no-business-replay semantics.
- **F5** Reconfirm current cross-repo provider compatibility.
- **F6** Close W5 documentation and produce the W6 handoff.

## Constraints

- Do not add a new broker, policy engine, semantic IR or cryptographic implementation.
- Do not make M2SF and VEL mandatory together.
- Do not reopen W5-C solely for the accepted 82.9% vs 83.0% coverage delta.
- Work directly on `main`.
