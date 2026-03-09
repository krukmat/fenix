# AGENT_SPEC — Documento de Diseño

> **Fecha**: 2026-03-09
> **Derivado de**: `docs/agent-spec-use-cases.md` (behaviors B1-B8 con 3 niveles de detalle)
> **Informa a**: `docs/agent-spec-transition-plan.md` (Parte 3: Implementacion)
> **Principio**: Cada decision de diseño esta justificada por al menos un caso de uso.

---

## 1. Mapa: Casos de Uso → Componentes

La siguiente tabla muestra que componente satisface cada behavior (y sus sub-casos).

| Behavior | Componente principal | Componentes de soporte |
|---|---|---|
| B1: Definicion de Workflow | `WorkflowService` | `WorkflowRepository`, validador de campos |
| B2: Verificacion | `Judge` | `DSLParser`, `SpecParser`, `JudgeChecks` |
| B3: Ejecucion | `DSLRunner` + `DSLRuntime` | `AgentRunner`, `RunnerRegistry`, `RunContext`, `ExpressionEvaluator`, `VerbMapper` |
| B4: Deteccion de Signals | `SignalService` | `SignalRepository`, `EventBus` |
| B5: Accion Diferida | `Scheduler` | `ScheduledJobRepository`, `WorkflowResumeHandler` |
| B6: Override Humano | `ApprovalService` | `OverrideRecord` en `AgentRun` |
| B7: Versionado y Rollback | `WorkflowVersionService` | `WorkflowRepository` (parent_version_id) |
| B8: Delegacion | `ProtocolHandler` | `DispatchClient`, `DelegationRecord` en `AgentRun` |

---

## 2. Catalogo de Componentes

### 2.1 WorkflowService

**Satisface**: B1 (todos los sub-casos), B7.

**Responsabilidades**:
- Crear workflow como draft (v1) con validacion de campos obligatorios → B1.3
- Validar DSL source no vacio → B1.5
- Validar limite de tamano (64KB) → B1.7
- Rechazar creacion con nombre duplicado → B1.2
- Permitir edicion libre del DSL en draft → B1 principal
- Rechazar edicion de workflow no-draft → B1.4
- Last-write-wins en ediciones concurrentes → B1.8

**Interfaz**:

```go
type WorkflowService interface {
    Create(ctx context.Context, input CreateWorkflowInput) (*Workflow, error)
    Update(ctx context.Context, workspaceID, id string, input UpdateWorkflowInput) (*Workflow, error)
    Get(ctx context.Context, workspaceID, id string) (*Workflow, error)
    List(ctx context.Context, workspaceID string, filters WorkflowFilters) ([]*Workflow, error)
    GetActiveByAgent(ctx context.Context, workspaceID, agentDefinitionID string) (*Workflow, error)
    Activate(ctx context.Context, workspaceID, id string) (*Workflow, error)   // B2.6, B7
    Archive(ctx context.Context, workspaceID, id string) (*Workflow, error)    // B7
    NewVersion(ctx context.Context, workspaceID, id string) (*Workflow, error) // B7
    Rollback(ctx context.Context, workspaceID, id string) (*Workflow, error)   // B7.1
    DeleteDraft(ctx context.Context, workspaceID, id string) error              // B7.5
}
```

**Decision de diseño** — DSL no se valida en save, solo en verify (B2):
> B1.6 establece que el draft permite DSL con errores de sintaxis. La validacion ocurre en B2 (Judge). Separar save de validate permite edicion incremental sin friction.

---

### 2.2 Judge

**Satisface**: B2 (todos los sub-casos).

**Responsabilidades**:
- Ejecutar solo validacion sintactica cuando no hay spec_source → B2.2
- Reportar TODAS las violaciones en un solo pass (no fail-fast) → B2.1
- Distinguir violaciones (bloqueantes) de warnings (no bloqueantes) → B2.4
- Cada verificacion es independiente, sin cache → B2.5
- Re-verificar en activacion como safety net → B2.6
- Rechazar verificacion de workflow no-draft → B2.7
- Parsear spec con bloques faltantes sin abortar → B2.8

**Interfaz**:

```go
type Judge interface {
    Verify(ctx context.Context, workflow *Workflow) (*JudgeResult, error)
}

type JudgeResult struct {
    Passed     bool
    Violations []Violation
    Warnings   []Warning
}

type Violation struct {
    CheckID     int    // 1-8 segun AGENT_SPEC judge checks
    Type        string // behavior_no_coverage, constraint_contradiction, actor_undefined, given_unreachable, dsl_mismatch
    Description string
    Location    string // "DSL linea 12" o "BEHAVIOR detect_intent"
}

type Warning struct {
    CheckID     int
    Description string
}
```

**Checks implementados por fase**:

| Check | Descripcion | Fase |
|---|---|---|
| 1 | THEN clause observable sin conocer implementacion | 2 |
| 2 | BEHAVIOR no contradice CONSTRAINT | 2 |
| 3 | ACTORS en BEHAVIOR estan definidos en ACTORS | 2 |
| 4 | GIVEN states son producidos por otro BEHAVIOR o son eventos externos | 2 |
| 5 | DSL BLOCK implementa todos los BEHAVIOR | 2 |
| 6 | DSL usa solo conceptos BPMN-grounded | 3 |
| 7 | Protocol responses (ACCEPTED/REJECTED/DELEGATED) estan cubiertos | 3 |
| 8 | Terminos que pueden interpretarse de mas de una forma | 3 |

