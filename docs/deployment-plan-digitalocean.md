---
title: "FenixCRM - Plan de despliegue en DigitalOcean"
version: "2.1"
date: "2026-03-27"
timezone: "Europe/Madrid"
language: "es-ES"
status: "proposed"
audience: ["engineering", "platform", "security", "product"]
tags: ["deployment", "digitalocean", "gradient", "open-models", "costs", "poc"]
canonical_id: "fenix-deploy-do-v2.1"
source_of_truth: ["docs/requirements.md", "repo state verified on 2026-03-27"]
---

# FenixCRM - Plan de despliegue en DigitalOcean

> **Proposito**: definir un plan realista y low-cost para desplegar FenixCRM en DigitalOcean durante los proximos meses, cubriendo backend, BFF, SQLite y AI con modelos abiertos, sin depender de modelos comerciales.

> **Criterio**: este documento parte del estado real del repositorio a fecha **2026-03-27** y de precios/documentacion oficial de DigitalOcean validados el **2026-03-27**.

> **Referencia de contrato mobile/BFF/API**: la auditoria especifica de interfaz mobile, BFF y API se mantiene separada en `docs/mobile-bff-api-audit-remediation-plan.md` para no mezclar decisiones de despliegue con remediacion de contrato.

---

## 1 -- Resumen ejecutivo

### Decision recomendada

La opcion principal para la POC pasa a ser:

- **`app-droplet` unico** para `reverse proxy + BFF + backend Go + SQLite`
- **`Volume Block Storage`** para DB y adjuntos
- **DigitalOcean Gradient serverless inference** para chat con **modelos abiertos**
- **sin GPU dedicada** en la primera POC

### Por que esta es ahora la opcion recomendada

Porque reduce el coste fijo de forma muy agresiva:

- evita un `gpu-droplet` 24x7
- evita operar Ollama grande en produccion desde el primer dia
- mantiene todo el hosting principal dentro de DigitalOcean
- sigue cumpliendo la restriccion de **no usar modelos comerciales**

### Coste objetivo mensual

Precios en **USD/mes**, sin IVA/VAT ni conversion EUR/USD.

- **Perfil POC minimo recomendado**: **$33/mes + uso de tokens**
- **Perfil POC mas comodo**: **$57/mes + uso de tokens**
- **Perfil self-hosted CPU-only**: **$57-$66/mes**

### Conclusiones clave

1. El documento anterior sobrecosteaba la POC al asumir infraestructura AI demasiado pesada para una primera salida.
2. El repo actual ya tiene mucho runtime de negocio, pero su capa AI productiva sigue acoplada a Ollama.
3. La forma mas barata de salir a una POC realista en DigitalOcean es **mover chat/completions a Gradient serverless con modelos abiertos**.
4. Para minimizar riesgo tecnico, conviene **mantener embeddings locales** en una primera iteracion y separar `chat provider` de `embed provider`.
5. La recomendacion principal es **Droplet pequeno + Gradient serverless open-source**.

---

## 2 -- Estado real verificado del repo

### Wiring actual

El estado real del codigo hoy es:

- entrypoint Go en `cmd/fenix/main.go`
- servidor HTTP en `internal/server/server.go`
- router principal en `internal/api/routes.go`
- BFF separado en `bff/`
- imagen Docker del backend en `deploy/Dockerfile`
- imagen Docker del BFF en `deploy/Dockerfile.bff`
- `docker-compose.yml` solo levanta `backend + bff`

### Runtime actual del backend

El backend ya monta:

- `PolicyEngine`
- `ToolRegistry`
- `ApprovalService`
- `RunnerRegistry`
- `Orchestrator`
- `DSLRunner`
- `WorkflowService`
- `SchedulerService + Worker`
- `SignalService`
- `Eval services`
- agentes `support`, `prospecting`, `kb`, `insights`

### Estado actual de AI

Hoy la capa AI real del repo es:

- provider productivo acoplado a **Ollama**
- `internal/infra/llm/ollama.go` implementa `ChatCompletion`, `Embed` y `HealthCheck`
- `docker-compose.yml` no levanta Ollama dentro de Docker
- el despliegue local actual apunta a `OLLAMA_BASE_URL=http://host.docker.internal:11434`

### Implicacion directa para el plan low-cost

El plan low-cost **no** es "cambiar solo infraestructura". Requiere un cambio tecnico concreto:

- el repo debe dejar de asumir que chat y embeddings salen del mismo provider Ollama

Ese es el punto mas importante del nuevo plan.

### Observabilidad real hoy

Hoy existen:

- `/health`
- `/metrics`
- `/bff/health`
- `/bff/metrics`

Pero siguen faltando:

- `/readyz`
- logging estructurado consistente
- contadores reales de 5xx conectados al flujo
- stack centralizado de logs
- alertas SMTP de aplicacion

### Persistencia y adjuntos

Persistencia actual:

- SQLite local en fichero
- WAL activado
- migraciones al arranque

Adjuntos:

- hoy se guarda `storage_path`
- no existe una capa blob storage completa dentro del backend

Implicacion:

- para la POC low-cost es suficiente un volumen local
- `Spaces` queda como opcion posterior, no como prerequisito

---

## 3 -- Arquitectura objetivo low-cost

### Topologia recomendada

```text
Internet
  |
  v
Caddy o Nginx
  |
  v
BFF :3000
  |
  v
Backend Go :8080
  |
  +--> SQLite en Volume Block Storage
  |
  +--> Embeddings locales de bajo coste (fase 1)
  |
  \--> DigitalOcean Gradient Serverless Inference
        usando modelos abiertos
```

### Que se queda en DigitalOcean

- `app-droplet`
- `Volume Block Storage`
- DNS, firewall, monitoring, uptime
- AI via **DigitalOcean Gradient**

### Que eliminamos de la recomendacion principal

- Ollama grande 24x7

### Restricciones operativas

- backend Go no expuesto publicamente
- un solo nodo de aplicacion por SQLite
- sin balanceador gestionado en dia 1
- sin Kubernetes

---

## 4 -- Perfiles de despliegue

