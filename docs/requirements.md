---
title: "Governed AI CRM Operations Layer — Requerimientos (Agent-Ready)"
version: "2.0"
date: "2026-03-19"
timezone: "Europe/Madrid"
language: "es-ES"
status: "active"
audience: ["product", "architecture", "engineering", "security", "revops"]
tags: ["crm", "llm", "agents", "rag", "governance", "self-hosted", "byo-model"]
canonical_id: "acrmos-req-v2.0"
supersedes: ["agentic_crm_requirements_agent_ready.md"]
---

# Governed AI CRM Operations Layer — Requerimientos (Agent-Ready)

> **Propósito**: Este documento consolida el análisis de mercado (CRM + IA/LLM + Agents) y deriva un set completo de requerimientos **funcionales (FR)**, **no funcionales (NFR)** y **casos de uso (UC)** para un producto que cubra el hueco: **capa de IA gobernada para customer operations, evidence-first, action-capable, gobernada por políticas y con trazabilidad end-to-end**, con opción **self-hosted** y/o **BYO-model**.

> **Nota**: Este documento reemplaza y consolida `agentic_crm_requirements_agent_ready.md`.

> **Alineación estratégica 2026-04-06**: este documento sigue cubriendo el espacio completo de capacidades, pero la prioridad comercial y arquitectónica vigente es la definida en `docs/architecture.md` y `docs/plans/fenixcrm_strategic_repositioning_spec.md`: FenixCRM se posiciona como **governed AI layer for customer operations**, no como broad CRM replacement; el wedge inicial es **Support Copilot / Support Agent**, con **Sales Copilot** como segundo wedge. Cuando este documento entre en conflicto con esa priorización, prevalece la arquitectura actualizada.

---

## Índice