**Decision de diseño** — Judge re-verifica en activacion (B2.6):
> El gap entre verify y activate puede incluir ediciones del DSL. Re-verificar en activate garantiza que lo que se activa es lo que se verifico. Costo: latencia extra en activate. Beneficio: garantia de consistencia.

---

### 2.3 AgentRunner + RunnerRegistry + RunContext

**Satisface**: B3 (contrato de ejecucion para todos los runners), B8 (sub-agente interno).

**Responsabilidades**:
- Contrato unico para Go agents, SkillRunner y DSLRunner → B3 principal
- RunContext pasado por metodo (no constructor) para soportar runners creados dinamicamente → B3.11
- RunnerRegistry hace lookup por agent_type → B3 principal

**Interfaces**:

```go
// RunContext lleva todas las dependencias que cualquier runner necesita.
// Forward-compatible: el DSL Runtime usa el mismo contexto.
type RunContext struct {
    Orchestrator  *Orchestrator
    ToolRegistry  *tool.ToolRegistry
    PolicyEngine  *policy.Evaluator
    EventBus      eventbus.EventBus
    AuditLogger   audit.Service
    SignalService signal.Service    // para SURFACE verb → B4
    Scheduler     scheduler.Service // para WAIT verb → B5
    RunnerRegistry *RunnerRegistry  // para AGENT verb → B3.5
    DB            *sql.DB
    // Contexto de ejecucion para detectar loops
    CallDepth     int               // limite: 5 → B3.12
    CallChain     []string          // agentes en la cadena → B8.6
}

// AgentRunner es el contrato de ejecucion para cualquier tipo de agente.
type AgentRunner interface {
    Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error)
}

type RunnerRegistry struct {
    runners map[string]AgentRunner
}

func (r *RunnerRegistry) Register(agentType string, runner AgentRunner)
func (r *RunnerRegistry) Get(agentType string) (AgentRunner, bool)
```

**Decision de diseño** — RunContext por metodo, no por constructor:
> Los DSLRunners se crean dinamicamente (uno por workflow, bajo demanda). Si el contexto se inyectara en el constructor del runner, cada workflow necesitaria una instancia diferente pre-configurada al startup. Pasarlo por metodo permite crear runners dinamicamente y compartir la misma instancia para multiples ejecuciones.

---

### 2.4 DSLParser

**Satisface**: B2 (genera AST para verificacion), B3 (genera AST para ejecucion).

**Responsabilidades**:
- Tokenizar DSL con soporte de indentacion significativa (INDENT/DEDENT) → B2.3
- Producir AST completo si no hay errores de sintaxis
- Reportar errores con linea y columna → B2.3
- Rechazar verbos no permitidos → B2.9
- Validar duraciones negativas en WAIT → B5.3
- Ser puro: sin side effects, sin imports de domain packages

**Tokens**:

```
WORKFLOW  ON      IF      SET     AGENT
NOTIFY    SURFACE WAIT    DISPATCH
IDENT     STRING  NUMBER  NEWLINE INDENT  DEDENT  EOF
EQ(==)  NEQ(!=)  GT(>)   LT(<)   GTE(>=) LTE(<=) IN  AND  OR  ASSIGN(=)
```

**Nodos AST**:

```go
type WorkflowNode struct {
    Name       string
    Statements []Statement
}

type Statement interface{ statementNode() }

type OnNode       struct{ Event DottedIdent }
type IfNode       struct{ Condition Expression; Body []Statement }
type SetNode      struct{ Target DottedIdent; Value Expression }
type AgentNode    struct{ FuncName string; Args []Expression }
type NotifyNode   struct{ Actor string; Data Expression }
type SurfaceNode  struct{ Entity string; View DottedIdent; Reason string }
type WaitNode     struct{ Duration int; Unit string } // unit: hours|minutes|days|seconds
type DispatchNode struct{ AgentName string; WorkflowName string }

type Expression interface{ expressionNode() }
type BinaryExpr  struct{ Left Expression; Op string; Right Expression }
type DottedIdent struct{ Parts []string } // e.g. contact.intent_signal
type StringLit   struct{ Value string }
type NumberLit   struct{ Value float64 }
type ArrayLit    struct{ Elements []Expression }
```

---

### 2.5 DSLRuntime + ExpressionEvaluator + VerbMapper

**Satisface**: B3 (todos los sub-casos de ejecucion).

**Responsabilidades**:

**DSLRuntime**:
- Interpretar AST statement por statement → B3 principal
- Omitir body de IF cuando condicion es false → B3.1
- Fallar en tool error (no retry automatico) → B3.2
- Propagar fallo de sub-agente → B3.5
- Ejecutar ejecuciones de workflows multiples independientemente → B3.6
- Tratar abstained del sub-agente como informacion, no error → B3.8
- Detectar y abortar loops circulares de AGENT calls → B3.12
- Tratar campo inexistente como null (no abortar) → B3.14

**ExpressionEvaluator** — operadores soportados (sin coercion de tipos):
- Comparacion: `==`, `!=`, `>`, `<`, `>=`, `<=` → B3 principal
- Logicos: `AND`, `OR`
- Membership: `IN`
- Campo inexistente → null → B3.14
- Tipos incompatibles → error → B3.13

