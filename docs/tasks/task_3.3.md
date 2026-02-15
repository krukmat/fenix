# Task 3.3 — Tool Definition & Registry

**Status**: ✅ Completed
**Phase**: 3 — AI Layer
**Goal**: Implementar registro de tools con definición persistida, validación de parámetros y endpoints admin para gestión básica.

---

## Objetivos

1. Crear persistencia `tool_definition` en SQLite.
2. Implementar dominio `ToolRegistry` con registro runtime de ejecutores.
3. Implementar validación de params contra esquema JSON (subset MVP).
4. Exponer API admin para listar/crear tools.
5. Cubrir tests unitarios + integración focalizados.

---

## Scope implementado (as-built)

### Migraciones

- `internal/infra/sqlite/migrations/016_tools.up.sql`
  - Tabla `tool_definition` con:
    - `id`, `workspace_id`, `name`
    - `description`, `input_schema`
    - `required_permissions`, `is_active`, `created_by`
    - `created_at`, `updated_at`
  - `UNIQUE(workspace_id, name)`
  - Índices:
    - `idx_tool_definition_workspace`
    - `idx_tool_definition_workspace_active`

- `internal/infra/sqlite/migrations/016_tools.down.sql`

### Dominio

- `internal/domain/tool/executor.go`
  - `ToolExecutor` con contrato:
    - `Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error)`

- `internal/domain/tool/registry.go`
  - `ToolRegistry{ db, executors }`
  - `Register(name, executor)`
  - `Get(name)`
  - `CreateToolDefinition(...)`
  - `ListToolDefinitions(...)`
  - `ValidateParams(...)` (validación JSON mínima sobre `required` + `additionalProperties` + `properties`)
  - Errores de dominio:
    - `ErrToolExecutorAlreadyRegistered`
    - `ErrToolExecutorNotRegistered`
    - `ErrToolDefinitionNotFound`
    - `ErrToolValidationFailed`

### API

- `internal/api/handlers/tool.go`
  - `GET /api/v1/admin/tools`
  - `POST /api/v1/admin/tools`

- `internal/api/routes.go`
  - wiring bajo `/api/v1/admin/tools` en rutas protegidas.

---

## Tests agregados

### Dominio

- `internal/domain/tool/registry_test.go`
  - Register + Get executor
  - ValidateParams con JSON inválido → error esperado
  - ListToolDefinitions recupera esquema/permisos desde DB
  - Cobertura de validadores de schema extraídos del refactor:
    - required faltante
    - `additionalProperties=false` rechaza campos desconocidos
    - `additionalProperties=true` permite campos desconocidos
    - comportamiento default de `additionalProperties`
    - `extractStringSlice` (filtrado de input inválido)

### API handlers

- `internal/api/handlers/tool_test.go`
  - Create tool (POST) + List tools (GET) end-to-end sobre DB migrada

---

## Validación ejecutada

Comandos ejecutados:

```bash
make complexity
golangci-lint run ./internal/domain/tool/... ./internal/api/...
go test ./internal/domain/tool ./internal/api/handlers ./internal/api
go test -race ./internal/domain/tool ./internal/api/handlers ./internal/api
COVERAGE_MIN=79 make coverage-gate
COVERAGE_APP_MIN=79 make coverage-app-gate
TDD_COVERAGE_MIN=79 make coverage-tdd
```

Resultado:

- ✅ `make complexity` OK
- ✅ `golangci-lint` OK
- ✅ `internal/domain/tool` OK
- ✅ `internal/api/handlers` OK
- ✅ `internal/api` OK
- ✅ `-race` OK en los 3 paquetes
- ✅ coverage gate global/app/TDD en verde

---

## Cierre final de CI

- Run final en GitHub Actions: **`22033174931`**
- Commit asociado: **`5a18aa54fc2a5e0bab73c1cefd9eb5ee733b35f4`**
- Estado: **`success`**
- Jobs:
  - ✅ Complexity Gate
  - ✅ Lint and Test (incluyendo race + coverage gates)
  - ✅ E2E Tests (skip esperado por proyecto E2E no presente)

---

## Checklist

- [x] Migración de tool registry creada (up/down)
- [x] Dominio ToolRegistry implementado
- [x] Endpoints admin tools implementados y ruteados
- [x] Tests unitarios/integración focalizados en verde
- [x] Validación completa con gates locales y CI remoto en verde
