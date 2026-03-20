# TODO.RESULT — Hardening de endpoints públicos de autenticación

## Estado: COMPLETADO

Tarea ejecutada y commiteada en `agent-spec-transition` (commit `6eb5158`).

---

## Verificación por criterio de aceptación

| Criterio | Estado | Evidencia |
|---|---|---|
| Requests desde origen no permitido no reciben headers CORS | OK | `cors.go` — solo setea headers si `origin == allowedOrigin` |
| Requests desde origen permitido reciben headers CORS correctos | OK | `cors.go` — setea `Access-Control-Allow-Origin`, `Methods`, `Headers`, `Credentials`, `Vary` |
| `POST /auth/register` falla con `400` si password < 12 chars | OK | `auth.go` — `minPasswordLen = 12`, retorna `400` |
| `POST /auth/login` responde `429` al exceder límite | OK | `ratelimit.go` — 429; `routes.go` — login 5/min |
| `POST /auth/register` responde `429` al exceder límite | OK | `ratelimit.go` — 429; `routes.go` — register 3/hora |
| Tests cubren origen permitido, bloqueado, password débil y throttling | OK | 23 tests nuevos — todos PASS |
| Comportamiento actual de login/registro no roto | OK | `go test ./internal/api/... ./internal/server/...` — todos PASS |

---

## Archivos creados o modificados

| Archivo | Cambio |
|---|---|
| `internal/api/middleware/cors.go` | NUEVO — `CORSMiddleware(allowedOrigin)`, strict allowlist, preflight 204 |
| `internal/api/middleware/cors_test.go` | NUEVO — 8 tests unitarios |
| `internal/api/middleware/ratelimit.go` | NUEVO — `RateLimitMiddleware(limit, window)`, in-memory per-IP, 429 |
| `internal/api/middleware/ratelimit_test.go` | NUEVO — 5 tests unitarios |
| `internal/api/handlers/auth.go` | MODIFICADO — `minPasswordLen = 12` en `validateRegisterRequest` |
| `internal/api/handlers/auth_test.go` | MODIFICADO — 3 tests nuevos de password |
| `internal/infra/config/config.go` | MODIFICADO — campo `BFFOrigin`, env var `BFF_ORIGIN`, default `http://localhost:3000` |
| `internal/api/routes.go` | MODIFICADO — CORS global, rate limiters por ruta, `newRouterWithConfig` para testabilidad |
| `internal/api/routes_test.go` | MODIFICADO — 7 tests de integración nuevos (CORS, password, rate limit) |

---

## Resultado de tests

```
ok  github.com/matiasleandrokruk/fenix/internal/api/middleware  0.674s
ok  github.com/matiasleandrokruk/fenix/internal/api/handlers    1.786s
ok  github.com/matiasleandrokruk/fenix/internal/api             1.665s
ok  github.com/matiasleandrokruk/fenix/internal/server          0.344s
```

---

## Decisiones de diseño respetadas

- CORS configurable por env var (`BFF_ORIGIN`), no hardcodeado.
- Rate limiting en memoria — válido para MVP/single instance; no cluster-safe (documentado en código).
- Sin dependencia externa para CORS — implementación stdlib mínima.
- `minPasswordLen` como constante nombrada.