### Perfil A -- POC minima recomendada

#### Arquitectura

- `app-droplet` Basic `4GB / 2 vCPU`
- `Volume Block Storage` `50 GiB`
- AI en **DigitalOcean Gradient serverless inference**
- sin GPU dedicada

#### Cuando usarlo

- POC externa
- demos con usuarios reales
- presupuesto controlado
- necesidad de salir rapido

#### Ventajas

- el menor coste total razonable
- suficiente para una POC real
- no hay que operar GPU
- se mantiene el stack principal dentro de DigitalOcean

#### Limites

- menos margen de RAM y CPU
- requiere cambios de codigo en la capa LLM
- ya no es AI self-hosted pura
- dependes de un servicio gestionado de DigitalOcean para inferencia

### Perfil A2 -- POC mas comoda

#### Arquitectura

- `app-droplet` Basic `8GB / 4 vCPU`
- `Volume Block Storage` `50 GiB`
- AI en **DigitalOcean Gradient serverless inference**

#### Uso recomendado

Usarla si:

- vas a hacer demos frecuentes
- esperas algo mas de concurrencia
- quieres menos riesgo operativo

### Perfil B -- Self-hosted CPU-only

#### Arquitectura

- `app-droplet` Basic `8GB / 4 vCPU`
- `Volume Block Storage` `50 GiB`
- backend + BFF + SQLite
- Ollama local solo en CPU

#### Cuando usarlo

Solo si priorizamos:

- cero cambio de provider
- despliegue totalmente self-hosted de AI

#### Ventajas

- no requiere integrar Gradient
- mantiene el wiring actual mas cerca del repo

#### Limites

- peor latencia
- peor calidad si usamos modelos pequeños
- para una POC externa suele sentirse mas debil

## 5 -- Modelos abiertos recomendados

### Restriccion de producto

No se usaran modelos comerciales. Por tanto:

- no OpenAI cerrado
- no Anthropic
- no Gemini
- no APIs propietarias como opcion principal

### Recomendacion para el perfil low-cost

La recomendacion se separa en dos capas:

#### Chat / generation

Usar **DigitalOcean Gradient serverless inference** con modelos abiertos.

#### Embeddings

Mantener **locales** en una primera fase para no rediseñar RAG completo de golpe.

### Modelos de chat candidatos en DigitalOcean Gradient

Segun la documentacion oficial validada el **2026-03-27**, Gradient ofrece varios modelos abiertos para serverless inference.

| Modelo | Tipo | Coste oficial | Recomendacion |
|---|---|---:|---|
| `llama3-8b-instruct` | open-weight | `$0.198` input / `$0.198` output por 1M tokens | **Mejor coste** |
| `alibaba-qwen3-32b` | open-license | `$0.25` input / `$0.55` output por 1M tokens | **Mejor calidad probable** |
| `mistral-nemo-instruct-2407` | open-weight | `$0.30` input / `$0.30` output por 1M tokens | Alternativa equilibrada |

### Decision recomendada de modelos

#### Opcion principal low-cost

- chat: `llama3-8b-instruct`
- embeddings: `nomic-embed-text` local

Motivo:

- es la opcion con menor coste operativo en Gradient
- encaja bien con el uso real del repo

#### Opcion alternativa si priorizamos licencia mas abierta y mejor calidad

- chat: `alibaba-qwen3-32b`
- embeddings: `nomic-embed-text` local

Motivo:

- `Qwen3-32B` aparece como modelo abierto hospedado por DigitalOcean
- es mejor opcion si preferimos una licencia mas cercana a open-source que Llama

### Embeddings recomendados

#### Fase 1

- mantener `nomic-embed-text` local en Ollama o en el host

Motivo:

- el repo ya esta cableado para embeddings por provider propio
- evita rehacer la parte de RAG y sqlite-vec de golpe
- su peso operativo es muy pequeno comparado con servir un chat model grande

#### Fase 2 opcional

Evaluar migracion de embeddings a Gradient si:

- validamos compatibilidad tecnica del endpoint
- queremos quitar Ollama por completo
- nos compensa pagar el coste por token de embeddings

### Uso real del repo y sizing de modelo

El uso real del LLM hoy es acotado:

- `copilot/chat`
- `copilot/suggest-actions`
- `copilot/summarize`
- `prospecting` para draft corto

Y ademas:

- `support` no depende de chat largo
- `kb` no necesita gran modelo para la primera POC
- `insights` hoy es mas retrieval/metrics que generacion pesada

Implicacion:

- no hace falta un modelo 70B para la primera POC

---

## 6 -- Costes detallados

### Notas de coste

- precios validados contra documentacion oficial de DigitalOcean el **2026-03-27**
- importes en **USD/mes**
- `Monitoring` y `Cloud Firewalls` vienen incluidos con Droplets
- una `Uptime check` es gratis; las adicionales cuestan `$1/mes`

### Costes fijos de infraestructura

| Recurso | Precio oficial |
|---|---:|
| Basic Droplet 4GB / 2 vCPU | $24/mes |
| Basic Droplet 8GB / 4 vCPU | $48/mes |
| Volume Block Storage 50 GiB | $5/mes |
| Volume Snapshot 50 GiB | $3/mes |
| Uptime adicional | $1/mes |

### Escenario 1 -- POC minima recomendada

Supuestos:

- 1 x Basic Droplet `4GB / 2 vCPU`
- 1 x Volume `50 GiB`
- 1 x snapshot mensual estimado sobre `50 GiB`
- 2 uptime checks en total -> 1 gratis + 1 de pago

| Concepto | Importe |
|---|---:|
| Basic Droplet 4GB | $24 |
| Volume 50 GiB | $5 |
| Snapshot 50 GiB | $3 |
| Uptime extra | $1 |
| **Total fijo base** | **$33/mes** |

### Escenario 2 -- POC mas comoda

Supuestos:

- 1 x Basic Droplet `8GB / 4 vCPU`
- 1 x Volume `50 GiB`
- 1 x snapshot mensual estimado sobre `50 GiB`
- 2 uptime checks en total -> 1 gratis + 1 de pago

| Concepto | Importe |
|---|---:|
| Basic Droplet 8GB | $48 |
| Volume 50 GiB | $5 |
| Snapshot 50 GiB | $3 |
| Uptime extra | $1 |
| **Total fijo base** | **$57/mes** |

### Escenario 3 -- Self-hosted CPU-only

Supuestos:

- mismo perfil de infraestructura que la POC mas comoda
- sin coste variable de Gradient

| Concepto | Importe |
|---|---:|
| Total fijo base | $57 |

Comentario:

- es barato
- pero la experiencia de IA probablemente sera peor que el perfil low-cost con Gradient

### Coste variable de AI en Gradient

#### Modelo mas barato: `llama3-8b-instruct`

Precio oficial:

- input: `$0.198` por 1M tokens
- output: `$0.198` por 1M tokens

Ejemplo de referencia:

- `1,000` interacciones de `4k input + 1k output`
- input total: `4M tokens`
- output total: `1M tokens`

Calculo:

- input: `4 * 0.198 = $0.792`
- output: `1 * 0.198 = $0.198`
- **total = $0.99**

Por tanto:

- `1,000` interacciones medianas -> **~$0.99**
- `10,000` interacciones medianas -> **~$9.90**

#### Alternativa: `alibaba-qwen3-32b`

Precio oficial:

- input: `$0.25` por 1M tokens
- output: `$0.55` por 1M tokens

Mismo ejemplo:

- input: `4M * 0.25 = $1.00`
- output: `1M * 0.55 = $0.55`
- **total = $1.55**

Por tanto:

- `1,000` interacciones medianas -> **~$1.55**
- `10,000` interacciones medianas -> **~$15.50**

### Lectura de negocio

Para una POC pura, el suelo fijo mas razonable es **$33/mes** y el resto crece con uso real.

Si esa configuracion se queda corta, el siguiente salto natural es **$57/mes**.

---

## 7 -- Cambios tecnicos necesarios en el repo

### Cambio principal

Para implantar la opcion low-cost hay que dejar de asumir que un unico provider resuelve todo.

Hoy el repo asume:

- un `LLMProvider`
- ese provider hace **chat + embeddings**

La opcion low-cost necesita:

- **chat/completions en Gradient**
- **embeddings locales** en fase 1

### Cambio recomendado de arquitectura en codigo

Separar la configuracion en dos proveedores:

- `CHAT_PROVIDER`
- `EMBED_PROVIDER`

Ejemplo:

```env
CHAT_PROVIDER=openai-compat
EMBED_PROVIDER=ollama
OPENAI_COMPAT_BASE_URL=https://inference.do-ai.run
OPENAI_COMPAT_API_KEY=...
OPENAI_COMPAT_MODEL=llama3-8b-instruct
OLLAMA_BASE_URL=http://127.0.0.1:11434
OLLAMA_MODEL=nomic-embed-text
```

### Trabajo tecnico minimo

1. integrar Gradient a traves de un provider `openai-compat`
2. aprovechar formato OpenAI-compatible en chat completions
3. separar provider de chat y provider de embeddings
4. mantener `OllamaProvider` solo para embeddings en la primera salida
5. exponer `/readyz` para DB + chat provider + embed provider
6. conectar metricas reales de 5xx
7. introducir logging estructurado

### Trabajo tecnico opcional posterior

1. mover embeddings tambien a Gradient si validamos compatibilidad y coste
2. retirar Ollama por completo
3. evaluar storage de adjuntos en `Spaces`

---

## 8 -- Plan de implantacion

### Fase 0 -- Preparacion del repo

Antes de desplegar, el repo debe cumplir todos los puntos siguientes. Cada punto tiene un criterio de aceptacion verificable.

#### 0.1 -- Separacion de providers (bloqueante)

El repo actualmente asume un unico provider que resuelve chat y embeddings. La opcion low-cost requiere separarlos.

Trabajo minimo:

1. introducir `CHAT_PROVIDER` y `EMBED_PROVIDER` como variables de configuracion independientes
2. implementar provider `openai-compat` para chat (compatible con la API de Gradient)
3. mantener `OllamaProvider` solo para embeddings en la primera salida
4. el backend arranca con error explicito si `CHAT_PROVIDER` o `EMBED_PROVIDER` no estan configurados

Criterio de aceptacion: el backend levanta con `CHAT_PROVIDER=openai-compat` y `EMBED_PROVIDER=ollama` configurados, y falla con mensaje claro si falta alguno.

#### 0.2 -- Endpoint `/readyz` (bloqueante)

Hoy existen `/health` y `/metrics`. Falta `/readyz`, que es el endpoint que valida que el sistema esta listo para recibir trafico real.

`/readyz` ya esta implementado en `internal/api/handlers/readyz.go` y valida:

- conexion a SQLite activa (`database`)
- chat provider responde (`chat`)
- embed provider responde (`embed`)

Comportamiento real del endpoint:

- Si SQLite falla → devuelve `503` con `"status": "degraded"` y `"database": "error"`. El sistema no puede servir requests.
- Si chat o embed fallan → devuelve `200` con `"status": "degraded"` y el campo correspondiente en `"error"`. La API sigue operativa pero sin capacidad de IA.
- Si todo pasa → devuelve `200` con `"status": "ready"`.

Esto es comportamiento correcto: un fallo de provider LLM no debe impedir que el CRM base funcione.

Criterio de aceptacion: `curl /readyz` devuelve `200` con todos los campos en `"ok"` cuando backend + Ollama + Gradient estan accesibles. Devuelve `503` solo si la DB no responde.

Este punto esta recogido en **NFR-033** de `docs/requirements.md`.

#### 0.3 -- Suite BDD pasa en local antes de desplegar (bloqueante)

La suite `@stack-go` debe pasar limpia en local antes de hacer el primer despliegue.

```bash
go test ./tests/bdd/... -v
```

Todos los scenarios `@stack-go` deben completar sin `ErrPending` ni fallos.

Criterio de aceptacion: salida del runner muestra todos los scenarios en verde o skipped por tag, sin ninguno en rojo.

