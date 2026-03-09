# AGENT_SPEC — Analisis de Casos de Uso

> **Fecha**: 2026-03-09
> **Source**: `docs/agent-spec-transition-plan.md` (Parte 1: Behaviors B1-B8)
> **Formato**: 3 niveles de detalle por behavior
>   - **Nivel 1**: Flujo principal (happy path)
>   - **Nivel 2**: Flujos alternativos y de error
>   - **Nivel 3**: Edge cases y condiciones de borde
> **Naming source of truth**: `docs/agent-spec-overview.md`

## Convencion de naming

Este documento conserva `BEHAVIOR` en `snake_case` para escenarios detallados.

Los casos de uso top-level quedan armonizados con la convencion historica del repo:

| UC | Capability | Behavior family |
|---|---|---|
| `UC-A2` | Workflow Authoring | `define_workflow*` |
| `UC-A3` | Workflow Verification and Activation | `verify_workflow*` |
| `UC-A4` | Workflow Execution | `execute_workflow*` |
| `UC-A5` | Signal Detection and Lifecycle | `detect_signal*` |
| `UC-A6` | Deferred Actions | `defer_action*` |
| `UC-A7` | Human Override and Approval | `human_override*` |
| `UC-A8` | Workflow Versioning and Rollback | `version_workflow*` |
| `UC-A9` | Agent Delegation | `delegate_workflow*` |

Set canonico relacionado:
- `docs/agent-spec-overview.md`
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-design.md`
- `docs/agent-spec-integration-analysis.md`
- `docs/agent-spec-development-plan.md`
- `docs/agent-spec-traceability.md`

---

## B1: Definicion de Workflow

### Nivel 1 — Flujo principal

```
BEHAVIOR define_workflow
  GIVEN   un Admin quiere automatizar un proceso de negocio
  WHEN    el Admin escribe un workflow en lenguaje DSL
  THEN    el sistema almacena el workflow como borrador versionado (version=1, status=draft)
  AND     el Admin puede editar el DSL hasta estar conforme
```

**Ejemplo**: El admin crea `qualify_lead` con DSL source. Se almacena como v1/draft. El admin edita el DSL 3 veces. Cada edicion actualiza el draft in-place.

### Nivel 2 — Flujos alternativos y de error

#### B1.1: Creacion con spec incluido

```
BEHAVIOR define_workflow_with_spec
  GIVEN   un Admin quiere crear un workflow con verificacion futura
  WHEN    el Admin envia DSL source + spec source (CONTEXT/ACTORS/BEHAVIOR/CONSTRAINTS)
  THEN    el sistema almacena ambos en el workflow borrador
  AND     el spec_source queda disponible para el Judge en B2
```

**Nota**: `spec_source` es opcional. Si no se provee, el Judge solo puede hacer validaciones sintacticas del DSL (no consistency checks).

#### B1.2: Nombre duplicado

```
BEHAVIOR define_workflow_duplicate_name
  GIVEN   ya existe un workflow con el mismo nombre y version en el workspace
  WHEN    el Admin intenta crear otro workflow con ese nombre
  THEN    el sistema rechaza la creacion con error de unicidad
  AND     el Admin recibe el detalle del conflicto (workspace, name, version)
```

**Constraint aplicado**: UNIQUE(workspace_id, name, version).

#### B1.3: Campos requeridos faltantes

```
BEHAVIOR define_workflow_missing_fields
  GIVEN   el Admin envia una solicitud de creacion
  WHEN    faltan campos obligatorios (name o dsl_source)
  THEN    el sistema rechaza con error de validacion
  AND     el error lista los campos faltantes
```

**Campos obligatorios**: `name`, `dsl_source`. Todo lo demas es opcional en draft.

#### B1.4: Edicion de workflow no-draft

```
BEHAVIOR define_workflow_edit_non_draft
  GIVEN   un workflow en estado active o archived
  WHEN    el Admin intenta editar el DSL directamente
  THEN    el sistema rechaza la edicion
  AND     informa que debe crear una nueva version (→ B7)
```

**Constraint aplicado**: Solo drafts son editables. Activar o archivar hace el DSL inmutable.

### Nivel 3 — Edge cases

#### B1.5: DSL source vacio

```
BEHAVIOR define_workflow_empty_dsl
  GIVEN   el Admin envia un workflow con dsl_source vacio o solo whitespace
  WHEN    el sistema valida la solicitud
  THEN    el sistema rechaza con error de validacion
  AND     el error indica que dsl_source no puede estar vacio
```

**Razon**: Un draft sin DSL no tiene sentido — no hay nada que verificar o activar. Diferente de DSL con errores de sintaxis, que si se permite almacenar como draft.

#### B1.6: DSL con errores de sintaxis

```
BEHAVIOR define_workflow_invalid_syntax
  GIVEN   el Admin escribe DSL con errores de sintaxis
  WHEN    el Admin guarda el workflow como draft
  THEN    el sistema almacena el DSL tal cual (el draft no requiere sintaxis valida)
  AND     los errores se detectaran en B2 cuando el Admin solicite verificacion
```

**Razon**: El draft es un espacio de trabajo. Forzar sintaxis valida en cada save frustra la experiencia de edicion.

#### B1.7: DSL source excede limite de tamano

```
BEHAVIOR define_workflow_size_limit
  GIVEN   el Admin envia un dsl_source que excede el limite permitido
  WHEN    el sistema valida la solicitud
  THEN    el sistema rechaza con error de tamano
  AND     el error indica el limite maximo y el tamano enviado
```

**Limite sugerido**: 64KB para `dsl_source`, 64KB para `spec_source`. Suficiente para workflows complejos, protege contra abuso.

#### B1.8: Ediciones concurrentes al mismo draft

```
BEHAVIOR define_workflow_concurrent_edit
  GIVEN   dos sesiones del mismo Admin editan el mismo workflow draft
  WHEN    ambas sesiones envian actualizaciones
  THEN    la ultima escritura gana (last-write-wins)
  AND     el campo updated_at refleja el timestamp mas reciente