**VerbMapper** — tabla estatica DSL verb → tool:

```
SET case.status     → update_case(status)
SET case.priority   → update_case(priority)
SET lead.status     → update_lead(status)
SET deal.stage      → update_deal(stage)
NOTIFY contact      → send_reply(contact_id, content)
NOTIFY salesperson  → create_task(owner_id, title)
SURFACE entity      → signal.Service.Create(...)       → B4
WAIT duration       → scheduler.Service.Schedule(...)  → B5
DISPATCH            → protocol_handler.Dispatch(...)   → B8.9 (stub en Fase 2)
```

**Decision de diseño** — campo inexistente = null (B3.14):
> En un sistema con entidades dinamicas, las condiciones pueden referenciar campos que aun no existen en una instancia concreta. Abortar seria demasiado estricto. Tratar ausencia como null permite workflows mas robustos (e.g., `IF deal.close_date IS NULL`).

**Decision de diseño** — abstained no es error fatal (B3.8):
> Un sub-agente que abstiene esta comunicando informacion valida ("no encontre evidencia suficiente"). El workflow padre puede tener ramas para ese caso (`IF evidence.top_score < 0.55`). Propagarlo como error destruiria esa logica.

---

### 2.6 DSLRunner

**Satisface**: B3 (runner para agent_type="dsl"), B7 (usa version activa del workflow).

**Responsabilidades**:
- Cargar workflow activo por agent_definition_id → B7 principal
- Parsear DSL → AST (con cache en memoria, invalidado en Activate()) → B3.11
- Ejecutar via DSLRuntime con RunContext
- Registrar cada statement ejecutado como agent_run_step
- Permitir que ejecuciones en curso completen aunque el workflow se archive → B3.11

**Interfaz**: Implementa `AgentRunner`.

```go
type DSLRunner struct {
    workflowService workflow.Service
    astCache        map[string]*dsl.WorkflowNode // key: workflow.id
    cacheMu         sync.RWMutex
}

func (r *DSLRunner) Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error)
func (r *DSLRunner) InvalidateCache(workflowID string) // llamado en Workflow.Activate()
```

---

### 2.7 SignalService

**Satisface**: B4 (todos los sub-casos).

**Responsabilidades**:
- Crear signal con evidencia → B4 principal
- Rechazar signal sin evidence → B4.1 (constraint)
- Permitir multiples signals del mismo tipo/entidad → B4.2
- Publicar evento `signal.created` en EventBus → B4 principal
- Registrar quien y cuando descarto un signal → B4.4
- Publicar evento `signal.dismissed` → B4.4
- Rechazar signal con entidad inexistente → B4.5
- Validar confianza en [0.0, 1.0] → B4.6
- No validar existencia de cada evidence_id en creacion → B4.7

**Interfaz**:

```go
type SignalService interface {
    Create(ctx context.Context, input CreateSignalInput) (*Signal, error)
    List(ctx context.Context, workspaceID string, filters SignalFilters) ([]*Signal, error)
    GetByEntity(ctx context.Context, workspaceID, entityType, entityID string) ([]*Signal, error)
    Dismiss(ctx context.Context, workspaceID, signalID, actorID string) error
}

type CreateSignalInput struct {
    WorkspaceID string
    EntityType  string   // contact, lead, deal, case
    EntityID    string
    SignalType  string   // intent_high, churn_risk, upsell_opportunity, ...
    Confidence  float64  // [0.0, 1.0]
    EvidenceIDs []string // al menos 1 requerido
    SourceType  string   // agent_run, workflow, manual
    SourceID    string
    Metadata    map[string]any
    ExpiresAt   *time.Time
}
```

**Decision de diseño** — no deduplicar signals (B4.2):
> Multiples evaluaciones del mismo tipo/entidad con diferente confianza o evidencia son informacion valiosa. La deduplicacion perderia el rastro de como evoluciona la confianza en el tiempo. La UI puede mostrar el mas reciente destacado.

---

### 2.8 Scheduler

**Satisface**: B5 (todos los sub-casos).

**Responsabilidades**:
- Persistir jobs en DB (recovery ante restart) → B5.1
- Procesar jobs pendientes en ciclos de polling (cada 10s) → B5 principal
- Cancelar jobs cuando el workflow se archiva → B5.2
- WAIT 0 = yield (resume en el proximo ciclo) → B5.3
- Limite de concurrencia: 10 resumes por ciclo → B5.7
- Marcar job como executed aunque el resume falle (no reintentar) → B5.8

**Interfaz**:

```go
type Scheduler interface {
    Schedule(ctx context.Context, job ScheduleJobInput) (*ScheduledJob, error)
    Cancel(ctx context.Context, workspaceID, jobID string) error
    CancelBySource(ctx context.Context, workspaceID, sourceID string) error // B5.2: cancela todos los jobs del workflow
}

type ScheduleJobInput struct {
    WorkspaceID string
    JobType     string    // workflow_resume
    Payload     any       // {workflow_id, run_id, resume_step_index}
    ExecuteAt   time.Time
}

// Callback registrado en startup para manejar cada tipo de job
type JobHandler func(ctx context.Context, job *ScheduledJob) error
```

