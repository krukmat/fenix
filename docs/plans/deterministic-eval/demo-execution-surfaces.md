---
doc_type: summary
title: Demo Ejecutable — Superficies Reales Mobile y BFF Admin
status: complete
created: 2026-05-02
---

# Demo Ejecutable — Superficies Reales Mobile y BFF Admin

## Propósito

Este documento responde a una pregunta concreta:

> Si queremos dirigir la demo de soporte gobernado usando superficies reales del producto, ¿qué puede hacer hoy una persona en `mobile/`, qué puede hacer en `bff/admin`, y qué falta completar para que la demo sea realmente ejecutable de punta a punta?

No describe el packet técnico en abstracto. Describe la demo operativa real.

## Respuesta Corta

Hoy la demo **no está cerrada end-to-end como flujo 100% ejecutable por un humano desde las superficies actuales**.

Sí existe bastante superficie real para mostrar el caso, las aprobaciones, los runs y la auditoría:

- `mobile`: casos de soporte, detalle del caso, inbox de aprobaciones, activity log, copilot contextual
- `bff/admin`: agent runs, approvals, audit, workflows, policy, tools, métricas

Pero hay un bloqueo importante:

- el disparo real del support agent desde mobile **no está alineado** con el contrato backend visible

Por tanto, la demo hoy queda así:

1. **Demo plenamente reproducible**: la demo determinista por fixtures y Review Packet
2. **Demo parcialmente ejecutable en superficies reales**: inspección del caso, approvals, runs, audit
3. **Demo operativa end-to-end en vivo**: pendiente de completar

## Superficies Reales Que Ya Existen

### Mobile

Estas superficies ya existen en la app mobile:

- lista de casos de soporte
- detalle del caso de soporte
- botón para lanzar el support agent
- activity log con detalle del run
- inbox con approvals, handoffs, signals y runs rechazados
- pantalla contextual de copilot para el caso
- CRM hub con creación de casos

### BFF Admin

Estas superficies ya existen en el shell `bff/admin`:

- lista y detalle de `agent runs`
- cola de `approvals` con acciones inline
- `audit`
- `workflows`
- `policy`
- `tools`
- `metrics`

### Lo Que No Existe Como Superficie Visible De Demo

No encontré una superficie real ya hecha para:

- disparar el demo de soporte desde BFF admin
- generar o visualizar el `Review Packet` de una corrida real desde mobile o admin
- ejecutar la evaluación determinista del run real desde una UI del producto

## Flujo Ideal De Demo Que Queremos Tener

La demo que realmente queremos dirigir con un humano debería verse así:

1. El operador abre la app mobile.
2. Entra en un caso de soporte de alta prioridad.
3. Dispara el support agent desde la pantalla del caso.
4. El sistema analiza el caso, recupera evidencia y decide que la mutación sensible requiere aprobación.
5. El aprobador entra en Inbox mobile o en BFF admin Approvals y aprueba.
6. El presentador abre el run en Activity o en BFF admin Agent Runs.
7. El presentador enseña evidencia, tools, audit y estado final.
8. El presentador abre el Review Packet generado para esa corrida y cierra la demo con el veredicto.

Ese es el flujo objetivo.

## Flujo Que Sí Puede Hacerse Hoy

Hoy se puede dirigir una demo **híbrida**, no totalmente cerrada.

### Opción A — Mobile + artefactos deterministas

La persona que dirige la demo puede hacer esto:

1. Abrir la app mobile.
2. Navegar a Support.
3. Abrir un caso de soporte existente.
4. Mostrar el contexto del caso, la cuenta y las señales.
5. Mostrar Inbox para explicar cómo se ven las aprobaciones.
6. Mostrar Activity para explicar cómo se ve un agent run.
7. Cerrar la demo mostrando el Review Packet determinista ya generado por fixtures.

Esta opción sirve para narrativa de producto, pero no demuestra todavía el disparo real del support agent desde la UI actual.

### Opción B — Mobile + BFF Admin

La persona que dirige la demo puede hacer esto:

1. Mostrar el caso en mobile.
2. Explicar el punto donde el sistema debería disparar el support agent.
3. Cambiar a BFF admin.
4. Mostrar la cola de approvals.
5. Mostrar agent runs.
6. Mostrar audit.
7. Cerrar con el Review Packet determinista.

Esta opción es mejor que la anterior para una audiencia técnica, porque enseña superficies reales de operación y control.

### Opción C — Demo completamente determinista

La persona que dirige la demo no usa la ejecución viva del producto. En su lugar:

1. Presenta el escenario.
2. Presenta la traza sintética.
3. Presenta la evaluación determinista.
4. Presenta el Review Packet.

Esta opción es la más estable hoy, pero no es la más fuerte si el objetivo es enseñar operación real del sistema.

## Flujo Real Por Surface

## 1. Qué puede hacer hoy un operador en Mobile

### Paso 1 — Abrir la lista de casos

Ruta real:

- `Support` tab

Qué demuestra:

- el operador sí tiene una entrada real a los casos de soporte
- se ve el estado del caso
- se ve la prioridad
- se ve si hay señales activas

Qué no demuestra todavía:

- no demuestra por sí mismo la automatización gobernada

### Paso 2 — Abrir el detalle del caso

Ruta real:

- `/support/[id]`

Qué demuestra:

- detalle del caso
- prioridad visible
- cuenta asociada
- señales
- sección de actividad de agentes

Qué debería pasar idealmente en la demo:

- desde aquí el operador lanza el support agent

Qué problema hay hoy:

- el botón existe, pero el payload móvil visible no coincide con el contrato backend visible

### Paso 3 — Lanzar el support agent

Superficie actual:

- botón `Run Support Agent` en el detalle del caso

Problema detectado:

- mobile envía `entity_type` y `entity_id`
- el handler visible de backend espera `case_id`, `customer_query` y opcionalmente `priority`

Conclusión:

- esta parte del flujo **está bloqueada o, como mínimo, incompleta**
- hoy no conviene prometer una demo en vivo basada en ese botón sin corregir antes el contrato

### Paso 4 — Ver aprobaciones

Superficie actual:

- `Inbox` en mobile

Qué demuestra:

- sí existe una cola real de approvals
- el aprobador puede aprobar o rechazar desde la app
- la aprobación aparece como objeto operativo del producto, no como nota técnica

Qué limitación tiene:

- depende de que el approval exista previamente
- sin trigger real correcto, la demo depende de datos sembrados o de una generación manual previa

### Paso 5 — Ver el run

Superficie actual:

- `Activity`
- detalle de `/activity/[id]`

Qué demuestra:

- estado público del run
- latencia
- cost
- tool calls
- output
- audit events
- usage

Qué no demuestra todavía de forma nativa:

- no muestra el Review Packet
- no muestra el expected-vs-actual determinista
- no expone de forma explícita la comparación contra el escenario dorado

## 2. Qué puede hacer hoy un operador en BFF Admin

### Paso 1 — Abrir Approvals

Ruta real:

- `/bff/admin/approvals`

Qué demuestra:

- existe una cola web de approvals
- el aprobador puede decidir directamente
- la cola ya sirve como superficie de demo

Valor para la demo:

- es una buena superficie para el momento “aquí interviene un humano”

### Paso 2 — Abrir Agent Runs

Ruta real:

- `/bff/admin/agent-runs`
- `/bff/admin/agent-runs/:id`

Qué demuestra:

- existe una vista web de runs
- se puede abrir detalle
- se ven tool calls, cost, evidence y reasoning trace

Valor para la demo:

- es la mejor superficie actual para explicar observabilidad del run

### Paso 3 — Abrir Audit

Ruta real:

- `/bff/admin/audit`

Qué demuestra:

- la demo tiene rastro auditable
- el comportamiento se puede reconstruir desde eventos

Valor para la demo:

- refuerza mucho la parte de gobernanza

### Lo que no puede hacer hoy BFF Admin

BFF Admin hoy no ofrece una pantalla clara para:

- crear el caso de soporte del demo
- disparar el support agent del demo
- abrir un Review Packet generado desde un run real
- ejecutar la evaluación determinista del run real desde la propia UI

