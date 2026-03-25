# Carta — Plan de Implementación

> Fecha: 2026-03-23
> Status: pending
> Spec: `docs/carta-spec.md`
> Task files: `docs/tasks/task_carta_<fase>_<num>.md`
> Sources of truth: `docs/carta-spec.md`, `docs/agent-spec-design.md`, `docs/architecture.md`

---

## Resumen

Carta es la evolución machine-readable del campo `spec_source` en el `Workflow`. Integra dos capas nuevas sobre la infraestructura existente:

1. **Análisis estático (Judge)**: el CartaParser parsea `spec_source`; el Judge verifica que el DSL sea consistente con los PERMIT y BEHAVIOR declarados en Carta
2. **Enforcement dinámico (Runtime)**: `DelegateEvaluator` y `GroundsValidator` se insertan como preflight antes del `DSLRuntime.ExecuteProgram`

No hay cambio de esquema de DB. No hay cambio al DSL existente. La retrocompatibilidad es total: `spec_source` en formato libre sigue por la ruta existente.

---

## Fases y Tasks

### Fase FC-L — Lexer

Base de todas las fases del parser. Sin dependencias externas.

| Task | Archivo | Descripción |
|---|---|---|
| FC-L.1 | `carta_token.go` | Constantes de token Carta + `cartaKeywords` map + `IsCartaKeyword()` |
| FC-L.2 | `carta_lexer.go` | `CartaLexer` wrappea `Lexer`, override keyword resolution vía `cartaKeywords` |

**Dep**: FC-L.1 → FC-L.2

---

### Fase FC-P — Parser

Structs (FC-P.1/2/3) son independientes entre sí y se pueden hacer en paralelo.
Block parsers (FC-P.4..8) pueden hacerse en paralelo una vez FC-P.1/2/3 + FC-L.2 estén listos.

| Task | Archivo | Descripción |
|---|---|---|
| FC-P.1 | `carta_ast.go` | Structs `CartaSummary` + `CartaGrounds` (reusar `knowledge.ConfidenceLevel`) |
| FC-P.2 | `carta_ast.go` | Structs `CartaPermit` + `CartaRate` + `CartaApprovalConfig` |
| FC-P.3 | `carta_ast.go` | Structs `CartaDelegate` + `CartaInvariant` + `CartaBudget` |
| FC-P.4 | `carta_parser.go` | `CartaParser` struct + `parseGroundsBlock()` |
| FC-P.5 | `carta_parser.go` | `parsePermitBlock()` — PERMIT sin cláusulas válido |
| FC-P.6 | `carta_parser.go` | `parseDelegateBlock()` — sequence `DELEGATE TO HUMAN` obligatoria |
| FC-P.7 | `carta_parser.go` | `parseInvariantBlock()` — múltiples `never`/`always` |
| FC-P.8 | `carta_parser.go` | `parseBudgetBlock()` — validar `on_exceed` |
| FC-P.9 | `carta_parser.go` | `ParseCarta()` + `parseAgentBlock()` + `parseProgram()` — orquestador |
| FC-P.10 | `carta_parser_test.go` | Golden tests por bloque + full Carta (Escenario A) + casos de error |

**Deps**:
- FC-P.1, FC-P.2, FC-P.3 independientes
- FC-L.2 + FC-P.1/2/3 → FC-P.4, FC-P.5, FC-P.6, FC-P.7, FC-P.8
- todos los block parsers → FC-P.9 → FC-P.10

---

### Fase FC-J — Judge

Análisis estático. Depende de FC-P.9 para tener `CartaSummary` tipado.