**Decision de diseño** — no reintentar resume fallido (B5.8):
> El resume puede haber ejecutado steps parcialmente. Reintentar puede duplicar side effects (e.g., enviar el mismo email dos veces). Mejor loguear y dejar al Admin re-trigger manualmente con control total.

---

### 2.9 ApprovalService (extendido para B6)

**Satisface**: B6 (todos los sub-casos).

**Responsabilidades** (extension del servicio existente):
- Crear approval_request antes de tool call sensible → B3.4, B6
- Manejar aprobacion, rechazo y modificacion → B6 principal, B6.1
- Timeout = rechazo implicito (no aprobacion silenciosa) → B6.3
- Primer respondente es definitivo → B6.5
- Registrar override como feedback → B6.2
- Aceptar override sin razon (razon opcional) → B6.7

**Tipos de override**:

```go
const (
    OverrideTypeRejected           = "rejected"             // B6 principal
    OverrideTypeModified           = "modified"             // B6.1
    OverrideTypePostExecution      = "post_execution_feedback" // B6.2
    OverrideTypeDelegationOverride = "delegation_override"  // B6.4
)
```

**Decision de diseño** — no compensacion automatica post-ejecucion (B6.2):
> Revertir automaticamente (e.g., "desenviar un email") es imposible en muchos casos. Para los casos donde si es posible (e.g., cambiar un campo), el riesgo de compensar incorrectamente supera el beneficio. El override post-ejecucion se registra como feedback para mejorar el agente, no como rollback.

---

### 2.10 ProtocolHandler (Fase 3)

**Satisface**: B8 (todos los sub-casos).

**Responsabilidades**:
- Serializar DSL + headers de protocolo → B8 principal
- Enviar via HTTP (adaptable a MCP/A2A) → B8 principal
- Timeout configurable → B8.3
- Incluir cadena de delegacion en headers para deteccion de loops → B8.6
- Stub en Fase 2: retorna REJECTED("not_implemented") → B8.9
- Dispatch interno: invoca RunnerRegistry directamente sin HTTP → B8.5

**Interfaz**:

```go
type ProtocolHandler interface {
    Dispatch(ctx context.Context, input DispatchInput) (*DispatchResponse, error)
}

type DispatchInput struct {
    TargetAgent   string
    WorkflowName  string
    DSLSource     string
    CallChain     []string // para deteccion de loops circulares → B8.6
    TimeoutSec    int
}

type DispatchResponse struct {
    Status  string // ACCEPTED, REJECTED, DELEGATED
    Reason  string // requerido si REJECTED → B8.1
    Target  string // requerido si DELEGATED → B8.2
}
```

**Decision de diseño** — no seguir cadena DELEGATED automaticamente (B8.2):
> Seguir automaticamente puede crear loops o ejecutar en agentes no autorizados. El control humano sobre la cadena de delegacion es explicito: el Admin decide si re-dispatch al agente sugerido.

---

### 2.10.1 Direccion de Interoperabilidad

- `DISPATCH` externo debe ser A2A-first.
- HTTP queda como transporte del estandar, no como contrato propietario.
- La frontera externa para tools, resources y contexto debe ser MCP-first.
- `ProtocolHandler` se mantiene como puerto interno, pero sus adapters externos deben alinearse con A2A y MCP.

## 3. Modelo de Datos

### Diagrama ERD

```mermaid
erDiagram
    workspace ||--o{ workflow : "tiene"
    workspace ||--o{ signal : "tiene"
    agent_definition ||--o{ workflow : "referenciado por"
    agent_definition ||--o{ agent_run : "ejecutado por"
    workflow ||--o{ agent_run : "ejecutado como"
    workflow ||--o| workflow : "parent_version_id"
    agent_run ||--o{ agent_run_step : "contiene"
    agent_run ||--o{ approval_request : "genera"
    signal }o--|| agent_run : "generado por"
    scheduled_job }o--|| agent_run : "resume de"

    workflow {
        TEXT id PK
        TEXT workspace_id FK
        TEXT name
        TEXT description
        TEXT dsl_source
        TEXT spec_source
        INTEGER version
        TEXT status "draft|testing|active|archived"
        TEXT parent_version_id FK
        TEXT agent_definition_id FK
        TEXT created_by FK
        DATETIME created_at
        DATETIME updated_at
    }

    signal {
        TEXT id PK
        TEXT workspace_id FK
        TEXT entity_type "contact|lead|deal|case"
        TEXT entity_id
        TEXT signal_type
        REAL confidence
        TEXT evidence_ids "JSON array"
        TEXT source_type "agent_run|workflow|manual"
        TEXT source_id
        TEXT metadata "JSON"
        TEXT status "active|expired|dismissed"
        TEXT dismissed_by FK
        DATETIME dismissed_at
        DATETIME created_at
        DATETIME expires_at
    }

    scheduled_job {
        TEXT id PK
        TEXT workspace_id
        TEXT job_type "workflow_resume"
        TEXT payload "JSON"
        DATETIME execute_at
        TEXT status "pending|executed|cancelled"
        TEXT source_id "agent_run_id"
        DATETIME created_at
        DATETIME executed_at
    }

    agent_run {
        TEXT id PK
        TEXT agent_definition_id FK
        TEXT workflow_id FK
        TEXT status "running|accepted|rejected|delegated|success|partial|failed|abstained|escalated"
        TEXT trigger_type "event|schedule|manual|copilot"
        TEXT triggered_by FK
        TEXT inputs "JSON"
        TEXT output "JSON"
        TEXT tool_calls "JSON"
        TEXT reasoning_trace "JSON"
        INTEGER total_tokens
        REAL total_cost
        INTEGER latency_ms
        DATETIME created_at
        DATETIME updated_at
    }
```

