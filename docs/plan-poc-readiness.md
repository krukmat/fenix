# Plan: Preparar repo para POC deployment low-cost

## Contexto

El plan de deployment (`docs/deployment-plan-digitalocean.md`) identifica que el repo está acoplado a Ollama como único LLM provider. Para una POC low-cost con Gradient/Groq/Together.ai para chat y Ollama local solo para embeddings, hay que separar providers. Además faltan `/readyz`, reverse proxy (Caddy), y un docker-compose de producción.

**Separación limpia confirmada** — ningún servicio usa Chat + Embed del mismo provider:

| Servicio | Usa Chat | Usa Embed |
|---|---|---|
| EmbedderService | - | ✓ |
| SearchService | - | ✓ |
| ChatService | ✓ | - |
| ActionService | ✓ | - |
| ProspectingAgent | ✓ | - |
| KBAgent | ✓ | - |
| SupportAgent | - | - (no toma LLMProvider) |
| InsightsAgent | - | - (no toma LLMProvider) |

---

## Tareas para agentes de código

### Tarea 1: Config split provider
**Scope**: Solo `internal/infra/config/`
**Archivos a modificar**: `config.go`, `config_test.go`
**Dependencias**: Ninguna
**Instrucciones**:
1. Añadir campos a `Config`: `ChatProvider`, `EmbedProvider`, `OpenAICompatBaseURL`, `OpenAICompatAPIKey`, `OpenAICompatModel`
2. Añadir constantes `envKey*` correspondientes
3. En `Load()`: leer env vars. Si `CHAT_PROVIDER` no está seteado, fallback a `LLM_PROVIDER`, luego a `"ollama"`. `EMBED_PROVIDER` default `"ollama"`.
4. Tests: defaults, overrides, fallback chain
**LOC**: ~60
**Verificación**: `go test ./internal/infra/config/...`

---

### Tarea 2: OpenAI-Compatible Provider
**Scope**: Solo `internal/infra/llm/`
**Archivos a crear**: `openai_compat.go`, `openai_compat_test.go`
**Dependencias**: Ninguna (usa `LLMProvider` interface y types que ya existen)
**Archivos de referencia**: `ollama.go` (copiar patrón de helpers, doPost, tipos internos)
**Instrucciones**:
1. Crear `OpenAICompatProvider` struct: `baseURL`, `apiKey`, `model`, `httpClient`
2. Constructor: `NewOpenAICompatProvider(baseURL, apiKey, model string)`
3. `ChatCompletion()`: POST `{baseURL}/v1/chat/completions` formato estándar OpenAI. Parsear `choices[0].message.content`, `finish_reason`, `usage.total_tokens`
4. `Embed()`: devolver error descriptivo ("use ollama for embeddings")
5. `HealthCheck()`: GET `{baseURL}/v1/models` con `Authorization: Bearer {apiKey}`
6. `ModelInfo()`: metadata estática, provider `"openai-compat"`
7. Reusar constantes `mimeJSON`, `headerContentType` del paquete
8. Tests con `httptest`: success, server error, auth header presente, temperatura/maxTokens en request body, embed-returns-error, health ok/down, model info
**LOC**: ~360
**Verificación**: `go test ./internal/infra/llm/...`

---

### Tarea 3: Factory de providers
**Scope**: Solo `internal/infra/llm/`
**Archivos a crear**: `factory.go`, `factory_test.go`
**Dependencias**: Tarea 1 (config) + Tarea 2 (openai-compat provider)
**Instrucciones**:
1. `NewChatProvider(cfg config.Config) (LLMProvider, error)`: switch en `cfg.ChatProvider` → `"openai-compat"` crea `NewOpenAICompatProvider`, `"ollama"` crea `NewOllamaProvider`, otro → error
2. `NewEmbedProvider(cfg config.Config) (LLMProvider, error)`: solo `"ollama"` soportado, otro → error
3. Tests: factory devuelve tipo correcto para cada config, error en provider desconocido
**LOC**: ~90
**Verificación**: `go test ./internal/infra/llm/...`

---

### Tarea 4: Split inyección + /readyz
**Scope**: `internal/api/routes.go` + `internal/api/handlers/`
**Archivos a modificar**: `routes.go`
**Archivos a crear**: `handlers/readyz.go`, `handlers/readyz_test.go`
**Dependencias**: Tarea 3 (factory)
**Archivos de referencia**: `handlers/health.go` (patrón para readyz)
**Instrucciones**:

**Parte A — Split en routes.go**:
1. Reemplazar `llmProvider := llm.NewOllamaProvider(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.OllamaChatModel)` por llamadas a `llm.NewChatProvider(cfg)` y `llm.NewEmbedProvider(cfg)` con manejo de error (`log.Fatalf`)
2. Mover creación de providers ANTES del bloque de rutas públicas (necesario para registrar `/readyz`)
3. Inyectar `embedProvider` → `EmbedderService`, `SearchService`
4. Inyectar `chatProvider` → `ChatService`, `ActionService`, `ProspectingAgent`, `KBAgent`

**Parte B — /readyz handler**:
1. `NewReadyzHandler(db *sql.DB, chat, embed llm.LLMProvider) http.HandlerFunc`
2. Checks con timeout 5s: `db.Ping()`, `chat.HealthCheck()`, `embed.HealthCheck()`
3. 200 + `{"status":"ready",...}` si todo OK, 503 + status individual si algo falla
4. Registrar en routes.go zona pública junto a `/health`
5. Tests: mock providers, all-ok, db-down, chat-down, embed-down

**LOC**: ~155
**Verificación**: `go test ./internal/api/handlers/... && go test ./internal/api/...`

---

### Tarea 5: Deploy configs (Caddy + compose + env)
**Scope**: `deploy/`, `docker-compose*.yml`, `.env.example`
**Archivos a crear**: `deploy/Caddyfile`, `docker-compose.prod.yml`
**Archivos a modificar**: `docker-compose.yml`, `.env.example`
**Dependencias**: Tarea 4 (readyz endpoint existe)
**Instrucciones**:

**Parte A — Caddyfile** (`deploy/Caddyfile`):
```
{$DOMAIN:localhost} {
    handle /bff/* {
        reverse_proxy bff:3000
    }
    handle {
        reverse_proxy backend:8080
    }
}
```

**Parte B — docker-compose.prod.yml**:
- **caddy**: `caddy:2-alpine`, puertos 80/443, monta Caddyfile, volumes para data/config
- **backend**: `expose: 8080` (NO `ports`), env vars split provider, healthcheck `/readyz`
- **bff**: `expose: 3000` (NO `ports`), depends_on backend healthy
- **ollama**: perfil `with-ollama` (opt-in), `expose: 11434`
- `restart: unless-stopped` en todos

**Parte C — docker-compose.yml** (dev):
Añadir al backend:
```yaml
- CHAT_PROVIDER=${CHAT_PROVIDER:-ollama}
- OPENAI_COMPAT_BASE_URL=${OPENAI_COMPAT_BASE_URL:-}
- OPENAI_COMPAT_API_KEY=${OPENAI_COMPAT_API_KEY:-}
- OPENAI_COMPAT_MODEL=${OPENAI_COMPAT_MODEL:-}
```

**Parte D — .env.example**:
Añadir sección documentada con los nuevos env vars.

**LOC**: ~100
**Verificación**: `docker compose config` y `docker compose -f docker-compose.prod.yml config` (valida sintaxis)

---

## Grafo de dependencias

```
Tarea 1 (config) ──┐
                    ├──> Tarea 3 (factory) ──> Tarea 4 (split + readyz) ──> Tarea 5 (deploy)
Tarea 2 (provider) ┘
```

**Tareas 1 y 2 son paralelas** — sin dependencias entre sí.
**Tarea 3** depende de ambas.
**Tarea 4** depende de 3.
**Tarea 5** depende de 4.

---

## Verificación end-to-end

1. `go test ./internal/infra/llm/... ./internal/infra/config/... ./internal/api/handlers/...`
2. `CHAT_PROVIDER=ollama docker compose up` → backward compatible
3. `CHAT_PROVIDER=openai-compat OPENAI_COMPAT_BASE_URL=https://api.groq.com/openai OPENAI_COMPAT_API_KEY=xxx OPENAI_COMPAT_MODEL=llama3-8b-8192 go run ./cmd/fenix serve` → chat via Groq, embeddings via Ollama
4. `curl localhost:8080/readyz` → 200/503
5. `DOMAIN=app.example.com docker compose -f docker-compose.prod.yml config` → válido
6. `make ci` → pasa lint + test + coverage gates

---

## Total

~790 LOC | 5 archivos nuevos | 4 archivos modificados | 5 tareas (2 paralelas + 3 secuenciales)