| Task | Archivo | Descripción |
|---|---|---|
| FC-J.1 | `judge.go` | `isCartaSource()` + branch en `appendInitialSpecConsistencyFindings` |
| FC-J.2 | `verb_mapper.go` | `ToolNameForStatement(stmt Statement) string` — sin ejecutar |
| FC-J.3 | `judge_carta.go` | Check 10: `RunCartaPermitChecks` — `tool_not_permitted` |
| FC-J.4 | `judge_carta.go` | Check 11: `RunCartaCoverageChecks` — `behavior_no_permit_or_delegate` |
| FC-J.5 | `judge_carta.go` | Check 12: `RunCartaGroundsPresenceCheck` — warning (no violation) |
| FC-J.6 | `judge_carta.go` | `RunCartaSpecDSLChecks` orquestador + wire en `judge.go` |
| FC-J.7 | `judge_carta_test.go` | Escenarios A/B/C como tests + backward compat spec libre format |

**Deps**:
- FC-P.9 → FC-J.1, FC-J.4, FC-J.5
- FC-P.9 + FC-J.2 → FC-J.3
- FC-J.3 + FC-J.4 + FC-J.5 → FC-J.6
- FC-J.6 → FC-J.7

Paralelo seguro: FC-J.2, FC-J.3, FC-J.4, FC-J.5 (una vez FC-P.9 listo)

---

### Fase FC-R — Runtime

Enforcement dinámico. Depende de FC-P.9 para `CartaSummary`. Usa servicios existentes.

| Task | Archivo | Descripción |
|---|---|---|
| FC-R.1 | `grounds_validator.go` | `GroundsValidator` + `Validate()` usando `EvidencePackService.BuildEvidencePack` |
| FC-R.2 | `runner.go` | +campo `GroundsValidator *GroundsValidator` a `RunContext` |
| FC-R.3 | `delegate_evaluator.go` | `EvaluateDelegate()` usando `ExpressionEvaluator` existente |
| FC-R.4 | `dsl_runner.go` | Preflight DelegateEvaluator: si trigerea → `HandoffService` + return delegated |
| FC-R.5 | `dsl_runner.go` | Preflight GroundsValidator: si falla → `HandoffService` + return abstained |
| FC-R.6 | `carta_policy_bridge.go` | `CartaBudgetToLimits(budget *CartaBudget) map[string]any` |
| FC-R.7 | `carta_policy_bridge.go` | `CartaInvariantsAsPolicyRules(invariants []CartaInvariant) []map[string]any` |
| FC-R.8 | `workflow/service.go` | Sync Carta budget → `AgentDefinition.Limits` en `Activate()` |
| FC-R.9 | `workflow/service.go` | Inject Carta invariants → policy version en `Activate()` |
| FC-R.10 | `grounds_validator_test.go` | Mock `EvidencePackService`, 5 casos (nil grounds, fuentes insuf., confidence, staleness, OK) |
| FC-R.11 | `delegate_evaluator_test.go` | 4 casos (condition true, false, When vacío, múltiples delegates) |
| FC-R.12 | `integration_test.go` | End-to-end Escenarios A (success), D (abstained), E (delegated) |

**Deps**:
- FC-P.9 → FC-R.1 → FC-R.2, FC-R.10
- FC-P.9 → FC-R.3 → FC-R.4, FC-R.11
- FC-R.2 + FC-R.4 → FC-R.5
- FC-P.9 → FC-R.6 → FC-R.8
- FC-P.9 → FC-R.7 → FC-R.9
- FC-R.5 + FC-R.9 → FC-R.12

Paralelo seguro: FC-R.1 + FC-R.3 | FC-R.6 + FC-R.7

---

## Diagrama de dependencias completo

```
FC-L.1 → FC-L.2 ─────────────────────────────────────────────────────────┐
                                                                           │
FC-P.1 ──┐                                                                 │
FC-P.2 ──┼──────────────────────────────────────────────────────→ FC-P.4 ←┘
FC-P.3 ──┘                                                    → FC-P.5
                                                              → FC-P.6
                                                              → FC-P.7
                                                              → FC-P.8
                                                                 │
FC-P.4+FC-P.5+FC-P.6+FC-P.7+FC-P.8 ──────────────────→ FC-P.9 → FC-P.10
                                                            │
           ┌────────────────────────────────────────────────┤
           │                │               │               │
           ▼                ▼               ▼               ▼
         FC-J.1           FC-R.1          FC-R.3          FC-R.6
           │             ↙      ↘        ↙     ↘            │
         FC-J.3       FC-R.2  FC-R.10 FC-R.4 FC-R.11     FC-R.8
           │            │        │       │
         FC-J.4       FC-R.5 ←──────────┘
           │                     │              FC-R.7 → FC-R.9
         FC-J.5                  │                         │
           │                     └────────────────→ FC-R.12 ←┘
           ▼
         FC-J.6 → FC-J.7
```

