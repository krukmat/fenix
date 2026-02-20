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

El gate `make pattern-refactor-gate` valida en modo MVP:

- presencia de `docs/refactors/template.md`,
- presencia de al menos una evidencia (`*.md` distinta de `template.md`),
- secciones obligatorias por evidencia,
- señales de patrones (Strategy/Factory/Decorator) en código Go,
- señales de duplicación estructural en código Mobile TypeScript.

### Checks Go

| Check | Qué detecta |
|-------|-------------|
| `strategy` | `type XStrategy interface` |
| `factory` | `type XFactory interface` o `func NewXFactory(` |
| `decorator` | `type XDecorator struct` o `func NewXDecorator(` |
| `type_switches` | `switch x.(type)` sin contraparte Strategy |

### Checks Mobile TypeScript

| Check | Umbral | Patrón a aplicar |
|-------|--------|-----------------|
| `useThemeColors/useColors` inline defs | ≥ 3 | Extract Custom Hook → `src/hooks/useThemeColors.ts` |
| `formatLatency/formatCost/formatTokens` defs | ≥ 2 | Utility Extract → `src/utils/format.ts` |
| `getStatusColor/getPriorityColor/getStatusLabel` defs | ≥ 3 | Strategy Lookup → `src/utils/statusColors.ts` |
| `useInfiniteQuery` calls sin `createInfiniteListHook` | ≥ 4 | Factory Method → `useCRM.ts` |

En fase inicial se ejecuta en modo `warn` en CI:

```bash
make pattern-refactor-gate PATTERN_GATE_MODE=warn
```

Cuando el equipo lo decida, puede endurecerse a `strict`.
