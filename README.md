# FenixCRM

> **Agentic CRM OS** — A self-hosted, AI-native CRM platform combining operational CRM with evidence-based agents, RAG retrieval, and policy-driven governance.

---

## Overview

FenixCRM closes the gap between traditional CRMs (no agentic layer) and enterprise suites (vendor lock-in). It enables teams to build trustworthy AI workflows with:

- **Evidence-first AI**: No action without grounded evidence. Abstain when uncertain.
- **Tools, not mutations**: AI executes via registered, allowlisted tools. Never direct data writes.
- **Governance**: RBAC/ABAC, PII/no-cloud policies, approval chains, immutable audit logs.
- **Model-agnostic**: Local (Ollama/vLLM) or cloud (OpenAI/Anthropic) LLMs.

---

## Quick Start

### Prerequisites

- Go 1.22+
- SQLite (embedded)
- Docker (optional, for Ollama)
- Node.js 20+ (for frontend)

### Installation

```bash
# Clone repository
git clone https://github.com/matiasleandrokruk/fenix.git
cd fenix

# Install dev tools
make install-tools

# Build binary
make build

# Run tests
make test

# Start server
make run
```

### Verify Installation

```bash
# Check version
./fenix --version
# Output: fenix version dev (built ...)
```

---

## Development

### Project Structure

```
fenix/
├── cmd/fenix/           # Entry point
├── internal/            # Private application code
│   ├── domain/         # Business logic (CRM, Knowledge, Agent, etc.)
│   ├── infra/          # Infrastructure adapters (SQLite, LLM, etc.)
│   ├── api/            # HTTP handlers
│   ├── config/         # Configuration
│   ├── server/         # HTTP server setup
│   └── version/        # Version info
├── pkg/                # Public shared libraries
├── tests/              # Integration and E2E tests
└── docs/               # Documentation
```

### Available Commands

```bash
make test              # Run all tests
make test-unit         # Run unit tests only
make test-integration  # Run integration tests
make build             # Build binary
make run               # Run server (dev mode)
make lint              # Run linter
make fmt               # Format code
make migrate-up        # Apply database migrations
make migrate-down      # Rollback last migration
make sqlc-generate     # Generate Go code from SQL
make docker-build      # Build Docker image
make docker-run        # Run Docker container
make ci                # Run all CI checks
make help              # Show all commands
```

---

## Architecture

- **Stack**: Go 1.22+ / go-chi | SQLite (WAL) + sqlite-vec + FTS5 | React 19 + TypeScript + shadcn/ui
- **LLM**: Ollama (local) + OpenAI/Anthropic (cloud)
- **Deployment**: Single binary — `./fenix serve --port 8080`

See `docs/architecture.md` for full technical design.

---

## Implementation Plan

- **Duration**: 13 weeks (3 months)
- **Approach**: TDD (Test-Driven Development), incremental delivery
- **Phases**:
  1. Foundation (Weeks 1-3): CRM CRUD, Auth, Audit
  2. Knowledge & Retrieval (Weeks 4-6): Hybrid search, Evidence packs
  3. AI Layer (Weeks 7-10): Copilot, Agents, Tools, Policy
  4. Integration & Polish (Weeks 11-13): React UI, E2E tests

See `docs/implementation-plan.md` for detailed tasks.

---

## Core Principles

1. **Evidence-first**: All AI responses cite sources. Abstain if uncertain.
2. **Tools, not mutations**: AI only executes through registered tools.
3. **Governed**: Every action checked against policies.
4. **Auditable**: Immutable logs of all actions.
5. **Operable**: Full tracing, metrics, replay capability.

---

## Documentation

| Document | Description |
|----------|-------------|
| `docs/architecture.md` | Technical design, ERD, API specs |
| `docs/implementation-plan.md` | 13-week execution plan |
| `docs/CORRECTIONS-APPLIED.md` | Audit report of plan corrections |
| `CLAUDE.md` | Project guidance for Claude Code |
| `agentic_crm_requirements_agent_ready.md` | Requirements (FR/NFR/UC) |

---

## License

[License TBD]

---

**Status**: Phase 1, Task 1.1 — Project Setup (In Progress)
