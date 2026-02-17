# Task 4.1 — BFF Setup: Express.js Gateway

**Status**: ✅ Done
**Phase**: 4 — Mobile App + BFF + Polish
**Duration**: 2.5 días
**Depends on**: Phase 1-3 ✅ (Go backend corriendo en :8080)
**Resolves**: FR-301 (BFF Gateway)

---

## Objetivo

Construir el **BFF (Backend-for-Frontend)** en Express.js 5 + TypeScript dentro del directorio `/bff/` del monorepo. El BFF actúa como gateway entre la app mobile (React Native) y el Go backend. Es un **proxy thin y stateless**: no tiene lógica de negocio, no accede a SQLite directamente, y no almacena estado.

Responsabilidades:
1. **Proxy transparente** — `GET/POST/PUT/DELETE /bff/api/v1/*` → Go `:8080/api/v1/*`
2. **Auth relay** — `POST /bff/auth/login`, `/bff/auth/register` → Go `/auth/*`
3. **Aggregated routes** — combina múltiples llamadas Go en una respuesta mobile-optimizada
4. **SSE proxy** — relay del stream del Copilot desde Go hacia la app mobile
5. **Health check** — `/bff/health` con ping al Go backend

---

## Scope

### 1. Inicialización del proyecto BFF
- `mkdir bff && cd bff && npm init -y`
- Deps producción: `express@^5`, `http-proxy-middleware@^3`, `axios@^1.7`, `helmet@^8`, `cors@^2.8`, `dotenv@^16`
- Deps dev: `typescript@^5.6`, `@types/express@^5`, `@types/cors`, `ts-node`, `ts-jest`, `jest`, `@types/jest`, `supertest@^7`, `@types/supertest`, `nodemon`
- `tsconfig.json` con `strict: true`, `target: ES2022`, `module: commonjs`, `outDir: dist`
- `jest.config.ts` con `ts-jest`, cobertura ≥80%
- `.env.example`: `BFF_PORT=3000`, `BACKEND_URL=http://localhost:8080`
- `package.json` scripts: `build`, `dev`, `test`, `test:coverage`, `start`

### 2. Configuración central
- `src/config.ts` — lee env vars, valida que `BACKEND_URL` exista, exporta objeto tipado
- `src/services/goClient.ts` — instancia Axios con `baseURL = BACKEND_URL`, timeouts

### 3. Middleware stack
- `src/middleware/authRelay.ts` — extrae `Authorization: Bearer <token>` del request y lo propaga a las llamadas a Go. Si es request a ruta protegida y no hay token → 401 inmediato
- `src/middleware/mobileHeaders.ts` — reenvía `X-Device-Id`, `X-App-Version` como headers a Go
- `src/middleware/errorHandler.ts` — captura errores (de Go o internos), normaliza al envelope `{error: {code: string, message: string, details?: unknown}}`

### 4. Routes: Auth relay
- `src/routes/auth.ts`
  - `POST /bff/auth/login` → relay a `POST /auth/login`
  - `POST /bff/auth/register` → relay a `POST /auth/register`
  - Retorna exactamente la respuesta de Go (token, user, etc.)

### 5. Routes: Proxy transparente
- `src/routes/proxy.ts`
  - `ALL /bff/api/v1/*` → `createProxyMiddleware({ target: BACKEND_URL, changeOrigin: true, pathRewrite: {'^/bff': ''} })`
  - Headers de auth y mobile propagados automáticamente por middlewares anteriores
  - Reenvía status codes de Go sin modificar

### 6. Routes: Aggregated (mobile-optimized)
- `src/routes/aggregated.ts`
  - `GET /bff/accounts/:id/full` — `Promise.all([getAccount, getContacts, getDeals, getTimeline])` → merge en `{account, contacts, deals, timeline}`
  - `GET /bff/deals/:id/full` — `Promise.all([getDeal, getAccount, getContact, getActivities])` → merge en `{deal, account, contact, activities}`
  - `GET /bff/cases/:id/full` — `Promise.all([getCase, getAccount, getContact, getActivities, getHandoff?])` → merge en `{case, account, contact, activities, handoff}`
  - Si alguna sub-llamada falla: incluir `null` en ese campo + warning en respuesta

### 7. Routes: SSE Proxy (Copilot)
- `src/routes/copilot.ts`
  - `POST /bff/copilot/chat` → abre request SSE a `POST /api/v1/copilot/chat` en Go
  - `res.setHeader('Content-Type', 'text/event-stream')`
  - `res.setHeader('Cache-Control', 'no-cache')`
  - `res.flushHeaders()` para iniciar stream inmediatamente
  - Relay de cada chunk `data:` conforme llega desde Go
  - Cierra cuando Go cierra o hay error

