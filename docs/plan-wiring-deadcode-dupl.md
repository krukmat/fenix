# Plan: Wiring de subsistemas + deadcode blocking + reducción de dupl

## Context

El CI tiene 183 hallazgos de deadcode porque subsistemas completos están implementados pero no conectados a `cmd/fenix/main.go`. El gate de deadcode corre con `|| true` (no bloquea). El threshold de `dupl` en la pattern gate es 150.

**Objetivo**: conectar los subsistemas faltantes → hacer deadcode gate bloqueante → reducir threshold de dupl directo a 120.

---

## Subsistemas no conectados (fuente de los 183 hallazgos)

| Subsistema | Hallazgos | Archivos clave |
|---|---|---|
| Scheduler | ~31 | `domain/scheduler/service.go`, `worker.go`, `repository.go` |
| SkillRunner | 42 | `domain/agent/skill_runner.go` |
| A2A ProtocolHandler | 39 | `domain/agent/protocol_handler_a2a.go` |
| MCP Gateway | 10 | `domain/tool/mcp_adapter.go` |
| Handler utils (context.go) | 2 | `api/context.go` — ya reachable via tests con `-test` |

---

## Paso 1 — Wiring del Scheduler + WorkflowService (elimina ~31 hallazgos)

**Archivos a modificar**: `internal/api/routes.go`

Agregar antes de la creación de `workflowService`:

```go
import schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"

schedulerRepo := schedulerdomain.NewRepository(db)
schedulerSvc  := schedulerdomain.NewService(schedulerRepo)
```

Reemplazar:
```go
// Before
workflowService := workflowdomain.NewService(db)

// After
workflowRepo    := workflowdomain.NewRepository(db)
workflowService := workflowdomain.NewServiceWithDependencies(workflowRepo, schedulerSvc)
```

`workflowdomain.NewRepository(db)` existe en `internal/domain/workflow/repository.go:69`.
`workflowdomain.NewServiceWithDependencies` ya existe y está testeado. La interfaz `workflowScheduler { CancelBySource }` es satisfecha por `scheduler.Service`.

**Riesgo**: Bajo — `cancelScheduledJobsForWorkflow` ya guarda nil-check del scheduler.

---

## Paso 2 — WorkflowResumeHandler + Worker goroutine (completa Scheduler)

**Archivos a modificar**: `internal/api/routes.go`

Agregar después de que `toolRegistry`, `agentOrchestrator`, `dslRunner`, `signalService`, `policyEngine`, `approvalService`, `auditService` estén creados:

```go
resumeRC := &agent.RunContext{
    Orchestrator:    agentOrchestrator,
    ToolRegistry:    toolRegistry,
    PolicyEngine:    policyEngine,
    ApprovalService: approvalService,
    Scheduler:       schedulerSvc,
    SignalService:   signalSvc,
    AuditService:    auditService,
    DB:              db,
}
resumeHandler   := agent.NewWorkflowResumeHandler(dslRunner, resumeRC)
schedulerWorker := schedulerdomain.NewWorker(schedulerRepo, resumeHandler.Handle)
go func() {
    if err := schedulerWorker.Start(context.Background()); err != nil && !errors.Is(err, context.Canceled) {
        _ = err
    }
}()
```

`agent.NewWorkflowResumeHandler` existe en `internal/domain/agent/workflow_resume_handler.go`.
`RunContext` ya tiene los campos `Scheduler schedulerdomain.Scheduler` y `SignalService *signaldomain.Service`.
Patrón de goroutine idéntico al de `embedder.Start(context.Background(), ...)` existente.

**Riesgo**: Bajo-medio — el Worker auto-limita concurrencia (default: 10 jobs simultáneos, poll cada 10s).

---

## Paso 3 — SkillRunner en RunnerRegistry (elimina 42 hallazgos)

**Archivos a modificar**:
1. `internal/domain/agent/agents/registry.go` — agregar constante + función
2. `internal/api/routes.go` — wiring

**3a — `agents/registry.go`** (igual al patrón de `RegisterDSLRunner` existente):

```go
const AgentTypeSkill = "skill"

func RegisterSkillRunner(registry *agent.RunnerRegistry, runner agent.Runner) error {
    if registry == nil {
        return ErrRunnerRegistryNil
    }
    if runner == nil {
        return ErrDSLRunnerNil
    }
    return registry.Register(AgentTypeSkill, runner)
}
```

**3b — `routes.go`** (después de `dslRunner`):

```go
skillRunner := agent.NewSkillRunner(db)
_ = agents.RegisterSkillRunner(runnerRegistry, skillRunner)
```

No se necesita nuevo endpoint HTTP: el orchestrator ya resuelve runners por `agent_type` en `RunnerRegistry`. Un agente con `agent_type="skill"` se dispara vía el flujo genérico existente.

**Riesgo**: Bajo — únicamente wiring + registro.

---

## Paso 4 — A2A ProtocolHandler en RunContext (elimina 39 hallazgos)

`ProtocolHandler` es la interfaz en `internal/domain/agent/protocol_handler.go:19`.
`A2AProtocolHandler` la implementa (`protocol_handler_a2a.go`).
`RunContext` en `internal/domain/agent/runner.go` **no tiene** aún el campo `ProtocolHandler`.

