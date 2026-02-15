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

### API handlers

- `internal/api/handlers/tool_test.go`
  - Create tool (POST) + List tools (GET) end-to-end sobre DB migrada

---

## Validación ejecutada

Comandos ejecutados:

```bash
go test ./internal/domain/tool ./internal/api/handlers ./internal/api
go test -race ./internal/domain/tool ./internal/api/handlers ./internal/api
```

Resultado:

- ✅ `internal/domain/tool` OK
- ✅ `internal/api/handlers` OK
- ✅ `internal/api` OK
- ✅ `-race` OK en los 3 paquetes

Nota:

- `golangci-lint` no disponible en el entorno actual (`command not found`), por lo que lint global queda para CI/local con tooling instalado.

---

## Checklist

- [x] Migración de tool registry creada (up/down)
- [x] Dominio ToolRegistry implementado
- [x] Endpoints admin tools implementados y ruteados
- [x] Tests unitarios/integración focalizados en verde
- [x] Validación con `go test` + `-race`