---

## Specs por task (guía para el coder al crear task files)

### FC-L.1

- **Objetivo**: tabla de keywords Carta y helper `IsCartaKeyword`
- **Scope**: solo `carta_token.go`. Constantes `TokenCarta`, `TokenHuman`, `TokenGrounds`, `TokenPermit`, `TokenDelegate`, `TokenInvariant`, `TokenBudget`, `TokenSkill`, `TokenNever`, `TokenAlways`. Map `cartaKeywords map[string]TokenType`. `IsCartaKeyword(s string) bool`.
- **Out of scope**: no tocar `token.go`, no implementar el lexer
- **Acceptance criteria**: `IsCartaKeyword("GROUNDS")` → true, `IsCartaKeyword("IF")` → false, `go build` verde
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaToken`

---

### FC-L.2

- **Objetivo**: lexer que tokeniza fuente Carta con keywords propios
- **Scope**: `carta_lexer.go`. Struct `CartaLexer`. `NewCartaLexer() *CartaLexer`. `Lex(source string) ([]Token, error)` — usa `Lexer.Lex` internamente pero sobreescribe resolución de identifiers via `cartaKeywords`.
- **Constraint**: reusar `emitIndentationTokens` del `Lexer` existente. No duplicar lógica INDENT/DEDENT.
- **Acceptance criteria**: `"GROUNDS\n  min_sources: 2\n"` → `[TokenGrounds, TokenNewline, TokenIndent, TokenIdent("min_sources"), TokenColon, TokenNumber(2), TokenNewline, TokenDedent]`
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaLexer`

---

### FC-P.1

- **Objetivo**: structs base `CartaSummary` + `CartaGrounds`
- **Scope**: `carta_ast.go`. Reusar `knowledge.ConfidenceLevel` de `internal/domain/knowledge/models.go`.
- **Constraint**: no redefinir `ConfidenceLevel`
- **Acceptance criteria**: `go build ./...` verde
- **Quality gate**: `go build ./internal/domain/agent/...`

---

### FC-P.2

- **Objetivo**: structs `CartaPermit` + `CartaRate` + `CartaApprovalConfig`
- **Scope**: añadir a `carta_ast.go`
- **Acceptance criteria**: `go build` verde
- **Quality gate**: `go build ./internal/domain/agent/...`

---

### FC-P.3

- **Objetivo**: structs `CartaDelegate` + `CartaInvariant` + `CartaBudget`
- **Scope**: añadir a `carta_ast.go`
- **Acceptance criteria**: `go build` verde
- **Quality gate**: `go build ./internal/domain/agent/...`

---

### FC-P.4

- **Objetivo**: parsear el bloque GROUNDS
- **Scope**: crear `carta_parser.go` con `CartaParser` struct + `parseGroundsBlock() (*CartaGrounds, error)`. Consume INDENT…DEDENT, parsea los 4 tipos de grounds_field.
- **Constraint**: errores reportan línea+columna (patrón `ParserError` de `syntax_error.go`). Campos desconocidos → Warning. `min_confidence: invalid` → error.
- **Acceptance criteria**: GROUNDS válido → `CartaGrounds` poblado; campo desconocido → warning; `min_confidence: foobar` → error
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaParserGrounds`

---

### FC-P.5

- **Objetivo**: parsear el bloque PERMIT
- **Scope**: añadir `parsePermitBlock() (*CartaPermit, error)` a `carta_parser.go`. PERMIT sin cláusulas (sin INDENT) es válido. Parsear `when:`, `rate:`, `approval:`.
- **Constraint**: `rate: -5 / hour` → error (valor negativo)
- **Acceptance criteria**: PERMIT sin cláusulas compila; `rate: 10 / hour` → `CartaRate{10, "hour"}`
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaParserPermit`