```

**Razon**: Para MVP no se requiere conflict resolution. Si en el futuro se necesita, se puede agregar `etag`/`if-match` headers.

---

## B2: Verificacion de Workflow

### Nivel 1 — Flujo principal

```
BEHAVIOR verify_workflow
  GIVEN   un workflow en estado draft con spec_source y dsl_source
  WHEN    el Admin solicita verificacion
  THEN    el Judge parsea el DSL y valida consistencia contra el spec
  AND     si todas las verificaciones pasan, el workflow transiciona a status=testing
  AND     el resultado incluye passed=true
```

**Ejemplo**: Admin solicita verificacion de `resolve_support_case`. El Judge parsea el DSL (sintaxis OK), verifica que cada BEHAVIOR del spec tiene cobertura en el DSL, no encuentra violaciones. Resultado: passed=true, workflow status=testing.

### Nivel 2 — Flujos alternativos y de error

#### B2.1: Judge encuentra violaciones

```
BEHAVIOR verify_workflow_violations
  GIVEN   un workflow con spec y DSL inconsistentes
  WHEN    el Admin solicita verificacion
  THEN    el Judge reporta TODAS las violaciones (no se detiene en la primera)
  AND     cada violacion incluye: tipo de check, descripcion, ubicacion en DSL/spec
  AND     el workflow permanece en status=draft
  AND     el Admin corrige y puede volver a solicitar verificacion
```

**Tipos de violacion (Fase 2)**:
1. BEHAVIOR sin cobertura DSL — "BEHAVIOR detect_intent no tiene camino de ejecucion en el DSL"
2. BEHAVIOR contradice CONSTRAINT — "BEHAVIOR X produce SET sin tool registrado (viola CONSTRAINT 2)"
3. ACTOR no definido — "Actor 'Manager' referenciado en BEHAVIOR pero no definido en ACTORS"
4. GIVEN inalcanzable — "Estado 'lead.qualified' nunca es producido por ningun BEHAVIOR"
5. DSL no matchea BEHAVIOR — "DSL contiene NOTIFY pero ningun BEHAVIOR lo describe"

#### B2.2: Verificacion sin spec_source

```
BEHAVIOR verify_workflow_no_spec
  GIVEN   un workflow con dsl_source pero sin spec_source
  WHEN    el Admin solicita verificacion
  THEN    el Judge ejecuta solo validaciones sintacticas del DSL
  AND     los checks de consistencia spec↔DSL se omiten con warnings
  AND     el resultado puede ser passed=true (solo sintaxis) con warnings
```

**Razon**: Permitir workflows sin spec reduce la barrera de entrada. El spec es recomendado pero no bloqueante.

#### B2.3: DSL con errores de sintaxis

```
BEHAVIOR verify_workflow_syntax_error
  GIVEN   un workflow con DSL que tiene errores de sintaxis
  WHEN    el Admin solicita verificacion
  THEN    el Judge reporta los errores de sintaxis con linea y columna
  AND     no se ejecutan los checks de consistencia (dependen del AST)
  AND     el workflow permanece en status=draft
```

**Ejemplo**: "Error de sintaxis en linea 4, columna 12: se esperaba NEWLINE despues de ON, se encontro '='"

#### B2.4: Judge con warnings (no bloqueantes)

```
BEHAVIOR verify_workflow_warnings
  GIVEN   un workflow valido pero con terminos ambiguos o patrones riesgosos
  WHEN    el Judge completa la verificacion
  THEN    el resultado incluye passed=true + warnings[]
  AND     cada warning describe el termino o patron y su riesgo
  AND     el workflow puede activarse a pesar de los warnings
```

**Ejemplo de warning**: "El termino 'high priority' en BEHAVIOR 3 podria interpretarse de mas de una forma (Judge check 8)"

#### B2.5: Re-verificacion despues de correccion

```
BEHAVIOR verify_workflow_re_verify
  GIVEN   un workflow cuya verificacion fallo y el Admin corrigio el DSL
  WHEN    el Admin solicita verificacion nuevamente
  THEN    el Judge ejecuta todos los checks desde cero
  AND     el resultado refleja el estado actual del DSL (no el anterior)
```

**Nota**: No hay cache de verificaciones previas. Cada verificacion es completa e independiente.

#### B2.6: Activacion post-verificacion

```
BEHAVIOR verify_workflow_activate
  GIVEN   un workflow en status=testing (verificacion pasada)
  WHEN    el Admin solicita activacion
  THEN    el Judge re-verifica como safety net
  AND     si pasa, el workflow transiciona a status=active
  AND     si ya existe un workflow activo con el mismo nombre, el anterior se archiva (→ B7)
  AND     solo puede haber 1 activo por (workspace, name) en todo momento
```

**Constraint aplicado**: UNIQUE activo por (workspace, name). Activar uno nuevo archiva el anterior automaticamente.

### Nivel 3 — Edge cases

#### B2.7: Verificacion de workflow no-draft

```
BEHAVIOR verify_workflow_wrong_status
  GIVEN   un workflow en status=active o status=archived
  WHEN    el Admin solicita verificacion
  THEN    el sistema rechaza con error de estado invalido
  AND     informa que solo workflows en draft pueden verificarse
```

#### B2.8: Spec con bloques incompletos

```
BEHAVIOR verify_workflow_incomplete_spec
  GIVEN   un workflow con spec_source que tiene bloques CONTEXT y ACTORS pero no BEHAVIOR
  WHEN    el Judge intenta parsear el spec
  THEN    el Judge reporta warning: "bloques faltantes en spec: BEHAVIOR, CONSTRAINTS"
  AND     los checks que dependen de esos bloques se omiten
  AND     la verificacion puede pasar (solo checks disponibles)
