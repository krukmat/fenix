# Go Agents Baseline

> Fecha: 2026-03-09
> Objetivo: resolver F0.4 con una referencia minima de comportamiento para los agentes Go actuales

---

## Uso

Este documento fija que no debe cambiar cuando los agentes Go se adapten a `AgentRunner`
en Fase 1. Si un refactor cambia alguno de estos comportamientos, ya no es un cambio
tecnico aislado: es una regresion funcional.

---

## Support Agent

Referencia: `internal/domain/agent/agents/support_test.go`

- Allowed tools: `update_case`, `send_reply`, `create_task`, `search_knowledge`, `get_case`, `get_contact`.
- Si no hay evidencia relevante, escala.
- Si el score de conocimiento es alto, resuelve el caso.
- Si el score es intermedio, abstiene.
- Requiere `workspace_id` y `case_id`.
- En escalacion, el `agent_run` queda `escalated`.
- En resolucion, el `agent_run` queda `success` y el caso termina en `resolved`.
- En abstention, el `agent_run` queda `abstained`.

Tests de referencia:

- `TestSupportAgent_Run_EscalatesWhenNoKnowledge`
- `TestSupportAgent_Run_ResolvesWhenHighConfidence`
- `TestSupportAgent_Run_AbstainsWhenConfidenceIsMedium`

---

## Prospecting Agent

Referencia: `internal/domain/agent/agents/prospecting_test.go`

- Allowed tools: `search_knowledge`, `create_task`, `get_lead`, `get_account`.
- Requiere `workspace_id` y `lead_id`.
- Con confianza alta, genera un draft de outreach y crea una tarea.
- Con confianza baja, no actua y deja salida de tipo `skip`.
- Si el lead no existe, falla con error funcional.
- Si el lead tiene sensibilidad alta, no muta nada: crea `approval_request` y deja el run en `escalated`.
- En sensibilidad alta, la salida debe incluir `action=pending_approval`, `reason=high_sensitivity` y `approval_id`.

Tests de referencia:

- `TestProspectingAgent_Run_HighConfidence_DraftsOutreach`
- `TestProspectingAgent_Run_LowConfidence_Skips`
- `TestProspectingAgent_Run_MissingLead_Error`
- `TestProspectingAgent_Run_HighSensitivity_CreatesApprovalAndBlocks`

---

## KB Agent

Referencia: `internal/domain/agent/agents/kb_test.go`

- Allowed tools: `search_knowledge`, `create_knowledge_item`, `update_knowledge_item`.
- Requiere `workspace_id` y `case_id`.
- Solo opera sobre casos resueltos.
- Si no encuentra duplicado claro, crea articulo.
- Si encuentra duplicado fuerte, actualiza articulo existente.
- Si el caso no esta resuelto, falla con error funcional.
- Si el caso tiene sensibilidad alta, bloquea la mutacion, crea `approval_request` y deja el run en `escalated`.
- En sensibilidad alta, la salida debe incluir `action=pending_approval`, `reason=high_sensitivity` y `approval_id`.

Tests de referencia:

- `TestKBAgent_Run_ResolvedCase_CreatesArticle`
- `TestKBAgent_Run_DuplicateFound_Updates`
- `TestKBAgent_Run_UnresolvedCase_Error`
- `TestKBAgent_Run_HighSensitivity_CreatesApprovalAndBlocksMutation`

---

## Insights Agent

Referencia: `internal/domain/agent/agents/insights_test.go`

- Allowed tools: `search_knowledge`, `query_metrics`.
- Requiere `workspace_id` y `query`.
- Para consultas de funnel o backlog, responde con metricas y salida no abstained.
- Para consultas sin datos suficientes, abstiene con `action=abstain` y `confidence=low`.
- Debe registrar `tool_calls` cuando consulta metricas.
- El parseo de intencion debe priorizar `case_backlog` frente a consultas genericas de volumen cuando aplique.

Tests de referencia:

- `TestInsightsAgent_Run_SalesFunnelQuery`
- `TestInsightsAgent_Run_CaseBacklogQuery`
- `TestInsightsAgent_Run_EmptyData_Abstains`
- `TestParseQueryIntent_BacklogPriorityOverCaseVolume`

---

## Regla de transicion

Durante Fase 1:

- no cambiar allowed tools por agente sin decision explicita
- no cambiar las condiciones de `success`, `abstained`, `escalated` o error funcional
- no eliminar approvals en casos de sensibilidad alta
- no cambiar side effects esperados en CRM o `approval_request`

Si alguno de estos puntos cambia, el cambio debe tratarse como funcional y no como simple
adaptacion al nuevo contrato runtime.