---

### FC-P.6

- **Objetivo**: parsear el bloque DELEGATE TO HUMAN
- **Scope**: añadir `parseDelegateBlock() (*CartaDelegate, error)`. Secuencia `DELEGATE TO HUMAN` obligatoria (3 tokens: TokenDelegate, TokenTo, TokenHuman). Parsear `when:`, `reason:`, `package:`.
- **Constraint**: `package:` parsea como lista de identifiers. `DELEGATE TO HUMAN` sin body → error.
- **Acceptance criteria**: `package: [evidence_ids, case_summary]` → `[]string{"evidence_ids", "case_summary"}`
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaParserDelegate`

---

### FC-P.7

- **Objetivo**: parsear el bloque INVARIANT
- **Scope**: añadir `parseInvariantBlock() ([]CartaInvariant, error)`. Múltiples `never:`/`always:` → múltiples structs.
- **Constraint**: sólo string literals como statement. `never:` sin string → error.
- **Acceptance criteria**: 2 líneas `never:` → slice con 2 `CartaInvariant`
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaParserInvariant`

---

### FC-P.8

- **Objetivo**: parsear el bloque BUDGET
- **Scope**: añadir `parseBudgetBlock() (*CartaBudget, error)`. `on_exceed` acepta sólo "pause"|"degrade"|"abort". Campos faltantes → zero value (no error).
- **Acceptance criteria**: budget parcial (solo `daily_tokens`) → resto en zero value; `on_exceed: invalid` → error
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaParserBudget`

---

### FC-P.9

- **Objetivo**: entry point público `ParseCarta()`
- **Scope**: añadir `ParseCarta(source string) (*CartaSummary, error)`. Flujo: CartaLexer.Lex → parseProgram (consume `CARTA identifier`) → detecta bloques AGENT/BUDGET → parseAgentBlock llama a los 4 block parsers.
- **Constraint**: source que no empieza con `CARTA ` → error inmediato. Bloques GROUNDS duplicados → error. BUDGET puede estar fuera de AGENT.
- **Acceptance criteria**: full Carta fuente (Escenario A) → `CartaSummary` completo; Carta sin GROUNDS → `Grounds: nil` + Warning; sintaxis rota → error con línea/columna
- **Quality gate**: `go test ./internal/domain/agent/... -run TestParseCarta`

---

### FC-P.10

- **Objetivo**: suite de tests dorada del parser
- **Scope**: `carta_parser_test.go`. Un test por cada bloque. Test full Carta = Escenario A. Tests de error: keyword desconocido, `min_confidence` inválido, `on_exceed` inválido, CARTA sin AGENT.
- **Acceptance criteria**: todos pasan, cobertura ≥ 80% de `carta_parser.go`
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaParser -v -cover`

---

### FC-J.1

- **Objetivo**: dispatch en judge.go según tipo de spec_source
- **Scope**: `judge.go`. Añadir `isCartaSource(source string) bool`. En `appendInitialSpecConsistencyFindings`: si `isCartaSource` → llamar `ParseCarta` + `RunCartaSpecDSLChecks`. Error de parse → `Violation{Code: "carta_parse_error"}`.
- **Constraint**: else branch existente (free format → `ParsePartialSpec`) queda intacto.
- **Acceptance criteria**: spec que empieza con `CARTA ` → branch nuevo; spec con `CONTEXT\n` → branch existente
- **Quality gate**: `go test ./internal/domain/agent/... -run TestWorkflowJudge`

---

### FC-J.2

- **Objetivo**: resolución de tool name desde statement sin ejecutar
- **Scope**: `verb_mapper.go`. Añadir `ToolNameForStatement(stmt Statement) string`. Reusar tabla estática existente. `IF`, `WAIT`, `ON` → `""`. `SET case.status` → `"update_case"`. `NOTIFY salesperson` → `"send_reply"`.
- **Constraint**: sólo lectura de tabla. No ejecutar, no acceder a DB.
- **Acceptance criteria**: los 3 ejemplos del scope retornan los valores esperados
- **Quality gate**: `go test ./internal/domain/agent/... -run TestToolNameForStatement`