```

#### B2.9: DSL referencia verbo no soportado

```
BEHAVIOR verify_workflow_unknown_verb
  GIVEN   un DSL contiene un verbo que no esta en el conjunto permitido
  WHEN    el parser intenta tokenizar el DSL
  THEN    el parser reporta error de sintaxis con el token desconocido
  AND     la verificacion falla
```

**Verbos permitidos**: WORKFLOW, ON, IF, SET, AGENT, NOTIFY, SURFACE, WAIT, DISPATCH. Cualquier otro es error.

---

## B3: Ejecucion de Workflow

### Nivel 1 — Flujo principal

```
BEHAVIOR execute_workflow
  GIVEN   un workflow activo esta asociado a un agent_definition con agent_type="dsl"
  WHEN    ocurre un evento que matchea la clausula ON del workflow
  THEN    el Orchestrator crea un agent_run (status=running)
  AND     el DSLRunner parsea el DSL, genera AST, ejecuta via Runtime
  AND     cada accion (SET, NOTIFY, AGENT) se ejecuta via ToolRegistry
  AND     cada paso se registra como agent_run_step
  AND     al completar, el agent_run transiciona a status=success
```

**Ejemplo**: Evento `case.created` → workflow `resolve_support_case` se dispara → Runtime evalua `IF case.status == "open"` → busca evidence → SET case.status = "resolved" → agent_run success.

### Nivel 2 — Flujos alternativos y de error

#### B3.1: Condicion IF evalua a false

```
BEHAVIOR execute_workflow_condition_false
  GIVEN   un workflow en ejecucion llega a una clausula IF
  WHEN    la condicion evalua a false
  THEN    el body del IF se omite (no se ejecutan sus statements)
  AND     la ejecucion continua con el siguiente statement al mismo nivel de indentacion
  AND     el paso se registra como agent_run_step con status=skipped
```

**Ejemplo**: `IF case.priority IN ["high","urgent"]` evalua a false porque la prioridad es "low" → se salta el bloque de escalation, continua con el siguiente statement.

#### B3.2: Tool call falla

```
BEHAVIOR execute_workflow_tool_failure
  GIVEN   un workflow en ejecucion invoca un SET/NOTIFY que se traduce a un tool call
  WHEN    el ToolRegistry retorna error (tool no encontrado, params invalidos, tool inactivo)
  THEN    el agent_run transiciona a status=failed
  AND     el error se registra en el agent_run_step correspondiente
  AND     los pasos posteriores no se ejecutan
  AND     se emite un audit event con el detalle del error
```

**No hay retry automatico a nivel DSL** (MVP). El admin puede re-ejecutar manualmente.

#### B3.3: Policy engine bloquea tool call

```
BEHAVIOR execute_workflow_policy_blocked
  GIVEN   un workflow ejecuta SET que se traduce a un tool call
  WHEN    el PolicyEngine deniega la ejecucion (permisos, PII, no-cloud)
  THEN    el agent_run transiciona a status=rejected con la razon de la policy
  AND     el tool call no se ejecuta
  AND     se emite audit event con policy_violation
```

**Constraint aplicado**: "Un agente no puede ejecutar una herramienta sin permisos validos"

#### B3.4: Tool call requiere aprobacion

```
BEHAVIOR execute_workflow_approval_required
  GIVEN   un workflow ejecuta un tool call marcado como sensible
  WHEN    el PolicyEngine determina que requiere aprobacion humana
  THEN    se crea un approval_request con la accion propuesta
  AND     la ejecucion del workflow se pausa
  AND     el agent_run permanece en status=accepted (esperando aprobacion)
  AND     cuando el aprobador acepta, la ejecucion resume
  AND     cuando el aprobador rechaza, → B6 (override humano)
```

**Constraint aplicado**: "Una accion sensible no puede ejecutarse sin aprobacion humana previa"

#### B3.5: Sub-agente (AGENT verb) falla

```
BEHAVIOR execute_workflow_subagent_failure
  GIVEN   un workflow ejecuta AGENT evaluate_intent(...)
  WHEN    el sub-agente (via RunnerRegistry) retorna error o status=failed
  THEN    el agent_run del workflow padre transiciona a status=failed
  AND     el error del sub-agente se propaga al paso correspondiente
  AND     el agent_run del sub-agente queda registrado independientemente
```

#### B3.6: Multiples workflows matchean el mismo evento

```
BEHAVIOR execute_workflow_multiple_match
  GIVEN   dos o mas workflows activos tienen la misma clausula ON
  WHEN    ocurre el evento
  THEN    cada workflow se ejecuta de forma independiente
  AND     cada uno crea su propio agent_run
  AND     los tool calls de cada ejecucion son independientes
  AND     si hay conflicto (ambos hacen SET al mismo campo), el ultimo en ejecutar gana
```

**Nota**: No hay prioridad entre workflows. Para MVP, el orden de ejecucion es no determinista. Si se necesita ordenamiento, se agrega en futuro.

#### B3.7: Ningun workflow matchea el evento

```
BEHAVIOR execute_workflow_no_match
  GIVEN   un evento se publica en el EventBus
  WHEN    ningun workflow activo tiene una clausula ON que matchee
  THEN    el evento se descarta sin efecto
  AND     no se crea ningun agent_run
```

#### B3.8: Sub-agente (AGENT verb) retorna abstained

```
BEHAVIOR execute_workflow_subagent_abstained
  GIVEN   un workflow ejecuta AGENT search_knowledge(...)
  WHEN    el sub-agente retorna status=abstained (evidencia insuficiente)
  THEN    el resultado del AGENT se refleja en el contexto de ejecucion
  AND     los IF siguientes pueden evaluar el resultado (e.g., evidence.top_score < threshold)
  AND     la ejecucion continua — abstained no es un error fatal
```

**Razon**: Abstention es informacion valida ("no se encontro evidencia suficiente"). El workflow puede tener ramas que manejan ese caso.

#### B3.9: Trigger manual via API

```
BEHAVIOR execute_workflow_manual_trigger
  GIVEN   un workflow activo
  WHEN    un Admin/User invoca POST /workflows/{id}/execute con un payload de input
  THEN    el workflow se ejecuta igual que si fuera disparado por evento
  AND     el agent_run registra trigger_type=manual y triggered_by=user_id
  AND     el input del payload se inyecta en el contexto de ejecucion
