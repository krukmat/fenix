# Plan de Remediacion de `eslint-disable`

## Objetivo

Este documento define la remediacion de suppressions inline de ESLint
detectadas en `mobile` y `bff`.

Objetivos concretos:

- reducir suppressions a las estrictamente necesarias;
- eliminar comments obsoletos o usados para ocultar deuda evitable;
- introducir guardrails para detectar disables muertos.

Objetivo adicional de politica:

- no permitir disables inline en codigo salvo excepciones muy fundamentadas y
  centralizadas fuera del archivo afectado.

## Estado Actual Auditado

Conteo actual por regla:

- `no-require-imports`: 33
- `no-explicit-any`: 11
- `no-var-requires`: 1
- `no-unused-vars`: 1

Distribucion principal:

- tests mobile con `require()` diferido;
- tests BFF con `any` repetido;
- 2 casos en codigo de produccion BFF.

## Hallazgos Priorizados

- suppression obsoleta en `bff/src/routes/copilot.ts`;
- suppression obsoleta y mal aplicada en
  `mobile/__tests__/stores/authStore.test.ts`;
- helper no usado en `mobile/__tests__/screens/copilot.test.tsx`;
- uso repetido de `as any` y `req/res/next: any` en tests BFF;
- suppressions aceptables que deben mantenerse, en particular el error
  middleware de Express.

## Politica Objetivo

Politica objetivo para el repositorio:

- no se permiten `eslint-disable`, `eslint-disable-next-line`,
  `eslint-disable-line` ni `eslint-enable` dentro del codigo fuente o tests;
- las excepciones no deben resolverse con comentarios inline;
- toda excepcion debe centralizarse en configuracion de ESLint o allowlist
  documentada;
- toda excepcion aprobada debe indicar archivo, regla, motivo tecnico, referencia
  a issue o ADR, y fecha de revision.

Aplicacion practica:

- el comentario inline deja de ser un mecanismo autorizado;
- la excepcion, cuando exista, se mueve a configuracion central y queda
  auditable;
- el objetivo por defecto es cero disables inline en el repo.

## Plan de Remediacion

### Fase 1

Eliminar suppressions obsoletas y comentarios muertos.

### Fase 2

Reemplazar `require()` diferido por imports estaticos o `jest.isolateModules()`
cuando sea realmente necesario.

### Fase 3

Crear helper tipado compartido para stubs de proxy en BFF y eliminar `any`
repetidos.

### Fase 4

Endurecer lint con `--report-unused-disable-directives`.

### Fase 5

Bloquear disables inline en pipeline y branch protection.

## Tareas Individuales y Dependencias

### T0. Congelar la politica

- confirmar que la politica objetivo es cero disables inline por defecto;
- usar este documento como fuente de verdad para aprobacion y ejecucion.

Dependencias:

- ninguna.

### T1. Inventario final y baseline

- generar el inventario definitivo de `eslint-disable*` y `eslint-enable` en
  `mobile/` y `bff/`;
- clasificar cada caso como obsoleto, reemplazable o excepcion real.

Dependencias:

- `T0`.

### T2. Endurecer ESLint en mobile

- activar `linterOptions.noInlineConfig = true`;
- activar `linterOptions.reportUnusedDisableDirectives = "error"`;
- verificar que `npm run lint` falla ante disables inline o muertos.

Dependencias:

- `T1`.

### T3. Crear gate textual de repo

- crear un script que falle si detecta:
  - `eslint-disable`
  - `eslint-disable-next-line`
  - `eslint-disable-line`
  - `eslint-enable`
- revisar como minimo `mobile/` y `bff/`.

Dependencias:

- `T1`.

### T4. Integrar el gate en CI

- anadir un job nuevo en GitHub Actions que ejecute el gate textual;
- colocarlo antes de jobs de calidad para fallar pronto;
- dejarlo preparado para branch protection.

Dependencias:

- `T3`.

### T5. Extender el control a BFF

- cubrir `bff` con la misma politica que `mobile`;
- como minimo, hacer pasar `bff` por el gate textual;
- opcion recomendada: incorporar ESLint en `bff` con la misma politica de
  inline config deshabilitada.

Dependencias:

- `T1`.

### T6. Remediar suppressions obsoletas de bajo riesgo

- eliminar primero los casos claramente muertos u obsoletos;
- casos iniciales:
  - `bff/src/routes/copilot.ts`
  - `mobile/__tests__/stores/authStore.test.ts`
  - `mobile/__tests__/screens/copilot.test.tsx`

Dependencias:

- `T2`;
- `T3`;
- `T5` si `bff` queda bajo ESLint completo.

### T7. Remediar tests mobile con `require()` diferido

- sustituir `require()` por imports estaticos donde no haga falta reevaluacion;
- cuando haga falta reevaluacion, usar `jest.isolateModules()` o `import()`;
- priorizar:
  - `mobile/__tests__/screens/workflows.test.tsx`
  - `mobile/__tests__/screens/home.test.tsx`
  - `mobile/__tests__/navigation/drawer.test.tsx`
  - `mobile/__tests__/components/copilot/CopilotPanel.context.test.tsx`

Dependencias:

- `T2`;
- `T6`.

### T8. Remediar tests BFF con `any` repetido

