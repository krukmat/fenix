# Task 1.6 — Authentication Middleware: Plan de Implementación

**Proyecto:** FenixCRM
**Task:** 1.6 — Authentication Middleware
**Estado:** ✅ Completado
**Fuente de verdad:** `tasks_1.6.md` + `docs/architecture.md` (Auth JWT MVP)
**Fecha inicio:** 2026-02-09

---

## Objetivo

Habilitar autenticación JWT end-to-end para proteger `/api/v1/*` y preparar la base para auditoría/políticas. Los claims JWT pasan a ser la fuente primaria de `workspace_id` y `user_id` en el contexto de cada request.

---

## Decisiones de diseño (pre-código)

### 1. Login multi-tenant
- `email` en `user_account` es `UNIQUE` global (no por workspace).
- Login requiere solo `email + password`. No hay `email + workspace` en el request.
- El `workspace_id` se incluye en el JWT claim al momento del login/register.
- Consecuencia: un email solo puede pertenecer a un workspace en el MVP.

### 2. Migración 009
- `password_hash TEXT` ya existe en `001_init_schema.up.sql:35` — no se necesita agregar la columna.
- `idx_user_account_email` ya existe en `001_init_schema.up.sql:46`.
- Lo que sí falta: índice compuesto `(workspace_id, email)` para lookup eficiente en login multi-tenant.
- Migración `009_auth_index` agrega ese índice y nada más.

### 3. Contexto (`ctxkeys`)
- Se agrega `UserID Key = "user_id"` al paquete `internal/api/ctxkeys/ctxkeys.go` existente.
- El `AuthMiddleware` inyecta `workspace_id` + `user_id` usando los mismos keys tipados.
- `WorkspaceMiddleware` actual se **elimina** del router de producción; queda solo para helpers de test.

### 4. Compatibilidad backward (tests existentes)
- Los tests en `account_test.go`, `contact_test.go`, etc. usan `contextWithWorkspaceID()` directamente.
- Se migran a un helper `tokenForTest(t, db, wsID, userID)` que genera un JWT real de test.
- Esto valida el flujo completo sin mockear el middleware.

### 5. Estructura de paquetes
- `pkg/auth/` — bcrypt + JWT puro, sin dependencias de dominio. Reutilizable.
- `internal/domain/auth/` — lógica de negocio: register (crea workspace + user), login (lookup + verify).
- `internal/api/middleware/` — Bearer middleware que llama a `pkg/auth.ParseJWT`.
- `internal/api/handlers/auth.go` — handlers HTTP register + login.

### 6. Seguridad
- `JWT_SECRET` leído de variable de entorno. Si no está seteada, el servidor falla al arrancar (no silencioso).
- Mensajes de error genéricos en login: siempre `"invalid credentials"` (no revelar si el email existe).
- Nunca loguear password ni token en texto plano.
- bcrypt cost = 12 (balance seguridad/performance para MVP).

### 7. Rutas públicas vs protegidas (post-1.6)
```
/health                    → público (sin auth)
/api/v1/auth/register      → público (sin auth)
/api/v1/auth/login         → público (sin auth)
/api/v1/*                  → AuthMiddleware (JWT Bearer obligatorio)
```

---

## Lista de tareas (TDD — test primero)

