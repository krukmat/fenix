# Refactors con Patrones de Diseño

Este directorio contiene evidencia de refactors estructurales guiados por patrones de diseño.

## Objetivo

Dar trazabilidad a cambios de arquitectura/refactor que no siempre quedan claros en un diff de código:

- qué patrón se aplicó,
- por qué se aplicó,
- qué riesgo reduce,
- y cómo se valida con tests/métricas.

## Uso

1. Copiar `template.md`.
2. Crear un archivo nuevo con prefijo ordenable, por ejemplo:
   - `2026-02-20_strategy-evaluator.md`
3. Completar todas las secciones obligatorias.

## Gate asociado

El gate `make pattern-opportunities-gate` (alias de `pattern-refactor-gate`) valida en modo MVP:

- presencia de `docs/refactors/template.md`,
- presencia de al menos una evidencia (`*.md` distinta de `template.md`),
- secciones obligatorias por evidencia,
- oportunidades estructurales de refactor por duplicación:
  - **Go** con `dupl` (vía `golangci-lint --enable-only=dupl`)
  - **TypeScript (mobile + bff)** con `jscpd`

### Umbrales actuales

- **Go / dupl**: `threshold=120` (configurado en `.golangci.yml`)
- **TS / jscpd**: `PATTERN_GATE_TS_DUP_THRESHOLD=2` (%)

En fase inicial se ejecuta en modo `warn` en CI:

```bash
make pattern-opportunities-gate PATTERN_GATE_MODE=warn
```

Cuando el equipo lo decida, puede endurecerse a `strict`.

## Evolución recomendada (WARN → FAIL)

1. **Semana 1-2 (warn):** levantar baseline y revisar falsos positivos.
2. **Semana 3 (warn):** ajustar exclusiones/umbrales por módulo.
3. **Semana 4 (strict):** activar bloqueo en CI para nuevos PRs.
