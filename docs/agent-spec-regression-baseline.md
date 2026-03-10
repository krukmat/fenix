# AGENT_SPEC Regression Baseline

> Fecha: 2026-03-09
> Objetivo: resolver F0.3 identificando la red de seguridad minima para la transicion AGENT_SPEC

---

## Alcance

Este documento identifica las pruebas actuales que deben considerarse baseline de no regresion
antes y durante la transicion AGENT_SPEC, con foco en:

- orquestacion
- tools
- policy
- approval
- audit

No intenta listar toda la suite del proyecto. Solo el subconjunto que protege el nucleo que va a
ser tocado por `AgentRunner`, `RunnerRegistry`, `Workflow`, `Judge` y `DSLRunner`.

---

## Conclusion operativa

La transicion AGENT_SPEC debe considerar como red de seguridad minima estas 5 areas:

1. Estado y ciclo de vida de `agent_run`
2. Ejecucion de agentes actuales via orquestador
3. Registro, validacion, permisos y auditoria de tools
4. Evaluacion de policy y approvals
5. Persistencia y exposicion del audit trail

Si cualquiera de estas areas se rompe, la Fase 1 deja de ser segura.

---

## Suites criticas

### 1. Orquestacion

#### Suite principal

- `internal/domain/agent/orchestrator_test.go`

#### Cobertura relevante

- creacion de runs validos
- rechazo por agente inexistente o inactivo
- validacion de trigger type
- listado y consulta de runs
- transiciones de estado
- creacion del paso inicial
- recuperacion de runs en curso
- reglas de terminalidad

#### Tests mas importantes

- `TestTriggerAgent_Success`
- `TestTriggerAgent_AgentNotFound`
- `TestTriggerAgent_AgentNotActive`
- `TestTriggerAgent_InvalidTriggerType`
- `TestUpdateAgentRunStatus_Success`
- `TestTriggerAgent_CreatesInitialPendingStep`
- `TestUpdateAgentRunStatus_InvalidTerminalTransition`
- `TestUpdateAgentRun_SynthesizesStepsForCompletedRun`
- `TestRecoverRun_RetryableRunningStepCreatesRetryAttempt`
- `TestRecoverRun_NonRetryableRunningStepFailsRun`

#### Riesgo que cubre

Protege el refactor de `orchestrator.go` cuando se introduzcan `AgentRunner`,
`RunContext` y `RunnerRegistry`.

---

### 2. Agentes actuales como baseline de comportamiento

#### Suites principales

- `internal/domain/agent/agents/support_test.go`
- `internal/domain/agent/agents/prospecting_test.go`
- `internal/domain/agent/agents/kb_test.go`
- `internal/domain/agent/agents/insights_test.go`

#### Cobertura relevante

- herramientas permitidas por agente
- comportamiento funcional principal de cada agente
- manejo de errores de entrada
- abstention, escalation o side effects esperados
- approvals en casos sensibles

#### Tests mas importantes

- Support
  - `TestSupportAgent_Run_EscalatesWhenNoKnowledge`
  - `TestSupportAgent_Run_ResolvesWhenHighConfidence`
  - `TestSupportAgent_Run_AbstainsWhenConfidenceIsMedium`
- Prospecting
  - `TestProspectingAgent_Run_HighConfidence_DraftsOutreach`
  - `TestProspectingAgent_Run_LowConfidence_Skips`
  - `TestProspectingAgent_Run_HighSensitivity_CreatesApprovalAndBlocks`
- KB
  - `TestKBAgent_Run_ResolvedCase_CreatesArticle`
  - `TestKBAgent_Run_DuplicateFound_Updates`
  - `TestKBAgent_Run_HighSensitivity_CreatesApprovalAndBlocksMutation`
- Insights
  - `TestInsightsAgent_Run_SalesFunnelQuery`
  - `TestInsightsAgent_Run_CaseBacklogQuery`
  - `TestInsightsAgent_Run_EmptyData_Abstains`

#### Riesgo que cubre

Protege contra regresiones de comportamiento cuando los agentes Go se adapten al nuevo
contrato runtime en Fase 1.

---

### 3. Tools

#### Suites principales

- `internal/domain/tool/registry_test.go`
- `internal/domain/tool/builtin_executors_test.go`
- `internal/api/handlers/tool_test.go`

#### Cobertura relevante

- registro y lookup de tools
- validacion de schema e inputs
- lifecycle create/update/activate/deactivate/delete
- enforcement de permisos
- contrato de errores
- auditoria de ejecucion y denegacion
- ejecutores built-in con side effects reales
- handlers HTTP de administracion de tools

#### Tests mas importantes

- Registry
  - `TestToolRegistry_RegisterAndGet`
  - `TestToolRegistry_ValidateParams_InvalidJSON_ReturnsError`
  - `TestToolRegistry_UpdateActivateDeactivateDeleteLifecycle`
  - `TestToolRegistry_Execute_EnforcesActiveValidationAndPermissions`
  - `TestToolRegistry_Execute_BuiltinAuditAndErrorContract`
  - `TestToolRegistry_Execute_MissingUserContext`
- Built-in executors
  - `TestCreateTaskExecutor_Execute_CreatesActivity`
  - `TestUpdateCaseExecutor_Execute_UpdatesCase`
  - `TestSendReplyExecutor_Execute_CreatesNote`
  - `TestRegisterBuiltInExecutors`
- API
  - `TestToolHandler_CreateAndListTools`
  - `TestToolHandler_ToolLifecycle`
  - `TestToolHandler_CreateTool_ForbiddenByAuthorizer`

