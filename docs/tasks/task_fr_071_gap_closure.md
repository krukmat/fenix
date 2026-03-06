# Task - FR-071 Gap Closure (Policy Engine)

**Status**: Closed  
**Depends on**: Base de policies existente

## Objetivo
Estandarizar evaluacion determinista por `policy_set`/`policy_version` con precedencia consistente y traza de regla aplicada en todos los decision paths relevantes.

## Entregables
- Resolucion runtime por workspace + policy set/version activa.
- Matching uniforme recurso/accion/efecto en paths API y tools.
- Trazabilidad de decision (regla/effect/rule trace) auditable.
- Suite de regresion para conflictos de policy y precedencia.

## Criterios de cierre (DoD FR-071)
- Todas las decisiones de autorizacion criticas pasan por el entrypoint unificado de policy engine.
- Resolucion determinista documentada y validada (`priority`, `deny-overrides` en empate, orden estable).
- Evidencia de logging/auditoria para decisiones allow/deny con policy set/version.
- Tests de regresion en verde para conflictos, precedencia y fallback controlado.

---

## Avance actual

### Hecho
- Evaluacion determinista por `policy_set`/`policy_version` activa en `internal/domain/policy/evaluator.go`.
- Resolucion de reglas con precedencia estable (`priority`, `deny` sobre `allow` en empate, orden estable por ID).
- Traza de decision (`PolicyDecisionTrace`) con `matched_rule`, `matched_effect`, `rule_trace`.
- Logging de decision (`policy.evaluated`) con metadatos de set/version/regla en auditoria.
- Integracion en `CheckToolPermission`/`CheckActionPermission` con fallback RBAC explicito cuando no hay policy activa.
- Adopcion del entrypoint unificado en handlers admin del scope:
  - `internal/api/handlers/prompt.go`
  - `internal/api/handlers/eval.go`
- Wiring runtime actualizado con `policyEngine` en `internal/api/routes.go`.

### Gap pendiente
- Ninguno en los decision paths admin/API inventariados para esta fase de FR-071.

---

## Ejecucion realizada en esta iteracion

### Cambios aplicados
- `internal/domain/policy/evaluator.go`
  - `EvaluatePolicyDecision(...)` ahora devuelve `deny` con `Trace` explicita y audita `policy.evaluated` cuando hay policy activa pero no hay regla que matchee.
- `internal/domain/policy/evaluator_test.go`
  - Nuevo test `TestEvaluatePolicyDecision_NoMatchingRule_DeniesWithTraceAndAudit`.
- `internal/api/handlers/authorization.go`
  - Nuevo helper comun para enforcement `CheckActionPermission(...)` en handlers.
- `internal/api/handlers/tool.go`
  - Reutiliza el helper comun de autorizacion.
- `internal/api/handlers/eval.go`
  - Gating unificado para `admin.eval.suites.create`, `admin.eval.suites.list`, `admin.eval.suites.get`, `admin.eval.run`, `admin.eval.runs.list`, `admin.eval.runs.get`.
- `internal/api/handlers/prompt.go`
  - Gating unificado para `admin.prompts.list`, `admin.prompts.create`, `admin.prompts.promote`, `admin.prompts.rollback`.
- `internal/api/handlers/eval_test.go`
  - Nuevos tests de `403 forbidden` y `401 missing user` con authorizer activo.
- `internal/api/handlers/prompt_test.go`
  - Nuevos tests de `403 forbidden` y `401 missing user` con authorizer activo.

### Evidencia de cobertura FR-071
- Resolucion por `policy_set`/`policy_version`: OK
- Determinismo de precedencia (`priority`, `deny-overrides`, orden estable): OK
- Trazabilidad de decision con auditoria (`policy.evaluated`): OK, incluyendo no-match
- Regresion de conflictos/precedencia/no-match: OK
- Adopcion del entrypoint unificado en paths admin visibles del scope: OK

### Validacion ejecutada en este entorno
- `go test ./internal/domain/policy`: OK
- `go test ./internal/api/handlers`: OK
- Nota: `./internal/api/handlers` requirio ejecucion fuera del sandbox por acceso denegado al directorio temporal de build de Go; no hubo fallo funcional del codigo.

---

## Estado actual vs cierre
- **Estado**: Closed.
- **Cierre realizado**: Fase 2 + Fase 3 completadas y validadas con tests en verde.

### Checklist para cierre definitivo
- [x] Implementacion tecnica FR-071 aplicada.
- [x] Pruebas de regresion anadidas/actualizadas.
- [x] Ejecutar suite en entorno con Go (local/CI) y adjuntar salida verde.
- [x] Cambiar estado a Closed tras validacion de tests.