## Qué Falta Completar Para Tener La Demo De Verdad

Estos son los gaps concretos.

### Gap 1 — Corregir el trigger real del support agent en mobile

Este es el gap principal.

Hoy el flujo visual existe, pero el contrato visible no cierra:

- mobile trigger: `entity_type/entity_id`
- backend support trigger: `case_id/customer_query/priority`

Qué hay que completar:

- decidir el contrato final
- o adaptar mobile para enviar `case_id`, `customer_query` y `priority`
- o añadir en BFF una traducción explícita, documentada y testeada

Sin esto, la demo end-to-end desde mobile no es fiable.

### Gap 2 — Definir de dónde sale `customer_query`

Aunque se corrija el payload, falta cerrar la UX:

- ¿sale de la descripción del caso?
- ¿sale de un campo editable en el detalle?
- ¿sale de una acción del copilot?
- ¿sale de un modal antes de lanzar el agent?

Ahora mismo ese paso no está definido como interacción humana completa.

### Gap 3 — Decidir quién aprueba y desde dónde

La demo necesita un actor humano claro:

- aprobador en mobile Inbox
- aprobador en BFF Admin Approvals

Hay que elegir una de las dos como recorrido oficial del demo. Si no se decide, la demo queda ambigua.

### Gap 4 — Exponer el Review Packet desde una superficie real

Hoy el Review Packet existe como artefacto de dominio y fixture.

Pero no existe una superficie clara para:

- pedir el packet de un run real
- abrirlo desde Activity
- abrirlo desde Agent Runs admin

Qué hay que completar:

- endpoint o servicio que genere/recupere packet por `run_id` + `scenario_id`
- enlace visible desde mobile o admin

### Gap 5 — Definir cómo se mapea un run real a un escenario dorado

La evaluación determinista necesita saber contra qué escenario se compara el run.

Eso hoy está claro en fixtures, pero no está claro como flujo operativo visible de producto.

Qué hay que completar:

- convención explícita de `scenario_id`
- mapping run -> scenario
- forma de seleccionar el escenario correcto para la demo

### Gap 6 — Preparación de datos del demo

Para que la demo salga bien, hace falta cerrar la preparación:

- caso de soporte
- cuenta asociada
- evidencia o conocimiento disponible
- usuario operador
- usuario aprobador

Si esto no está sembrado y verificado antes, la demo depende demasiado del entorno.

## Recomendación Operativa

La recomendación pragmática es esta:

### Corto plazo — Demo utilizable ya

Usar un recorrido híbrido:

1. mostrar el caso en mobile
2. mostrar approvals y/o agent runs en BFF admin
3. cerrar con el Review Packet determinista generado

Esto da una demo honesta y fuerte sin vender una automatización viva que aún no está cerrada.

### Medio plazo — Demo real mínima

Completar solo estas piezas:

1. corregir el trigger support mobile/BFF/backend
2. definir cómo se aporta `customer_query`
3. elegir la superficie oficial de aprobación
4. exponer el Review Packet desde una UI real

Con eso ya existe una demo end-to-end creíble.

## Checklist De Preparación Para La Demo

Antes de demoar, alguien debería poder marcar esto:

- backend Go levantado
- BFF levantado
- mobile levantado y autenticado
- token disponible para `bff/admin`
- workspace de demo definido
- caso de soporte de demo creado o sembrado
- cuenta y contacto asociados al caso
- evidencia/knowledge necesaria disponible
- usuario aprobador disponible
- runbook elegido: mobile-only, admin-only o híbrido
- Review Packet accesible para el cierre

## Decisión Recomendada

La decisión recomendada hoy es:

- **superficie principal de narrativa**: mobile
- **superficie principal de control y trazabilidad**: BFF admin
- **artefacto final de cierre**: Review Packet

Y la afirmación correcta frente a terceros sería:

> “La demo operativa del flujo gobernado está parcialmente disponible en superficies reales. La validación determinista y el Review Packet ya existen. Para cerrar la demo end-to-end en vivo faltan el trigger correcto del support agent y la exposición del packet desde una UI real.”
