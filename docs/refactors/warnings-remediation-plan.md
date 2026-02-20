# Plan de Remediación de Warnings — Pattern Opportunities Gate

## Contexto

Estado inicial observado al ejecutar:

```bash
make pattern-opportunities-gate PATTERN_GATE_MODE=warn PATTERN_GATE_TS_DUP_THRESHOLD=2
```

- Evidencia documental: **0** archivos de refactor.
- Go (`dupl`): **32** coincidencias estructurales.
- TypeScript (`jscpd`): **90%** de duplicación (6 clones detectados en reporte actual).

---

## Objetivo

Reducir warnings del gate de forma incremental sin bloquear entregas, para luego migrar de `warn` a `strict` con riesgo controlado.

---

## Fase 0 — Baseline y trazabilidad (1-2 días)

### Acciones
1. Crear evidencia inicial: `docs/refactors/2026-02-XX_baseline.md` usando `template.md`.
2. Registrar baseline de métricas (dupl/jscpd) en ese documento.
3. Mantener CI en `PATTERN_GATE_MODE=warn`.

### Criterio de salida
- Desaparece warning de “No hay evidencias de refactor”.

---

## Fase 1 — Reducción de duplicación Go (3-5 días)

### Scope inicial
- Prioridad en `internal/domain/crm/*` (según muestra del gate).

### Acciones
1. Identificar bloques duplicados top por volumen (primeras 10 coincidencias de `dupl`).
2. Extraer helpers comunes (normalización, validaciones repetidas, mapeos compartidos).
3. Aplicar refactor por lotes pequeños (2-3 archivos por PR).
4. Validar cada lote con tests + lint + gate en warn.

### Patrones objetivo
- Extract Method
- Template Method (liviano)
- Factory helper para constructores repetidos

### Criterio de salida
- Bajar `dupl` de ~32 a **<=12** en primera iteración.

---

## Fase 2 — Reducción de duplicación TypeScript (4-6 días)

### Scope inicial
- `mobile/src`
- `bff/src`

### Acciones
1. Ejecutar `jscpd` con reporte y atacar primero top clones.
2. En Mobile: extraer utilidades y hooks comunes.
3. En BFF: unificar flujos repetidos de proxy/aggregated handlers.
4. Integrar pruebas para asegurar no regresión funcional.

### Patrones objetivo
- Extract Utility Module
- Strategy Lookup Table
- Hook/Factory reuse para queries/listados

### Criterio de salida
- Bajar duplicación jscpd de ~90% a **<=20%** en primera iteración.

---

## Fase 3 — Calibración y endurecimiento (1 semana)

### Acciones
1. Ajustar exclusiones justificadas (tests, generated, fixtures si aplica).
2. Endurecer umbrales en etapas para TS (`20 -> 10 -> 5`).
3. Mantener `dupl` en `threshold=120` inicialmente, luego evaluar bajar si el ruido es bajo.
4. Definir “Definition of Done” para activar `strict`.

### Criterio para activar `strict`
- Evidencia documental consistente por PR de refactor.
- Métricas estables por 2 semanas consecutivas.
- Bajo nivel de falsos positivos.

---

## Roadmap recomendado

1. Semana 1: Fase 0 + comienzo Fase 1.
2. Semana 2: terminar Fase 1 + inicio Fase 2.
3. Semana 3: consolidar Fase 2.
4. Semana 4: Fase 3 y decisión `warn` -> `strict`.

---

## Comandos útiles

```bash
# Gate completo en modo warn
make pattern-opportunities-gate PATTERN_GATE_MODE=warn PATTERN_GATE_TS_DUP_THRESHOLD=2

# Gate en modo strict (para validación final)
make pattern-opportunities-gate PATTERN_GATE_MODE=strict PATTERN_GATE_TS_DUP_THRESHOLD=5

# Linter Go enfocado en duplicación
golangci-lint run --enable-only=dupl ./...
```

---

## Patrón aplicado

- Patrón principal: **N/A (documento de planificación)**
- Componentes impactados: N/A

## Problema previo

El plan no estaba explícito en el repositorio para seguir una ejecución por fases.

## Motivación

Tener trazabilidad del roadmap para reducir warnings sin bloquear entregas.

## Before

- No había documento operativo detallando fases, metas y criterios de salida.

## After

- Existe un plan versionado con fases, objetivos y comandos operativos.

## Riesgos y rollback

- Riesgo: desactualización del plan respecto del estado real del código.
- Mitigación: actualizar el documento al cierre de cada fase.

## Tests

- N/A (documentación).

## Métricas

- Baseline documentado en este mismo plan.