| # | Tarea | Estado | Archivos |
|---|-------|--------|---------|
| 1.6.1 | pkg/auth: tests bcrypt (hash/verify) | ✅ | `pkg/auth/auth_test.go` |
| 1.6.2 | pkg/auth: implementación bcrypt | ✅ | `pkg/auth/auth.go` |
| 1.6.3 | pkg/auth: tests JWT (generate/parse/expiry) | ✅ | `pkg/auth/auth_test.go` |
| 1.6.4 | pkg/auth: implementación JWT | ✅ | `pkg/auth/auth.go` — cobertura 86.5% (20 tests) |
| 1.6.5 | Migración 009: índice compuesto (workspace_id, email) | ✅ | `internal/infra/sqlite/migrations/009_auth_index.up.sql`, `.down.sql` |
| 1.6.6 | ctxkeys: agregar UserID key | ✅ | `internal/api/ctxkeys/ctxkeys.go` |
| 1.6.7 | domain/auth: tests de AuthService (register, login) | ✅ | `internal/domain/auth/service_test.go` — 10 tests |
| 1.6.8 | domain/auth: AuthService implementación + SQL queries | ✅ | `internal/domain/auth/service.go` |
| 1.6.9 | middleware/auth: tests del Bearer middleware | ✅ | `internal/api/middleware/auth_test.go` — 11 tests, 100% coverage |
| 1.6.10 | middleware/auth: implementación Bearer middleware | ✅ | `internal/api/middleware/auth.go` |
| 1.6.11 | handlers/auth: tests handlers register + login | ✅ | `internal/api/handlers/auth_test.go` — 17 tests |
| 1.6.12 | handlers/auth: implementación handlers | ✅ | `internal/api/handlers/auth.go` |
| 1.6.13 | routes.go: restructurar rutas públicas vs protegidas | ✅ | `internal/api/routes.go` — /auth/* public, /api/v1/* JWT-protected |
| 1.6.14 | Adaptar tests existentes al nuevo AuthMiddleware | ✅ | TestMain JWT_SECRET en pkg/auth, middleware, domain/auth |
| 1.6.15 | go test ./... en verde + make complexity pasa | ✅ | Todos los paquetes OK, complexity avg 2.9 |

---

## Archivos afectados (inventario completo)

### Nuevos
| Archivo | Descripción |
|---------|-------------|
| `pkg/auth/auth.go` | bcrypt (HashPassword, VerifyPassword) + JWT (GenerateJWT, ParseJWT, Claims) |
| `pkg/auth/auth_test.go` | Tests unitarios de bcrypt y JWT |
| `internal/infra/sqlite/migrations/009_auth_index.up.sql` | Índice compuesto `(workspace_id, email)` en `user_account` |
| `internal/infra/sqlite/migrations/009_auth_index.down.sql` | Rollback del índice |
| `internal/domain/auth/service.go` | AuthService: Register (workspace+user) y Login (lookup+verify+JWT) |
| `internal/domain/auth/service_test.go` | Tests de integración de AuthService (con DB real) |
| `internal/api/middleware/auth.go` | Bearer JWT middleware: extrae y valida token, inyecta claims en ctx |
| `internal/api/middleware/auth_test.go` | Tests del middleware (token válido, inválido, expirado, ausente) |
| `internal/api/handlers/auth.go` | Handlers HTTP: POST /auth/register y POST /auth/login |
| `internal/api/handlers/auth_test.go` | Tests de integración de los handlers auth |

### Modificados
| Archivo | Cambio | Líneas afectadas |
|---------|--------|-----------------|
| `internal/api/ctxkeys/ctxkeys.go` | Agregar `UserID Key = "user_id"` | ~línea 14 |
| `internal/api/routes.go` | Reemplazar `WorkspaceMiddleware` por `AuthMiddleware`; separar rutas públicas/protegidas; registrar `/auth` | Todo el bloque `/api/v1` |
| `internal/api/handlers/account_test.go` | Migrar `contextWithWorkspaceID` → `tokenForTest` helper | helper y todos los tests |
| `internal/api/handlers/contact_test.go` | Idem | ídem |
| `internal/api/handlers/lead_test.go` | Idem | ídem |
| `internal/api/handlers/deal_test.go` | Idem | ídem |
| `go.mod` / `go.sum` | Agregar `golang.org/x/crypto` y `github.com/golang-jwt/jwt/v5` | — |

### Eliminados / deprecados
| Archivo | Cambio |
|---------|--------|
| `WorkspaceMiddleware()` en `internal/api/routes.go` | Reemplazado por `AuthMiddleware`. La función se mueve a helper de tests si es necesario. |

---

## Interfaces clave

### pkg/auth — Claims y funciones públicas
```go
// Task 1.6: JWT claims mínimas según architecture.md Section 8
type Claims struct {
    UserID      string `json:"user_id"`
    WorkspaceID string `json:"workspace_id"`
    jwt.RegisteredClaims
}

func HashPassword(password string) (string, error)
func VerifyPassword(hash, password string) bool
func GenerateJWT(userID, workspaceID string) (string, error)  // usa JWT_SECRET + JWT_EXPIRY env
func ParseJWT(token string) (*Claims, error)
```

### internal/domain/auth — AuthService
```go
// Task 1.6: Business logic para register y login
type RegisterInput struct {
    Email         string
    Password      string
    DisplayName   string
    WorkspaceName string
}

type LoginInput struct {
    Email    string
    Password string
}

type AuthResult struct {
    Token       string
    UserID      string
    WorkspaceID string
}

type AuthService interface {
    Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
    Login(ctx context.Context, input LoginInput) (*AuthResult, error)
}
```

### internal/api/middleware — AuthMiddleware
```go
// Task 1.6: Bearer JWT middleware — inyecta user_id y workspace_id en context
func AuthMiddleware(next http.Handler) http.Handler
// Inyecta: ctxkeys.UserID y ctxkeys.WorkspaceID
// Responde 401 en: token ausente, inválido, o expirado
```

---

## Criterio de aceptación (DoD)

- [ ] `POST /api/v1/auth/register` crea workspace + user, devuelve JWT válido
- [ ] `POST /api/v1/auth/login` verifica password, devuelve JWT válido
- [ ] `POST /api/v1/auth/login` con password incorrecta devuelve 401 con `"invalid credentials"`
- [ ] `GET /api/v1/accounts` sin token devuelve 401
- [ ] `GET /api/v1/accounts` con JWT válido devuelve 200 (workspace isolation vía claims)
- [ ] `GET /health` sin token devuelve 200 (ruta pública)
- [ ] `go test ./...` en verde (0 tests rotos)
- [ ] `make complexity` pasa (todas las funciones ≤ 7)
- [ ] No se loguea password ni token en texto plano

---

## Evidencia post-implementación

> *Completar al cerrar la task*

- Resultado `go test ./...`:
- Resultado `make complexity`:
- Endpoints verificados manualmente:
