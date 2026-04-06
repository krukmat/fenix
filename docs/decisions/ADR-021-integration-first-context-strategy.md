---
doc_type: adr
id: ADR-021
title: "Integration-first context strategy: native CRM tables plus external system provenance"
date: 2026-04-06
status: accepted
deciders: [matias]
tags: [adr, architecture, integrations, crm, context]
related_tasks: []
related_frs: [FR-050, FR-051, FR-091]
---

# ADR-021 — Integration-first context strategy: native CRM tables plus external system provenance

## Status

`accepted`

## Context

FenixCRM already has native CRM entities and knowledge ingestion. The repositioned product, however, must operate over customer context that may come from native records, external CRM/ticketing systems, or mixed sources. Treating integrations as peripheral would weaken the wedge.

## Decision

The architecture adopts an **integration-first context strategy**:

- native CRM tables remain valid context sources
- external system references are first-class and must be preserved in metadata/provenance
- connector contracts are part of the target architecture even if they start inside the knowledge layer
- the system of context may be native, integrated, or mixed per customer

## Consequences

### Positive

- the product can sit on top of existing customer systems
- ingestion and evidence can preserve explainable provenance
- the architecture no longer assumes native CRM ownership is required for value

### Tradeoffs

- connector contracts need stable source-identity fields
- some current metadata fields will later need extraction into stronger contracts

## References

- `docs/plans/fenixcrm_strategic_repositioning_spec.md`
- `docs/architecture.md`
- `docs/openapi.yaml`