### Entidad Workflow — Campos y Reglas

| Campo | Tipo | Regla | Caso de uso |
|---|---|---|---|
| `id` | TEXT UUID | PK inmutable | — |
| `name` | TEXT | NOT NULL | B1.3 |
| `dsl_source` | TEXT | NOT NULL, max 64KB | B1.3, B1.5, B1.7 |
| `spec_source` | TEXT | NULL permitido | B1.1 |
| `version` | INTEGER | DEFAULT 1, NOT NULL | B7 |
| `status` | TEXT | draft\|testing\|active\|archived | B1, B2, B7 |
| `parent_version_id` | TEXT | NULL para v1, FK a self para v2+ | B7 |
| `agent_definition_id` | TEXT | NULL permitido (workflow standalone) | B3 |
| `updated_at` | DATETIME | Updated en cada UPDATE | B1.8 |
| UNIQUE | (workspace_id, name, version) | Enforcement en DB | B1.2 |

### Entidad AgentRun — Estados extendidos

Nuevos estados para AGENT_SPEC (adicionales a los existentes):

| Estado | Descripcion | Terminal | Caso de uso |
|---|---|---|---|
| `running` | En ejecucion | No | B3 |
| `accepted` | DSL validado, ejecutando | No | B3 (nuevo) |
| `rejected` | Judge/policy denego la ejecucion | Si | B2.1, B8.1 (nuevo) |
| `delegated` | DISPATCH a agente externo | Si | B8 (nuevo) |
| `success` | Completado exitosamente | Si | B3 |
| `partial` | Algunos pasos completados | Si | B3 |
| `failed` | Error en ejecucion | Si | B3 |
| `abstained` | Evidencia insuficiente | Si | B3 |
| `escalated` | Handoff a humano (Go agents) | Si | B6 |

---

## 4. Maquinas de Estado

### 4.1 Workflow Lifecycle

```mermaid
stateDiagram-v2
    [*] --> draft : B1 — Create(name, dsl_source)

    draft --> draft : B1 — Update(dsl_source)\nLast-write-wins
    draft --> testing : B2 — Verify() → passed=true
    draft --> draft : B2 — Verify() → violations\nAdmin corrige DSL

    testing --> active : B2.6 — Activate() → re-verify + swap\narchiva version activa anterior
    testing --> draft : B2.6 — Activate() → re-verify fallida

    active --> archived : B7 — NewVersion() crea draft\nautomatico al activar nueva version
    active --> active : B3 — Execute() crea agent_runs

    archived --> active : B7.1 — Rollback()\narchiva version activa actual
    archived --> [*]: permanente

    note right of draft
        Unico estado editable.
        DSL puede tener sintaxis invalida.
        No ejecutable.
    end note

    note right of testing
        DSL sintacticamente valido.
        Verificacion de consistencia pasada.
        No ejecutable aun.
    end note

    note right of active
        UNIQUE(workspace_id, name) activo.
        Ejecutable. DSL inmutable.
        Ejecuciones en curso completan
        aunque se archive (B3.11).
    end note

    note right of archived
        No ejecutable.
        Ejecutable via rollback.
        Draft purgable (B7.5).
    end note
```

### 4.2 AgentRun States

```mermaid
stateDiagram-v2
    [*] --> running : Orchestrator.TriggerAgent()

    running --> accepted : DSLRunner: DSL parseado\ny validado, comenzando ejecucion
    running --> rejected : Judge/policy\ndengo la ejecucion
    running --> failed : Error antes de\nparsear DSL
    running --> escalated : Go agent\nhandoff a humano

    accepted --> success : Todos los steps OK
    accepted --> partial : Algunos steps OK\n(run parcial intencional)
    accepted --> failed : Error en step\n(tool fail, type mismatch, quota)
    accepted --> abstained : Sub-agente abstained\ny workflow no tiene rama
    accepted --> delegated : DISPATCH aceptado\npor agente externo

    success --> [*]
    partial --> [*]
    failed --> [*]
    abstained --> [*]
    escalated --> [*]
    rejected --> [*]
    delegated --> [*]

    note left of accepted
        NUEVO. DSL parseado exitosamente.
        Ejecucion en progreso.
        No es terminal — puede fallar despues.
    end note

    note right of rejected
        NUEVO. Debe incluir razon.
        No es lo mismo que failed
        (failed = error tecnico,
        rejected = decision deliberada).
    end note

    note right of delegated
        NUEVO. DISPATCH completado.
        El receptor tiene el control.
        No implica success ni failure.
    end note
```

### 4.3 ApprovalRequest States (B6)

