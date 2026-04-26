---
id: ADR-010
title: "Use modernc.org/sqlite (pure-Go) to guarantee CGO_ENABLED=0 builds"
date: 2026-02-20
status: accepted
deciders: [matias]
tags: [adr, go, sqlite, deploy, docker]
related_tasks: [task_post_mvp_docker_uat]
related_frs: [FR-300]
---

# ADR-010 — Use modernc.org/sqlite (pure-Go) to guarantee CGO_ENABLED=0 builds

## Status

`accepted`

## Context

SQLite in Go has two common driver options:

1. `mattn/go-sqlite3` — CGO-based. Requires a C compiler at build time. Produces a
   native binary linked against the SQLite C library. Standard and widely used.
2. `modernc.org/sqlite` — Pure-Go port of SQLite (transpiled from C to Go via `cgo`-free
   output). No C compiler required at build time.

For the Docker deployment of FenixCRM (`deploy/Dockerfile`), the build stage uses a
minimal Go image. The target runtime image is `distroless/static` — it has no C runtime,
no `libc`, and no `libsqlite3`. A CGO-linked binary would fail to start in this image.

## Decision

Use `modernc.org/sqlite` as the SQLite driver throughout the project. Build with:

```
CGO_ENABLED=0 go build ./...
```

This guarantees:
- The binary is fully self-contained (no external shared libraries)
- The binary runs in `distroless/static` or any Alpine image without additional packages
- Cross-compilation (e.g., `GOOS=linux GOARCH=arm64`) works without a cross-C-compiler

## Rationale

- `distroless/static` is the smallest secure runtime image for Go — enables the smallest
  possible Docker image
- `CGO_ENABLED=0` eliminates an entire class of build environment issues (missing
  `gcc`, missing `libsqlite3-dev`, cross-compilation toolchains)
- `modernc.org/sqlite` is a direct port of the same SQLite source — functionally
  identical for FTS5, WAL mode, and sqlite-vec extension usage
- Pure-Go also enables `go test` without CGO on any developer machine

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| `mattn/go-sqlite3` + CGO | Requires C compiler in build image; fails in distroless/static |
| `mattn/go-sqlite3` + debian runtime image | Larger image; includes C runtime (unnecessary attack surface) |
| PostgreSQL instead of SQLite | Valid for P1 scale, but adds infrastructure complexity for MVP |
| Build with CGO, load sqlite3 as shared lib in image | Complex multi-stage build; breaks distroless target |

## Consequences

**Positive:**
- `go build` works on any machine with Go installed, no C toolchain needed
- Docker image uses `distroless/static` — minimal, secure, ~20MB final image
- Cross-compilation to Linux/ARM64 (e.g., Raspberry Pi, DigitalOcean ARM) works out of the box

**Negative / tradeoffs:**
- `modernc.org/sqlite` is slightly slower than `mattn/go-sqlite3` for CPU-intensive
  queries (pure-Go interpretation vs. native C). Acceptable for the MVP workload.
- Some SQLite extensions that require C compilation (e.g., custom loadable extensions)
  are not available via this driver

## References

- `modernc.org/sqlite` — https://pkg.go.dev/modernc.org/sqlite
- `deploy/Dockerfile` — multi-stage build using `CGO_ENABLED=0`
- `docker-compose.yml` — local development stack
