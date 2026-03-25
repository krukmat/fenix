---
title: "FenixCRM - Plan de despliegue e instalacion en DigitalOcean"
version: "1.0"
date: "2026-03-23"
timezone: "Europe/Madrid"
language: "es-ES"
status: "proposed"
audience: ["engineering", "platform", "security", "revops"]
tags: ["deployment", "digitalocean", "operations", "observability", "logging", "alerts"]
canonical_id: "fenix-deploy-do-v1"
source_of_truth: ["docs/requirements.md", "docs/architecture.md"]
---

# FenixCRM - Plan de despliegue e instalacion en DigitalOcean

> **Proposito**: Definir un plan operativo y tecnico para publicar una version funcional de FenixCRM en DigitalOcean, incluyendo backend Go, runtime agentico, BFF, observabilidad, logging robusto y alertas criticas al admin por email.

> **Importante**: Este documento no reemplaza `docs/requirements.md`. `docs/requirements.md` sigue siendo la fuente de requisitos de producto y arquitectura funcional. Este documento cubre despliegue, operacion y hardening.

---

## Indice

1. [Objetivo y alcance](#1--objetivo-y-alcance)
2. [Estado actual verificado del repo](#2--estado-actual-verificado-del-repo)
3. [Decision de despliegue en DigitalOcean](#3--decision-de-despliegue-en-digitalocean)
4. [Arquitectura objetivo](#4--arquitectura-objetivo)
5. [Configuracion y despliegue](#5--configuracion-y-despliegue)
6. [Observabilidad y operacion](#6--observabilidad-y-operacion)
7. [Logging robusto](#7--logging-robusto)
8. [Alertas criticas al admin por email](#8--alertas-criticas-al-admin-por-email)
9. [Backups, restauracion y continuidad](#9--backups-restauracion-y-continuidad)
10. [Checklist de aceptacion](#10--checklist-de-aceptacion)
11. [Riesgos y limites actuales](#11--riesgos-y-limites-actuales)
12. [Cambios tecnicos previstos](#12--cambios-tecnicos-previstos)

---

## 1 -- Objetivo y alcance

### Objetivo

Publicar una version funcional de FenixCRM en DigitalOcean con:

- backend Go
- runtime agentico embebido en el backend
- BFF Node/TypeScript
- base de datos SQLite persistida en volumen
- runtime AI basado en Ollama sobre infraestructura DigitalOcean
- operacion observable con logs, metricas, health checks y alertas

### Alcance incluido

- despliegue del backend, BFF y capa agentica
- configuracion de red, persistencia, TLS y variables de entorno
- observabilidad self-hosted
- logging estructurado
- alertas criticas al admin por email
- estrategia de backups y restauracion

### Fuera de alcance

- publicacion de la app mobile en stores
- migracion a PostgreSQL
- despliegue multi-region
- escalado horizontal del backend
- sustitucion del runtime AI actual por otro proveedor

---

## 2 -- Estado actual verificado del repo

### Arquitectura actual

El estado real del codigo confirma:

- backend principal en Go con entrypoint en `cmd/fenix/main.go`
- servidor HTTP en `internal/server/server.go`
- router en `internal/api/routes.go`
- BFF separado en `bff/`
- despliegue previsto de dos procesos, documentado en `docs/architecture.md`
- imagen Docker para backend en `deploy/Dockerfile`
- imagen Docker para BFF en `deploy/Dockerfile.bff`
- `docker-compose.yml` ya orientado a levantar backend + BFF

### Runtime agentico

La parte agentica no es un servicio aparte. Vive dentro del backend Go:

- el router protegido instancia `PolicyEngine`, `ToolRegistry`, `ApprovalService`, `RunnerRegistry`, `Orchestrator` y `DSLRunner`
- el runtime real actual usa `OllamaProvider` en `internal/infra/llm/ollama.go`
- la abstraccion de proveedor existe, pero el wiring productivo actual esta acoplado a Ollama

### Persistencia

- el backend usa SQLite embebido
- `DATABASE_URL` apunta a un fichero local
- el `docker-compose.yml` ya persiste la DB en un volumen Docker

### Logging y observabilidad actuales

El estado actual es funcional pero basico:

- backend:
  - usa `middleware.Logger` y `middleware.Recoverer` de chi (`internal/api/routes.go:57-60`)
  - usa `fmt.Printf`/`fmt.Println` en `internal/server/server.go` (lineas 63, 69, 81)
  - usa `log.Printf` en `internal/infra/llm/ollama.go` (linea 223) y `internal/api/handlers/copilot_chat.go` (linea 50)
  - expone `/health` y `/metrics`
  - `MetricsCollector` con `IncErrors()` existe en `internal/api/handlers/metrics.go` pero nunca es llamado en el flujo real
- BFF:
  - usa `console.log` en `bff/src/server.ts` (lineas 9, 10, 11, 17)
  - expone `/bff/health` y `/bff/metrics`
  - `incErrors()` existe en `bff/src/routes/metrics.ts` pero no es llamada desde `bff/src/middleware/errorHandler.ts`
  - tiene middleware de errores para envelope HTTP

### Limitaciones verificadas en codigo

- **no existe logging JSON estructurado** — `middleware.Logger` de chi emite texto plano; `fmt.Printf`/`log.Printf` no tienen campos estructurados
- **contadores de error desconectados** — `Metrics.IncErrors()` (backend) e `incErrors()` (BFF) existen pero no se invocan en ningun flujo real; los contadores en `/metrics` siempre marcan 0
- **no existe endpoint `/readyz`** — `/health` solo valida DB con `db.Ping()`; no valida disponibilidad de Ollama ni estado del runtime agentico
- **OllamaProvider tiene `HealthCheck()` implementado** (`internal/infra/llm/ollama.go`) pero no se usa en ningun endpoint publico
- **no existe agregacion centralizada de logs** — no hay Loki, Promtail ni equivalente configurado
- **no existe notificacion por email al admin** — no hay codigo SMTP ni `IncidentNotifier`
- **docker-compose.yml actual** solo tiene `backend` y `bff`; no incluye stack de observabilidad

Estas limitaciones condicionan el plan y deben tratarse como trabajo tecnico explicito, no como capacidades ya resueltas.

---

## 3 -- Decision de despliegue en DigitalOcean

### Decision

La opcion elegida para v1 es:

- **DigitalOcean Droplets + Docker Compose**

### Justificacion

Esta es la mejor opcion para el estado actual del repo porque:

1. `SQLite` necesita persistencia local fiable y controlada.
2. El repo ya esta organizado para un despliegue de dos procesos con Docker.
3. `Ollama` encaja mejor en un host controlado por nosotros que en una plataforma mas opinionada.
4. El backend no esta preparado para multi-instancia con `SQLite`.
5. La capa agentica vive dentro del backend, por lo que no hay beneficio real en fragmentar el despliegue.

### Por que no App Platform

No se recomienda `App Platform` para esta v1 porque:

- complica el uso de almacenamiento local persistente
- no encaja bien con `SQLite` como datastore principal
- complica un runtime AI autoservido como Ollama
- forzaria decisiones de arquitectura prematuras

### Por que no Kubernetes

No se recomienda `Kubernetes` en esta fase porque:

- incrementa mucho la complejidad operativa
- no resuelve la limitacion principal de `SQLite`
- el sistema sigue siendo logicamente single-node

---

## 4 -- Arquitectura objetivo

### Topologia

Se define una arquitectura de dos droplets:

#### 1. app-droplet

Responsabilidades:

- reverse proxy con TLS
- BFF Node/TypeScript
- backend Go
- volumen persistente para SQLite y adjuntos
- stack de observabilidad self-hosted

#### 2. ai-droplet

Responsabilidades:

- servicio Ollama
- exposicion solo por red privada de DigitalOcean

### Diagrama logico

```text
Internet
  |
  v
Reverse Proxy (TLS)
  |
  v
BFF :3000
  |
  v
Backend Go :8080
  | \
  |  \-- SQLite + adjuntos en volumen persistente
  |
  \----> Ollama en ai-droplet por red privada

Promtail --> Loki
Prometheus --> Backend/BFF/host metrics
Grafana --> Loki + Prometheus
Alertmanager --> Email admin
DigitalOcean Monitoring/Uptime --> Email admin
```

### Exposicion de puertos

- publico:
  - `443` HTTPS
  - `80` solo para redireccion a HTTPS
- privado:
  - `3000` BFF
  - `8080` backend Go
  - `11434` Ollama
  - puertos internos del stack de observabilidad

### Restricciones operativas

- un solo nodo de aplicacion para no romper consistencia de SQLite
- backend Go no debe exponerse directamente a Internet
- Ollama no debe exponerse publicamente

---

## 5 -- Configuracion y despliegue

### Prerequisitos de infraestructura

- cuenta activa en DigitalOcean
- VPC privada creada
- dominio o subdominio para exponer el BFF
- `app-droplet` con Ubuntu LTS
- `ai-droplet` con Ubuntu LTS
- Docker Engine + Compose Plugin en ambos hosts
- volumen persistente adjunto al `app-droplet`

### Layout propuesto en app-droplet

```text
/srv/fenix/
  compose/
  config/
  data/
    fenixcrm.db
    attachments/
  backups/
  observability/
```

### Variables de entorno requeridas

#### Aplicacion

- `JWT_SECRET`
- `DATABASE_URL`
- `BFF_ORIGIN`
- `BACKEND_URL`
- `BFF_PORT`
- `PORT`
- `NODE_ENV`
- `LLM_PROVIDER`
- `OLLAMA_BASE_URL`
- `OLLAMA_MODEL`
- `OLLAMA_CHAT_MODEL`

#### Logging y operacion

- `LOG_LEVEL`
- `LOG_FORMAT`
- `SERVICE_NAME`
- `ENVIRONMENT`

#### Alertas al admin

- `ADMIN_ALERT_EMAILS`
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USER`
- `SMTP_PASS`
- `SMTP_FROM`
- `ALERT_COOLDOWN_MINUTES`

#### Readiness y dependencias criticas

- `OLLAMA_HEALTH_TIMEOUT_MS`
- `READINESS_FAIL_THRESHOLD`

### Valores operativos recomendados

```env
NODE_ENV=production
ENVIRONMENT=production
LOG_FORMAT=json
LOG_LEVEL=info
DATABASE_URL=/srv/fenix/data/fenixcrm.db
BACKEND_URL=http://backend:8080
OLLAMA_BASE_URL=http://10.0.0.20:11434
ALERT_COOLDOWN_MINUTES=15
OLLAMA_HEALTH_TIMEOUT_MS=2000
READINESS_FAIL_THRESHOLD=3
```

### Estrategia de volumen

- montar un `Block Storage Volume` en el `app-droplet`
- alojar `fenixcrm.db` y adjuntos en ese volumen
- separar configuracion del codigo de los datos persistentes

### TLS y proxy inverso

El reverse proxy debe:

- terminar TLS
- redirigir `80 -> 443`
- exponer solo el BFF
- inyectar cabeceras seguras
- mantener timeouts compatibles con SSE

### Flujo de despliegue

1. crear VPC, droplets y volumen
2. instalar Docker y Compose
3. montar volumen en `app-droplet`
4. desplegar Ollama en `ai-droplet`
5. descargar modelos requeridos en Ollama
6. desplegar backend y BFF en `app-droplet`
7. desplegar stack de observabilidad
8. configurar DNS y TLS
9. ejecutar smoke checks
10. habilitar alertas de infra y uptime

### Smoke checks minimos

- `GET /health` backend devuelve `200`
- `GET /bff/health` devuelve `200`
- login y registro funcionan
- CRUD basico responde
- Copilot SSE abre stream
- agent runs pueden dispararse y consultarse

---

## 6 -- Observabilidad y operacion

### Objetivos

La plataforma debe poder responder a estas preguntas operativas:

- esta vivo el servicio publico
- esta disponible el backend
- esta disponible Ollama
- hay errores 5xx sostenidos
- los agent runs estan fallando
- hay degradacion de latencia o saturacion
- los logs necesarios para diagnostico estan accesibles

### Componentes self-hosted

#### Prometheus

Responsabilidades:

- scrappear metricas del backend
- scrappear metricas del BFF
- scrappear metricas del host
- alimentar reglas de alerta

#### Loki

Responsabilidades:

- almacenamiento y consulta centralizada de logs

#### Promtail

Responsabilidades:

- recoger logs de contenedores y sistema
- etiquetar por servicio y entorno
- enviarlos a Loki

#### Grafana

Responsabilidades:

- dashboards operativos
- exploracion de logs y metricas

#### Alertmanager

Responsabilidades:

- deduplicacion de alertas
- agrupacion
- enrutado al email del admin

### Dashboards minimos

- disponibilidad del BFF
- disponibilidad del backend
- estado de Ollama
- tasa de 5xx
- latencia de endpoints criticos
- errores de copilot SSE
- errores de agent runs
- consumo de CPU, RAM y disco
- crecimiento del volumen SQLite

### Checks operativos

#### Health

`/health` debe responder rapido y validar:

- proceso vivo
- conectividad con DB

#### Readiness

`/readyz` debe validar:

- DB disponible
- Ollama disponible
- servicios criticos inicializados
- estado apto para servir trafico

#### Metrics

Debe existir cobertura de:

- total de requests
- total de 5xx
- panics recuperados
- fallos de Ollama
- fallos de copilot SSE
- agent runs fallidos
- latencia por endpoint
- uptime de proceso

### Alertas de infraestructura en DigitalOcean

Ademas del stack self-hosted, se deben configurar alertas de DigitalOcean para:

- CPU alta sostenida
- memoria alta sostenida
- disco casi lleno
- droplet no reachable
- uptime check fallando

Estas alertas sirven como fallback si la propia aplicacion esta demasiado degradada para emitir sus propios avisos.

---

## 7 -- Logging robusto

### Objetivo

Pasar del logging actual, basado en logs de consola y middleware generico, a un sistema de logging estructurado y correlacionable.

### Estado actual

Hoy el repo usa:

- `chi/middleware.Logger`
- `chi/middleware.Recoverer`
- `fmt.Printf`
- `fmt.Println`
- `log.Printf`
- `console.log`

Eso es suficiente para desarrollo, pero insuficiente para operacion seria.

### Objetivo tecnico

Implementar logs JSON estructurados y consistentes en backend y BFF.

### Campos minimos del log

Cada linea de log debe incluir, segun disponibilidad:

- `ts`
- `level`
- `service`
- `environment`
- `message`
- `request_id`
- `trace_id`
- `workspace_id`
- `user_id`
- `agent_run_id`
- `method`
- `route`
- `status`
- `latency_ms`
- `error`

### Backend Go

Cambios requeridos:

- introducir logger estructurado unico
- reemplazar `fmt.*` y `log.Printf` en runtime productivo
- añadir middleware de access log propio
- añadir middleware de recovery con logging estructurado de panic
- propagar `request_id` y `trace_id` al contexto
- registrar errores internos con suficiente contexto

### BFF

Cambios requeridos:

- sustituir `console.log` por logger JSON
- loggear inicio, parada, errores de proxy y fallos SSE
- instrumentar el middleware de errores para incrementar el contador real de errores
- propagar `request_id` desde proxy a backend cuando sea posible

### Correlacion

La correlacion debe funcionar entre:

- reverse proxy
- BFF
- backend
- runtime agentico
- tool execution
- auditoria

El mismo `request_id` y, cuando aplique, `trace_id`, deben aparecer en todo el recorrido.

### Politica de niveles

- `debug`: diagnostico fino en entornos controlados
- `info`: trafico normal, lifecycle y eventos esperados
- `warn`: degradaciones recuperables
- `error`: errores que afectan una operacion
- `fatal`: errores que fuerzan parada o impiden el arranque

### Retencion

- logs calientes en Loki con retencion corta o media segun coste
- snapshots o exportes periodicos si compliance lo requiere

---

## 8 -- Alertas criticas al admin por email

### Objetivo

Disponer de un sistema explicito de notificacion por email para incidencias criticas.

### Canal principal

- email

### Canal de respaldo

- alertas de DigitalOcean por email

### Eventos que deben notificar

El sistema debe enviar email al admin cuando ocurra alguno de estos casos:

- fallo de arranque del backend
- fallo de migraciones
- base de datos no disponible
- `readyz` en estado no apto de forma sostenida
- Ollama no disponible durante una ventana configurable
- panic recuperado en backend o BFF
- tasa de 5xx sostenida por encima del umbral definido
- fallo sostenido en Copilot SSE
- agent runs fallando por encima del umbral definido
- disco del volumen en nivel critico

### Eventos que no deben notificar directamente por email

No deben generar email por si solos:

- errores 4xx normales
- timeouts puntuales aislados
- una unica peticion fallida sin patron sostenido
- logs de negocio no criticos

### Anti-spam, cooldown y deduplicacion

El sistema debe:

- deduplicar alertas iguales dentro de una ventana temporal
- agrupar alertas repetidas
- aplicar cooldown por tipo de incidente
- enviar aviso de resolucion cuando el sistema vuelva a estado sano

### Fallback operativo

Si la propia aplicacion no puede enviar correos:

- las alertas de infraestructura y uptime de DigitalOcean deben seguir notificando
- el stack Prometheus + Alertmanager debe poder emitir el correo aunque el backend este degradado

### Implementacion recomendada

Separar dos capas:

#### Capa 1. Alertas de plataforma

- emitidas por Prometheus/Alertmanager
- cubren salud del sistema y umbrales

#### Capa 2. Alertas de aplicacion

- emitidas por un `IncidentNotifier` o `CriticalAlertService`
- cubren fallos de arranque, migraciones y eventos internos de alta severidad

### Formato minimo del email

Cada email critico debe incluir:

- servicio afectado
- entorno
- severidad
- timestamp
- resumen del incidente
- clave de deduplicacion
- contexto tecnico basico
- enlace al dashboard o query de logs

---

## 9 -- Backups, restauracion y continuidad

### SQLite

La DB requiere estrategia explicita de backup porque es single-file y single-node.

### Politica recomendada

- snapshot programado del volumen
- backup logico de SQLite antes de despliegues
- retencion definida segun RPO/RTO

### Restauracion

Debe existir runbook para:

- detener trafico
- restaurar snapshot o backup logico
- validar integridad de DB
- levantar backend y BFF
- ejecutar smoke checks

### Validacion periodica

No basta con hacer backups. Debe probarse la restauracion.

---

## 10 -- Checklist de aceptacion

### Salud basica

- [ ] `GET /health` responde correctamente
- [ ] `GET /bff/health` responde correctamente
- [ ] `GET /readyz` refleja salud real de DB + Ollama
- [ ] `GET /metrics` backend disponible
- [ ] `GET /bff/metrics` BFF disponible

### Funcionalidad

- [ ] registro y login funcionan
- [ ] CRUD CRM basico funciona
- [ ] knowledge ingest/reindex funciona
- [ ] Copilot SSE funciona extremo a extremo
- [ ] agent runs pueden ejecutarse y consultarse

### Persistencia

- [ ] la DB persiste tras reinicio
- [ ] los adjuntos persisten tras redeploy

### Observabilidad

- [ ] logs de backend visibles en Loki
- [ ] logs de BFF visibles en Loki
- [ ] dashboards base visibles en Grafana
- [ ] metricas reales de error se incrementan ante fallos

### Alertas

- [ ] caida simulada de Ollama degrada `readyz`
- [ ] la degradacion sostenida de Ollama envia email al admin
- [ ] un panic controlado genera alerta y log estructurado
- [ ] un fallo de uptime genera alerta de DigitalOcean por email
- [ ] las alertas repetidas no generan spam

### Continuidad

- [ ] se puede restaurar backup de SQLite
- [ ] tras restauracion, los smoke checks vuelven a pasar

---

## 11 -- Riesgos y limites actuales

### SQLite implica single-node

Mientras la persistencia principal siga en SQLite:

- no debe escalarse horizontalmente el backend
- no debe plantearse HA activa-activa

### Ollama es dependencia critica

La capa agentica y de copilot depende operativamente de Ollama:

- si cae Ollama, la app puede seguir viva pero perder funcionalidad clave
- eso obliga a readiness real y alertas dedicadas

### Logging y alertas hoy son insuficientes

El repo no tiene aun:

- logging estructurado consistente (backend usa `middleware.Logger` chi + `fmt`/`log`; BFF usa `console.log`)
- contadores de error reales en metricas (los contadores existen pero no se invocan — `fenixcrm_request_errors_total` y `bff_request_errors_total` siempre son 0)
- endpoint `/readyz` (solo existe `/health` que valida DB pero no Ollama)
- agregacion centralizada de logs (Loki/Promtail no configurados)
- correos criticos al admin (no existe codigo SMTP ni servicio de alertas de aplicacion)

Trabajo tecnico identificado antes de operacion seria:
1. logging JSON estructurado en backend (`slog`) y BFF (`pino`)
2. conectar `IncErrors()`/`incErrors()` al flujo real de 5xx
3. implementar `/readyz` que valide DB + Ollama
4. docker-compose de observabilidad (Prometheus + Loki + Promtail + Grafana + Alertmanager)
5. `IncidentNotifier` con SMTP para alertas criticas al admin

Por tanto, la operacion propuesta requiere implementacion adicional y no debe asumirse como ya disponible.

### Coste operativo

El stack self-hosted de observabilidad añade:

- mas consumo de recursos
- mas componentes a operar
- mas trabajo de mantenimiento

Se acepta este coste porque se ha priorizado observabilidad completa self-hosted.

---

## 12 -- Cambios tecnicos previstos

### Documento y alcance

- nuevo documento tecnico: `docs/deployment-plan-digitalocean.md`

### Variables operativas previstas

- `LOG_LEVEL`
- `LOG_FORMAT`
- `SERVICE_NAME`
- `ENVIRONMENT`
- `ADMIN_ALERT_EMAILS`
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USER`
- `SMTP_PASS`
- `SMTP_FROM`
- `ALERT_COOLDOWN_MINUTES`
- `OLLAMA_HEALTH_TIMEOUT_MS`
- `READINESS_FAIL_THRESHOLD`

### Endpoint operativo previsto

- `/readyz`

### Cambios funcionales esperados en codigo

#### Fase A — logging + metricas + readyz (prerequisito para observabilidad util)

- `internal/infra/config/config.go` — añadir `LOG_LEVEL`, `LOG_FORMAT`, `SERVICE_NAME`, `ENVIRONMENT`, `OLLAMA_HEALTH_TIMEOUT_MS`
- `cmd/fenix/main.go` — inicializar `slog` segun `LOG_FORMAT`/`LOG_LEVEL` al arranque
- `internal/server/server.go` — reemplazar `fmt.Printf`/`fmt.Println` por `slog`
- `internal/infra/llm/ollama.go` — reemplazar `log.Printf` por `slog.Warn`
- `internal/api/handlers/copilot_chat.go` — reemplazar `log.Printf` por `slog.Error`
- `internal/api/routes.go` — reemplazar `middleware.Logger` por middleware de access log JSON propio; añadir middleware que conecta `IncErrors()` al flujo real de 5xx
- `internal/api/handlers/health.go` — nuevo `NewReadyzHandler` (valida DB + Ollama con timeout)
- `internal/api/routes.go` — registrar `/readyz` como ruta publica
- `bff/src/config.ts` — añadir `logLevel`, `serviceName`, `environment`
- `bff/src/server.ts` — reemplazar `console.log` por `pino`
- `bff/src/middleware/errorHandler.ts` — llamar `incErrors()` en ramas 5xx
- `bff/package.json` — añadir dependencia `pino`

#### Fase B — stack de observabilidad (post Fase A)

- docker-compose de observabilidad: Prometheus + Loki + Promtail + Grafana + Alertmanager
- reglas de alerta en Prometheus/Alertmanager

#### Fase C — alertas al admin por email (post Fase B)

- servicio de alertas criticas por email (`IncidentNotifier` / `CriticalAlertService`)
- integracion SMTP en backend Go

---

## Notas finales

- Este plan asume `Droplet + Docker Compose` como decision cerrada para v1.
- Este plan asume `Ollama` en DigitalOcean como runtime AI de v1.
- Este plan asume stack de observabilidad self-hosted.
- Este plan asume alertas criticas por email como canal principal al admin.

Si en una fase posterior cambia la persistencia a PostgreSQL o se externaliza el runtime AI, este documento debera revisarse porque varias restricciones operativas dejarian de aplicar.