```mermaid
stateDiagram-v2
    [*] --> pending : B3.4 — tool sensible\ndetectado en ejecucion

    pending --> approved : B6 — aprobador acepta\n(ejecuta tool con params originales)
    pending --> modified : B6.1 — aprobador modifica\n(ejecuta tool con params nuevos)
    pending --> rejected_ap : B6 — aprobador rechaza\n(tool no se ejecuta)
    pending --> expired : B6.3 — timeout sin respuesta\n(= rechazo implicito)

    approved --> [*]
    modified --> [*]
    rejected_ap --> [*]
    expired --> [*]

    note right of pending
        Ejecucion del workflow pausa.
        AgentRun permanece en accepted.
    end note

    note right of expired
        Silencio != aprobacion.
        CONSTRAINT aplicado.
        AgentRun → failed.
    end note
```

---

## 5. Diagramas de Secuencia

### 5.1 Verificacion y Activacion (B2)

```mermaid
sequenceDiagram
    participant Admin
    participant API
    participant WS as WorkflowService
    participant J as Judge
    participant SP as SpecParser
    participant DP as DSLParser
    participant DB

    Admin->>API: POST /workflows/{id}/verify
    API->>WS: Get(id)
    WS->>DB: SELECT workflow WHERE id
    DB-->>WS: workflow (status=draft)
    WS-->>API: workflow

    API->>J: Verify(workflow)

    alt spec_source presente (B2 principal)
        J->>SP: Parse(spec_source)
        SP-->>J: ParsedSpec {actors, behaviors, constraints}
        J->>DP: Parse(dsl_source)
        DP-->>J: AST o SyntaxError
        J->>J: RunChecks(1-5): violations[], warnings[]
    else sin spec_source (B2.2)
        J->>DP: Parse(dsl_source)
        DP-->>J: AST o SyntaxError
        J->>J: SyntaxCheckOnly + add warnings["no spec provided"]
    end

    J-->>API: JudgeResult{passed, violations[], warnings[]}

    alt passed=true
        API->>WS: UpdateStatus(id, testing)
        WS->>DB: UPDATE workflow SET status=testing
    end

    API-->>Admin: JudgeResult

    Note over Admin,DB: ... Admin revisa resultado y solicita activacion ...

    Admin->>API: PUT /workflows/{id}/activate
    API->>J: Verify(workflow) -- re-verify safety net (B2.6)

    alt re-verify pasa
        API->>WS: Activate(id)
        WS->>DB: BEGIN TRANSACTION
        WS->>DB: UPDATE existing active → archived (mismo workspace+name)
        WS->>DB: UPDATE workflow SET status=active
        WS->>DB: COMMIT
    else re-verify falla
        API-->>Admin: 422 {violations}
        WS->>DB: UPDATE workflow SET status=draft
    end

    API-->>Admin: 200 workflow activo
```

### 5.2 Ejecucion con Tool Call y Approval (B3 + B6)

```mermaid
sequenceDiagram
    participant EVT as EventBus
    participant ORC as Orchestrator
    participant RR as RunnerRegistry
    participant DR as DSLRunner
    participant RT as DSLRuntime
    participant EE as ExpressionEvaluator
    participant VM as VerbMapper
    participant TR as ToolRegistry
    participant PE as PolicyEngine
    participant AS as ApprovalService
    participant DB

    EVT->>ORC: Event{topic: "case.created", payload}
    ORC->>DB: INSERT agent_run (status=running)
    ORC->>RR: Get("dsl")
    RR-->>ORC: DSLRunner

    ORC->>DR: Run(ctx, RunContext, input)
    DR->>DB: SELECT workflow WHERE agent_id AND status=active
    DB-->>DR: workflow.dsl_source

    DR->>DR: Parse(dsl_source) → AST (cache hit si disponible)
    DR->>DB: UPDATE agent_run (status=accepted)

    loop Cada statement en AST
        DR->>RT: Interpret(statement, executionCtx)

        alt IF statement (B3.1)
            RT->>EE: Evaluate(condition, ctx)
            EE-->>RT: true | false
            Note right of RT: false → skip body, log step=skipped
        end

        alt SET statement
            RT->>VM: Resolve(target_field)
            VM-->>RT: tool_name, param_mapping
            RT->>TR: Execute(tool_name, params)
            TR->>PE: CheckPolicy(before_tool)

            alt Policy bloquea (B3.3)
                PE-->>TR: denied + reason
                TR-->>RT: PolicyError
                RT->>DB: UPDATE agent_run (status=rejected)
            else Requiere approval (B3.4)
                PE-->>TR: requires_approval
                TR->>AS: CreateApprovalRequest(action, params)
                AS-->>TR: approval_request_id
                Note over DR,AS: Ejecucion pausa. AgentRun permanece en accepted.
                AS->>DB: INSERT approval_request (status=pending)
                Note over DR,AS: ... Salesperson responde ...
                AS->>DB: UPDATE approval_request (status=approved|rejected|modified)
                AS-->>TR: ApprovalResult
                alt Aprobado (B6 principal)
                    TR->>DB: Execute tool
                else Rechazado (B6 override)
                    TR-->>RT: OverrideError
                    RT->>DB: UPDATE agent_run (status=failed, override_recorded=true)
                end
            else Permitido
                PE-->>TR: allowed
                TR->>DB: Execute tool (UPDATE/INSERT)
                TR-->>RT: ToolResult
            end
        end

        DR->>DB: INSERT agent_run_step (step_index, type, status, input, output)
    end

    DR->>DB: UPDATE agent_run (status=success, output, tool_calls)
    DR-->>ORC: *Run
```

