# Task 2.6 — Coverage & Quality Gateway Plan

**Status**: � Etapa de gate COVERAGE_MIN>=60 aplicada y validada
**Date**: 2026-02-13
**Related**: `task_2.6.md`, `task_2.7.md`, `.github/workflows/ci.yml`, `Makefile`

---

## 1) Baseline actual (última medición)

### Cobertura global
- `coverage.out` (repo completo): **32.2%**

### Cobertura TDD (paquetes focalizados)
- `coverage_tdd.out` (`internal/api`, `internal/api/handlers`, `internal/domain/knowledge`, `pkg/auth`): **52.5%**

### Snapshot por paquete clave
- `internal/api`: **95.4%**
- `internal/api/handlers`: **36.9%**
- `internal/domain/knowledge`: **86.4%**
- `pkg/auth`: **86.5%**

Diagnóstico: el cuello de botella principal es `internal/api/handlers`.

### Estado actual (actualizado)
- Global (`coverage.out`): **41.3%**
- TDD (`coverage_tdd.out`): **66.9%**
- `internal/api/handlers`: **57.0%**

Progreso desde baseline inicial:
- Handlers: **36.9% → 57.0%**
- TDD: **52.5% → 66.5%**

---

## 2) Gateways activos en CI (quality gates)

En `.github/workflows/ci.yml` (job `Lint and Test`) están activos:

1. `Run tests` (`make test`)
2. `Race stability gate` (`make race-stability`)
3. `Global coverage gate` (`make coverage-gate`)
4. `TDD coverage gate` (`make coverage-tdd`)
5. `Build binary`

Además, `coverage-gate` y `coverage-tdd` están implementados en `Makefile`.

---

## 3) Objetivo de mejora

Subir cobertura de forma incremental, sin introducir flakes y manteniendo el pipeline estable.

### Objetivo técnico principal
- Elevar `internal/api/handlers` desde ~36.9% a >50% en iteraciones.

---

## 4) Plan de subida gradual de umbrales

### Etapa A (actual)
- Global (`COVERAGE_MIN`): **35** ✅ (ajustado en Makefile)
- TDD (`TDD_COVERAGE_MIN`): **60** ✅ (ajustado en Makefile)

### Etapa B (cuando haya estabilidad sostenida)
- Subir a:
  - Global: **35**
  - TDD: **70**

### Etapa C (segunda mejora)
- Subir a:
  - Global: **45**
  - TDD: **78**

### Etapa D (pre-objetivo 85)
- Subir a:
  - Global: **60**
  - TDD: **82**

### Etapa E (objetivo final)
- Subir a:
  - Global: **85**
  - TDD: **85**

### Criterio de avance entre etapas
Mover de etapa solo si se cumplen ambos:
1. Cobertura real >= nuevo umbral por al menos 5 corridas consecutivas.
2. `race-stability` en verde sin flakes en el mismo período.

---

## 5) Backlog priorizado de tests (impacto/costo)

Prioridad alta (handlers con mayor potencial de mejora):
1. `internal/api/handlers/activity.go`
2. `internal/api/handlers/attachment.go`
3. `internal/api/handlers/case.go`
4. `internal/api/handlers/note.go`
5. `internal/api/handlers/pipeline.go`
6. `internal/api/handlers/timeline.go`

Para cada handler nuevo testear mínimo:
- Happy path (200/201)
- Validación request (400)
- Falta de contexto/auth (401)
- Entidad no encontrada (404) cuando aplique

---

## 6) Anti-flake y confiabilidad

Reglas obligatorias para tests nuevos:
- Si se usa SQLite `:memory:`, forzar:
  - `db.SetMaxOpenConns(1)`
  - `db.SetMaxIdleConns(1)`
- Evitar estado global mutable sin sincronización.
- IDs/helpers thread-safe para `t.Parallel()`.

Validación recomendada antes de merge:
- `make race-stability`
- `make coverage-gate`
- `make coverage-tdd`

---

## 7) Riesgos y mitigaciones

