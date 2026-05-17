# Prompt — Blackboard Multi-Agent Architecture

Read:
- ADR-100-agentic-blackboard-architecture.md

Objectives:
- evolve the current runtime toward collaborative multi-agent cognition
- preserve existing governance and auditability
- avoid distributed-system overengineering

Tasks:
1. Analyze current agent runtime
2. Design a shared cognitive workspace
3. Define:
   - workspace event model
   - reasoning events
   - hypothesis model
   - confidence propagation
4. Add:
   - shared workspace interfaces
   - reasoning timeline
   - collaborative signal registry
5. Preserve:
   - audit trail
   - policy enforcement
   - approval model

Deliverables:
- updated architecture docs
- ADR references
- implementation phases
- dependency graph
- migration strategy
