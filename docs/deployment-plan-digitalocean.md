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

Antes de desplegar:

1. verificar `openai-compat` para chat serverless en Gradient
2. separar `chat` y `embed` provider
3. mantener `nomic-embed-text` local para embeddings
4. verificar `/readyz`
5. mejorar logging y metricas

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

Smoke checks minimos:

- `GET /health`
- `GET /readyz`
- `GET /bff/health`
- login y registro
- CRUD base
- `knowledge/ingest`
- `knowledge/search`
- `copilot/chat`
- `copilot/suggest-actions`
- `copilot/summarize`
- `agents/prospecting/trigger`

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

- [ ] `app-droplet` desplegado con TLS
- [ ] backend no expuesto publicamente
- [ ] `GET /health` responde
- [ ] `GET /readyz` valida DB + provider de chat + embeddings
- [ ] `GET /bff/health` responde
- [ ] login y registro funcionan
- [ ] CRUD base funciona
- [ ] `knowledge/ingest` y `knowledge/search` funcionan
- [ ] `copilot/chat` funciona via Gradient
- [ ] `copilot/suggest-actions` funciona via Gradient
- [ ] `prospecting` genera draft
- [ ] snapshots configurados
- [ ] uptime alerts por email configuradas

---

## 11 -- Decision final recomendada

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

## 12 -- Fuentes externas usadas

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