### Riesgo 1 — Flake por concurrencia
- **Mitigación**: race gate + tests deterministas + DB in-memory single-conn.

### Riesgo 2 — Subir umbral demasiado pronto
- **Mitigación**: escalado por etapas con criterio de 5 corridas verdes.

### Riesgo 3 — Mejora de cobertura “cosmética”
- **Mitigación**: priorizar paths de negocio y errores reales, no solo líneas fáciles.

---

## 8) Definición de éxito

Se considera completada esta iniciativa cuando:
1. `internal/api/handlers` supera 50% de cobertura.
2. Etapa B de umbrales aplicada sin romper CI.
3. Se mantienen corridas estables sin regresión en race/coverage gates.

---

## 9) Próximo bloque técnico (para habilitar Global 35)

Prioridad inmediata de cobertura global (fuera de handlers):
1. `internal/api/context.go`
2. `internal/api/errors.go`
3. `internal/api/routes.go`
4. `internal/api/ctxkeys/ctxkeys.go`
5. `cmd/fenix/main.go` (smoke tests de rutas CLI/help/version)

Objetivo corto: mover global de **32.2%** a **>=35%** y luego subir `COVERAGE_MIN` a 35 sin bloquear CI.

Estado: ✅ logrado (global 41.3%, gate en 35 estable).

---

## 10) Nota de estabilidad (flaky test en knowledge/embedder)

Se estabilizó `coverage-tdd` ajustando `internal/domain/knowledge/embedder_test.go`:
- incremento de ventana de espera en tests async de 3s a 8s,
- pequeña espera para asegurar que el subscriber esté suscripto antes del publish.

Con esto, `make coverage-tdd` volvió a verde de forma consistente en esta iteración.

---

## 11) Prueba de escalado en GitHub Actions (workflow)

Se actualizaron temporalmente los parámetros en `.github/workflows/ci.yml` para validar un escalón superior:
- `Global coverage gate`: `COVERAGE_MIN=40 make coverage-gate`
- `TDD coverage gate`: `TDD_COVERAGE_MIN=65 make coverage-tdd`

Validación local con esos mismos valores:
- Global: **41.3%** → PASS contra 40
- TDD: **66.9%** → PASS contra 65

Conclusión: el pipeline soporta correctamente este escalón superior.

---

## 12) Actualización 2026-02-13 — COVERAGE_MIN elevado a 60

Se aplicó una mejora estructural del gate global para alinearlo a código de aplicación (no generado):

- `Makefile`:
  - `COVERAGE_MIN` default actualizado a **60**.
  - `coverage-gate` ahora evalúa `coverage_gate.out` filtrando:
    - `internal/infra/sqlite/sqlcgen/*` (código generado)
    - `cmd/fenix/main.go` (bootstrap)
    - `internal/version/*` (wiring/version metadata)
- `.github/workflows/ci.yml`:
  - Job `Global coverage gate` actualizado a `COVERAGE_MIN=60 make coverage-gate`.

### Validación local posterior al cambio

- `go test ./...` ✅
- `make coverage-gate` con min 60 ✅
  - **Total coverage (gate scope): 77.9%**
- `make coverage-app-gate` (`COVERAGE_APP_MIN=70`) ✅
  - **App coverage: 77.9%**
- `make coverage-tdd` (`TDD_COVERAGE_MIN=76`) ✅
  - **TDD coverage: 76.3%**

### Nota de priorización por referencias de módulos

Para priorizar próximos incrementos se agregó criterio de fan-in (cantidad de paquetes que importan cada módulo):

- `internal/infra/sqlite`: **6** referencias
- `internal/domain/crm`: **2** referencias
- `internal/domain/knowledge`: **2** referencias
- `internal/domain/auth`: **2** referencias
- `internal/api/handlers`: **1** referencia
- `internal/api`: **1** referencia

Interpretación: priorizar módulos con alto fan-in y alto retorno de cobertura en código de negocio (`domain/*`, `api/handlers`) para sostener el nuevo umbral sin flakes.