#### 0.4 -- Logging estructurado (recomendado, no bloqueante)

Hoy los logs no son estructurados de forma consistente. Para la POC es suficiente con:

- todos los errores HTTP 5xx loguean `method`, `path`, `status`, `error`, `latency_ms`
- arranque del backend loguea provider configurado (`chat_provider`, `embed_provider`, `db_path`)

Sin esto los errores en produccion son dificiles de diagnosticar.

#### 0.5 -- Variables de entorno documentadas

El fichero `.env.example` en la raiz debe reflejar la nueva separacion de providers antes de desplegar:

```env
CHAT_PROVIDER=openai-compat
EMBED_PROVIDER=ollama
OPENAI_COMPAT_BASE_URL=https://inference.do-ai.run
OPENAI_COMPAT_API_KEY=
OPENAI_COMPAT_MODEL=llama3-8b-instruct
OLLAMA_BASE_URL=http://127.0.0.1:11434
OLLAMA_MODEL=nomic-embed-text
```

### Fase 1 -- Provisioning

Recursos:

- 1 proyecto DigitalOcean
- 1 VPC
- 1 Cloud Firewall
- 1 `app-droplet`
- 1 `Volume Block Storage`
- 1 o 2 Uptime checks

### Fase 2 -- App host

En el `app-droplet`:

- Ubuntu LTS
- Docker Engine + Compose plugin
- `Caddy` o `Nginx`
- volumen montado para SQLite y adjuntos
- backend + BFF
- Ollama local solo si mantenemos embeddings locales

Layout recomendado:

```text
/srv/fenix/
  compose/
  config/
  data/
    fenixcrm.db
    attachments/
  backups/
```

### Fase 3 -- Configuracion recomendada

```env
NODE_ENV=production
ENVIRONMENT=production
BFF_PORT=3000
BACKEND_URL=http://backend:8080
DATABASE_URL=/srv/fenix/data/fenixcrm.db
JWT_SECRET=...
BFF_ORIGIN=https://app.tudominio.com

CHAT_PROVIDER=openai-compat
EMBED_PROVIDER=ollama

OPENAI_COMPAT_BASE_URL=https://inference.do-ai.run
OPENAI_COMPAT_API_KEY=...
OPENAI_COMPAT_MODEL=llama3-8b-instruct

OLLAMA_BASE_URL=http://127.0.0.1:11434
OLLAMA_MODEL=nomic-embed-text
```

### Fase 4 -- Validacion funcional

La validacion se divide en dos niveles: checks de infraestructura y checks funcionales. Ambos deben pasar antes de considerar la POC lista.

#### Nivel 1 -- Infraestructura y conectividad

Estos checks no requieren datos ni LLM. Verifican que la topologia esta bien:

| Check | Comando | Resultado esperado |
|---|---|---|
| Backend vivo | `curl https://tudominio.com/health` | `200 OK` |
| Backend listo | `curl https://tudominio.com/readyz` | `200 OK` con `"status":"ready"` y los tres campos en `"ok"` |
| BFF vivo | `curl https://tudominio.com/bff/health` | `200 OK` |
| TLS activo | `curl -I https://tudominio.com` | certificado valido, sin warnings |
| Backend no expuesto | `curl http://IP_DROPLET:8080/health` | timeout o connection refused |

Si `/readyz` devuelve `503`, revisar el detalle del cuerpo antes de continuar.

#### Nivel 2 -- Funcional basico (smoke)

Estos checks requieren un usuario de prueba creado previamente. Ejecutar en orden.

**Autenticacion:**

- login con usuario de prueba devuelve JWT valido
- registro de nuevo usuario funciona

**CRM base (FR-001):**

- crear un Account
- crear un Contact ligado al Account
- crear un Deal
- crear un Case

**Knowledge (FR-090, FR-091):**

- `POST /knowledge/ingest` con un documento de prueba devuelve `201`
- `POST /knowledge/search` con query simple devuelve resultados con `score`

**Copilot (FR-200, FR-202):**

- `POST /copilot/chat` devuelve respuesta con `sources` en el body (evidence pack)
- `POST /copilot/suggest-actions` devuelve lista de acciones con tool names
- `POST /copilot/summarize` devuelve resumen con evidencias citadas

**Agentes (FR-231):**

- `POST /agents/prospecting/trigger` acepta el request y devuelve `run_id`
- consultar el `run_id` devuelve estado `completed` o `running`

#### Nivel 3 -- BDD post-deploy (recomendado)

Si el entorno de produccion es accesible desde el runner de CI, ejecutar la suite BDD contra el host real:

```bash
FENIX_BASE_URL=https://tudominio.com go test ./tests/bdd/... -v -run TestFeatures
```

Los scenarios `@stack-go` que solo validan logica interna (workflow authoring, versioning, etc.) seguiran pasando porque usan SQLite en memoria. Los scenarios que dependan de LLM real quedaran como indicadores de calidad, no como gate de despliegue.

Los scenarios `@stack-mobile` no tienen runner activo en esta fase. Quedan pendientes para cuando el runner movil (Detox o Maestro) este configurado.

### Fase 5 -- Operacion

Dia 1:

- snapshots de volumen
- Uptime checks
- alertas de DigitalOcean por email
- logs locales del host

Despues:

- observabilidad mas completa
- backup de adjuntos
- posible migracion de embeddings a servicio gestionado

---

## 9 -- Riesgos y limites

### Riesgo 1 -- Hay cambio de provider

El mayor riesgo del nuevo plan es tecnico, no de infraestructura:

- el repo hoy solo esta cableado a Ollama
- la opcion low-cost requiere introducir Gradient

### Riesgo 2 -- SQLite sigue siendo single-node

Mientras usemos SQLite:

- no hay horizontal scaling real
- seguimos siendo single-node

### Riesgo 3 -- Embeddings quedan hibridos en fase 1

Durante la primera salida:

- chat en Gradient
- embeddings locales

Eso es una solucion pragmatica, no una arquitectura final.

### Riesgo 4 -- 4GB es agresivo

El perfil de `4GB` puede quedarse corto si:

- la base crece rapido
- hay varias sesiones simultaneas
- mantenemos demasiados procesos auxiliares en el mismo host

Por eso el `4GB` pasa a ser la recomendacion principal de POC, y `8GB` queda como mejora inmediata si hiciera falta.

### Riesgo 5 -- Observabilidad sigue basica en dia 1

No recomiendo meter Loki/Grafana/Prometheus como requisito previo del low-cost. Primero hay que arreglar:

- logs estructurados
- `/readyz`
- metricas de error reales

---

## 10 -- Checklist de aceptacion de la POC low-cost

### Pre-despliegue (Fase 0)

- [ ] `CHAT_PROVIDER` y `EMBED_PROVIDER` separados en el codigo
- [ ] provider `openai-compat` implementado y verificado contra Gradient
- [ ] `/readyz` implementado: valida DB + chat provider + embed provider
- [ ] suite BDD `@stack-go` pasa en local sin errores
- [ ] `.env.example` actualizado con nueva separacion de providers
- [ ] logs de arranque incluyen provider configurado y db path

### Infraestructura (Fases 1-3)

- [ ] `app-droplet` desplegado con TLS activo
- [ ] backend no expuesto publicamente (solo accesible via BFF o proxy)
- [ ] volumen montado en `/srv/fenix/data/`
- [ ] Ollama local corriendo con `nomic-embed-text`
- [ ] variables de entorno de produccion configuradas (sin secrets en disco plano)

### Validacion funcional (Fase 4 -- Nivel 1)

- [ ] `GET /health` responde `200`
- [ ] `GET /readyz` responde `200` con `"status":"ready"` y los tres campos en `"ok"`; devuelve `503` si SQLite falla, `200` degradado si solo falla un provider LLM
- [ ] `GET /bff/health` responde `200`
- [ ] backend no responde en puerto directo desde exterior

### Validacion funcional (Fase 4 -- Nivel 2)

- [ ] login y registro funcionan
- [ ] CRUD de Account, Contact, Deal, Case funciona
- [ ] `knowledge/ingest` acepta documento y devuelve `201`
- [ ] `knowledge/search` devuelve resultados con score
- [ ] `copilot/chat` devuelve respuesta con `sources` (evidence pack) via Gradient
- [ ] `copilot/suggest-actions` devuelve tool actions via Gradient
- [ ] `copilot/summarize` devuelve resumen con evidencias citadas
- [ ] `agents/prospecting/trigger` acepta request y devuelve `run_id`

### Operacion (Fase 5)

- [ ] snapshots de volumen configurados
- [ ] uptime alerts por email configuradas
- [ ] logs del host accesibles via SSH

### Deuda tecnica de testing identificada

Estado real de la cobertura por FR — para saber que falta antes y despues de la POC.

| FR | Que existe | Que falta | Tipo de deuda |
|---|---|---|---|
| FR-001 (CRM CRUD) | Unit + integration + API handler tests | Nada | Sin deuda |
| FR-002 (Pipelines/etapas) | Handler tests en `internal/api/handlers/` | Sin BDD feature | Aceptado por diseno (enabler, no UC) |
| FR-060 (Auth/RBAC base) | Unit + integration + E2E Detox (`auth.e2e.ts`) | Nada | Sin deuda |
| FR-070 (Audit trail) | Integration + API tests | Sin test de export | Menor |
| FR-090/091/092 (Knowledge) | Unit + integration parcial | Pipeline completo sin test; CDC/reindex pendiente | Partial coverage |
| FR-200 (Copilot chat SSE) | Detox spec en `copilot.e2e.ts`; no conectado a Gherkin | Feature `@stack-mobile` + runner activo | Detox-Gherkin separados |
| FR-201 (Resumenes copilot) | Solo verificacion manual (Paso 4) | Sin BDD feature ni smoke check | Feature pendiente |
| FR-210 (Abstencion) | UC-C1 tiene scenario `@abstention` en feature file | Tag `@FR-210` faltante | Tag missing |
| FR-211 (Safe tool routing) | Cubierto implicitamente en agent runs | Sin feature BDD dedicada; crear `uc-b1-safe-tool-routing.feature` (`@stack-go`) | Feature pendiente |
| FR-230/231/232 (Agentes) | Support + Prospecting + KB agents implementados; Detox `agent-runs.e2e.ts` | Insights agent no implementado; Detox no conectado a Gherkin | Parcial |
| FR-242 (Evals) | `/admin/eval` implementado y CRUD funcional | Sin smoke check en Fase 4 ni BDD feature | Smoke + feature pendiente |
| FR-300/301 (Mobile+BFF) | 8 suites Detox en `mobile/e2e/` + BFF Supertest 80% cobertura | Gherkin runner `@stack-mobile` no activo; Detox no conecta con `features/` | Activar runner CI |
| NFR-070 (Frame rate mobile) | Nada | Tests de performance automatizados | Post-POC |
| NFR-071 (Latencia de carga) | Nada | Benchmarks de latencia en navegacion | Post-POC |
| NFR-072 (Primer token SSE ≤500ms) | Verificacion manual en Paso 0 | Sin check automatizado | Post-POC |

### Pendientes aceptados para post-POC

- [ ] Detox esta completamente configurado: `mobile/.detoxrc.js` (Android emulator Pixel 7 API 33), `mobile/e2e/jest.config.ts`, 8 suites escritas (`auth.e2e.ts`, `accounts.e2e.ts`, `deals.e2e.ts`, `cases.e2e.ts`, `copilot.e2e.ts`, `agent-runs.e2e.ts`, `workflows.e2e.ts`) + directorio `bdd/`. Lo que no existe es la integracion con el runner Gherkin `features/` — los dos sistemas viven separados
- [ ] Para activar el runner `@stack-mobile`: conectar cucumber-js con los feature files de `features/` apuntando a las Detox specs existentes (direccion acordada)
- [ ] Deuda de naming: `copilot-uc-s1.e2e.ts` y `uc_s1.helper.ts` usan nombres internos de UC — renombrar a `copilot-sales.e2e.ts` y `copilot_sales.helper.ts` en la siguiente iteracion de mobile
- [ ] Features BDD Gherkin pendientes de crear para flujos que ya tienen Detox tests: crear Account desde mobile, crear Deal desde mobile, ver Case + Copilot en contexto, trigger agent desde mobile y ver run en Activity Log
- [ ] FR-210: anadir tag `@FR-210` al scenario `@abstention` en `uc-c1-support-agent.feature`
- [ ] FR-211 (safe tool routing): sin feature BDD — crear `uc-b1-safe-tool-routing.feature` (`@stack-go`)
- [ ] FR-242 (evals): endpoint `/admin/eval` existe pero sin smoke check en Fase 4 ni BDD
- [ ] observabilidad completa (Loki/Grafana/Prometheus) fuera de scope de la POC