1. [Resumen ejecutivo](#1--resumen-ejecutivo)
2. [Hallazgos del mercado y patrones](#2--hallazgos-del-mercado-y-patrones)
3. [Definición del producto y alcance](#3--definición-del-producto-y-alcance)
4. [Actores y JTBD](#4--actores-y-jtbd)
5. [Mapa de capacidades](#5--mapa-de-capacidades)
6. [Casos de uso (UC)](#6--casos-de-uso-uc)
7. [Requerimientos funcionales (FR)](#7--requerimientos-funcionales-fr)
8. [Requerimientos no funcionales (NFR)](#8--requerimientos-no-funcionales-nfr)
9. [Arquitectura conceptual](#9--arquitectura-conceptual)
10. [Modelo de datos conceptual](#10--modelo-de-datos-conceptual)
11. [Gobernanza de IA](#11--gobernanza-de-ia)
12. [Calidad y evaluación (Evals)](#12--calidad-y-evaluación-evals)
13. [Roadmap por fases](#13--roadmap-por-fases)
14. [Riesgos y mitigaciones](#14--riesgos-y-mitigaciones)
15. [Apéndices](#15--apéndices)

---

## 1 — Resumen ejecutivo

### Problema

Los CRMs se están "IA-izando" de dos maneras:
- **Copilots** embebidos en la UI que redactan/resumen y, en algunos casos, ejecutan acciones.
- **Agentes** especializados (support, prospecting, content, insights) con algún grado de orquestación, "studio", y control.

Sin embargo, el mercado queda fragmentado entre:
- suites enterprise con fuerte "agent layer" pero **vendor lock-in**,
- CRMs OSS/modernos con buena base operativa pero **sin runtime agentic consistente**,
- add-ons de IA puntuales que no resuelven **gobernanza ni trazabilidad**.

### Oportunidad

Una **capa de IA gobernada para customer operations** que combine:
1. CRM core extensible
2. Capa de conocimiento (RAG) **con evidencia y citación obligatoria**
3. Runtime agentic (acciones, tools, triggers) con **políticas**
4. Contratos de comportamiento, versionado y evals para gobernar workflows
5. Auditoría y observabilidad end-to-end
6. Data sovereignty (self-host/BYO-model, "no-cloud" por policy)
7. Control de costos (presupuestos/quotas por rol/agente/tenant)

### Principios de diseño (no negociables)

- **Evidence-first**: si no hay evidencia suficiente → **abstención** o **handoff humano**.
- **Actions via tools**: la IA no "escribe directo"; ejecuta **herramientas registradas**.
- **Policy-governed**: RBAC/ABAC + PII/no-cloud + approvals.
- **Operabilidad real**: logs, métricas, tracing, replay/dry-run, gating por evals.
- **Model-agnostic**: local o cloud sin reescribir el producto.

---

## 2 — Hallazgos del mercado y patrones

> Nota: Se listan patrones por su relevancia, no como dependencias de vendors.

### Patrón A — Copilot "in-flow" + acciones ejecutables
**Síntesis**: el asistente vive dentro del record/pipeline, con acciones "one-click" y chaining.
**Implicación**: requiere **tool registry** + **audit log** + **idempotencia**.

### Patrón B — Agentes + Studio + guardrails + analítica
**Síntesis**: agentes por dominio y un "studio" para adaptar habilidades/políticas y medir rendimiento.
**Implicación**: la plataforma necesita contratos, evaluación y gobernanza; un **Agent Studio** amplio puede llegar después del wedge inicial.

### Patrón C — Customer support agentic: handoff humano + reglas
**Síntesis**: escalado configurable, control por canal, transferencia de contexto.
**Implicación**: "human-in-the-loop" es un módulo core, no accesorio.

### Patrón D — Grounding/Trust: respuestas con evidencia
**Síntesis**: tendencia a exigir fuentes/citas, especialmente en enterprise.
**Implicación**: imponer **evidence pack** como contrato de salida.

### Patrón E — IA como add-on puntual en CRMs tradicionales/OSS
**Síntesis**: features aisladas (dashlets, generadores) sin coherencia agentic.
**Implicación**: el hueco está en la **capa agentic y de gobernanza**.

---

## 3 — Definición del producto y alcance

### Nombre de trabajo
**Governed AI CRM Operations Layer (self-hosted + AI-native)**

### Propuesta de valor
1. **Support Copilot** y **Support Agent** para casos con evidencia, approvals y audit trail.
2. **Sales Copilot** para account/deal context con grounding y next steps.
3. **RAG** sobre datos estructurados y no estructurados, con **citas obligatorias**.
4. **Governance runtime**: RBAC/ABAC, PII/no-cloud, approvals, auditoría, handoff.
5. **Cost governance**: budgets, quotas y atribución por run/workspace/tool.
6. **Model-agnostic**: LLM local y/o cloud con budgets explícitos.

### Fuera de alcance (por defecto)
- ERP completo
- Marketing automation tipo "suite" (solo features mínimas integrables)
- Contact center/CTI completo (solo conectores e integración)
- Broad CRM replacement en la estrategia inicial
- Mobile parity como criterio de salida a mercado
- Marketplace o Studio breadth antes de validar el wedge

---

## 4 — Actores y JTBD

### Actores
- **Sales Rep**
- **Sales Manager**
- **Support Agent**
- **Marketing Manager**
- **RevOps / Data Analyst**
- **Admin / SecOps**
- **Automation/AI Builder** (rol mixto, típicamente Admin/RevOps)

### JTBD (jobs-to-be-done)
- "Quiero saber **qué hacer ahora** y **por qué** (evidencia)."
- "Quiero que el sistema ejecute tareas repetitivas **con límites y aprobación**."
- "Quiero confianza operacional: **abstención** si no hay evidencia."
- "Quiero trazabilidad: quién hizo qué, cuándo, con qué datos, y cómo revertirlo."

---

## 5 — Mapa de capacidades

1. **CRM Core & Extensibility**
2. **Knowledge & Retrieval Layer (RAG)**
3. **Copilot UI (in-flow)**
4. **Agent Runtime (triggers + orchestration + tools)**
5. **Agent Studio & Behavior Contracts (builder + policy + eval + deploy)**
6. **Governance & Security (RBAC/ABAC + audit + approvals)**
7. **Integrations (connectors + API/webhooks)**
8. **Observability & Analytics (usage, quality, cost, outcomes)**

---

## 6 — Casos de uso (UC)

### Catálogo de casos de uso

| ID | Nombre | Descripción | Prioridad |
|---|---|---|---|
| **UC-S1** | Sales Copilot | Resumen de account/deal + próximos pasos con evidencia | P0 |
| **UC-S2** | Prospecting Agent | Investigar + redactar outreach + crear tasks | P1 |
| **UC-S3** | Deal Risk Agent | Detectar riesgo de deal + sugerir mitigación | P1 |
| **UC-C1** | Support Agent | Responder case + actualizar case + handoff humano | P0 |
| **UC-K1** | KB Agent | Convertir soluciones en artículos + revisión | P1 |
| **UC-D1** | Data Insights Agent | Responder preguntas analíticas con evidencia | P1 |
| **UC-G1** | Governance | Auditar Agent Runs + replay + rollback | P0 |
| **UC-A1** | Agent Studio | Crear skill + policy + eval + contrato de comportamiento y promover a producción | P1 |

### Diagrama — Nivel 0 (Contexto)

```mermaid
flowchart LR
  subgraph Actores
    SR[Sales Rep]
    SM[Sales Manager]
    SA[Support Agent]
    MM[Marketing]
    RO[RevOps / Analyst]
    ADM[Admin / SecOps]
  end

  subgraph Externos
    IDP[SSO / IdP]
    COMMS[Email/Calendar/Telephony/WhatsApp]
    KB[Docs/KB Storage]
    DWH[Warehouse/Lake]
    LLM[LLM Provider\nLocal/Cloud]
  end

  SYS[(Governed AI CRM Operations Layer)]

  SR --> SYS
  SM --> SYS
  SA --> SYS
  MM --> SYS
  RO --> SYS
  ADM --> SYS

  SYS <--> IDP
  SYS <--> COMMS
  SYS <--> KB
  SYS <--> DWH
  SYS <--> LLM
```

### Diagrama — Nivel 1 (Casos de uso principales)

```mermaid
flowchart TB
  SR[Sales Rep] --- U1((Gestionar Leads/Deals))
  SM[Sales Manager] --- U2((Forecast & Coaching))
  SA[Support Agent] --- U3((Gestionar Cases/Tickets))
  MM[Marketing] --- U4((Campañas & Contenido))
  RO[RevOps] --- U5((Reporting & Calidad de Datos))
  ADM[Admin/SecOps] --- U6((Permisos, Políticas, Auditoría))

  SR --- U7((Copilot in-flow))
  SA --- U7
  MM --- U7
  SM --- U7
  RO --- U7

  SR --- U8((Prospecting/Deal Agents))
  SA --- U9((Customer Support Agent))
  MM --- U10((Content/KB Agents))
  RO --- U11((Data Insights Agent))

  ADM --- U12((Agent Studio: Skills/Policies/Evals))
  ADM --- U13((Integraciones: API/Webhooks/Connectors))
```

### Diagrama — Nivel 2 (Detalle: UC-C1 "Support Agent resuelve un Case")

```mermaid
flowchart TD
  SA[Support Agent] --> A((Iniciar ejecución / Autorizar agente))
  A --> B((Recolectar contexto del Case\n+ conversación + adjuntos))
  B --> C((Recuperar evidencia\nKB + casos similares + producto + SLAs))
  C --> D((Construir Evidence Pack\n+ dedupe + ranking + freshness))
  D --> E((Generar respuesta con citas\n+ propuesta de acción))
  E --> F{Policy Check\nPermisos/PII/Riesgo/Canal}
  F -- Bloquea --> G((Abstenerse + Escalar a humano\ncon evidence pack))
  F -- OK --> H((Proponer respuesta))
  H --> I{Requiere aprobación?}
  I -- Sí --> J((Workflow de aprobación))
  I -- No --> K((Enviar respuesta al cliente))
  J --> K
  K --> L((Actualizar Case\nestado, etiquetas, next steps))
  L --> M((Registrar Agent Run Log\ntrazas + métricas + costos))
```

---

## 7 — Requerimientos funcionales (FR)

> **Formato**: FR-XXX — *Título* (Prioridad)
> **Descripción**, **AC (Acceptance Criteria)**, **Dependencias**.
> **Prioridad**: P0 (MVP), P1 (v1), P2 (v2)

---

### 7.1 CRM Core & Extensibilidad

**FR-001 — Entidades core CRM (P0)**
**Descripción:** CRUD + relaciones para Account/Company, Contact, Lead, Opportunity/Deal, Case/Ticket, Activity (Task/Event), Note, Attachment.
**AC:**
- CRUD con búsqueda, filtros, paginación.
- Timeline por entidad (actividades + cambios).
- Auditoría de cambios (quién/cuándo/qué).
**Dep:** FR-060, FR-070.

**FR-002 — Pipelines y etapas (P0)**
**Descripción:** pipeline configurable, reglas de etapa (required fields, probabilidad, SLA interno).
**AC:**
- Configurar pipeline por equipo/unidad.
- Evento al cambiar etapa (trigger).
**Dep:** FR-120.

**FR-003 — Reporting base (P0/P1)**
**Descripción:** dashboards y KPIs: embudo, SLA, backlog, aging.
**AC:**
- Dashboards mínimos por dominio (Sales/Support).
- Export/CSV.
**Dep:** FR-070.

**FR-004 — Extensión del modelo (custom objects/fields) (P1)**
**Descripción:** crear objetos/campos, layouts y validaciones; reflejo en APIs y retrieval.
**AC:**
- Creación sin downtime.
- Indexación y reporting para campos extensibles.
**Dep:** FR-090, FR-061.

**FR-005 — Workflows no-IA (P1)**
**Descripción:** motor de reglas para notificaciones/asignación/SLAs sin usar LLM.
**AC:**
- Triggers + acciones estándar (create task, assign owner, webhook).
**Dep:** FR-051, FR-120.

---

### 7.2 Knowledge & Retrieval Layer (RAG)

**FR-090 — Indexación híbrida (P0)**
**Descripción:** keyword (BM25) + vector (embeddings) + filtros por permisos; indexación incremental.
**AC:**
- Query con filtros por entidad/tenant/owner.
- Ranking híbrido configurable.
- Incremental reindex por cambios.
**Dep:** FR-060, FR-100, FR-121.

**FR-091 — Ingesta multifuente (P0/P1)**
**Descripción:** ingesta de emails, calendar, llamadas/transcripciones, docs/KB, chat → normalización a KnowledgeItem.
**AC:**
- P0: emails + docs.
- Metadata mínima: fuente, timestamps, owner, sensibilidad, TTL.
**Dep:** FR-050, FR-061.

**FR-092 — Evidence Pack obligatorio (P0)**
**Descripción:** toda salida de IA (respuesta/recomendación/acción) adjunta evidencias: IDs + snippets + score + timestamp.
**AC:**
- UI muestra "fuentes" por defecto.
- Si evidencia insuficiente → FR-210 (abstención).
**Dep:** FR-200, FR-230.

**FR-093 — Freshness/TTL por tipo (P1)**
**Descripción:** políticas de frescura para datos volátiles (stage/status).
**AC:**
- TTL por tipo de KnowledgeItem.
- Advertencias cuando evidencia es antigua.
**Dep:** FR-091.

**FR-094 — Dedupe y consolidación (P1)**
**Descripción:** deduplicar chunks/evidencia; consolidar respuestas repetidas.
**AC:**
- Reducir duplicación en evidence packs.
- Métrica de "duplication rate" disponible.
**Dep:** FR-310.

---

### 7.3 Copilot (in-flow)

**FR-200 — Copilot embebido (P0)**
**Descripción:** UI contextual dentro del CRM mobile (pantallas de record/pipeline) + chat panel. Streaming via SSE proxied a través del BFF (Express.js) al cliente React Native.
**AC:**
- En account/deal/case (mobile screens): "Resumen", "Siguiente acción", "Draft email", "Actualizar campo".
- "Explain why" muestra evidencias en cards expandibles (React Native Paper).
- SSE streaming funcional en mobile (react-native-sse o EventSource polyfill).
**Dep:** FR-092, FR-060, FR-300.

**FR-201 — Resúmenes operativos (P0)**
**Descripción:** resúmenes de account/deal/case/meeting prep con riesgos y next steps.
**AC:**
- Resumen incluye evidencias citadas.
- Debe indicar "unknown" si faltan datos.
**Dep:** FR-092.

**FR-202 — Copilot Actions (tools) (P0)**
**Descripción:** biblioteca de acciones ejecutables y chaining (crear task, actualizar etapa, crear case, enviar email).
**AC:**
- Tool registry con schemas.
- La IA no muta datos sin invocar tool.
- Log de tool call (params/result).
**Dep:** FR-230, FR-070.

**FR-203 — Plantillas y voz de marca (P1)**
**Descripción:** drafting con plantillas por segmento y estilo/tono administrable.
**AC:**
- Configurable por unidad/tenant.
- Versionado de plantillas.
**Dep:** FR-240.

---

### 7.4 Contratos de comportamiento IA (enforcement)

> **Nota canónica**: `spec_source` soporta dos modos: texto libre legacy y contrato machine-readable tipo **Carta**. Carta complementa al DSL; no lo reemplaza. Detalle técnico: `docs/carta-spec.md` y `docs/carta-implementation-plan.md`.

**FR-210 — Abstención obligatoria (P0)**
**Descripción:** el sistema debe abstenerse si evidencia insuficiente/contradictoria o viola policy.
**AC:**
- Output incluye razón de abstención y próximos pasos sugeridos.
- Opción de "escalar a humano" con evidence pack.
**Dep:** FR-092, FR-232.

**FR-211 — Safe tool routing (P0)**
**Descripción:** el modelo solo puede ejecutar acciones via tools allowlisted; tool schemas validados.
**AC:**
- Validación de parámetros y scopes antes de ejecutar.
- Denegar si parámetros peligrosos/no permitidos.
**Dep:** FR-202, FR-060, FR-071.

**FR-212 — Behavior Contracts / Carta (P1)**
**Descripción:** `spec_source` debe poder declararse en formato machine-readable tipo Carta para expresar requisitos de evidencia, permisos, delegación, invariantes y límites operativos sobre un workflow/agent skill, manteniendo retrocompatibilidad con el formato libre actual.
**AC:**
- Judge valida consistencia estática entre DSL y Carta antes de activar o promover el workflow.
- `GROUNDS` puede forzar abstención previa al DSL si la evidencia no cumple requisitos mínimos.
- `DELEGATE TO HUMAN` puede activar escalado proactivo antes de retrieval/DSL.
- `BUDGET` sincroniza límites operativos al runtime del agente.
- `INVARIANT` se compila a reglas de policy enforcement en activación.
- Si `spec_source` no está en formato Carta, el sistema mantiene la ruta legacy de parsing sin romper compatibilidad.
**Dep:** FR-210, FR-211, FR-232, FR-233, FR-240.

---

### 7.5 Agent Runtime (agentes y orquestación)

**FR-230 — Runtime de agentes (P0/P1)**
**Descripción:** ejecutar agentes por evento/schedule/manual; soportar herramientas; mantener estado.
**AC:**
- Dry-run disponible.
- Reintentos con backoff; DLQ.
- Idempotencia para writes.
**Dep:** FR-121, FR-070, FR-060.

**FR-231 — Catálogo mínimo de agentes (P0)**
**Descripción:** incluir agentes: Prospecting, Support, KB, Insights.
**AC:**
- Cada agente tiene: objetivo, tools permitidas, límites, KPIs.
**Dep:** FR-240, FR-092.

**FR-232 — Handoff humano + reglas de escalado (P0)**
**Descripción:** escalado configurable por canal/audiencia/estado; transferencia de contexto.
**AC:**
- Se preserva conversación + evidence pack.
- Se registra motivo de escalado.
**Dep:** FR-070, FR-060.

**FR-233 — Límites operativos (quotas) (P1)**
**Descripción:** límites por agente/rol/tenant (tokens, costo, ejecuciones/día).
**AC:**
- Circuit breaker ante errores repetidos.
- Notificaciones por umbral.
**Dep:** FR-310.

**FR-234 — Scheduling y triggers (P1)**
**Descripción:** triggers por evento (record created/updated), schedule, y manual.
**AC:**
- UI para configurar triggers.
- Auditoría de configuración.
**Dep:** FR-120, FR-070.

---

### 7.6 Agent Studio (builder + policy + eval + deploy)

> **Nota**: Agent Studio incluye edición/versionado de contratos de comportamiento tipo Carta como capa declarativa complementaria al DSL. Ver `docs/carta-spec.md`.

**FR-240 — Prompt/Policy versioning (P0)**
**Descripción:** repositorio versionado de prompts/policies por agente/rol/tenant; rollback y entornos dev/test/prod.
**AC:**
- Diff + auditoría de cambios.
- Rollback en un click (con log).
**Dep:** FR-070.

**FR-241 — Skills/Tools Builder (P1)**
**Descripción:** crear tools y skills (workflows multi-step) low-code; librería reutilizable.
**AC:**
- Tool schema + auth + rate limit + retries.
- Skills componen tools con pasos.
**Dep:** FR-051, FR-230.

**FR-242 — Evals y gating de releases (P0/P1)**
**Descripción:** datasets por dominio y scoring (groundedness, exactitud, abstención, policy).
**AC:**
- Run eval antes de promover a prod.
- Umbrales configurables.
**Dep:** FR-092, FR-310.

**FR-243 — Simulación / replay (P1)**
**Descripción:** simular ejecuciones con snapshots; replay de agent runs.
**AC:**
- Reproducibilidad bajo modo determinista configurable.
**Dep:** FR-070, FR-090.

---

### 7.7 Integraciones y APIs

**FR-050 — Framework de conectores (P0/P1)**
**Descripción:** conectores (email, calendar, docs, telephony, WhatsApp) con scopes y auditoría.
**AC:**
- Gestión de tokens/secretos.
- Logs por conector.
**Dep:** FR-320.

**FR-051 — API pública + webhooks (P0)**
**Descripción:** API (REST/GraphQL) + webhooks de eventos.
**AC:**
- OAuth/SSO + rate limits.
- Documentación (OpenAPI).
**Dep:** FR-060.

**FR-052 — Plugins/Marketplace (P2)**
**Descripción:** empaquetar skills/agentes/widgets como plugins instalables.
**AC:**
- SDK con contracts estables.
**Dep:** FR-241, FR-240.

---

### 7.8 Seguridad, compliance y auditoría

**FR-060 — RBAC/ABAC (P0)**
**Descripción:** permisos por rol + atributos (equipo/territorio), por objeto/campo/registro; enforcement en UI/API/retrieval/tools.
**AC:**
- Ningún usuario/agente recupera evidencia de records sin permiso.
- Ningún tool call se ejecuta sin permiso.
**Dep:** FR-070.

**FR-061 — Clasificación de datos y PII/no-cloud (P0)**
**Descripción:** tags de sensibilidad (PII/PHI/secret), políticas de retención/anonimización/no-cloud.
**AC:**
- Enmascaramiento antes del prompt.
- Reglas por tenant/unidad.
**Dep:** FR-320.

**FR-070 — Audit trail + Agent Run Log (P0)**
**Descripción:** log auditable de: queries, evidencias, tool calls, outputs, costos, decisiones.
**AC:**
- Consultable con filtros + exportable.
- Modo append-only opcional.
**Dep:** FR-310.

**FR-071 — Approvals workflow (P0/P1)**
**Descripción:** aprobación para acciones sensibles (envío externo, cambios masivos, transferencias de ownership).
**AC:**
- Definir política por acción/rol.
- Auditoría de decisiones (approve/deny).
**Dep:** FR-060, FR-070.

---

### 7.9 Mobile App & BFF Gateway

**FR-300 — Mobile App (React Native) (P0)**
**Descripción:** Aplicación móvil nativa usando React Native (Expo managed workflow) + React Native Paper (Material Design 3). Android-first, iOS posterior.
**AC:**
- Navegación principal: Stack + Drawer (React Navigation).
- Pantallas CRM: listados y detalle de Account, Contact, Deal, Case con búsqueda, filtros, paginación.
- Deals y Cases deben incluir además pantallas de creación y edición (update) con validaciones, estados de carga/error y feedback de guardado.
- Flujos mínimos obligatorios en mobile para P0:
  - Deals: listar, crear, editar (campos comerciales), ver detalle.
  - Cases: listar, crear, editar (status/priority/owner/description), ver detalle.
- Panel Copilot integrado en pantallas de detalle.
- Autenticación: login/registro via BFF → Go backend.
- Soporte offline básico: cache local de últimos records consultados.
**Dep:** FR-301, FR-200.

**FR-301 — BFF Gateway (Express.js) (P0)**
**Descripción:** Backend-for-Frontend en Express.js/TypeScript que actúa como API gateway entre la mobile app y el Go backend. No contiene lógica de negocio ni accede a SQLite directamente.
**AC:**
- Proxy transparente de todas las llamadas al Go backend REST API.
- Relay de autenticación: JWT token management, refresh logic.
- Agregación de requests: combinar múltiples llamadas Go API en una sola respuesta mobile-optimizada (ej: pantalla de detalle = account + contacts + deals + timeline).
- Proxy SSE: retransmitir streaming de Copilot chat desde Go backend al cliente mobile.
- Headers mobile-specific: device info, app version, push token.
- Health check propio (`/bff/health`).
**Dep:** FR-051.

**FR-302 — Push Notifications (P1)**
**Descripción:** Notificaciones push para eventos críticos: approval requests, handoff asignado, agent run completado.
**AC:**
- Integración con Firebase Cloud Messaging (FCM) para Android.
- BFF despacha notificaciones al recibir eventos del Go backend (via polling o webhook).
- Usuario puede configurar preferencias de notificación por tipo de evento.
**Dep:** FR-301, FR-232, FR-071.

**FR-303 — Offline Cache (P1)**
**Descripción:** Cache local en dispositivo para permitir consulta de datos CRM sin conexión.
**AC:**
- Cache de últimos N records consultados (configurable, default 50).
- Indicador visual de datos offline/stale.
- Sincronización automática al recuperar conectividad.
- No se permiten mutaciones offline (solo lectura).
**Dep:** FR-300.

**FR-304 — CRM List Centralized CRUD and Bulk Delete (P1)**
**Descripción:** Las pantallas de listado CRM (Account, Contact, Lead, Deal, Case) son la única superficie operacional para crear, editar y eliminar entidades. Los detail screens pasan a modo read-only.
**AC:**
- Checkboxes always-visible por fila para multi-selección; estado de selección keyed by entity id.
- Control "Select all" aplica solo a filas visibles (filtered rows actuales, no paginación completa).
- Control "Clear" limpia la selección.
- Contador de selección visible en el header durante selección activa.
- Botón "Edit" (icono pencil) por fila navega a `/crm/<entity>/edit/<id>`; la navegación row-body navega al detail read-only.
- "Delete selected" visible solo cuando hay al menos una fila seleccionada; oculto con cero selección.
- Confirmación destructiva vía `Alert.alert` con conteo de items seleccionados antes de ejecutar.
- Delete ejecuta mutaciones por entidad vía `Promise.allSettled`; en fallo parcial mantiene seleccionados solo los ids fallidos para retry.
- Checkboxes, Select all, Clear, Edit y Delete selected deshabilitados durante delete pendiente.
- Detail screens (account, contact, lead, deal, case) no exponen acción primaria de edición; son read-only.
**Dep:** FR-300, FR-001, FR-002.

---

### Resumen FR por dominio y prioridad

| Dominio | P0 | P1 | P2 | Total |
|---|---|---|---|---|
| CRM Core | FR-001, FR-002, FR-003 | FR-003, FR-004, FR-005 | — | 5 |
| Knowledge & RAG | FR-090, FR-092 | FR-091, FR-093, FR-094 | — | 5 |
| Copilot | FR-200, FR-201, FR-202 | FR-203 | — | 4 |
| Comportamiento IA | FR-210, FR-211 | FR-212 | — | 3 |
| Agent Runtime | FR-230, FR-231, FR-232 | FR-233, FR-234 | — | 5 |
| Agent Studio | FR-242 | FR-240, FR-241, FR-243 | — | 4 |
| Integraciones | FR-050, FR-051 | — | FR-052 | 3 |
| Seguridad & Audit | FR-060, FR-061, FR-070, FR-071 | FR-071 | — | 4 |
| Mobile & BFF | — | FR-300, FR-301, FR-302, FR-303, FR-304 | — | 5 |
| **Total** | **19** | **~15** | **1** | **31** |

---

## 8 — Requerimientos no funcionales (NFR)

### Performance & Latency
- **NFR-001 (P0)** Copilot Q&A p95 ≤ **2.5s** (respuestas cortas).
- **NFR-002 (P0)** Resúmenes p95 ≤ **5s** (account/deal/case) con evidencia.
- **NFR-003 (P1)** Indexación incremental: cambios visibles ≤ **60s** (objetivo).

### Reliability & Correctness
- **NFR-010 (P0)** Idempotencia para escrituras via idempotency keys.
- **NFR-011 (P0)** Reintentos con backoff + DLQ.
- **NFR-012 (P1)** Consistencia eventual documentada por tipo.

### Security & Privacy
- **NFR-020 (P0)** Cifrado en tránsito y reposo.
- **NFR-021 (P0)** Secrets en vault + rotación.
- **NFR-022 (P0)** Enforcement permisos en retrieval y tools.
- **NFR-023 (P0/P1)** "No-cloud" por policy para datos sensibles.

### Observability & Governance
- **NFR-030 (P0)** Métricas por agente: éxito, abstención, escalado, latencia, costo.
- **NFR-031 (P0)** Tracing por request: retrieval → ranking → prompt → tools → output.
- **NFR-033 (P0)** Readiness operativa del backend: exponer `/readyz` que valide base de datos y dependencias críticas de IA antes de recibir tráfico.
- **NFR-032 (P1)** Alertas por regresión de calidad/costos.

### Cost Control
- **NFR-040 (P0/P1)** Presupuestos por tenant/agente/rol (tokens/€).
- **NFR-041 (P1)** Degradación controlada (modelo más barato, menor contexto, abstención).

### Portabilidad & Deployment
- **NFR-050 (P0)** Self-host (Docker/K8s) + opción SaaS; BYO-LLM.
- **NFR-051 (P1)** Multi-tenant con aislamiento (keys por tenant, namespaces).

### UX & Operabilidad
- **NFR-060 (P0)** IA "in-flow" en la interfaz operativa principal del wedge, con acciones contextuales y evidencia visible. Mobile puede cumplir esto cuando el workflow lo requiera, pero no es un gate universal de P0.
- **NFR-061 (P0)** "Explain why" siempre disponible con evidencia.
- **NFR-062 (P0)** Handoff y reversión/undo para acciones críticas.

### Mobile Performance
- **NFR-070 (P1)** Tiempo de carga inicial de mobile app ≤ **3s** en dispositivos Android mid-range (cold start).
- **NFR-071 (P1)** Navegación entre pantallas CRM ≤ **300ms** (perceived transition).
- **NFR-072 (P1)** Copilot SSE streaming: primer token visible ≤ **500ms** después de envío de query (excluye latencia LLM).
- **NFR-073 (P1)** Tamaño del APK ≤ **50MB** (sin assets de onboarding).
- **NFR-074 (P1)** Consumo de memoria ≤ **200MB** en uso típico (lista + detalle + copilot activo).

---

## 9 — Arquitectura conceptual

### Componentes
- **CRM Store (OLTP)**
- **Event Bus / CDC**
- **Connectors & Ingestion Workers**
- **Hybrid Index (BM25 + Vector)**
- **Policy Engine (RBAC/ABAC, PII/no-cloud, approvals)**
- **Copilot Service (evidence builder)**
- **Agent Orchestrator (planner + state machine + queues)**
- **Tool Registry (schemas, auth, rate limits)**
- **Audit/Telemetry**
- **Eval Service (datasets + gating)**

### Diagrama

```mermaid
flowchart LR
  MOBILE[Mobile App\n(React Native + Expo)] --> BFF[BFF Gateway\n(Express.js)]
  BFF --> API[Go CRM API]
  API --> OLTP[(CRM Store)]
  OLTP --> BUS[(Event Bus)]
  API --> POL[Policy Engine\nRBAC/ABAC + PII + Approvals]

  subgraph Knowledge
    CONN[Connectors\nEmail/Docs/Calls] --> ING[Ingestion/Normalize]
    ING --> IDX[Hybrid Index\nBM25 + Vector]
  end

  BFF -->|"SSE proxy"| COP
  API --> COP[Copilot Service\nEvidence Builder]
  COP --> IDX
  COP --> LLM[LLM Adapter\nLocal/Cloud]
  COP --> POL

  API --> ORCH[Agent Orchestrator]
  ORCH --> TOOL[Tool Registry\nActions/Skills]
  ORCH --> POL
  ORCH --> IDX
  ORCH --> LLM

  COP --> AUD[Audit & Telemetry]
  ORCH --> AUD
  POL --> AUD
```

---

## 10 — Modelo de datos conceptual

### Objetos core
- Workspace/Tenant
- User
- Role / PolicySet
- Account/Company
- Contact
- Lead
- Opportunity/Deal
- Case/Ticket
- Activity (Task/Event)
- Note / Attachment

### Objetos AI-native
- KnowledgeItem (normalizado: email/doc/call/chat)
- EmbeddingDocument (chunk + vector + metadata)
- Evidence (ref record + snippet + score + timestamp + permissions snapshot)
- AgentDefinition (objetivo + tools + límites + policies)
- SkillDefinition (workflows multi-step)
- ToolDefinition (schema + auth + rate limits)
- AgentRun (inputs/outputs/tools/costos/estado)
- ApprovalRequest (acción propuesta, aprobadores, decisión)
- AuditEvent (actor, recurso, cambio)
- EvalSuite / EvalRun (datasets, scores, gating)

---

## 11 — Gobernanza de IA

### Políticas esenciales
- **Abstención** si evidencia insuficiente/contradictoria o viola policy.
- **Handoff humano** por sensibilidad, riesgo, loop, baja confianza.
- **Allowlist de tools** + validación de schemas.
- **Approvals** para acciones sensibles.
- **No-cloud** para datos marcados (PII/secret).

### Puntos de enforcement
1. Antes de retrieval (filtro por permisos)
2. Antes de prompt (redacción/enmascaramiento de PII)
3. Antes de tool call (permisos + approvals)
4. Después de ejecución (auditoría + métricas)

---

## 12 — Calidad y evaluación (Evals)

### Métricas offline
- Groundedness (% outputs con evidencia suficiente)
- Exactitud vs CRM truth (comparación con campos)
- Abstención correcta
- Policy adherence (0 violaciones)
- Tool success rate + rollback correctness

### Métricas online
- Resolution/deflection (support)
- Time-to-update (sales)
- Adoption (WAU/MAU de copilot + actions)
- Cost per outcome (€/ticket resuelto, €/deal movido)
- Escalation rate + reasons

### Success Metrics (NFR)
- **Speed**: Copilot Q&A ≤2.5s p95, summaries ≤5s p95
- **Quality**: Groundedness >95%, abstention correctness >98%
- **Cost**: <€0.10 per copilot interaction on average
- **Adoption**: >50% WAU of copilot + actions in first quarter
- **Reliability**: 99.5% uptime, <0.1% tool failures

---

## 13 — Roadmap por fases

### P0 (MVP)
- FR-001, FR-002, FR-090, FR-092, FR-200, FR-202, FR-060, FR-070, FR-071
- FR-210, FR-211, FR-230, FR-231, FR-232, FR-242
- FR-050, FR-051, FR-061
- Base de metering/cost attribution y reporting de uso por workspace/run/tool
- Casos de uso: UC-C1 (Support Agent con handoff), UC-S1 (Sales Copilot), UC-G1 (Governance)
- Telemetría mínima: NFR-030, NFR-031
- Mobile y BFF: opcionales cuando el workflow del wedge lo requiera, no como gate universal de salida

### P1 (v1)
- FR-091, FR-093, FR-094, FR-203, FR-233, FR-234, FR-241, FR-242, FR-243
- FR-003 (Reporting extendido), FR-004, FR-005
- Catálogo completo de agentes (FR-231 extendido): UC-S2, UC-S3, UC-K1, UC-D1
- Agent Studio completo: UC-A1
- FR-300, FR-301, FR-302, FR-303 (Mobile + BFF breadth y experiencia extendida)
- Cost control completo: NFR-040, NFR-041
- iOS support (FR-300 extendido)
- Mobile performance optimization: NFR-073, NFR-074

### P2 (v2)
- Plugins/Marketplace (FR-052)
- Multi-tenant fuerte (NFR-051)
- Packs verticales (skills/templates por industria)

---

## 14 — Riesgos y mitigaciones

1. **Hallucinations (sin evidencia)**
   - Mitigación: FR-092 + FR-210 + eval gating (FR-242).

2. **Data leakage (PII/no-cloud)**
   - Mitigación: FR-061 + enforcement + LLM adapter policy aware.

3. **Acciones peligrosas**
   - Mitigación: FR-071 approvals + FR-211 allowlist tools + idempotencia (NFR-010).

4. **Cost blow-up**
   - Mitigación: quotas/presupuestos (FR-233, NFR-040) + degradación controlada (NFR-041).

5. **Baja adopción**
   - Mitigación: in-flow UI (NFR-060) + acciones de 1-click + "why" visible (NFR-061).

---

## 15 — Apéndices

### A) Glosario

- **RAG**: Retrieval-Augmented Generation (generación asistida por recuperación).
- **Evidence pack**: paquete de evidencias (IDs + snippets + scores + timestamps).
- **Tool/Action**: operación ejecutable (CRUD, enviar email, webhook, etc.) accesible al agente.
- **Dry-run**: ejecución simulada sin escribir/mutar datos.
- **RBAC/ABAC**: control de acceso por roles / por atributos.
- **No-cloud policy**: regla de no enviar datos sensibles a proveedores cloud.
- **Evals**: pruebas sistemáticas de calidad y compliance (offline).
- **UC**: Caso de uso de alto nivel (User Case).
- **FR**: Requerimiento funcional.
- **NFR**: Requerimiento no funcional.

### B) Checklist "Agent-ready"

- [x] IDs estables (FR-xxx, NFR-xxx, UC-xx)
- [x] Prioridades (P0/P1/P2)
- [x] AC claros por requerimiento
- [x] Dependencias explícitas
- [x] Diagramas Mermaid L0/L1/L2
- [x] Roadmap y riesgos
- [x] Tabla de UCs con prioridad
- [x] Resumen FR por dominio

### C) Historial de versiones

| Versión | Fecha | Cambio |
|---|---|---|
| 1.0 | 2026-02-08 | Documento original (`agentic_crm_requirements_agent_ready.md`) |
| 2.0 | 2026-03-19 | Consolidación en `docs/requirements.md`: UC con prioridades, resumen FR, roadmap expandido, secciones numeradas |