- crear helper tipado compartido para el stub de `http-proxy-middleware`;
- eliminar `req/res/next: any` y `as any` repetidos;
- priorizar:
  - `bff/tests/proxy.test.ts`
  - `bff/tests/mobileHeaders.test.ts`
  - `bff/tests/errorHandler.test.ts`
  - `bff/tests/copilot.test.ts`
  - `bff/tests/e2e/fullstack.test.ts`

Dependencias:

- `T5`;
- `T6`.

### T9. Registrar excepciones reales fuera del codigo

- si sobrevive alguna excepcion, moverla a configuracion centralizada;
- documentar archivo, regla, motivo tecnico y fecha de revision;
- no dejar comentarios inline como mecanismo final.

Dependencias:

- `T7`;
- `T8`.

### T10. Validacion final

- ejecutar lint de `mobile`;
- ejecutar el gate textual;
- correr los tests afectados de `mobile` y `bff`;
- confirmar que no quedan disables inline en codigo.

Dependencias:

- `T4`;
- `T7`;
- `T8`;
- `T9`.

### T11. Proteccion de rama

- marcar el nuevo job de bloqueo como required check en GitHub;
- impedir merge si falla el gate de disables inline.

Dependencias:

- `T4`;
- `T10`.

## Orden de Ejecucion

Orden practico recomendado:

1. `T0`
2. `T1`
3. `T2` y `T3` en paralelo
4. `T4` y `T5`
5. `T6`
6. `T7` y `T8` en paralelo
7. `T9`
8. `T10`
9. `T11`

Ruta critica:

- `T0 -> T1 -> T3 -> T4 -> T6 -> T7/T8 -> T9 -> T10 -> T11`

Trabajo paralelizable:

- `T2` con `T3`;
- `T7` con `T8`;
- la documentacion de excepciones de `T9` puede prepararse mientras cierran
  `T7` y `T8`.

## Controles Recomendados

### Controles en ESLint

- activar `linterOptions.noInlineConfig = true` para que los comentarios inline
  no tengan efecto;
- activar `linterOptions.reportUnusedDisableDirectives = "error"` para tratar
  disables muertos como fallo;
- aplicar la misma politica a `mobile` y `bff`, no solo a `mobile`.

### Controles en Pipeline

- crear un script de validacion de repo que falle si detecta:
  - `eslint-disable`
  - `eslint-disable-next-line`
  - `eslint-disable-line`
  - `eslint-enable`
- ejecutar ese script en CI sobre `mobile/` y `bff/`;
- hacer obligatorio el job correspondiente para merge en pull requests.

### Controles de Gobernanza

- si una excepcion es inevitable, moverla a override centralizado de ESLint;
- documentar las excepciones aprobadas en un documento dedicado o en este mismo
  plan;
- rechazar en code review cualquier PR con disables inline no registrados.

## Criterios de Aprobacion

- no introducir cambios funcionales;
- mantener verdes las suites afectadas;
- dejar documentadas las suppressions que sigan siendo necesarias;
- no permitir que se pueda fusionar codigo nuevo con disables inline activos.

## Validacion

- lint de `mobile`;
- tests afectados de `mobile`;
- tests afectados de `bff`;
- verificacion de SSE relay y middleware de error;
- validacion del nuevo job de CI que bloquea disables inline;
- comprobacion de que la proteccion de rama exige el job de bloqueo.

## Riesgos y Limites

- no tocar suppressions dentro de mocks hoisted de Jest salvo que exista
  alternativa clara;
- no modificar docs funcionales o de arquitectura no relacionadas;
- no reestructurar reglas globales de ESLint mas alla de guardrails concretos;
- si alguna excepcion sobrevive, debe estar fuera del codigo y quedar
  explicitamente justificada.

## Cambios de Implementacion Posteriores a la Aprobacion

Cuando se apruebe la remediacion tecnica, la implementacion debe hacer
exactamente esto:

- crear este documento como referencia previa;
- no editar `docs/mobile-agent-spec-transition-gap-closure-plan.md` ni
  `docs/handoffs/mobile-bff-remediation-pipeline-handoff.md`;
- dejar el documento como referencia de aprobacion previa antes de tocar codigo;
- endurecer la configuracion de ESLint para desactivar inline config;
- introducir un script de gate para bloquear inline disables en CI;
- extender el control a `bff` aunque hoy no tenga el mismo circuito de lint que
  `mobile`;
- esperar despues de crear el doc, sin ejecutar la remediacion tecnica, hasta
  nueva instruccion.

## Pruebas y Verificacion

La fase de solo documentacion no requiere cambios de codigo ni ejecucion
funcional adicional.

Tras la futura aprobacion de implementacion:

- ejecutar ESLint con deteccion de disables no usados;
- correr las suites de test afectadas en `mobile` y `bff`;
- ejecutar el nuevo gate de deteccion textual de inline disables;
- verificar que no cambia el comportamiento observable de rutas, mocks ni
  streaming;
- verificar que cualquier excepcion residual esta centralizada y documentada.

## Supuestos

- se documenta en un archivo nuevo para evitar conflictos con docs abiertas o ya
  modificadas;
- la aprobacion del documento no implica aprobacion automatica de la
  remediacion tecnica;
- la politica objetivo del repo pasa a ser cero disables inline por defecto;
- las excepciones, si existen, deben aprobarse fuera del archivo afectado y con
  trazabilidad;
- hasta nueva orden, el trabajo queda detenido despues de dejar la
  documentacion creada.