---

## 11 -- Guia de prueba funcional post go-live

> **Proposito**: recorrido ejecutable que simula el ciclo de vida real de un CRM. Cubre desde el primer registro hasta el agente resolviendo un caso con evidencia. Cada paso es una llamada HTTP real con payload exacto.
>
> **Prerequisito**: el checklist de Fase 4 Nivel 1 paso completamente.

### Convenciones

- `$BASE` = URL base del BFF (ej. `https://app.tudominio.com`) o del backend directo en local (`http://localhost:8080`)
- Los IDs devueltos en cada paso se reusan en pasos siguientes — guardar en variables de entorno o en un fichero temporal
- Todos los endpoints protegidos requieren `Authorization: Bearer $TOKEN`

---

### Paso 0 -- Verificar que la app mobile conecta con el backend

Este es el paso del end-user real. Antes de cualquier curl, verificar que la app Android arranca y conecta.

**Configurar la URL del BFF en la app:**

En `mobile/.env` o via `eas.json` en el build de desarrollo:

```env
EXPO_PUBLIC_BFF_URL=https://app.tudominio.com
```

**Verificacion en dispositivo o emulador Android (6 pasos):**

1. Abrir la app — debe mostrar la pantalla de login
2. Login con las credenciales creadas en el Paso 1
3. La pantalla Home carga correctamente (puede estar vacia en una POC nueva)
4. Navegar a **CRM → Accounts** — debe aparecer la lista con los accounts del Paso 2
5. Abrir un Account → pulsar el boton Copilot — debe arrancar el streaming SSE y mostrar respuesta con evidencia citada
   - NFR-072: el primer token debe aparecer en ≤500ms desde que se envia la query (excluye latencia LLM)
6. Navegar a un **Case** → el panel Copilot debe mostrar acciones sugeridas (suggest-actions)

**Si la app no conecta:**

- Verificar que `EXPO_PUBLIC_BFF_URL` apunta al dominio correcto
- Verificar CORS en el BFF: `BFF_ORIGIN` debe incluir el origen de la app o estar en modo desarrollo
- Las rutas que usa la app: `/bff/auth/login`, `/bff/api/v1/*`, `/bff/copilot/chat`

**Estado real del runner mobile:**

La app mobile esta mas avanzada de lo que sugiere el checklist. Lo que existe hoy:

- `mobile/` — 47 screens React Native + Expo (auth, CRM, copilot, workflows, agents, signals, approvals)
- `bff/` — Express.js proxy funcional con SSE relay, Supertest al 80% de cobertura
- `mobile/e2e/` — 8 suites Detox completamente configuradas (`auth.e2e.ts`, `accounts.e2e.ts`, `deals.e2e.ts`, `cases.e2e.ts`, `copilot.e2e.ts`, `agent-runs.e2e.ts`, `workflows.e2e.ts`) + directorio `bdd/`
- Los dos sistemas (Gherkin `features/` y Detox specs) viven separados — no hay runner que los conecte

Lo que falta para cerrar el loop BDD mobile automatizado:

1. Conectar los feature files Gherkin (`features/@stack-mobile`) con las Detox specs existentes via cucumber-js (direccion acordada)
2. Crear feature files Gherkin para los flujos mobile que ya tienen Detox tests pero no tienen escenario en `features/`:
   - crear Account desde mobile
   - crear Deal desde mobile
   - ver Case + Copilot en contexto
   - trigger agent desde mobile y ver run en Activity Log
3. Resolver deuda de naming: `copilot-uc-s1.e2e.ts` → `copilot-sales.e2e.ts` y `uc_s1.helper.ts` → `copilot_sales.helper.ts`

Para la POC: el flujo mobile se verifica manualmente con este Paso 0.
La automatizacion queda como deuda tecnica aceptada post-POC.

---

### Paso 1 — Registro y login (auth)

Esto simula el primer usuario de la POC creando su workspace.

```bash
# 1a. Registrar workspace + usuario
curl -s -X POST $BASE/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@fenixpoc.com",
    "password": "SuperSecure12345!",
    "displayName": "Admin POC",
    "workspaceName": "Fenix POC"
  }'
# Esperar: 201 con { token, userId, workspaceId }
# Guardar: TOKEN, USER_ID, WORKSPACE_ID

# 1b. Login (verificar que funciona por separado)
curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@fenixpoc.com",
    "password": "SuperSecure12345!"
  }'
# Esperar: 200 con { token, userId, workspaceId }
```

**Que estamos probando**: FR-060 (RBAC base), FR-051 (API publica), auth service completo.

**Si falla**: revisar logs de arranque, verificar que JWT_SECRET esta configurado.

---

### Paso 2 — CRM core: crear el grafo de datos basico

Esto es el ciclo clasico de un CRM: empresa → contacto → deal → caso.