**4a — `internal/domain/agent/runner.go`**: agregar campo al struct:

```go
type RunContext struct {
    // ... campos existentes ...
    ProtocolHandler ProtocolHandler  // nil = solo dispatch interno
}
```

**4b — `internal/api/routes.go`**: construir e inyectar:

```go
a2aHandler := agent.NewA2AProtocolHandler()
```

Agregar `ProtocolHandler: a2aHandler` a `resumeRC` (Paso 2) y verificar si el `workflowHandler` también construye un `RunContext` — si es así, inyectarlo también.

**Nota sobre dispatch routing**: agregar el campo hace el `A2AProtocolHandler` completo reachable por deadcode (elimina los 39 hallazgos). La lógica de routing en `dslRuntimeExecutor.executeDispatchOperation` (usar A2A cuando hay endpoint externo vs dispatch interno) puede ir en un commit posterior — la función es nil-guardeable.

**Riesgo**: Bajo (solo el campo en RunContext + construcción). El routing condicional tiene riesgo medio y se puede diferir.

---

## Paso 5 — MCP Gateway: diferir

`NewMCPGateway` requiere `MCPResourceProvider`, interfaz para la que **no existe ninguna implementación** en el codebase. Es infraestructura P1.

Los 10 hallazgos de MCP se filtran en el gate de CI (ver Paso 6).

---

## Paso 6 — Deadcode gate: de warn a blocking

**Archivo**: `.github/workflows/ci.yml`

Cambiar el step "Dead code report (warn only)":

```yaml
- name: Dead code gate (blocking)
  run: |
    go install golang.org/x/tools/cmd/deadcode@latest
    deadcode -test ./cmd/fenix/... ./cmd/frtrace/... 2>&1 \
      | grep -v "mcp_adapter\|MCPGateway\|BuildServer\|MCPResourceProvider\|MCPResourceDescriptor\|MCPResourcePayload" \
      | tee deadcode-report.txt || true
    LINES=$(grep -c "." deadcode-report.txt 2>/dev/null || echo 0)
    echo "Dead code findings (after MCP allowlist): $LINES"
    if [ "$LINES" -gt 0 ]; then
      echo "::error::$LINES unexpected dead code finding(s) — see deadcode-report artifact"
      exit 1
    fi
    echo "Dead code gate: PASSED"
```

El allowlist es simbólico (por nombre de archivo/función). Cuando MCP se conecte en P1, se elimina el grep.

**Riesgo**: Bajo — se ejecuta después de que los pasos 1-4 reducen el resto a 0.

---

## Paso 7 — Reducción de dupl threshold (directo a 120)

El threshold de `dupl` está en `.golangci.yml` bajo `linters-settings.dupl.threshold`.
La pattern gate ya corre `golangci-lint run --enable-only=dupl` usando esta config.

**Cambio**: `.golangci.yml` línea `threshold: 150` → `120` (salto directo, sin paso intermedio a 130).

Antes de commitear, correr localmente y fijar cualquier hallazgo nuevo (especialmente funciones `finalize*` en `dsl_runner.go` que tienen estructura paralela):
```powershell
golangci-lint run --enable-only=dupl ./...
```

Si hay hallazgos, refactorizar primero (mismo approach: Extract Method, helpers genéricos). Solo commitear cuando `dupl` reporte 0 hallazgos a threshold 120.

**Riesgo**: Medio — funciones `finalize*` en `dsl_runner.go` tienen estructura paralela y pueden requerir extracción de helpers.

---

## Orden de commits

```
Commit 1: Paso 1 + Paso 2 (scheduler + workflow + worker goroutine)
Commit 2: Paso 3 (skill runner)
Commit 3: Paso 4 (A2A RunContext field)
Commit 4: Paso 6 (deadcode gate blocking)
Commit 5: Paso 7 — dupl 150→120 + refactor de hallazgos nuevos + verificación
```

Cada commit pasa Backend Gate local (`go test ./internal/...`) antes de push.

---

## Archivos críticos

| Archivo | Cambio |
|---|---|
| `internal/api/routes.go` | Wiring scheduler, workflow, skill runner, A2A (Pasos 1-4) |
| `internal/domain/agent/runner.go` | Agregar `ProtocolHandler ProtocolHandler` a RunContext (Paso 4a) |
| `internal/domain/agent/agents/registry.go` | `AgentTypeSkill` + `RegisterSkillRunner` (Paso 3a) |
| `.github/workflows/ci.yml` | Deadcode gate blocking con MCP allowlist (Paso 6) |
| `.golangci.yml` | `dupl.threshold` 150→120 directo (Paso 7) |

---

## Verificación end-to-end

1. `go test ./internal/domain/scheduler/... ./internal/domain/agent/... ./internal/domain/workflow/...`
2. `go test ./internal/api/...`
3. `go build ./cmd/fenix/...` — confirma que `routes.go` compila
4. CI verde: todos los jobs incluyendo "Dead code gate (blocking)"
5. Pattern gate strict: 0 hallazgos a threshold 120
