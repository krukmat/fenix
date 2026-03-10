# AGENT_SPEC Phase 1 Regression Status

> Fecha: 2026-03-09
> Objetivo: documentar el estado real de F1.8 sobre la cobertura de no regresion de Fase 1

---

## Estado

`F1.8` queda documentada como **completada**.

La base de Fase 1 ya tiene evidencia suficiente para validar el refactor principal
de contrato, wiring y compatibilidad del nuevo camino de ejecucion.

---

## Cobertura ya presente

### 1. Contrato runtime

Archivos:
- `internal/domain/agent/runner_test.go`
- `internal/domain/agent/runner_registry_test.go`

Cubierto:
- `AgentRunner` compila como contrato comun
- `RunContext` clona dependencias y call chain
- `RunnerRegistry` registra, consulta y resuelve runners
- errores deterministas para tipos desconocidos o invalidos
- concurrencia basica de lecturas en registry

### 2. Orquestador

Archivo:
- `internal/domain/agent/orchestrator_test.go`

Cubierto:
- `ResolveRunner` falla si no hay registry
- `ResolveRunner` resuelve por `agent_type`
- `ExecuteAgent` delega al runner resuelto
- nuevos estados `accepted`, `rejected`, `delegated`
- transicion `running -> accepted`
- transiciones terminales nuevas
- compatibilidad con transiciones legacy

### 3. Agentes Go adaptados

Archivos:
- `internal/domain/agent/agents/runner_adapters_test.go`
- suites existentes de `support`, `prospecting`, `kb`, `insights`

Cubierto:
- adapters `SupportRunner`, `ProspectingRunner`, `KBRunner`, `InsightsRunner`
- decode de `TriggerAgentInput` hacia configs tipados
- fallback de `workspace_id` y `triggered_by`
- suites legacy de agentes Go siguen verdes

### 4. Wiring de registro de agentes Go

Archivo:
- `internal/domain/agent/agents/registry_test.go`

Cubierto:
- registro explicito de `support`, `prospecting`, `kb`, `insights`
- error si falta registry
- error si falta algun agente requerido

### 5. Gate corto ejecutado

Ejecutado y verde:

```powershell
go test ./internal/domain/agent/...
go test ./internal/domain/tool/...
go test ./internal/domain/policy/...
```

Nota de entorno:
- en esta maquina se uso `GOCACHE` local al repo por un problema de permisos en el cache global de Go

---

## Evidencia adicional de cierre

### 1. Gate de transicion completo

Ejecutado y verde:

```powershell
go test ./internal/domain/audit/...
go test ./internal/api/handlers/... ./internal/api/middleware/...
```

### 2. Evidencia end-to-end del nuevo camino

Ya existe una prueba que combina explicitamente:
- `RunnerRegistry`
- `RegisterCurrentGoRunners`
- `ExecuteAgent`
- un agente Go real

Archivo:
- `internal/domain/agent/agents/registry_test.go`

---

## Lectura operativa

Hoy Fase 1 esta funcionalmente estable y validada.

Lo ya cubierto valida:
- contrato comun
- contexto runtime
- registro de runners
- delegacion por orquestador
- adapters de agentes Go
- registro explicito de agentes Go actuales
- estados nuevos de `agent_run`
- gate de transicion completo
- una prueba integrada del camino `registry -> execute -> runner real`

---

## Conclusion

`F1.8` queda cerrada.

Con esto, la Fase 1 queda cubierta por:

1. contrato comun de ejecucion
2. contexto runtime compartido
3. registry de runners
4. delegacion del orquestador
5. adaptacion de agentes Go
6. registro explicito de agentes Go actuales
7. nuevos estados de `agent_run`
8. gates de regresion de Fase 1

El siguiente paso recomendado es abrir la Fase 2.