---

### FC-J.3

- **Objetivo**: Check 10 — tool usado en DSL sin PERMIT en Carta
- **Scope**: crear `judge_carta.go`. Constantes `CartaCheckPermit = 10`, `CartaCheckCoverage = 11`, `CartaCheckGrounds = 12`. Función `RunCartaPermitChecks(carta *CartaSummary, program *Program) []Violation`. Por cada statement del AST: resolver tool → si tool != "" y no hay PERMIT → Violation.
- **Constraint**: tool `""` (IF, WAIT) → ignorar. Comparison case-insensitive.
- **Acceptance criteria**: NOTIFY sin PERMIT send_reply → 1 violation `tool_not_permitted`; tool en PERMIT → 0 violations
- **Quality gate**: `go test ./internal/domain/agent/... -run TestRunCartaPermitChecks`

---

### FC-J.4

- **Objetivo**: Check 11 — behavior del objetivo sin PERMIT ni DELEGATE en Carta
- **Scope**: añadir `RunCartaCoverageChecks(carta *CartaSummary, spec *SpecSummary) []Violation` a `judge_carta.go`. Por cada `SpecBehavior`: verificar PERMIT (fuzzy token match) o DELEGATE presente → si ninguno → Violation `behavior_no_permit_or_delegate`.
- **Constraint**: si no hay spec separado (spec es nil o sin BEHAVIOR blocks) → no-op.
- **Acceptance criteria**: behavior `escalate_unresolved` sin cobertura → violation; CartaSummary con DELEGATE → no violation para cualquier behavior
- **Quality gate**: `go test ./internal/domain/agent/... -run TestRunCartaCoverageChecks`

---

### FC-J.5

- **Objetivo**: Check 12 — warning si no hay GROUNDS declarados
- **Scope**: añadir `RunCartaGroundsPresenceCheck(carta *CartaSummary) []Warning` a `judge_carta.go`.
- **Constraint**: sólo Warning, nunca Violation.
- **Acceptance criteria**: Carta sin GROUNDS → 1 warning `carta_missing_grounds`; Carta con GROUNDS → 0 warnings
- **Quality gate**: `go test ./internal/domain/agent/... -run TestRunCartaGroundsPresenceCheck`

---

### FC-J.6

- **Objetivo**: orquestador `RunCartaSpecDSLChecks` + wire en judge.go
- **Scope**: añadir `RunCartaSpecDSLChecks(carta, program, spec) ([]Violation, []Warning)` que llama los 3 checks. Modificar `judge.go` branch `isCartaSource` para llamarlo.
- **Acceptance criteria**: Escenario B end-to-end: `judge.Verify(workflow_missing_permit)` → `JudgeResult{Passed: false, Violations[0].Code == "tool_not_permitted"}`
- **Quality gate**: `go test ./internal/domain/agent/... -run TestWorkflowJudge`

---

### FC-J.7

- **Objetivo**: suite de tests de integración del Judge con Carta
- **Scope**: `judge_carta_test.go`. Test Escenario A (passed=true). Test Escenario B (tool_not_permitted). Test Escenario C (behavior_no_permit_or_delegate). Test backward compat (spec libre format sin cambios).
- **Acceptance criteria**: 4 tests pasan
- **Quality gate**: `go test ./internal/domain/agent/... -run TestJudgeCarta -v`

---

### FC-R.1

- **Objetivo**: GroundsValidator que evalúa evidencia contra requisitos Carta
- **Scope**: `grounds_validator.go`. Struct `GroundsValidator{evidenceSvc *knowledge.EvidencePackService}`. Struct `GroundsResult{Met bool, EvidencePack, Reason}`. `Validate(ctx, grounds *CartaGrounds, query, workspaceID) (*GroundsResult, error)`.
  - `grounds==nil` → `{Met: true}` sin llamar al servicio
  - Llamar `BuildEvidencePack` → check MinSources, MinConfidence, MaxStaleness
  - Helper local `confidenceOrdinal`: low=0, medium=1, high=2
