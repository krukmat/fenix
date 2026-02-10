# Handoff Técnico — Task 1.6 (Authentication Middleware)

**Proyecto:** FenixCRM  
**Fuente de verdad:** `docs/implementation-plan.md` (Task 1.6) + `docs/architecture.md` (Auth JWT MVP)  
**Task:** **1.6 — Authentication Middleware**  
**Objetivo:** habilitar autenticación JWT end-to-end para proteger `/api/v1/*` y preparar base para auditoría/políticas.

---

## 1) Contexto para agente developer

- Task 1.5 quedó cerrada (CRM core + timeline + rutas + tests en verde).
- El router hoy usa `WorkspaceMiddleware` con header `X-Workspace-ID` para multitenancy.
- En Task 1.6 ese mecanismo debe converger hacia **claims JWT** como fuente primaria de `workspace_id` y `user_id`.
- Alcance MVP: **built-in auth con bcrypt + JWT**, dejando hook para OIDC a futuro.

---

## 2) Alcance exacto de Task 1.6

Implementar:

1. Password hashing/verificación (`bcrypt`)
2. Emisión y validación de JWT
3. Middleware auth para rutas API
4. Endpoints de autenticación:
   - `POST /api/v1/auth/register`
   - `POST /api/v1/auth/login`
5. Tests unit + integración de auth

No incluye en esta task:

- RBAC/ABAC completo (Task 3.x)
- OIDC/SSO real (futuro)
- Refresh token avanzado (opcional fuera de 1.6)

---

## 3) Entregables obligatorios

## A) Migración de esquema auth

Crear migración (si no existe):

- `009_auth.up.sql` (o siguiente número disponible)

Cambios mínimos:

- agregar `password_hash` a `user_account` (nullable al inicio)
- índices útiles para lookup por `workspace_id + email`

Rollback (`*.down.sql`) coherente.

---

## B) Paquete de autenticación

Crear `pkg/auth/` con:

- `HashPassword(password string) (string, error)`
- `VerifyPassword(hash, password string) bool`
- `GenerateJWT(userID, workspaceID string) (string, error)`
- `ParseJWT(token string) (*Claims, error)`

Requisitos:

- bcrypt (cost razonable)
- expiración configurable (`JWT_EXPIRY`)
- secreto desde env (`JWT_SECRET`)
- claims mínimas: `sub`/`user_id`, `workspace_id`, `exp`, `iat`

---

## C) Middleware de autenticación

Archivo sugerido: `internal/api/middleware/auth.go` (o equivalente actual en `internal/api`).

Comportamiento:

- leer header `Authorization: Bearer <token>`
- validar token
- extraer claims
- inyectar en `context.Context`:
  - `user_id`
  - `workspace_id`
- responder `401` en token ausente/inválido/expirado

Compatibilidad:

- `/health` debe permanecer público
- `/api/v1/auth/*` público (register/login)
- resto de `/api/v1/*` protegido

---

## D) Handlers de autenticación

Crear handlers:

### `POST /api/v1/auth/register`

Request mínimo:

```json
{ "email": "user@acme.com", "password": "secret", "displayName": "User", "workspaceName": "Acme" }
```

Acciones:

- crear workspace (si aplica en MVP)
- crear user_account con `password_hash`
- devolver JWT + datos básicos de sesión

### `POST /api/v1/auth/login`

Request:

```json
{ "email": "user@acme.com", "password": "secret" }
```

Acciones:

- buscar usuario por email + workspace (según modelo actual)
- verificar password
- devolver JWT y claims de sesión

---

## E) Integración en router

Actualizar `internal/api/routes.go`:

- mantener `/health` público
- registrar `/api/v1/auth/login` y `/api/v1/auth/register` sin auth middleware
- aplicar auth middleware al resto de rutas API
- asegurar que `workspace_id` se toma del JWT claims (no sólo de header)

Nota práctica:

- durante transición, se puede mantener fallback a `X-Workspace-ID` sólo en dev/tests, pero objetivo de 1.6 es claims JWT como principal.

---

## 4) Tests requeridos (DoD técnico)

## Unit tests

- hash + verify password
- generate + parse JWT
- expiración JWT

## Integration/API tests

- register exitoso crea usuario y retorna token
- login exitoso retorna token válido
- login con password inválida retorna 401
- endpoint protegido sin token retorna 401
- endpoint protegido con token válido retorna 2xx
- aislamiento tenant via `workspace_id` claim

Comandos:

```bash
go test ./internal/api ./internal/api/handlers ./internal/domain/... ./pkg/auth
go test ./...
```

---

## 5) Riesgos y decisiones para developer

1. **Compatibilidad con middleware actual** (`X-Workspace-ID`):
   - Decidir estrategia de migración sin romper tests existentes.
2. **Modelo de login multi-tenant**:
   - definir si login requiere sólo email (global unique) o email+workspace.
3. **Seguridad mínima**:
   - no loguear password/token en texto plano.
4. **Rendimiento / UX**:
   - mensajes de error consistentes (`invalid credentials`, `unauthorized`).

---

## 6) Checklist operativo (ejecución sugerida)

- [ ] Crear migración para `password_hash` en `user_account`
- [ ] Implementar `pkg/auth` (bcrypt + JWT)
- [ ] Implementar middleware auth (Bearer)
- [ ] Implementar handlers `register` y `login`
- [ ] Integrar rutas públicas/protegidas en router
- [ ] Adaptar contexto para `workspace_id`/`user_id` desde claims
- [ ] Escribir tests unit e integración auth
- [ ] Ejecutar `go test ./...` en verde
- [ ] Actualizar `docs/implementation-plan.md` con estado de Task 1.6

---

## 7) Criterio de aceptación final

Task 1.6 se considera cerrada cuando:

1. login/register funcionan,
2. middleware protege rutas correctamente,
3. claims JWT alimentan contexto de tenant/actor,
4. tests de auth y suite global pasan,
5. documentación queda actualizada con evidencia.