### 8. Health Check
- `src/routes/health.ts`
  - `GET /bff/health`
  - Ping `GET /health` en Go con timeout 2s
  - Si Go responde: `{status:"ok", backend:"reachable", latency_ms: N}`
  - Si Go no responde: HTTP 503, `{status:"degraded", backend:"unreachable"}`

### 9. App factory + Server
- `src/app.ts` — crea y configura Express app (sin `listen`), registra middlewares y rutas
- `src/server.ts` — importa `app`, llama `app.listen(BFF_PORT)`

### 10. Docker
- `deploy/Dockerfile.bff` — multi-stage:
  - Stage `builder`: `node:22-alpine`, `npm ci`, `npm run build`
  - Stage `runtime`: `node:22-alpine`, copia `dist/` + `node_modules/`, `EXPOSE 3000`, `CMD ["node", "dist/server.js"]`

### 11. Tests (TDD — Supertest)
Los tests se escriben **ANTES** de la implementación correspondiente.

---

## Sub-tareas Desglosadas

| # | Sub-tarea | Estado | Notas |
|---|-----------|--------|-------|
| 4.1.1 | Init `/bff/` — npm + TypeScript + deps | ❌ | `npm init -y`, `tsconfig.json` strict, jest.config.ts |
| 4.1.2 | `config.ts` + `services/goClient.ts` | ✅ | Axios con BACKEND_URL, timeouts |
| 4.1.3 | TEST: health endpoint | ✅ | TDD — test primero. 200 ok + 503 degraded |
| 4.1.4 | IMPL: `routes/health.ts` | ✅ | Ping Go /health, latency_ms |
| 4.1.5 | TEST: auth relay | ✅ | TDD — test primero. login relay |
| 4.1.6 | IMPL: `middleware/authRelay.ts` + `routes/auth.ts` | ✅ | JWT relay a Go /auth/* |
| 4.1.7 | TEST: proxy pass-through | ✅ | TDD — test primero. 200 + 401 pass-through |
| 4.1.8 | IMPL: `routes/proxy.ts` | ✅ | http-proxy-middleware v3 |
| 4.1.9 | TEST: aggregated endpoints | ✅ | TDD — test primero. /accounts/:id/full merge |
| 4.1.10 | IMPL: `routes/aggregated.ts` | ✅ | Promise.allSettled paralelo, null en fallo parcial |
| 4.1.11 | TEST: SSE proxy | ✅ | TDD — test primero. PassThrough stream mock |
| 4.1.12 | IMPL: `routes/copilot.ts` | ✅ | res.flushHeaders(), relay chunks |
| 4.1.13 | IMPL: `middleware/mobileHeaders.ts` + `errorHandler.ts` | ✅ | Completar middleware stack |
| 4.1.14 | `src/app.ts` + `src/server.ts` | ✅ | app factory separada de listen |
| 4.1.15 | `deploy/Dockerfile.bff` multi-stage | ✅ | Build TS → node:alpine |
| 4.1.16 | Quality gates: `npm run build` + `npm test` pasan | ✅ | TypeScript strict + coverage ≥80% statements/lines/functions, ≥75% branches |
| 4.1.17 | Marcar Task 4.1 ✅ en `docs/implementation-plan.md` | ✅ | Actualizar status en plan |

---

## Archivos a Crear

### Nuevos en `/bff/`:
- `bff/package.json` — npm config + scripts
- `bff/tsconfig.json` — TypeScript strict + ES2022
- `bff/jest.config.ts` — ts-jest, coverage threshold 80%
- `bff/.env.example` — BFF_PORT, BACKEND_URL
- `bff/src/app.ts` — Express app factory
- `bff/src/server.ts` — entry point con app.listen
- `bff/src/config.ts` — env vars tipados
- `bff/src/middleware/authRelay.ts` — JWT header relay
- `bff/src/middleware/mobileHeaders.ts` — X-Device-Id, X-App-Version
- `bff/src/middleware/errorHandler.ts` — error envelope
- `bff/src/routes/health.ts` — GET /bff/health
- `bff/src/routes/auth.ts` — POST /bff/auth/login, /register
- `bff/src/routes/proxy.ts` — ALL /bff/api/v1/* pass-through
- `bff/src/routes/aggregated.ts` — /full aggregated routes
- `bff/src/routes/copilot.ts` — SSE proxy
- `bff/src/services/goClient.ts` — Axios preconfigurado
- `bff/tests/health.test.ts` — tests Supertest health
- `bff/tests/auth.test.ts` — tests Supertest auth relay
- `bff/tests/proxy.test.ts` — tests Supertest proxy pass-through
- `bff/tests/aggregated.test.ts` — tests Supertest aggregated
- `bff/tests/copilot.test.ts` — tests Supertest SSE relay

### Nuevos en raíz/deploy/:
- `deploy/Dockerfile.bff` — multi-stage Docker build

---

## Dependencias Externas

- **Go backend** — debe estar corriendo en `BACKEND_URL` (`:8080`) para tests de integración y health check
- **`http-proxy-middleware` v3** — compatible con Express 5, usa `async` handlers nativos
- **Supertest** — HTTP assertions sobre el app Express sin abrir puerto real

---

## Resolutos (FR/NFR)

- **FR-301** — BFF Gateway: proxy transparente, auth relay, agregación, SSE proxy, health check

---

## Quality Gates (BFF-specific)

```bash
# Desde /bff/
npm run build          # TypeScript strict — cero errores de compilación
npm test               # Jest + Supertest — todos los tests pasan
npm run test:coverage  # Coverage ≥80%
```

> El BFF **no** corre los quality gates de Go (`make complexity`, `make trace-check`, `make lint` Go).
> Tiene sus propios gates: TypeScript strict mode + Jest coverage.

---

## Acceptance Criteria

- [ ] `bff/` directorio existe con estructura completa
- [ ] `npm run build` — TypeScript compila sin errores (strict mode)
- [ ] `npm test` — 7 tests Supertest todos en verde
- [ ] Coverage ≥ 80% (Jest)
- [ ] `GET /bff/health` retorna 200 + `{status:"ok"}` cuando Go está up
- [ ] `GET /bff/health` retorna 503 + `{status:"degraded"}` cuando Go está down
- [ ] `POST /bff/auth/login` hace relay correcto a Go y retorna JWT
- [ ] `GET /bff/api/v1/accounts` (con token) es transparente a Go
- [ ] `GET /bff/api/v1/accounts` (sin token) retorna 401 tal como Go
- [ ] `GET /bff/accounts/:id/full` retorna objeto agregado `{account, contacts, deals, timeline}`
- [ ] `POST /bff/copilot/chat` hace SSE relay (chunks llegan al cliente)
- [ ] `deploy/Dockerfile.bff` construye sin errores (`docker build`)
- [ ] `docs/implementation-plan.md` Task 4.1 marcada como ✅

---

## Files Affected (post-implementación)

| Archivo | Acción | Líneas estimadas |
|---------|--------|-----------------|
| `bff/package.json` | CREATE | ~40 |
| `bff/tsconfig.json` | CREATE | ~25 |
| `bff/jest.config.ts` | CREATE | ~20 |
| `bff/.env.example` | CREATE | ~5 |
| `bff/src/app.ts` | CREATE | ~40 |
| `bff/src/server.ts` | CREATE | ~10 |
| `bff/src/config.ts` | CREATE | ~20 |
| `bff/src/middleware/authRelay.ts` | CREATE | ~30 |
| `bff/src/middleware/mobileHeaders.ts` | CREATE | ~20 |
| `bff/src/middleware/errorHandler.ts` | CREATE | ~25 |
| `bff/src/routes/health.ts` | CREATE | ~35 |
| `bff/src/routes/auth.ts` | CREATE | ~35 |
| `bff/src/routes/proxy.ts` | CREATE | ~20 |
| `bff/src/routes/aggregated.ts` | CREATE | ~80 |
| `bff/src/routes/copilot.ts` | CREATE | ~50 |
| `bff/src/services/goClient.ts` | CREATE | ~25 |
| `bff/tests/health.test.ts` | CREATE | ~60 |
| `bff/tests/auth.test.ts` | CREATE | ~50 |
| `bff/tests/proxy.test.ts` | CREATE | ~60 |
| `bff/tests/aggregated.test.ts` | CREATE | ~80 |
| `bff/tests/copilot.test.ts` | CREATE | ~70 |
| `deploy/Dockerfile.bff` | CREATE | ~20 |
| `docs/implementation-plan.md` | EDIT | Task 4.1 status → ✅ |

---

## Source of Truth

1. `docs/implementation-plan.md` — Week 11, Task 4.1
2. `docs/architecture.md` — Section 3.1 (BFF Responsibilities), Section 10 (Deployment)
3. `agentic_crm_requirements_agent_ready.md` — FR-301

---

## Notas

- **`app.ts` vs `server.ts`**: Separación crítica para Supertest — tests importan `app` directamente sin `listen()`
- **Express 5**: Maneja `async` handlers nativamente — un `throw` en route async ya propaga al `errorHandler` sin `next(err)` manual
- **SSE + proxy**: Usar `res.flushHeaders()` inmediatamente después de setear headers SSE. Sin esto, Node.js puede bufferear chunks y el cliente mobile no los recibe en tiempo real
- **Promise.all en aggregated**: Las sub-llamadas a Go se hacen en paralelo. Si una falla, se incluye `null` en ese campo (no se aborta toda la respuesta) para máxima resiliencia en mobile
- **Dockerfile.bff ubicación**: Se guarda en `deploy/Dockerfile.bff` (no en `bff/`) para consistencia con convenciones futuras de Docker Compose
