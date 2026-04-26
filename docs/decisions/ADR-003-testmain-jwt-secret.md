---
id: ADR-003
title: "TestMain + os.Setenv para JWT_SECRET en tests (no t.Setenv)"
date: 2026-01-25
status: accepted
deciders: [matias]
tags: [adr, testing, auth]
related_tasks: [task_1.6]
related_frs: [FR-060]
---

# ADR-003 — TestMain + os.Setenv para JWT_SECRET en tests

## Status

`accepted`

## Context

`pkg/auth.getJWTSecret()` hace panic si la variable de entorno `JWT_SECRET` no está
seteada. Todos los packages que usan `GenerateJWT` o `ParseJWT` necesitan esta variable
disponible durante los tests.

El approach inicial usaba `t.Setenv("JWT_SECRET", "...")` dentro de cada test. Esto
causaba el panic de Go:

```
testing: test using t.Setenv can not use t.Parallel
```

`t.Setenv` y `t.Parallel` son mutuamente excluyentes porque `t.Setenv` hace rollback
de la variable al terminar el test, lo que es incompatible con goroutines paralelas que
podrían leer la variable después del rollback.

## Decision

Usar `TestMain` con `os.Setenv` para setear `JWT_SECRET` a nivel de package, una sola
vez antes de que corran todos los tests del package:

```go
func TestMain(m *testing.M) {
    os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!")
    os.Exit(m.Run())
}
```

Packages afectados: `pkg/auth`, `internal/domain/auth`, `internal/api/middleware`,
`internal/api/handlers`.

## Rationale

- `os.Setenv` persiste para todo el proceso — compatible con `t.Parallel()`
- `TestMain` es el hook canónico de Go para setup/teardown a nivel de package
- Una sola línea por package, sin duplicación en cada test
- El valor de test (`"test-secret-key-32-chars-min!!!"`) cumple el mínimo de 32 chars

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| `t.Setenv` en cada test | Incompatible con `t.Parallel()` — Go panic |
| `init()` en archivo `_test.go` | `init()` corre antes que TestMain, orden no garantizado |
| Mockear `getJWTSecret()` | Requiere inyección de dependencia innecesaria en production code |
| Deshabilitar `t.Parallel()` | Aumenta tiempo de tests significativamente |

## Consequences

**Positive:**
- Tests paralelos sin panics
- Setup mínimo por package (una función TestMain)
- Patrón estándar de Go, fácil de entender para nuevos colaboradores

**Negative / tradeoffs:**
- `JWT_SECRET` queda seteado en el entorno del proceso para toda la suite del package.
  Es un test secret sin valor en producción — riesgo negligible.

## References

- Go testing docs: `TestMain` — https://pkg.go.dev/testing#hdr-Main
- Go issue sobre `t.Setenv` + `t.Parallel`: https://github.com/golang/go/issues/52817