```

### Nivel 3 — Edge cases

#### B3.10: Ejecucion excede cuota de costo

```
BEHAVIOR execute_workflow_quota_exceeded
  GIVEN   un workflow esta en ejecucion y acumula costo (tokens LLM, tool calls)
  WHEN    el costo acumulado excede el limite del agent_definition
  THEN    la ejecucion se aborta con status=failed
  AND     el error indica quota_exceeded con el detalle de costos
  AND     los tool calls ya ejecutados no se revierten (no hay compensacion automatica)
```

#### B3.11: Workflow desactivado durante ejecucion

```
BEHAVIOR execute_workflow_deactivated_during_run
  GIVEN   un workflow se desactiva (status→archived) mientras tiene un agent_run en curso
  WHEN    el agent_run intenta ejecutar el siguiente paso
  THEN    el agent_run en curso se permite completar (no se aborta mid-execution)
  AND     nuevos triggers para ese workflow se rechazan
```

**Razon**: Abortar mid-execution puede dejar datos en estado inconsistente. Mejor dejar que la ejecucion actual termine.

#### B3.12: Referencia circular entre AGENT calls

```
BEHAVIOR execute_workflow_circular_agent
  GIVEN   un workflow A contiene AGENT B, y el workflow de B contiene AGENT A
  WHEN    el Runtime detecta la cadena circular
  THEN    la ejecucion se aborta con status=failed
  AND     el error indica la cadena de invocacion detectada
```

**Implementacion**: Depth counter en RunContext. Limite sugerido: 5 niveles de profundidad.

#### B3.13: Expression evaluator con tipos incompatibles

```
BEHAVIOR execute_workflow_type_mismatch
  GIVEN   un workflow tiene IF lead.score >= "high" (comparacion numero vs string)
  WHEN    el expression evaluator intenta evaluar
  THEN    la evaluacion falla con error de tipo
  AND     el agent_run transiciona a status=failed con el detalle
```

**Nota**: El expression evaluator no hace coercion de tipos. Comparaciones solo entre tipos compatibles (number-number, string-string).

#### B3.14: Campo inexistente en expression

```
BEHAVIOR execute_workflow_unknown_field
  GIVEN   un workflow tiene IF contact.nonexistent_field == "value"
  WHEN    el expression evaluator intenta resolver el campo
  THEN    el campo inexistente evalua a null
  AND     cualquier comparacion con null retorna false (excepto != null que retorna true)
  AND     la ejecucion continua — campo inexistente no es error fatal
```

**Razon**: En un sistema dinamico, las entidades pueden no tener todos los campos. Tratar ausencia como null es mas robusto que abortar.

---

## B4: Deteccion de Signals

### Nivel 1 — Flujo principal

```
BEHAVIOR detect_signal
  GIVEN   un workflow o agente evalua interacciones de una entidad CRM
  WHEN    el analisis produce un resultado con tipo y confianza
  THEN    se crea un signal asociado a la entidad
  AND     el signal incluye tipo, confianza, evidence IDs y source (agent_run o workflow)
  AND     se publica evento signal.created en el EventBus
  AND     el signal es visible para el Salesperson responsable
```

**Ejemplo**: Agente evalua interacciones de Lead #123 → confianza 0.92 de intent_high → Signal creado → EventBus publica → el salesperson ve el signal en su dashboard.

### Nivel 2 — Flujos alternativos y de error

#### B4.1: Signal sin evidencia suficiente

```
BEHAVIOR detect_signal_no_evidence
  GIVEN   un workflow intenta crear un signal
  WHEN    los evidence_ids estan vacios o las evidencias no existen
  THEN    el sistema rechaza la creacion del signal
  AND     el error indica que un signal requiere al menos una evidencia valida
```

**Constraint aplicado**: "Un signal no puede crearse sin evidencia que lo respalde"

#### B4.2: Signal duplicado para misma entidad y tipo

```
BEHAVIOR detect_signal_duplicate
  GIVEN   ya existe un signal activo con el mismo entity_id y signal_type
  WHEN    se intenta crear otro signal con los mismos valores
  THEN    el sistema crea el nuevo signal (no deduplica)
  AND     ambos signals coexisten con timestamps diferentes
  AND     el mas reciente tiene mayor relevancia en la UI
```

**Razon**: Multiples evaluaciones pueden generar signals del mismo tipo con diferente confianza/evidencia. No deduplicar preserva la trazabilidad.

#### B4.3: Signal expirado

```
BEHAVIOR detect_signal_expired
  GIVEN   un signal tiene expires_at definido
  WHEN    la fecha actual supera expires_at
  THEN    el signal transiciona a status=expired
  AND     deja de mostrarse en vistas activas
  AND     sigue disponible en busquedas historicas
```

**Implementacion**: La expiracion se evalua en queries (WHERE status='active' AND (expires_at IS NULL OR expires_at > NOW())), no con un job de limpieza.

#### B4.4: Salesperson descarta signal

```
BEHAVIOR detect_signal_dismissed
  GIVEN   un signal activo visible para un Salesperson
  WHEN    el Salesperson descarta el signal explicitamente
  THEN    el signal transiciona a status=dismissed
  AND     se registra quien lo descarto y cuando
  AND     el signal deja de mostrarse en vistas activas
  AND     se emite evento signal.dismissed en EventBus
```

#### B4.5: Signal para entidad inexistente

```
BEHAVIOR detect_signal_invalid_entity
  GIVEN   un workflow intenta crear un signal referenciando un entity_id
  WHEN    la entidad no existe en el workspace
  THEN    el sistema rechaza la creacion con error de referencia invalida
