# Development Guide

[← Back to README](../README.md)

## Repository structure

```text
fenixcrm/
├── cmd/            entrypoints
├── internal/
│   ├── api/        HTTP handlers and middleware
│   ├── domain/     CRM, agents, Blackboard, tools, policy, audit, knowledge, workflow
│   └── infra/      SQLite, LLM and integration runtime
├── docs/           architecture, guides, ADRs, plans and task records
├── reqs/           requirements and traceability
├── tests/          contract and integration tests
├── mobile/         React Native / Expo app
├── bff/            Express.js backend-for-frontend and admin surface
└── pkg/            shared Go utilities
```

## Local setup

```bash
make run
make test
make build
make lint
make complexity
```

Generate mobile screenshots:

```bash
cd mobile
npm run screenshots
```

Generate admin screenshots:

```bash
cd bff
npm run admin-screenshots
```

## Local quality hooks

Install the repository pre-push gates once:

```bash
make install-hooks
```

For CI details and Linux/POSIX notes, see [CI](ci.md).

## Screenshot troubleshooting

- [Maestro debug runbook — English](maestro-debug-apk-runbook-en.md)
- [Maestro debug runbook — Spanish](maestro-debug-apk-runbook-es.md)

## Architecture references

- [Full architecture](architecture.md)
- [Agent overview](agent-spec-overview.md)
- [Agent design](agent-spec-design.md)
- [Carta workflow architecture](plans/carta-language-server-flow.md)
