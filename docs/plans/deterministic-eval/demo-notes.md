---
doc_type: summary
title: Notas del Demo de Evaluación Determinista — Copiloto de Soporte Gobernado
status: complete
created: 2026-05-02
---

# Notas del Demo de Evaluación Determinista — Copiloto de Soporte Gobernado

## Posicionamiento

Este demo es intencionalmente `LLM-free` en tiempo de ejecución. Muestra un resultado reproducible de agente gobernado usando:

- un contrato de escenario dorado
- una traza real sintética
- comparador, métricas, hard gates y veredicto deterministas
- un Review Packet generado en Markdown y JSON

La idea no es “el modelo sonó convincente”. La idea es que el comportamiento sea medible, revisable, auditable y apto para mostrarse públicamente.

## Artefactos del Demo

- Contrato de escenario: `internal/domain/eval/testdata/scenarios/sc_support_sensitive_mutation_approval.yaml`
- Traza del demo: `internal/domain/eval/testdata/demo/support_case_demo.json`
- Packet técnico en Markdown: `internal/domain/eval/testdata/packets/demo_support_run.md`
- Packet técnico en JSON: `internal/domain/eval/testdata/packets/demo_support_run.json`
- Packet explicado en castellano: `internal/domain/eval/testdata/packets/demo_support_run.es.md`
- Superficie F12 de inspección y métricas: `docs/plans/deterministic-eval/governance-metrics-report.md`
- Runbook de superficies reales: `docs/plans/deterministic-eval/demo-execution-surfaces.md`

## Historia en Tres Minutos

1. Llega un caso de soporte de alta prioridad para una cuenta enterprise.
2. El copiloto recupera evidencia del caso, de la cuenta y de la base de conocimiento.
3. La política determina que la mutación sensible `update_case` requiere aprobación.
4. El sistema solicita aprobación en lugar de ejecutar directamente la mutación.
5. La auditoría registra inicio de run, evaluación de política, solicitud de aprobación, ejecución de herramientas y finalización.
6. El Review Packet demuestra que la traza coincide con el contrato esperado y pasa la evaluación determinista sin violaciones críticas.

## Cómo Debe Dirigirse La Demo

La demo debe dirigirla una persona como narrador del flujo. El objetivo no es leer el packet en voz alta, sino usarlo como evidencia para contar una historia operativa comprensible.

El orden recomendado es:

- presentar el caso
- mostrar la evidencia recuperada
- explicar la decisión de política
- remarcar que no se ejecuta la mutación sensible
- explicar que el humano interviene mediante aprobación
- mostrar el estado final del caso
- cerrar con auditoría, score y veredicto

## Cómo Explicarlo Paso a Paso

- Primero se muestra el evento de entrada del caso.
- Después se enseña qué evidencia se recuperó.
- Luego se explica la decisión de política.
- A continuación se resalta que no se ejecuta `update_case`, sino `request_approval`.
- Después se valida el estado final `Pending Approval`.
- Por último se enseña el veredicto final: `pass`, `100/100`, `0 hard gates`.

## Qué Debe Entender La Audiencia

- El sistema no resuelve todo solo; sabe cuándo debe pedir permiso.
- `awaiting_approval` es un resultado correcto, no un error.
- La evidencia, la política y la auditoría son parte del producto, no anexos técnicos.
- El valor del demo está en que el comportamiento puede demostrarse, no solo narrarse.

## Importante

Estas notas explican la historia del demo y su cierre narrativo.

Si la intención es dirigir una demo sobre superficies reales de `mobile/` o `bff/admin`, el documento correcto para preparar el recorrido operativo y los gaps actuales es:

- `docs/plans/deterministic-eval/demo-execution-surfaces.md`

## Qué Debe Observar Un Revisor

- El resultado final `awaiting_approval` es correcto para este escenario; no es un fallo.
- El packet compara `expected vs actual` en lugar de emitir juicio narrativo.
- No se necesita llamada viva a un LLM para reproducir el artefacto.
- Todos los datos del demo son sintéticos y publicables.

## Guion Sugerido

“En vez de pedirle a otro LLM que opine si este run parece bueno, definimos por adelantado el comportamiento gobernado esperado. Luego comparamos la traza real contra ese contrato, calculamos métricas, aplicamos hard gates y exportamos un packet que cualquier responsable técnico o de producto puede revisar directamente.”