```

### Nivel 3 — Edge cases

#### B4.6: Signal con confianza en los limites

```
BEHAVIOR detect_signal_confidence_bounds
  GIVEN   un workflow crea un signal con confianza
  WHEN    la confianza es < 0.0 o > 1.0
  THEN    el sistema rechaza con error de validacion
  AND     el error indica que confianza debe estar en rango [0.0, 1.0]
```

#### B4.7: Evidence IDs referencian items eliminados

```
BEHAVIOR detect_signal_stale_evidence
  GIVEN   un workflow crea un signal con evidence_ids
  WHEN    algunos de los evidence_ids referencian knowledge_items eliminados
  THEN    el sistema crea el signal con los IDs tal cual (no valida existencia de cada evidence)
  AND     la UI muestra "evidencia no disponible" para los IDs faltantes
```

**Razon**: Validar cada evidence ID en creacion agrega latencia y crea race conditions. La UI maneja la degradacion.

#### B4.8: Multiples signals en una sola ejecucion de workflow

```
BEHAVIOR detect_signal_batch
  GIVEN   un workflow ejecuta SURFACE para multiples entidades en un loop logico
  WHEN    el Runtime crea multiples signals en la misma ejecucion
  THEN    cada signal se crea independientemente con su propio ID
  AND     todos comparten el mismo source_id (agent_run_id)
  AND     cada creacion publica su propio evento signal.created
```

---

## B5: Accion Diferida

### Nivel 1 — Flujo principal

```
BEHAVIOR defer_action
  GIVEN   un workflow en ejecucion llega a una clausula WAIT 48 hours
  WHEN    el Runtime procesa el statement WAIT
  THEN    el Runtime persiste el estado de ejecucion (step index + contexto)
  AND     crea un scheduled_job con execute_at = now + 48 hours
  AND     el agent_run permanece en status=accepted (no terminal)
  AND     cuando el scheduler detecta que execute_at ha pasado, resume la ejecucion
  AND     la ejecucion continua desde el paso siguiente al WAIT
```

**Ejemplo**: Workflow notifica al salesperson, WAIT 48 hours, luego evalua si el salesperson actuo. Si no, envia reminder.

### Nivel 2 — Flujos alternativos y de error

#### B5.1: Server restart durante WAIT

```
BEHAVIOR defer_action_restart_recovery
  GIVEN   un scheduled_job esta pendiente y el servidor se reinicia
  WHEN    el servidor arranca y el scheduler inicia su goroutine de polling
  THEN    el scheduler encuentra los jobs pendientes en la DB
  AND     los que tienen execute_at <= now se ejecutan inmediatamente
  AND     los que tienen execute_at en el futuro se procesan en el proximo ciclo de polling
```

**Garantia**: No se pierden jobs. La DB es la fuente de verdad, no la memoria.

#### B5.2: Workflow archivado durante WAIT

```
BEHAVIOR defer_action_workflow_archived
  GIVEN   un workflow tiene un agent_run pausado por WAIT
  WHEN    el Admin archiva el workflow
  THEN    los scheduled_jobs asociados se cancelan (status=cancelled)
  AND     los agent_runs en pausa transicionan a status=failed con razon "workflow_archived"
```

**Razon**: Ejecutar un workflow archivado viola el constraint "Un workflow archivado no puede recibir nuevas ejecuciones". Aunque tecnicamente es un resume (no una nueva ejecucion), es mas seguro cancelar.

#### B5.3: WAIT con duracion cero o negativa

```
BEHAVIOR defer_action_zero_duration
  GIVEN   un workflow tiene WAIT 0 hours o WAIT -1 hours
  WHEN    el Runtime procesa el statement
  THEN    WAIT 0: se trata como un yield — resume inmediatamente en el proximo ciclo de polling
  AND     WAIT negativo: se rechaza con error de validacion en parse time (no en runtime)
```

#### B5.4: Resume con estado de entidad cambiado

```
BEHAVIOR defer_action_state_changed
  GIVEN   un workflow pauso con WAIT y durante la pausa alguien modifico la entidad
  WHEN    el scheduler resume la ejecucion
  THEN    la ejecucion continua con el estado ACTUAL de la entidad (no el snapshot de antes del WAIT)
  AND     los IF siguientes al WAIT evaluan contra datos frescos
```

**Ejemplo**: WAIT 48h, luego IF salesperson.has_not_acted. Si durante las 48h el salesperson actuo, la condicion evalua a false y se salta el reminder.

#### B5.5: Multiples WAITs en secuencia

```
BEHAVIOR defer_action_sequential_waits
  GIVEN   un workflow tiene WAIT 24 hours seguido de WAIT 48 hours
  WHEN    el Runtime procesa cada WAIT
  THEN    cada WAIT crea su propio scheduled_job
  AND     la ejecucion se pausa y resume secuencialmente
  AND     el agent_run acumula el tiempo total de espera
```

### Nivel 3 — Edge cases

#### B5.6: Duracion excede limite maximo

```
BEHAVIOR defer_action_max_duration
  GIVEN   un workflow tiene WAIT 365 days
  WHEN    el parser valida la duracion
  THEN    el parser rechaza con error: duracion maxima es 30 dias
  AND     el error se reporta en verificacion (B2)
```

**Limite sugerido**: 30 dias. Workflows con esperas mayores probablemente necesitan un diseño diferente.

#### B5.7: Alta concurrencia de scheduled jobs

```
BEHAVIOR defer_action_high_concurrency
  GIVEN   cientos de scheduled_jobs tienen execute_at en el mismo momento
  WHEN    el scheduler detecta los jobs pendientes
  THEN    los jobs se procesan en lotes secuenciales (no todos en paralelo)
  AND     cada lote ejecuta un numero limitado de resumes concurrentes
  AND     los jobs restantes se procesan en el siguiente ciclo de polling
```

**Limite sugerido**: 10 resumes concurrentes por ciclo de polling. Evita saturar el sistema.

#### B5.8: Scheduled job falla al resumir

```
BEHAVIOR defer_action_resume_failure
  GIVEN   un scheduled_job se ejecuta y el resume del workflow falla
  WHEN    el scheduler intenta invocar el DSLRunner
  THEN    el scheduled_job se marca como executed (no se reintenta)
  AND     el agent_run transiciona a status=failed
  AND     se emite audit event con el error
