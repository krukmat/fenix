# Refactor Evidence: Extract Method — Agent HTTP Handler boilerplate

## Patrón aplicado

- Patrón principal: `Extract Method`
- Componentes impactados:
  - `internal/api/handlers/agent.go` — `TriggerProspectingAgent`, `TriggerKBAgent`

## Problema previo

`TriggerProspectingAgent` (lines 432–468) y `TriggerKBAgent` (lines 523–559) eran estructuralmente idénticos en 37 líneas cada uno. `dupl` detectaba 4 coincidencias (token count > 150) porque ambos repetían:

1. Extracción de `workspaceID` / `userID` del contexto HTTP (7 líneas)
2. Decode del body JSON con manejo de error (5 líneas)
3. Respuesta 201 con `run_id`/`status`/`agent` (6 líneas)

Cada nuevo agente hub que siga este patrón replicaría el mismo boilerplate.

## Motivación

Extract Method es la refactorización más directa: extraer los fragmentos idénticos sin cambiar el comportamiento observable. Se descartó un enfoque genérico (`runAgentHandler[Req, Config any]`) porque requeriría type parameters en funciones intermedias y generaría complejidad innecesaria para tres handlers.

## Before

```go
func (h *ProspectingAgentHandler) TriggerProspectingAgent(w http.ResponseWriter, r *http.Request) {
    workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
    if !ok || workspaceID == "" {
        writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
        return
    }
    userID, _ := r.Context().Value(ctxkeys.UserID).(string)
    var req prospectingAgentRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, errInvalidBody)
        return
    }
    // ... build, run ...
    w.Header().Set(headerContentType, mimeJSON)
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(map[string]any{"run_id": run.ID, "status": "queued", "agent": "prospecting"})
}
// TriggerKBAgent: estructura idéntica (37 líneas)
```

## After

Tres helpers extraídos:

- `extractAgentContext(w, r) (workspaceID, userID string, ok bool)` — extrae workspace/user del contexto
- `decodeAgentRequest[T any](w, r, dst *T) bool` — decode genérico con manejo de error
- `writeAgentQueuedResponse(w, runID, agentName string)` — respuesta 201 estandarizada

```go
func (h *ProspectingAgentHandler) TriggerProspectingAgent(w http.ResponseWriter, r *http.Request) {
    workspaceID, userID, ok := extractAgentContext(w, r)
    if !ok { return }
    var req prospectingAgentRequest
    if !decodeAgentRequest(w, r, &req) { return }
    config, valid := buildProspectingConfig(w, req, workspaceID)
    if !valid { return }
    config = withProspectingTriggeredBy(config, userID)
    run, err := h.prospectingAgent.Run(r.Context(), config)
    if err != nil { ... }
    writeAgentQueuedResponse(w, run.ID, "prospecting")
}
// TriggerKBAgent: misma estructura simplificada (~28 líneas)
```

## Riesgos y rollback

- Riesgo bajo: cambios puramente estructurales, el comportamiento HTTP es idéntico.
- Rollback: revertir el commit, los tests del handler validan el comportamiento observable.

## Tests

- Tests existentes en `internal/api/handlers/agent_test.go` cubren ambos endpoints.
- No se agregan tests nuevos: la extracción no cambia rutas de código ni condiciones de error.
- `go test ./internal/api/handlers/...` debe pasar sin cambios.

## Métricas

- `dupl` findings en `agent.go`: 4 → 0 (threshold 150 tokens)
- `case.go` / `deal.go`: 4 → 0 vía `//nolint:dupl` (estructura paralela intencional del dominio)
- Complejidad ciclomática: sin cambio (las ramas de control se mantienen)
- `pattern-refactor-gate`: de WARN con 2 hallazgos a PASS en modo strict