#### Riesgo que cubre

Protege el punto mas sensible del futuro DSL: toda mutacion seguira pasando por tools
registradas y autorizadas.

---

### 4. Policy y approvals

#### Suites principales

- `internal/domain/policy/evaluator_test.go`
- `internal/domain/policy/evaluator_unit_test.go`
- `internal/domain/policy/approval_test.go`
- `internal/api/handlers/approval_test.go`

#### Cobertura relevante

- decision de policy con precedence deterministica
- fallback a permisos por rol
- auditoria de decisiones
- permission checks para tools y actions
- redaccion de PII
- ciclo de vida de approval requests
- expiracion, forbidden y estados terminales
- handlers HTTP de approvals

#### Tests mas importantes

- Evaluator
  - `TestCheckToolPermission`
  - `TestEvaluatePolicyDecision_DeterministicPrecedenceAndTrace`
  - `TestEvaluatePolicyDecision_NoMatchingRule_DeniesWithTraceAndAudit`
  - `TestCheckToolPermission_UsesActivePolicyVersionWhenAvailable`
  - `TestCheckActionPermission`
  - `TestEvaluatePolicyDecision_WildcardResource`
- Approval service
  - `TestApprovalService_CreateApprovalRequest`
  - `TestApprovalService_Decide_ApproveAndDeny`
  - `TestApprovalService_ExpiredAndPending`
  - `TestApprovalService_ForbiddenApprover`
  - `TestApprovalService_DecideExpiredRequest_ReturnsExpiredError`
  - `TestApprovalService_DecideAlreadyClosed_ReturnsAlreadyClosedError`
- Approval API
  - `TestApprovalHandler_ListPendingApprovals_Success`
  - `TestApprovalHandler_DecideApproval_SuccessNoContent`
  - `TestApprovalHandler_DecideApproval_Forbidden`
  - `TestApprovalHandler_DecideApproval_Expired`
  - `TestApprovalHandler_DecideApproval_AlreadyClosed`

#### Riesgo que cubre

Protege el constraint principal del plan: las acciones sensibles del runtime declarativo no
pueden saltarse policy ni approval.

---

### 5. Audit

#### Suites principales

- `internal/domain/audit/service_test.go`
- `internal/api/middleware/audit_test.go`
- `internal/api/handlers/audit_test.go`

#### Cobertura relevante

- append-only
- log con detalles
- filtros y query
- aislamiento por tenant
- ordering
- export CSV
- consumo de eventos
- middleware HTTP que deriva action y outcome
- handler de consulta y export de eventos

#### Tests mas importantes

- Service
  - `TestCreateAuditEvent_Success`
  - `TestAuditEvent_AppendOnly_UpdateAndDeleteBlocked`
  - `TestLogWithDetails_Success`
  - `TestAuditTenantIsolation`
  - `TestEventOrdering`
  - `TestRegisterEventSubscribers_ConsumesEvents`
  - `TestRegisterEventSubscribers_ConsumesTypedPayloadsAndNormalizesActions`
- Middleware
  - `TestAuditMiddleware_LogsActionAndOutcome`
  - `TestOutcomeFromStatus`
  - `TestActionFromRequest`
- Handler
  - `TestAuditHandler_Query_200_NoFilters`
  - `TestAuditHandler_Query_200_WithActionFilter`
  - `TestAuditHandler_GetByID_200`
  - `TestAuditHandler_Export_200_CSV`

#### Riesgo que cubre

Protege la trazabilidad de la transicion. Si la nueva capa runtime deja de auditar o audita mal,
se pierde la capacidad de operar con seguridad.

---

## Gates de no regresion recomendados

### Gate minimo para Fase 1

Ejecutar estas suites cada vez que se toque orquestacion o contratos runtime:

```powershell
go test ./internal/domain/agent/...
go test ./internal/domain/tool/...
go test ./internal/domain/policy/...
go test ./internal/domain/audit/...
go test ./internal/api/handlers/... ./internal/api/middleware/...
```

### Gate rapido para iteracion local

Subconjunto minimo cuando se este tocando `orchestrator.go`, `runner.go` o adaptadores de agentes:

```powershell
go test ./internal/domain/agent/...
go test ./internal/domain/tool/...
go test ./internal/domain/policy/...
```

### Gate ampliado para cambios que toquen trazabilidad o permisos

```powershell
go test ./internal/domain/agent/...
go test ./internal/domain/tool/...
go test ./internal/domain/policy/...
go test ./internal/domain/audit/...
go test ./internal/api/handlers/... ./internal/api/middleware/...
```

---

## Dependencias de esta baseline con AGENT_SPEC

| Area AGENT_SPEC | Baseline que la protege |
|---|---|
| `AgentRunner` / `RunnerRegistry` | `internal/domain/agent/orchestrator_test.go` + suites de agentes |
| DSL verbs -> tools | `internal/domain/tool/registry_test.go` + `builtin_executors_test.go` |
| constraints via policy | `internal/domain/policy/evaluator_test.go` |
| approvals humanas | `internal/domain/policy/approval_test.go` + `internal/api/handlers/approval_test.go` |
| audit del runtime | `internal/domain/audit/service_test.go` + middleware/handler audit |

---

## Cierre de F0.3

F0.3 puede considerarse resuelta si:

- estas suites quedan identificadas como baseline de no regresion
- se usan como gate minimo durante Fase 1
- cualquier refactor del orquestador o del pipeline de tools/policy/audit se valida contra ellas

La conclusion practica es que Fase 1 ya puede avanzar con una red de seguridad definida.