```

**Razon**: Reintentar un resume fallido puede causar side effects duplicados. Mejor loguear y permitir re-trigger manual.

---

## B6: Override Humano

### Nivel 1 — Flujo principal

```
BEHAVIOR human_override
  GIVEN   un agente ha propuesto una accion (via approval_request)
  WHEN    el Salesperson o Admin rechaza la accion
  THEN    la accion propuesta no se ejecuta
  AND     el override se registra en el agent_run con tipo=rejected, actor, razon
  AND     el agent_run refleja el override en su output
```

**Ejemplo**: Agente propone SET case.status = "resolved". El soporte humano rechaza porque necesita mas info. El override se registra, el case permanece abierto.

### Nivel 2 — Flujos alternativos y de error

#### B6.1: Override con modificacion

```
BEHAVIOR human_override_modify
  GIVEN   un agente ha propuesto una accion via approval_request
  WHEN    el aprobador modifica la accion propuesta (cambia parametros) y aprueba
  THEN    la accion se ejecuta con los parametros modificados
  AND     el override se registra como tipo=modified con parametros originales y modificados
  AND     el tool call en el agent_run refleja los parametros finales (modificados)
```

**Ejemplo**: Agente propone SET case.priority = "high". El manager modifica a "medium" y aprueba. Se ejecuta con priority="medium".

#### B6.2: Override despues de ejecucion (compensacion)

```
BEHAVIOR human_override_post_execution
  GIVEN   un agente ya ejecuto una accion (tool call completado)
  WHEN    el Salesperson quiere revertir la accion
  THEN    el sistema NO revierte automaticamente (no hay compensacion automatica)
  AND     el Salesperson debe ejecutar la accion correctiva manualmente (o via otro tool call)
  AND     el override se registra como tipo=post_execution_feedback en el agent_run
```

**Razon**: La compensacion automatica es compleja y riesgosa (e.g., revertir un email enviado no es posible). El override post-ejecucion se registra como feedback para mejorar futuras ejecuciones.

#### B6.3: Approval timeout

```
BEHAVIOR human_override_timeout
  GIVEN   un approval_request esta pendiente
  WHEN    pasa el tiempo limite sin respuesta del aprobador
  THEN    la accion no se ejecuta (timeout = rechazo implicito)
  AND     el approval_request transiciona a status=expired
  AND     el agent_run transiciona a status=failed con razon "approval_timeout"
```

**Constraint aplicado**: "Una accion sensible no puede ejecutarse sin aprobacion humana previa". El silencio no es aprobacion.

#### B6.4: Override de accion delegada (B8)

```
BEHAVIOR human_override_delegated
  GIVEN   un workflow delego via DISPATCH y el agente externo respondio ACCEPTED
  WHEN    el Admin quiere cancelar la delegacion
  THEN    el sistema no puede cancelar la ejecucion externa (no hay protocolo de cancelacion)
  AND     el override se registra como tipo=delegation_override
  AND     el agent_run local se marca con nota de override
```

**Razon**: Una vez delegado, el control esta en el agente receptor. El protocolo AGENT_SPEC no define CANCEL — eso queda para P2+.

### Nivel 3 — Edge cases

#### B6.5: Multiples aprobadores para la misma accion

```
BEHAVIOR human_override_multiple_approvers
  GIVEN   una accion requiere aprobacion y tiene multiples aprobadores potenciales
  WHEN    el primer aprobador responde (aprueba o rechaza)
  THEN    la decision del primer respondente es definitiva
  AND     el approval_request transiciona a terminal
  AND     los demas aprobadores ya no pueden responder
```

#### B6.6: Override sobre override

```
BEHAVIOR human_override_recursive
  GIVEN   un Salesperson hizo un override y un Admin quiere revertir ese override
  WHEN    el Admin actua sobre la misma entidad
  THEN    el Admin ejecuta una nueva accion directa (no "revierte" el override)
  AND     ambas acciones quedan registradas como eventos independientes en la timeline
```

**Razon**: No existe concepto de "revertir un override". Cada accion es independiente y trazable.

#### B6.7: Override sin razon

```
BEHAVIOR human_override_no_reason
  GIVEN   un actor rechaza una accion propuesta
  WHEN    no proporciona razon del rechazo
  THEN    el sistema acepta el override sin razon (la razon es opcional)
  AND     se registra que no se proporciono razon
```

**Nota**: A diferencia de REJECTED en el protocolo (que DEBE incluir razon), el override humano permite rechazo sin explicacion. El constraint "Un override humano no puede ser descartado silenciosamente" se refiere a que el sistema no puede ignorar el override, no a que necesite razon.

---

## B7: Versionado y Rollback

### Nivel 1 — Flujo principal

```
BEHAVIOR version_workflow
  GIVEN   un workflow activo (v1) necesita cambios
  WHEN    el Admin solicita nueva version
  THEN    se crea v2 como copia del DSL de v1 con status=draft
  AND     v2 tiene parent_version_id = v1.id
  AND     v1 permanece activo (ejecutable) hasta que v2 se active
  AND     cuando v2 se activa, v1 transiciona automaticamente a archived
```

**Ejemplo**: Workflow `qualify_lead` v1 activo. Admin crea v2 (draft). Edita DSL de v2. Verifica (B2). Activa v2. v1 se archiva automaticamente.

### Nivel 2 — Flujos alternativos y de error

#### B7.1: Rollback a version anterior

```
BEHAVIOR version_workflow_rollback
  GIVEN   workflow v2 esta activo y v1 esta archivado
  WHEN    el Admin solicita rollback a v1
  THEN    v1 transiciona a status=active
  AND     v2 transiciona a status=archived
  AND     nuevos triggers usan v1
  AND     ejecuciones en curso de v2 se permiten completar