- **Constraint**: error del servicio → propagar (no convertir a GroundsResult)
- **Acceptance criteria**: 1 fuente/need 2 → `Met=false, Reason` incluye counts; `grounds=nil` → `Met=true` sin llamar al mock
- **Quality gate**: `go test ./internal/domain/agent/... -run TestGroundsValidator`

---

### FC-R.2

- **Objetivo**: añadir GroundsValidator a RunContext
- **Scope**: `runner.go`. +1 campo `GroundsValidator *GroundsValidator` a struct `RunContext`.
- **Constraint**: campo nullable — downstream verifica `rc.GroundsValidator != nil`. No wiring aquí (eso es en routes).
- **Acceptance criteria**: `go build ./...` verde
- **Quality gate**: `go build ./internal/domain/agent/...`

---

### FC-R.3

- **Objetivo**: DelegateEvaluator que evalúa condiciones de delegación proactiva
- **Scope**: `delegate_evaluator.go`. Struct `DelegateResult{Triggered bool, Reason, Package}`. Fn `EvaluateDelegate(delegates []CartaDelegate, evalCtx map[string]any, evaluator *ExpressionEvaluator) (*DelegateResult, error)`.
  - Por cada delegate: si `When==""` → skip; parsear condición via `ParseDSL` + extraer de `IfStatement`; evaluar contra evalCtx
  - Error de parsing → log warning + skip (no abortar)
- **Constraint**: parsing en runtime es aceptable para MVP (P1 pre-compila)
- **Acceptance criteria**: `case.tier=="enterprise"` + evalCtx enterprise → `Triggered=true`; standard → `Triggered=false`; `When=""` → `Triggered=false`
- **Quality gate**: `go test ./internal/domain/agent/... -run TestDelegateEvaluator`

---

### FC-R.4

- **Objetivo**: preflight DelegateEvaluator en dsl_runner.go
- **Scope**: `dsl_runner.go`. Después de parsear Carta: construir evalCtx desde input del agente → llamar `EvaluateDelegate` → si `Triggered`: `HandoffService.InitiateHandoff` + update `AgentRun.status="delegated"` + return.
- **Constraint**: si `carta.Delegates` vacío → skip. `HandoffService` ya existe — no crear uno nuevo.
- **Acceptance criteria**: Escenario E: `AgentRun.status=="delegated"`, DSLRuntime nunca llamado
- **Quality gate**: `go test ./internal/domain/agent/... -run TestDSLRunnerDelegate`

---

### FC-R.5

- **Objetivo**: preflight GroundsValidator en dsl_runner.go
- **Scope**: `dsl_runner.go`. Después del delegate check: si `rc.GroundsValidator != nil && carta.Grounds != nil` → `Validate` → si `!Met`: `HandoffService.InitiateHandoff` + `AgentRun.status="abstained"` + return.
- **Constraint**: orden crítico: Delegate → Grounds → DSL. `rc.GroundsValidator==nil` → skip silencioso.
- **Acceptance criteria**: Escenario D: `AgentRun.status=="abstained"`, DSLRuntime nunca llamado; `GroundsValidator==nil` → flujo normal
- **Quality gate**: `go test ./internal/domain/agent/... -run TestDSLRunnerGrounds`

---

### FC-R.6

