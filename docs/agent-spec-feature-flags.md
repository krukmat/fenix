# AGENT_SPEC Feature Flags

> Fecha: 2026-03-09
> Objetivo: resolver F0.5 definiendo los feature flags minimos de la transicion

---

## Flags minimos

| Flag | Default | Habilita | No habilita |
|---|---|---|---|
| `workflows_enabled` | `false` | entidades y APIs de workflow | ejecucion DSL por si sola |
| `signals_enabled` | `false` | entidad `signal`, servicio y eventos | `SURFACE` completo en UI |
| `dsl_runner_enabled` | `false` | `agent_type="dsl"` y ejecucion por DSLRunner | activacion automatica global |
| `scheduler_enabled` | `false` | polling y resume de `WAIT` | dispatch |
| `dispatch_enabled` | `false` | `DISPATCH` interno y luego externo A2A-first | BPMN o NL -> DSL |

## Direction de interoperabilidad

- `dispatch_enabled` debe gobernar una integracion A2A-first, no un protocolo propietario.
- La integracion MCP para tools/contexto debe activarse sobre adapters compatibles con el estandar.

---

## Regla de uso

Los flags deben permitir activar AGENT_SPEC por capas, sin afectar a los agentes Go actuales.

Reglas:

- Ningun flag nuevo debe cambiar el comportamiento de los agentes Go cuando esta en `false`.
- `dsl_runner_enabled` depende operativamente de `workflows_enabled`.
- `scheduler_enabled` depende operativamente de `dsl_runner_enabled`.
- `dispatch_enabled` depende operativamente de `dsl_runner_enabled`.

---

## Orden recomendado de activacion

1. `workflows_enabled`
2. `signals_enabled`
3. `dsl_runner_enabled`
4. `scheduler_enabled`
5. `dispatch_enabled`

Este orden permite abrir primero persistencia y gobernanza, luego ejecucion, luego asincronia
y por ultimo delegacion.

---

## Alcance por fase

| Fase | Flags esperados |
|---|---|
| Fase 1 | ninguno activo |
| Fase 2 | `workflows_enabled`, `signals_enabled` |
| Fase 3 | `workflows_enabled`, `signals_enabled` |
| Fase 4 | `dsl_runner_enabled` |
| Fase 6 | `scheduler_enabled` |
| Fase 8 | `dispatch_enabled` |

---

## Punto de configuracion

La opcion mas simple para esta transicion es usar configuracion por workspace en `workspace.settings`
con fallback a configuracion global si hiciera falta.

La expectativa minima es:

- control por workspace
- defaults seguros en `false`
- lectura centralizada
- posibilidad de rollback sin deploy

---

## Cierre de F0.5

F0.5 puede considerarse resuelta si:

- los 5 flags quedan definidos
- su dependencia operativa queda explicitada
- el orden de activacion queda fijado
- los agentes Go permanecen intactos cuando todos estan en `false`
