# AGENT_SPEC Phase 1 Quality Gates

> Fecha: 2026-03-09
> Objetivo: fijar los quality gates minimos para la Fase 1 de implementacion

---

## Alcance

Aplica a cambios en:

- `orchestrator.go`
- nuevo contrato `AgentRunner`
- `RunContext`
- `RunnerRegistry`
- adaptacion de agentes Go al nuevo contrato

No sustituye la CI general. Define el gate minimo para aceptar cambios de Fase 1.

---

## Baselines que protegen la fase

- No regresion: `docs/agent-spec-regression-baseline.md`
- Agentes Go: `docs/agent-spec-go-agents-baseline.md`
- Contratos core: `docs/agent-spec-core-contracts-baseline.md`
- Feature flags: `docs/agent-spec-feature-flags.md`

---

## Reglas de aceptacion

Un cambio de Fase 1 no se acepta si:

- rompe comportamiento actual de agentes Go
- rompe `ToolRegistry.Execute`
- rompe enforcement de policy o approvals
- degrada `agent_run` o `agent_run_step`
- exige activar feature flags para que sigan funcionando agentes Go actuales

---

## Gates operativos

### Gate corto

Usar durante iteracion local cuando se toque runtime u orquestacion:

```powershell
go test ./internal/domain/agent/...
go test ./internal/domain/tool/...
go test ./internal/domain/policy/...
```

### Gate de transicion

Usar antes de cerrar cualquier slice de Fase 1:

```powershell
go test ./internal/domain/agent/...
go test ./internal/domain/tool/...
go test ./internal/domain/policy/...
go test ./internal/domain/audit/...
go test ./internal/api/handlers/... ./internal/api/middleware/...
```

### Gate de merge

Usar antes de integrar a rama principal:

- Gate de transicion en verde
- CI general del repo en verde

---

## Evidencia minima por cambio

Cada cambio de Fase 1 debe dejar evidencia de:

- compatibilidad hacia atras de agentes Go
- continuidad de trazabilidad en `agent_run`
- integracion intacta con tools, policy y audit

La evidencia minima aceptable es:

- tests en verde
- diff acotado
- sin cambios funcionales no planificados

---

## Orden recomendado de control

1. Validar contrato nuevo
2. Validar orquestador adaptado
3. Validar un agente Go adaptado
4. Validar resto de agentes
5. Ejecutar gate de transicion completo

---

## Cierre de Fase 1

Fase 1 puede considerarse cerrada si:

- existe `AgentRunner`
- existe `RunContext`
- existe `RunnerRegistry`
- el orquestador delega por contrato comun
- los agentes Go siguen comportandose igual
- los quality gates anteriores quedan verdes