- **Objetivo**: bridge función budget→Limits
- **Scope**: `carta_policy_bridge.go`. `CartaBudgetToLimits(budget *CartaBudget) map[string]any`. Keys exactas: `"daily_tokens"`, `"daily_cost_usd"`, `"executions_per_day"`. Solo incluir campos con valor > 0.
- **Constraint**: keys deben coincidir con las que `orchestrator.go` lee de `Definition.Limits`.
- **Acceptance criteria**: `{DailyTokens: 50000, DailyCostUSD: 5.0}` → `{"daily_tokens": 50000, "daily_cost_usd": 5.0}`; campo 0 → ausente del map
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaBudgetToLimits`

---

### FC-R.7

- **Objetivo**: bridge función invariants→policy rules
- **Scope**: añadir a `carta_policy_bridge.go`. `CartaInvariantsAsPolicyRules(invariants []CartaInvariant) []map[string]any`. `never` → `{effect: "deny", priority: 1000}`. `always` → `{effect: "allow", priority: 1000}`.
- **Acceptance criteria**: 2 invariants `never` → 2 reglas deny; slice vacío → slice vacío
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaInvariantsAsPolicyRules`

---

### FC-R.8

- **Objetivo**: sync Carta budget → AgentDefinition.Limits en Activate()
- **Scope**: `workflow/service.go`. En `Activate()`: si `isCartaSource` → `ParseCarta` → `CartaBudgetToLimits` → merge en `AgentDefinition.Limits` (UPDATE). Solo si `workflow.AgentDefinitionID != nil` y `limits != nil`.
- **Constraint**: merge (no reemplazar): valores Carta sobreescriben, pero no borran keys que Carta no declara.
- **Acceptance criteria**: `BUDGET daily_tokens: 50000` + Activate → `AgentDefinition.Limits["daily_tokens"] == 50000`; workflow sin BUDGET → Limits sin cambio
- **Quality gate**: `go test ./internal/domain/workflow/... -run TestWorkflowActivateBudgetSync`

---

### FC-R.9

- **Objetivo**: inject Carta invariants → active policy version en Activate()
- **Scope**: añadir a `WorkflowService.Activate()`. `CartaInvariantsAsPolicyRules` → rules → cargar PolicySet activo → merge en `policy_version.policy_json` (no duplicar por `action`).
- **Constraint**: si `PolicySetID == nil` → skip. No duplicar reglas en re-activaciones.
- **Acceptance criteria**: `INVARIANT never: "send_pii"` + Activate → policy version contiene `{action: "send_pii", effect: "deny"}`; re-activar → no duplica
- **Quality gate**: `go test ./internal/domain/workflow/... -run TestWorkflowActivateInvariantSync`

---

### FC-R.10

- **Objetivo**: tests unitarios de GroundsValidator
- **Scope**: `grounds_validator_test.go`. Mock de `EvidencePackService`. 5 tests: grounds=nil → Met=true sin llamar mock; 1 fuente need 2 → Met=false; confidence insuf. → Met=false; staleness excedida → Met=false; todo OK → Met=true.
- **Quality gate**: `go test ./internal/domain/agent/... -run TestGroundsValidator -v`

---

### FC-R.11

- **Objetivo**: tests unitarios de DelegateEvaluator
- **Scope**: `delegate_evaluator_test.go`. 4 tests: condición true → Triggered=true; condición false → Triggered=false; When="" → Triggered=false; múltiples delegates segundo trigerea → Triggered=true con reason del segundo.
- **Quality gate**: `go test ./internal/domain/agent/... -run TestDelegateEvaluator -v`

---

### FC-R.12

- **Objetivo**: test de integración end-to-end
- **Scope**: `integration_test.go` (o añadir a existente). Escenario A (success), Escenario D (abstained + DSLRuntime no llamado), Escenario E (delegated + 0 tokens).
- **Quality gate**: `go test ./internal/domain/agent/... -run TestCartaIntegration -v`

---

## Acceptance criteria globales

```bash
go test ./internal/domain/agent/... -v       # 0 failures
go test ./internal/domain/workflow/... -v    # 0 failures
go build ./...                               # 0 errors
```

- Backward compat: workflow con `spec_source` libre format (empieza con `CONTEXT`) → ruta existente sin cambios
- Escenario A: `AgentRun.status == "success"`
- Escenario D: `AgentRun.status == "abstained"`, `AbstentionReason != ""`
- Escenario E: `AgentRun.status == "delegated"`, 0 tokens de retrieval