```

#### B7.2: Rollback saltando versiones

```
BEHAVIOR version_workflow_skip_rollback
  GIVEN   workflow tiene v1(archived), v2(archived), v3(active)
  WHEN    el Admin solicita rollback a v1 (saltando v2)
  THEN    v1 transiciona a active
  AND     v3 transiciona a archived
  AND     v2 permanece archived (no se toca)
```

**Razon**: El rollback es "activar una version especifica", no "ir un paso atras". El admin elige la version destino.

#### B7.3: Crear nueva version desde archived

```
BEHAVIOR version_workflow_from_archived
  GIVEN   un workflow v1 esta archived y v2 esta active
  WHEN    el Admin quiere basarse en v1 para una nueva version
  THEN    se crea v3 con DSL copiado de v1 (no de v2)
  AND     v3 tiene parent_version_id = v1.id
  AND     v3 entra como draft
```

**Razon**: A veces el admin quiere bifurcar desde una version anterior, no desde la activa.

#### B7.4: Ejecuciones en curso durante cambio de version

```
BEHAVIOR version_workflow_inflight_runs
  GIVEN   workflow v1 tiene agent_runs en curso (status=running o accepted)
  WHEN    v2 se activa y v1 se archiva
  THEN    los agent_runs de v1 en curso se permiten completar
  AND     nuevos triggers crean agent_runs usando v2
  AND     los agent_runs de v1 que completan despues del switch se registran con version_id=v1
```

#### B7.5: Eliminar un draft que nunca se activo

```
BEHAVIOR version_workflow_delete_draft
  GIVEN   un workflow v2 en status=draft que nunca fue activado
  WHEN    el Admin decide descartarlo
  THEN    v2 se elimina (hard delete, no archive)
  AND     v1 permanece en su estado actual (no se altera)
  AND     la eliminacion se registra en audit trail
```

**Razon**: Un draft que nunca se activo no tiene ejecuciones ni dependencias. Eliminarlo mantiene limpia la lista de versiones.

### Nivel 3 — Edge cases

#### B7.6: Creacion de version concurrente

```
BEHAVIOR version_workflow_concurrent_version
  GIVEN   dos requests simultaneos crean nueva version del mismo workflow
  WHEN    ambos intentan INSERT con el mismo version number
  THEN    la constraint UNIQUE(workspace_id, name, version) permite solo uno
  AND     el segundo request falla con error de conflicto
  AND     el segundo request puede reintentar con version number incrementado
```

#### B7.7: Rollback durante verificacion (status=testing)

```
BEHAVIOR version_workflow_rollback_during_testing
  GIVEN   v2 esta en status=testing y v1 esta archived
  WHEN    el Admin solicita rollback a v1
  THEN    v1 transiciona a active
  AND     v2 transiciona a draft (no archived, ya que nunca fue activo)
```

**Razon**: Un workflow en testing nunca ejecuto nada. Volver a draft es mas intuitivo que archivarlo.

#### B7.8: Maximo de versiones por workflow

```
BEHAVIOR version_workflow_max_versions
  GIVEN   un workflow tiene N versiones (activas + archived)
  WHEN    N no tiene limite explicito
  THEN    el sistema permite crear versiones indefinidamente
  AND     las versiones archived se pueden purgar manualmente (hard delete batch)
```

**Nota**: No se impone limite de versiones en MVP. Si la acumulacion se vuelve problema, se agrega purge en futuro.

---

## B8: Delegacion entre Agentes

### Nivel 1 — Flujo principal

```
BEHAVIOR delegate_workflow
  GIVEN   un workflow en ejecucion contiene DISPATCH TO agent_x WITH workflow_name
  WHEN    el Runtime procesa la clausula DISPATCH
  THEN    el DSL se serializa y se envia al agente receptor
  AND     el agente receptor parsea, verifica y responde:
          - ACCEPTED: ejecutara el workflow
          - REJECTED: no puede ejecutar, incluye razon
          - DELEGATED: reenvio a otro agente, incluye destino
  AND     el agent_run del workflow original registra la delegacion y la respuesta
```

**Ejemplo**: Workflow de soporte encuentra que el caso requiere expertise de producto → DISPATCH TO product_specialist WITH case_analysis → el agente especialista responde ACCEPTED y lo ejecuta.

### Nivel 2 — Flujos alternativos y de error

#### B8.1: Respuesta REJECTED

```
BEHAVIOR delegate_workflow_rejected
  GIVEN   el workflow envia DISPATCH y el receptor responde REJECTED
  WHEN    el Runtime recibe la respuesta
  THEN    el agent_run registra la razon del rechazo
  AND     el workflow transiciona a status=failed (o delegated, dependiendo del contexto)
  AND     no se reintenta automaticamente
  AND     la razon queda disponible para analisis
```

**Constraint aplicado**: "Un REJECTED debe siempre incluir la razon — nunca puede ser silencioso"

#### B8.2: Respuesta DELEGATED (cadena de delegacion)

```
BEHAVIOR delegate_workflow_chain
  GIVEN   el workflow envia DISPATCH a agent_A y este responde DELEGATED con destino agent_B
  WHEN    el Runtime recibe la respuesta DELEGATED
  THEN    el Runtime NO sigue la cadena automaticamente
  AND     registra la delegacion (agent_A → agent_B) en el agent_run
  AND     el workflow transiciona a status=delegated
  AND     el Admin decide manualmente si re-dispatch a agent_B
```

**Razon**: Seguir cadenas automaticamente puede crear loops o delegaciones no autorizadas. Mejor control humano.

#### B8.3: Timeout del agente receptor

```
BEHAVIOR delegate_workflow_timeout
  GIVEN   el workflow envia DISPATCH y el receptor no responde
  WHEN    pasa el timeout configurado (default: 60 segundos)
  THEN    el agent_run transiciona a status=failed con razon "dispatch_timeout"
  AND     se registra el intento de dispatch y el timeout