```bash
# 2a. Crear Account (empresa)
curl -s -X POST $BASE/api/v1/accounts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corp",
    "domain": "acme.com",
    "industry": "Technology",
    "sizeSegment": "mid",
    "ownerId": "'$USER_ID'"
  }'
# Esperar: 201 con { id, name, ... }
# Guardar: ACCOUNT_ID

# 2b. Crear Contact (persona en la empresa)
curl -s -X POST $BASE/api/v1/contacts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "firstName": "Maria",
    "lastName": "Garcia",
    "email": "maria@acme.com",
    "accountId": "'$ACCOUNT_ID'",
    "ownerId": "'$USER_ID'"
  }'
# Esperar: 201
# Guardar: CONTACT_ID

# 2c. Crear Deal (oportunidad comercial)
curl -s -X POST $BASE/api/v1/deals \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Licencia Enterprise Acme",
    "accountId": "'$ACCOUNT_ID'",
    "contactId": "'$CONTACT_ID'",
    "stage": "qualification",
    "amount": 25000,
    "currency": "EUR",
    "ownerId": "'$USER_ID'"
  }'
# Esperar: 201
# Guardar: DEAL_ID

# 2d. Crear Case (ticket de soporte)
curl -s -X POST $BASE/api/v1/cases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "Error en integracion API",
    "description": "El cliente reporta error 500 al llamar a /webhooks. Adjunta logs del 25 de marzo.",
    "priority": "high",
    "accountId": "'$ACCOUNT_ID'",
    "contactId": "'$CONTACT_ID'",
    "ownerId": "'$USER_ID'"
  }'
# Esperar: 201
# Guardar: CASE_ID
```

**Que estamos probando**: FR-001 (CRUD CRM completo), relaciones entre entidades, workspace isolation.

**Si falla**: revisar migraciones SQLite, verificar que el workspace_id del JWT coincide.

---

### Paso 3 — Knowledge: ingestar informacion y buscar

Esto alimenta la capa RAG para que el Copilot y los agentes tengan evidencia real.

```bash
# 3a. Ingestar un documento tecnico ligado al account
curl -s -X POST $BASE/api/v1/knowledge/ingest \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sourceType": "document",
    "title": "Acme API Integration Guide v2",
    "rawContent": "Acme Corp utiliza webhooks para sincronizar pedidos. El endpoint /webhooks acepta POST con firma HMAC-SHA256. Errores comunes: 500 si el payload excede 1MB, 403 si la firma no coincide. Solucion: verificar header X-Acme-Signature y limitar payload a 512KB.",
    "entityType": "account",
    "entityId": "'$ACCOUNT_ID'"
  }'
# Esperar: 201 con { id, sourceType, title, createdAt }

# 3b. Ingestar un email relevante del contacto
curl -s -X POST $BASE/api/v1/knowledge/ingest \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sourceType": "email",
    "title": "Re: Error 500 en webhooks",
    "rawContent": "Hola equipo, seguimos viendo el error 500 en /webhooks desde el 25 de marzo. Adjunto los logs del servidor. El payload que enviamos pesa 1.2MB. Maria.",
    "entityType": "case",
    "entityId": "'$CASE_ID'"
  }'
# Esperar: 201

# 3c. Buscar en knowledge (keyword + vector hibrido)
curl -s -X POST $BASE/api/v1/knowledge/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "error 500 webhooks acme",
    "limit": 5
  }'
# Esperar: 200 con array de resultados, cada uno con:
#   - id, title, snippet, score
# Verificar: que devuelve los dos items ingestados con score > 0
```

**Que estamos probando**: FR-090 (indexacion hibrida), FR-091 (ingesta), FR-092 (evidence pack base).

**Si falla**: verificar que Ollama esta corriendo con `nomic-embed-text` y que `EMBED_PROVIDER=ollama` esta configurado.

---

### Paso 4 — Copilot: chat con evidencia grounded

Esto es el test mas critico del producto — el copilot respondiendo con citas reales.

```bash
# 4a. Chat sobre el case (con contexto)
curl -s -X POST $BASE/api/v1/copilot/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Cual es la causa probable del error 500 en webhooks de Acme y como lo resolvemos?",
    "entityType": "case",
    "entityId": "'$CASE_ID'"
  }'
# Esperar: SSE stream con chunks, el ultimo debe incluir:
#   - sources: array con al menos 1 item (evidence pack)
#   - cada source debe tener: id, snippet, score
# Si las sources estan vacias o el copilot inventa: hay un problema de retrieval

# 4b. Suggest actions
curl -s -X POST $BASE/api/v1/copilot/suggest-actions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "entityType": "case",
    "entityId": "'$CASE_ID'"
  }'
# Esperar: 200 con array de acciones sugeridas con tool names

# 4c. Summarize
curl -s -X POST $BASE/api/v1/copilot/summarize \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "entityType": "case",
    "entityId": "'$CASE_ID'"
  }'
# Esperar: 200 con resumen que cite evidencia
```

**Que estamos probando**: FR-200 (copilot), FR-202 (actions), FR-092 (evidence pack obligatorio), NFR-001 (latencia).

**Si falla**:
- Si no devuelve sources → problema en SearchService o EvidencePackService
- Si devuelve error 500 → verificar `CHAT_PROVIDER=openai-compat` y conectividad con Gradient
- Si la respuesta es lenta (>10s) → revisar modelo seleccionado y latencia de red a Gradient

---

### Paso 5 — Agente de soporte: resolver el case

Esto simula lo que describe UC-C1: el agente resuelve un caso usando evidencia y herramientas.

```bash
# 5a. Trigger support agent sobre el case
curl -s -X POST $BASE/api/v1/agents/support/trigger \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "caseId": "'$CASE_ID'"
  }'
# Esperar: 200/202 con { runId, status }
# Guardar: RUN_ID

# 5b. Consultar el estado del agent run
curl -s -X GET $BASE/api/v1/agents/runs/$RUN_ID \
  -H "Authorization: Bearer $TOKEN"
# Esperar: 200 con estado completed, partial, abstained o failed
# Verificar:
#   - si completed: hay tool_calls en el resultado (FR-202)
#   - si abstained: hay abstain_reason (FR-210) — correcto si la evidencia es insuficiente
#   - si failed: revisar error en el body
```

**Que estamos probando**: FR-230 (runtime), FR-231 (catalogo), FR-211 (safe tool routing), FR-092 (evidence pack).

---

### Paso 6 — Audit trail: verificar que todo quedo registrado