### 5.3 Accion Diferida: WAIT y Resume (B5)

```mermaid
sequenceDiagram
    participant RT as DSLRuntime
    participant SCH as Scheduler
    participant DB
    participant POLL as Scheduler Goroutine
    participant RES as ResumeHandler

    RT->>RT: Procesa statement: WAIT 48 hours

    RT->>DB: INSERT agent_run_step (type=wait, execute_at=now+48h)
    RT->>SCH: Schedule({job_type: "workflow_resume", run_id, step_index+1, execute_at})
    SCH->>DB: INSERT scheduled_job (status=pending, execute_at=now+48h)
    SCH-->>RT: scheduled_job_id

    RT-->>RT: Suspend — retorna sin ejecutar steps siguientes
    Note over RT,DB: AgentRun permanece en status=accepted (no terminal)

    Note over POLL: ... 48 horas despues (o al reiniciar si el server cayó — B5.1) ...

    POLL->>DB: SELECT scheduled_job WHERE status=pending AND execute_at <= now LIMIT 10
    DB-->>POLL: [scheduled_job]

    loop Para cada job (max concurrencia=10 — B5.7)
        POLL->>DB: UPDATE scheduled_job SET status=executed
        POLL->>RES: Handle(job.payload)

        RES->>DB: SELECT agent_run WHERE id
        DB-->>RES: agent_run (status=accepted)

        alt Workflow archivado durante WAIT (B5.2)
            RES->>DB: SELECT workflow WHERE id AND status=active
            DB-->>RES: null
            RES->>DB: UPDATE agent_run (status=failed, reason="workflow_archived")
        else Workflow activo
            RES->>DB: SELECT entidad fresca (estado actual — B5.4)
            RES->>RT: Resume(agent_run, step_index, fresh_entity_state)
            RT->>RT: Continua desde step_index con estado actual
        end
    end
```

### 5.4 Delegacion entre Agentes (B8)

```mermaid
sequenceDiagram
    participant RT as DSLRuntime
    participant PH as ProtocolHandler
    participant EXT as External Agent
    participant DB

    RT->>RT: Procesa statement: DISPATCH TO product_specialist WITH case_analysis

    alt Fase 2 (stub — B8.9)
        RT-->>RT: REJECTED("dispatch_not_implemented")
        RT->>DB: UPDATE agent_run (status=failed)
    else Fase 3 (implementado)
        RT->>PH: Dispatch({target: "product_specialist", dsl, call_chain: ["support_agent"]})

        alt Loop circular detectado (B8.6)
            PH->>PH: "product_specialist" IN call_chain?
            PH-->>RT: REJECTED("circular_delegation_detected")
            RT->>DB: UPDATE agent_run (status=failed)
        else Dispatch interno (B8.5)
            PH->>PH: target IN RunnerRegistry?
            PH->>PH: runner.Run(dsl, call_chain+["product_specialist"])
            PH-->>RT: ACCEPTED|REJECTED
        else Dispatch externo
            PH->>EXT: HTTP POST /dispatch {dsl, X-Delegation-Chain: "support_agent"}

            alt Timeout (B8.3)
                PH-->>RT: TIMEOUT
                RT->>DB: UPDATE agent_run (status=failed, reason="dispatch_timeout")
            else ACCEPTED (B8 principal)
                EXT-->>PH: 202 ACCEPTED
                PH-->>RT: DispatchResponse{status: ACCEPTED}
                RT->>DB: UPDATE agent_run (status=delegated)
                RT->>DB: INSERT agent_run_step (type=dispatch, result=ACCEPTED)
            else REJECTED (B8.1)
                EXT-->>PH: 409 REJECTED {reason}
                PH-->>RT: DispatchResponse{status: REJECTED, reason}
                RT->>DB: UPDATE agent_run (status=failed, reason=dispatch_rejected)
                RT->>DB: INSERT agent_run_step (type=dispatch, result=REJECTED, reason)
            else DELEGATED (B8.2)
                EXT-->>PH: 307 DELEGATED {target}
                PH-->>RT: DispatchResponse{status: DELEGATED, target}
                RT->>DB: UPDATE agent_run (status=delegated)
                Note over RT,DB: No se sigue la cadena automaticamente.
                Note over RT,DB: Admin decide si re-dispatch a target.
            end
        end
    end
```

---

## 6. API Design

Los endpoints derivan directamente de los behaviors y sus pre-condiciones.

### Workflow API

| Endpoint | Metodo | Behavior | Pre-condicion |
|---|---|---|---|
| `/api/v1/workflows` | POST | B1 | Admin autenticado |
| `/api/v1/workflows` | GET | B1 | — |
| `/api/v1/workflows/{id}` | GET | B1 | — |
| `/api/v1/workflows/{id}` | PUT | B1 | status=draft |
| `/api/v1/workflows/{id}` | DELETE | B7.5 | status=draft, nunca activado |
| `/api/v1/workflows/{id}/verify` | POST | B2 | status=draft |
| `/api/v1/workflows/{id}/activate` | PUT | B2.6 | status=testing |
| `/api/v1/workflows/{id}/new-version` | POST | B7 | status=active |
| `/api/v1/workflows/{id}/rollback` | PUT | B7.1 | tiene parent_version_id archived |
| `/api/v1/workflows/{id}/execute` | POST | B3.9 | status=active |

