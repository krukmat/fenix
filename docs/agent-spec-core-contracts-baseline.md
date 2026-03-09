# AGENT_SPEC Core Contracts Baseline

> Fecha: 2026-03-09
> Objetivo: resolver F0.6 fijando los contratos actuales que no deben romperse durante la transicion

---

## Uso

Este documento no redefine interfaces futuras. Solo fija el baseline actual de los contratos
que AGENT_SPEC va a reutilizar o envolver:

- `agent_run`
- `approval_request`
- `ToolRegistry`
- `PolicyEngine`
- `EventBus`

Si alguno de estos contratos cambia, debe hacerse de forma compatible o quedar explicitamente
tratado como cambio funcional.

---

## 1. `agent_run`

Referencia principal: `internal/domain/agent/orchestrator.go`

### Estado actual

Estados actualmente usados:

- `running`
- `success`
- `partial`
- `abstained`
- `failed`
- `escalated`

### Invariantes actuales

- `TriggerAgent` crea un `agent_run` con estado inicial `running`.
- `TriggerAgent` crea tambien el paso inicial en `agent_run_step`.
- `trigger_type` valido: `event`, `schedule`, `manual`, `copilot`.
- Un `agent_run` guarda `inputs`, `reasoning_trace`, `tool_calls`, `output`, costos y tiempos.
- Los agentes actuales persisten sus resultados actualizando el mismo `agent_run`.

### Implicacion para AGENT_SPEC

- `accepted`, `rejected` y `delegated` deben agregarse sin romper los estados actuales.
- La introduccion de `DSLRunner` no puede eliminar el modelo actual de trazabilidad del run.

---

## 2. `approval_request`

Referencia principal: `internal/domain/policy/approval.go`

### Estado actual

Estados:

- `pending`
- `approved`
- `denied`
- `expired`

### Contrato actual

Creacion:

- `CreateApprovalRequest(ctx, input)` crea una solicitud en `pending`
- si `payload` viene vacio, se normaliza a `{}`
- registra audit event `approval.requested`

Decision:

- `DecideApprovalRequest(ctx, id, decision, decidedBy)`
- acepta alias `approve/approved` y `deny/denied`
- valida ownership del aprobador
- rechaza solicitudes cerradas o expiradas
- registra `approval.approved`, `approval.denied` o `approval.expired`

Consulta:

- `GetPendingApprovals(ctx, userID)` devuelve pendientes del aprobador
- al consultar pendientes, puede expirar las vencidas

### Implicacion para AGENT_SPEC

- los flujos declarativos deben reutilizar este servicio; no crear un mecanismo paralelo
- `WAIT`, `DSLRuntime` o `Judge` no deben saltarse approval para acciones sensibles

---

## 3. `ToolRegistry`

Referencias principales:

- `internal/domain/tool/registry.go`
- `internal/domain/tool/execution_pipeline.go`

### Contrato actual

Registro:

- `Register(name, executor)` registra un ejecutor por nombre
- no permite duplicados

Definicion:

- persiste `tool_definition`
- valida schema minimo
- soporta lifecycle create/update/activate/deactivate/delete

Ejecucion:

- `Execute(ctx, workspaceID, toolName, params)` ejecuta por nombre
- valida que la tool exista y este activa
- valida parametros contra schema
- exige contexto de usuario para enforcement
- pasa por autorizacion de permisos
- emite auditoria de `tool.executed` o `tool.denied`

### Invariantes actuales

- las mutaciones pasan por tools registradas
- permisos y validacion viven en el pipeline de ejecucion
- el runtime no deberia llamar servicios de dominio saltandose `ToolRegistry`

### Implicacion para AGENT_SPEC

- `SET`, `NOTIFY`, `SURFACE` y mutaciones futuras deben mapear a tools
- `VerbMapper` debe apoyarse en este contrato, no reemplazarlo

---

## 4. `PolicyEngine`

Referencia principal: `internal/domain/policy/evaluator.go`

### Contrato actual

Capacidades observadas:

- `CheckToolPermission`
- `CheckActionPermission`
- `EvaluatePolicyDecision`
- `BuildPermissionFilter`
- `RedactPII`
- `LogAuditEvent`

### Invariantes actuales

- policy puede negar incluso si el rol permite
- la decision usa precedencia deterministica
- las decisiones relevantes generan traza y audit
- el control de acceso a tools ya esta integrado en el pipeline actual

### Implicacion para AGENT_SPEC

- `Judge` no reemplaza a `PolicyEngine`
- los constraints declarativos deben terminar aplicandose sobre este enforcement existente
- un workflow DSL no puede ejecutar acciones fuera del pipeline de policy actual

---

## 5. `EventBus`

Referencia principal: `internal/infra/eventbus/eventbus.go`

### Contrato actual

Interface:

- `Publish(topic string, payload any)`
- `Subscribe(topic string) <-chan Event`

Semantica:

- implementacion in-memory
- `Publish` no bloquea
- si el buffer del subscriber esta lleno, el evento se descarta
- no hay persistencia ni replay
- la suscripcion devuelve un canal de solo lectura

### Implicacion para AGENT_SPEC

- la fase inicial de workflows puede apoyarse en este bus
- no debe asumirse durabilidad de eventos
- triggers `ON <event>` en Fase 1-4 deben diseñarse con esta limitacion

---

## Reglas de compatibilidad para la transicion

- No romper firmas publicas ya usadas transversalmente salvo que exista adaptador compatible.
- No cambiar semantica de `ToolRegistry.Execute`.
- No cambiar semantica de `ApprovalService` para introducir aprobaciones declarativas.
- No degradar trazabilidad de `agent_run` ni `agent_run_step`.
- No asumir persistencia o entrega garantizada en `EventBus`.

---

## Cierre de F0.6

F0.6 puede considerarse resuelta si:

- estos contratos quedan explicitados como baseline
- Fase 1 reutiliza estos contratos en lugar de introducir duplicados
- cualquier extension nueva se hace como compatibilidad hacia adelante y no como ruptura del core actual