```

#### B8.4: Agente receptor no disponible

```
BEHAVIOR delegate_workflow_unavailable
  GIVEN   el workflow intenta DISPATCH a un agente externo
  WHEN    el agente no es alcanzable (network error, DNS failure, etc.)
  THEN    el agent_run transiciona a status=failed con razon "dispatch_unreachable"
  AND     no se reintenta automaticamente
```

#### B8.5: Dispatch interno (mismo sistema)

```
BEHAVIOR delegate_workflow_internal
  GIVEN   un workflow tiene DISPATCH TO support_agent WITH case_analysis
  WHEN    el agente receptor es interno (existe en el mismo RunnerRegistry)
  THEN    se invoca como sub-agente local (similar a AGENT verb)
  AND     no se usa protocolo HTTP/MCP — se ejecuta in-process
  AND     la respuesta es inmediata (ACCEPTED/REJECTED)
  AND     el registro es identico al dispatch externo para trazabilidad
```

### Nivel 3 — Edge cases

#### B8.6: Delegacion circular

```
BEHAVIOR delegate_workflow_circular
  GIVEN   agent_A envia DISPATCH a agent_B, y agent_B tiene un workflow que hace DISPATCH a agent_A
  WHEN    el Runtime detecta la cadena circular
  THEN    el segundo DISPATCH se rechaza con REJECTED y razon "circular_delegation_detected"
  AND     el agent_run de agent_B falla
```

**Implementacion**: Cada DISPATCH incluye un header `X-Delegation-Chain: [agent_A, agent_B]`. Si el receptor esta en la cadena, rechaza.

#### B8.7: Delegacion con DSL modificado

```
BEHAVIOR delegate_workflow_modified_dsl
  GIVEN   el workflow envia DSL al agente receptor
  WHEN    el receptor modifica el DSL antes de ejecutar
  THEN    el receptor ejecuta su version modificada (si pasa su propio Judge)
  AND     la version modificada NO se retroalimenta al workflow original
  AND     el workflow original solo conoce la respuesta (ACCEPTED/REJECTED/DELEGATED)
```

**Razon**: Cada agente es autonomo en su ejecucion. El dispatch es "aqui tienes un workflow, ejecutalo si puedes", no "ejecuta esto exactamente como te lo mando".

#### B8.8: Ejecucion parcial por agente receptor

```
BEHAVIOR delegate_workflow_partial_execution
  GIVEN   el agente receptor acepta el workflow y empieza a ejecutar
  WHEN    la ejecucion falla a mitad de camino
  THEN    el receptor completa su agent_run con status=partial o failed
  AND     la respuesta al workflow original refleja el estado final
  AND     los tool calls ya ejecutados por el receptor no se revierten
```

#### B8.9: DISPATCH en Fase 2 (stub)

```
BEHAVIOR delegate_workflow_stub
  GIVEN   el sistema esta en Fase 2 (antes de implementar Protocol Handler)
  WHEN    un workflow contiene DISPATCH
  THEN    el Runtime retorna status=rejected con razon "dispatch_not_implemented"
  AND     el agent_run falla con mensaje claro
  AND     el workflow puede verificarse (Judge no bloquea por DISPATCH)
```

**Razon**: DISPATCH es Fase 3. El parser y Judge lo reconocen, pero el Runtime no lo ejecuta hasta que el Protocol Handler exista.

---

## Matriz: Constraints × Behaviors

Cada constraint debe ser verificable en al menos un behavior. Esta matriz documenta donde se aplica cada constraint.

| Constraint | B1 | B2 | B3 | B4 | B5 | B6 | B7 | B8 |
|---|---|---|---|---|---|---|---|---|
| Workflow no ejecuta sin verificacion Judge | | **B2** | **B3** | | | | **B7** | |
| Mutacion solo via herramientas registradas | | | **B3** | | | | | |
| Agente sin permisos no ejecuta herramienta | | | **B3.3** | | | | | |
| Accion sensible requiere aprobacion humana | | | **B3.4** | | | **B6** | | |
| Signal requiere evidencia | | | | **B4.1** | | | | |
| Override no se descarta silenciosamente | | | | | | **B6** | | |
| Workflow archivado no recibe ejecuciones | | | **B3.11** | | **B5.2** | | **B7** | |
| REJECTED incluye razon | | **B2.1** | | | | | | **B8.1** |
| Agentes Go siguen funcionando | **B1** | | **B3** | | | | | |

---

## Apendice: Flujos end-to-end

### E2E-1: Lifecycle completo de un workflow

```
B1 (crear draft)
  → B1 (editar DSL)
  → B2 (verificar — falla, corregir)
  → B2 (verificar — pasa)
  → B2.6 (activar)
  → B3 (ejecutar via evento)
  → B7 (crear v2)
  → B2 (verificar v2)
  → B2.6 (activar v2, v1 se archiva)
  → B7.1 (rollback a v1 si v2 tiene problemas)
```

### E2E-2: Ejecucion con approval y override

```
B3 (evento dispara workflow)
  → B3 ejecuta steps hasta encontrar accion sensible
  → B3.4 (approval_request creada, ejecucion pausa)
  → B6 (humano rechaza — override)
  → B3 (agent_run status=failed por override)
```

### E2E-3: Ejecucion con WAIT y signal

```
B3 (evento dispara workflow)
  → B3 ejecuta AGENT (sub-agente evalua interacciones)
  → B4 (signal creado con confianza 0.92)
  → B3 ejecuta NOTIFY salesperson
  → B5 (WAIT 48 hours — scheduler programa resume)
  → B5 (resume — evalua si salesperson actuo)
  → B3 IF salesperson.has_not_acted → NOTIFY reminder
```

### E2E-4: Delegacion con fallback

```
B3 (evento dispara workflow)
  → B8 (DISPATCH a agente externo)
  → B8.3 (timeout — agente no responde)
  → B3 (agent_run status=failed)
  → B3.9 (admin re-trigger manual con parametros ajustados)
```