```bash
# 6a. Consultar eventos de auditoria
curl -s -X GET "$BASE/api/v1/audit/events?limit=20" \
  -H "Authorization: Bearer $TOKEN"
# Esperar: 200 con array de eventos
# Verificar que aparecen:
#   - registro del usuario (auth.register)
#   - creacion de account, contact, deal, case
#   - ingesta de knowledge items
#   - copilot chat / suggest-actions / summarize
#   - agent run del support agent
# Cada evento debe tener: actor, action, resource, timestamp
```

**Que estamos probando**: FR-070 (audit trail), FR-060 (RBAC — solo vemos eventos de nuestro workspace).

---

### Paso 7 — Prospecting agent (ciclo de ventas)

```bash
# 7a. Trigger prospecting sobre un lead o account
curl -s -X POST $BASE/api/v1/agents/prospecting/trigger \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": "'$ACCOUNT_ID'"
  }'
# Esperar: 200/202 con { runId }
# Guardar: PROSPECTING_RUN_ID

# 7b. Consultar resultado
curl -s -X GET $BASE/api/v1/agents/runs/$PROSPECTING_RUN_ID \
  -H "Authorization: Bearer $TOKEN"
# Esperar: status completed con insights o draft
```

**Que estamos probando**: FR-231 (prospecting agent), UC-S2.

---

### Paso 8 — Reports (validar datos acumulados)

```bash
# 8a. Embudo de ventas
curl -s -X GET $BASE/api/v1/reports/sales/funnel \
  -H "Authorization: Bearer $TOKEN"
# Esperar: 200 con datos del pipeline (debe incluir el deal creado en paso 2)

# 8b. Backlog de soporte
curl -s -X GET $BASE/api/v1/reports/support/backlog \
  -H "Authorization: Bearer $TOKEN"
# Esperar: 200 con el case del paso 2 en el backlog
```

**Que estamos probando**: FR-003 (reporting base).

---

### Resumen: que FRs cubre cada paso

| Paso | FRs cubiertos | UC |
|---|---|---|
| 1. Auth | FR-060, FR-051 | — |
| 2. CRM core | FR-001 | — |
| 3. Knowledge | FR-090, FR-091, FR-092 | — |
| 4. Copilot | FR-200, FR-202 | UC-S1 |
| 5. Support agent | FR-230, FR-231, FR-210, FR-211 | UC-C1 |
| 6. Audit | FR-070 | UC-G1 |
| 7. Prospecting | FR-231 | UC-S2 |
| 8. Reports | FR-003 | — |

### Que NO cubre esta guia (y por que)

| Area | Razon | Cuando cubrirlo |
|---|---|---|
| Paso 0 (mobile) como test automatizado | Detox configurado pero no conectado a Gherkin runner; gap es cucumber-js ↔ `features/` | Post-POC |
| Workflows DSL (UC-A2 a A9) | Requiere configuracion previa de workflow definitions | Siguiente iteracion post-POC |
| Approval flows (FR-071) | Requiere segundo usuario con rol distinto | Test con 2 usuarios |
| FR-002 (Pipelines y etapas) | Sin BDD; cubierto por handler tests en Go | Aceptado por diseno |
| FR-201 (Resumenes copilot) | Paso 4 cubre `copilot/summarize` manualmente; sin BDD | Post-POC |
| FR-210 (Abstencion explicita) | UC-C1 tiene el scenario pero sin tag `@FR-210` | Correccion de tag pendiente |
| FR-211 (Safe tool routing) | Sin feature BDD propia; cubierto implicitamente en agent runs | Feature `@stack-go` pendiente |
| FR-242 (Evals y gating) | Endpoint `/admin/eval` existe; sin smoke check ni BDD | Post-POC |
| PII / no-cloud (FR-061) | Requiere policies configuradas | Test de governance dedicado |
| NFR-070 (Frame rate mobile) | Sin tests automatizados de performance | Post-POC |
| NFR-071 (Latencia de carga mobile) | Sin benchmarks de latencia en navegacion | Post-POC |
| NFR-072 (Primer token SSE ≤500ms) | Solo verificacion manual en Paso 0 | Post-POC |
| Quotas / degradation (FR-233) | P1 | Post-POC |

---

## 12 -- Decision final recomendada

### Lo que recomiendo hacer ahora

Para la POC low-cost:

1. **usar `app-droplet` Basic 4GB**
2. **usar `Volume Block Storage` de 50 GiB**
3. **usar DigitalOcean Gradient serverless inference con modelos abiertos**
4. **separar `chat provider` y `embed provider`**
5. **mantener embeddings locales en fase 1**

### Coste a presupuestar

#### Recomendacion principal

- **$33/mes fijos**
- **+$1 a $20/mes tipicos de tokens** para una POC normal

Orden de magnitud realista:

- **~$35-$55/mes**

#### Opcion mas comoda

- **$57/mes fijos**
- menos riesgo operativo

---

## 13 -- Fuentes externas usadas

DigitalOcean:

- Droplets pricing: https://www.digitalocean.com/pricing/droplets
- Volumes pricing: https://docs.digitalocean.com/products/volumes/details/pricing/
- Uptime pricing: https://docs.digitalocean.com/products/uptime/details/pricing/
- VPC pricing: https://docs.digitalocean.com/products/networking/vpc/details/pricing/
- Gradient AI pricing: https://docs.digitalocean.com/products/gradient-ai-platform/details/pricing/
- Gradient available models: https://docs.digitalocean.com/products/gradient-ai-platform/details/models/
- Gradient serverless inference: https://docs.digitalocean.com/products/gradient-ai-platform/how-to/use-serverless-inference/
- Gradient limits: https://docs.digitalocean.com/products/gradient-ai-platform/details/limits/

Modelos:

- Qwen3-32B model card: https://huggingface.co/Qwen/Qwen3-32B
- Llama 3.1 8B Instruct model card: https://huggingface.co/meta-llama/Llama-3.1-8B-Instruct
- Mistral NeMo model page: https://mistral.ai/news/mistral-nemo/
- Nomic Embed v1.5 model card: https://huggingface.co/nomic-ai/nomic-embed-text-v1.5