### Signal API

| Endpoint | Metodo | Behavior | Pre-condicion |
|---|---|---|---|
| `/api/v1/signals` | GET | B4 | — |
| `/api/v1/signals/{id}/dismiss` | PUT | B4.4 | status=active |
| `/api/v1/signals?entity_type=X&entity_id=Y` | GET | B4 | — |

### Codigos de Respuesta

| Situacion | HTTP Code | Behavior de origen |
|---|---|---|
| Creado exitosamente | 201 | B1 principal |
| Verificacion OK | 200 + JudgeResult | B2 principal |
| Verificacion con violations | 200 + violations (no es 422) | B2.1 — el cliente decide como mostrar |
| Verificacion de workflow no-draft | 409 Conflict | B2.7 |
| Nombre duplicado | 409 Conflict | B1.2 |
| Campos faltantes | 422 Unprocessable | B1.3 |
| DSL vacio | 422 Unprocessable | B1.5 |
| Tamano excedido | 413 Payload Too Large | B1.7 |
| Edicion de no-draft | 409 Conflict | B1.4 |
| Dispatch aceptado | 202 Accepted | B8 principal |
| Dispatch rechazado | 409 Conflict + reason | B8.1 |
| Dispatch delegado | 307 Temporary Redirect + target | B8.2 |

**Decision de diseño** — verificacion devuelve 200 con violations (no 422):
> La verificacion con violations no es un error del cliente — el cliente envio una request valida. El Judge hizo su trabajo y retorno un resultado. Usar 422 confundiria la respuesta tecnica con el resultado de negocio. El cuerpo del 200 contiene `passed: false` + `violations[]`.

---

## 7. Decisiones de Diseño (Resumen)

| Decision | Alternativa considerada | Razon de eleccion | Caso de uso |
|---|---|---|---|
| RunContext por metodo, no constructor | Inyeccion en constructor | DSLRunners creados dinamicamente; constructor requeriria pre-instanciar uno por workflow | B3 (DSLRunner dinamico) |
| DSL no se valida en save, solo en verify | Validar sintaxis en cada save | Draft es espacio de trabajo; forzar sintaxis frustra edicion incremental | B1.6 |
| No reintentar dispatch fallido | Retry con backoff | Side effects en agentes externos no son idempotentes; control humano explicitamente | B8.3, B8.8 |
| No seguir cadena DELEGATED automaticamente | Seguir automaticamente | Riesgo de loops y ejecuciones no autorizadas en agentes intermedios | B8.2, B8.6 |
| Abstained del sub-agente no es error fatal | Propagar como error | Abstained es informacion valida que el workflow padre puede manejar con IF | B3.8 |
| Last-write-wins en ediciones concurrentes | Optimistic locking con etag | Complejidad no justificada para MVP; drafts raramente editados concurrentemente | B1.8 |
| No compensacion automatica post-override | Compensacion automatica | Imposible en muchos casos (emails); riesgosa en los casos posibles | B6.2 |
| Signal sin deduplicacion | Upsert de signal existente | Multiples evaluaciones son trazabilidad valiosa; UI puede filtrar por timestamp | B4.2 |
| Scheduler no reintenta resume fallido | Retry con backoff | Resume puede tener side effects parciales; duplicacion es peor que fallo | B5.8 |
| Re-verificacion en Activate() | Solo verificar en Verify() | Gap entre verify y activate puede incluir ediciones del DSL | B2.6 |
| Verificacion devuelve 200 con violations | 422 cuando hay violations | Verificacion es un resultado de negocio, no un error HTTP del cliente | B2.1 |
| Campo inexistente en expression = null | Abortar con error | Entidades dinamicas; workflows mas robustos ante campos opcionales | B3.14 |
| Limite de concurrencia en Scheduler (10) | Sin limite | Previene saturacion del sistema en picos de scheduled_jobs | B5.7 |

---

## 8. Restricciones del Diseño (derivadas de Constraints del spec)

Cada constraint del documento de casos de uso tiene un componente de diseño responsable de hacerlo cumplir.

| Constraint | Componente responsable | Punto de enforcement |
|---|---|---|
| Workflow no ejecuta sin verificacion del Judge | `WorkflowService.Activate()` | Activa solo si Judge pasa |
| Mutacion solo via herramientas registradas | `DSLRuntime` + `VerbMapper` | SET → VerbMapper → ToolRegistry.Execute() |
| Agente sin permisos no ejecuta herramienta | `PolicyEngine` | Enforcement en ToolRegistry pipeline |
| Accion sensible requiere aprobacion | `PolicyEngine` + `ApprovalService` | Detectado en enforcement point before_tool |
| Signal requiere evidencia | `SignalService.Create()` | Validacion en service layer antes de INSERT |
| Override no se descarta silenciosamente | `ApprovalService` + `AgentRun` | Registro de override como evento en agent_run |
| Workflow archivado no recibe ejecuciones | `Orchestrator.TriggerAgent()` | Valida status=active antes de crear agent_run |
| REJECTED incluye razon | `ProtocolHandler` + `Judge` | Enforced en DispatchResponse y JudgeResult structs |
| Agentes Go siguen funcionando | `RunnerRegistry` | Go agents registrados como runners validos permanentemente |
