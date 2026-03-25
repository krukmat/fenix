# Carta — Especificación de la Gramática de Comportamiento

> Status: draft
> Parte del set canónico: `docs/carta-implementation-plan.md`
> Sources of truth: `docs/architecture.md`, `docs/agent-spec-design.md`

---

## Contexto

El sistema tiene un DSL imperativo (`dsl_source`) que define *qué pasos ejecutar* en un workflow, y un campo `spec_source` (TEXT nullable en `Workflow`) para describirlo. El `SpecParser` actual (`internal/domain/agent/spec_parser.go`, 101 líneas) parsea bloques CONTEXT/ACTORS/BEHAVIOR/CONSTRAINTS como texto libre — no es machine-readable. El Judge no puede verificar restricciones de evidencia, permisos de herramientas ni condiciones de delegación porque no hay estructuras tipadas que examinar.

**Carta** es la evolución estructurada y machine-readable de `spec_source`.

```
DSL (dsl_source)   → imperativo: "qué pasos ejecutar"
Carta (spec_source)→ declarativo: "bajo qué evidencia, con qué autorizaciones, con qué límites"
```

El Judge ya está diseñado para verificar consistencia entre ambas capas. Carta le da al Judge algo concreto que verificar en lugar de texto libre.

---

## Gramática EBNF

La gramática es indentation-sensitive, usando el mismo modelo INDENT/DEDENT que el DSL existente (Python-like). El `CartaLexer` comparte la lógica de `emitIndentationTokens` del `Lexer` existente.

```ebnf
carta_program   ::= "CARTA" identifier NEWLINE
                    INDENT carta_body DEDENT

carta_body      ::= agent_block? budget_block? skill_block*

(* ── AGENT: bloque de comportamiento por agente ── *)
agent_block     ::= "AGENT" identifier NEWLINE
                    INDENT agent_directives DEDENT

agent_directives::= grounds_block permit_stmt* delegate_stmt? invariant_block?

(* ── GROUNDS: barrera de entrada de evidencia ─────────────────────────
   Sin este bloque, la skill se intenta sin restricción de evidencia.
   Con él, si los requisitos no se cumplen → abstain antes del DSL.      *)
grounds_block   ::= "GROUNDS" NEWLINE
                    INDENT grounds_field+ DEDENT

grounds_field   ::= "min_sources"    ":" integer NEWLINE
                  | "min_confidence" ":" confidence_level NEWLINE
                  | "max_staleness"  ":" integer duration_unit NEWLINE
                  | "types"          ":" "[" string {"," string} "]" NEWLINE

confidence_level::= "low" | "medium" | "high"
duration_unit   ::= "days" | "hours" | "minutes"

(* ── PERMIT: herramientas autorizadas ────────────────────────────────
   Si el DSL usa una herramienta sin PERMIT → Judge violation (Check 10). *)
permit_stmt     ::= "PERMIT" identifier NEWLINE
                    INDENT permit_clause+ DEDENT
                  | "PERMIT" identifier NEWLINE

permit_clause   ::= "when"     ":" condition NEWLINE
                  | "rate"     ":" integer "/" rate_unit NEWLINE
                  | "approval" ":" approval_mode NEWLINE

rate_unit       ::= "min" | "hour" | "day"
approval_mode   ::= "none" | "required"

(* ── DELEGATE TO HUMAN: escalado proactivo ───────────────────────────
   Evaluado ANTES del DSL. Si trigerea → return delegated, 0 tokens.    *)
delegate_stmt   ::= "DELEGATE" "TO" "HUMAN" NEWLINE
                    INDENT delegate_clause+ DEDENT

delegate_clause ::= "when"    ":" condition NEWLINE
                  | "reason"  ":" string NEWLINE
                  | "package" ":" "[" identifier {"," identifier} "]" NEWLINE

(* ── INVARIANT: restricciones absolutas ──────────────────────────────
   Se compilan a policy rules en WorkflowService.Activate().             *)
invariant_block ::= "INVARIANT" NEWLINE
                    INDENT invariant_field+ DEDENT

invariant_field ::= ("never" | "always") ":" string NEWLINE

(* ── BUDGET: cuotas → se sincronizan a AgentDefinition.Limits ─────── *)
budget_block    ::= "BUDGET" NEWLINE
                    INDENT budget_field+ DEDENT

budget_field    ::= "daily_tokens"       ":" integer NEWLINE
                  | "daily_cost_usd"     ":" number NEWLINE
                  | "executions_per_day" ":" integer NEWLINE
                  | "on_exceed"          ":" exceed_action NEWLINE

exceed_action   ::= "pause" | "degrade" | "abort"

(* ── SHARED ─────────────────────────────────────────────────────────── *)
condition       ::= dotted_ident comp_op literal
                  | dotted_ident "IN" "[" literal {"," literal} "]"
                  | "NOT" condition

comp_op         ::= "==" | "!=" | ">=" | "<=" | ">" | "<"
dotted_ident    ::= identifier {"." identifier}
identifier      ::= (letter | "_") { letter | digit | "_" }
string          ::= '"' { any_char } '"'
integer         ::= digit { digit }
number          ::= ["-"] integer ["." integer]
```

### Keywords propios de Carta
`CARTA`, `AGENT`, `GROUNDS`, `PERMIT`, `DELEGATE`, `INVARIANT`, `BUDGET`, `SKILL`, `HUMAN`, `NEVER`, `ALWAYS`

`TO` y `WITH` ya existen en `token.go` — no redefinir. El CartaLexer tiene su propia `cartaKeywords` map; no modifica `dslKeywords`.

---

## 5 Escenarios de Interacción

### Escenario A — Happy Path

**Carta (spec_source):**
```
CARTA resolve_support_case

BUDGET
  daily_tokens: 50000
  daily_cost_usd: 5.00
  executions_per_day: 100
  on_exceed: pause

AGENT search_knowledge
  GROUNDS
    min_sources: 2
    min_confidence: medium
    max_staleness: 30 days
    types: ["case", "kb_article", "email"]
  PERMIT update_case
    when: case.status != "resolved"
    approval: none
  PERMIT send_reply
    rate: 10 / hour
    approval: none
  DELEGATE TO HUMAN
    when: case.tier == "enterprise"
    reason: "Enterprise cases require senior review"
    package: [evidence_ids, case_summary]
  INVARIANT
    never: "send_pii_to_external_without_redaction"
```

**DSL (dsl_source):**
```
WORKFLOW resolve_support_case
ON case.created
IF case.priority IN ["high", "urgent"]:
  AGENT search_knowledge WITH case
SET case.status = "resolved"
NOTIFY salesperson WITH "case resolved"
```

**Judge (`POST /api/v1/workflows/{id}/verify`):**
```json
{ "passed": true, "violations": [], "warnings": [] }
```

**Runtime** (case.tier=standard, 4 fuentes encontradas):
1. `DelegateEvaluator`: `case.tier == "enterprise"` → false. No delegación.
2. `GroundsValidator`: 4 fuentes, confidence=high ≥ medium. OK.
3. `DSLRuntime.ExecuteProgram()` — `update_case` PERMIT declarado, OK. `send_reply` rate 1/10. OK.
4. `AgentRun.status = "success"`. AuditEvent logged.

---

### Escenario B — Judge Violation: tool sin PERMIT declarado

**Carta**: tiene `PERMIT update_case` pero NO tiene `PERMIT send_reply`.

**DSL**: tiene `NOTIFY salesperson` — `VerbMapper.ToolNameForStatement(NotifyStatement)` → `"send_reply"`.

**Judge:**
```json
{
  "passed": false,
  "violations": [{
    "checkId": 10,
    "code": "tool_not_permitted",
    "type": "carta_permit_missing",
    "description": "DSL NOTIFY maps to tool 'send_reply' — no PERMIT block declared in Carta",
    "location": "NOTIFY salesperson (line 4)"
  }]
}
```

**Runtime:** Workflow no puede activarse. `WorkflowService.Activate` rechaza `testing→active` mientras existan violations.

---

### Escenario C — Judge Violation: behavior del objetivo sin cobertura

**Contexto**: `AgentDefinition.Objective = {"goals": ["resolve_case", "escalate_unresolved"]}`.

Carta tiene `PERMIT update_case` pero nada cubre `escalate_unresolved` (ni PERMIT ni DELEGATE TO HUMAN). DSL tampoco lo cubre.

**Judge:**
```json
{
  "passed": false,
  "violations": [
    {
      "checkId": 5,
      "code": "behavior_no_coverage",
      "description": "BEHAVIOR escalate_unresolved has no DSL coverage",
      "location": "BEHAVIOR escalate_unresolved"
    },
    {
      "checkId": 11,
      "code": "behavior_no_permit_or_delegate",
      "description": "BEHAVIOR escalate_unresolved not covered by PERMIT or DELEGATE TO HUMAN in Carta",
      "location": "CARTA resolve_support_case"
    }
  ]
}
```

Las dos violations son complementarias: Check 5 es del Judge clásico, Check 11 es del CartaCheck.

---

### Escenario D — Runtime Abstain: GROUNDS no satisfechos

Judge pasó. Workflow activo. `case.created` llega.

`GroundsValidator.Validate()` llama `EvidencePackService.BuildEvidencePack`:
- Resultado: 1 fuente, confidence=low
- Carta exige: min_sources=2, min_confidence=medium
- `GroundsResult{Met: false, Reason: "insufficient evidence: 1 source (need 2), confidence=low (need medium)"}`

DSL **nunca ejecuta**. `HandoffService.InitiateHandoff` llamado automáticamente.

**AgentRun:**
```json
{
  "status": "abstained",
  "abstention_reason": "insufficient evidence: 1 source (need 2), confidence=low (need medium)"
}
```

AuditEvent `agent.abstained` con evidence pack metadata adjunto.

---

### Escenario E — Runtime Delegate: condición de negocio proactiva

`case.created` llega con `case.tier = "enterprise"`.

`DelegateEvaluator.EvaluateDelegate()` corre **antes** del GroundsValidator y antes del DSL:
- `case.tier == "enterprise"` evaluado contra evalCtx → true

`HandoffService.InitiateHandoff` llamado con reason+package declarados en Carta. 0 tokens gastados (no hay retrieval).

**AgentRun:**
```json
{
  "status": "delegated",
  "abstention_reason": "Enterprise cases require senior review"
}
```

**Diferencia clave vs Escenario D:**
- D = abstain **reactivo** (retrieval hecho, evidencia insuficiente)
- E = delegate **proactivo** (regla de negocio, sin retrieval)

---

## Mapa de integración

| Bloque Carta | Archivo existente | Cambio | Archivo nuevo |
|---|---|---|---|
| Lexer | `lexer.go` (reusar `emitIndentationTokens`) | sin cambio | `carta_token.go`, `carta_lexer.go` |
| Structs | `spec_parser.go` (patrón a seguir) | sin cambio | `carta_ast.go` |
| Parser | — | — | `carta_parser.go` |
| `GROUNDS` runtime | `runner.go` → +campo | +`GroundsValidator *GroundsValidator` | `grounds_validator.go` |
| `PERMIT` estático Check 10 | `judge_consistency.go` → +fn paralela | `RunCartaPermitChecks` | `judge_carta.go` |
| `DELEGATE TO HUMAN` | `dsl_runner.go` → preflight | DelegateEvaluator call antes del DSL | `delegate_evaluator.go` |
| `INVARIANT` | `policy/evaluator.go` + `workflow/service.go` | inject rules en `Activate()` | `carta_policy_bridge.go` |
| `BUDGET` | `agent/orchestrator.go` `Definition.Limits` | sync en `Activate()` | `carta_policy_bridge.go` |
| Judge dispatch | `judge.go` | `isCartaSource()` branch (~10 líneas) | — |
| Check 11 coverage | `judge_consistency.go` | nueva fn paralela | `judge_carta.go` |
| Check 12 grounds warning | — | — | `judge_carta.go` |
| VerbMapper | `verb_mapper.go` | +`ToolNameForStatement()` | — |

### Único campo nuevo en RunContext
```go
// internal/domain/agent/runner.go
GroundsValidator *carta.GroundsValidator  // nil = checks skipped (backward compat)
```

### Preflight order en dsl_runner.go (único cambio de flujo)
```
loadWorkflow()
parseCartaIfPresent()           ← si isCartaSource()
→ DelegateEvaluator.Check()     ← si triggers: HandoffService + return delegated
→ GroundsValidator.Validate()   ← si falla:    HandoffService + return abstained
→ DSLRuntime.ExecuteProgram()   ← sin cambio
```

---

## Scope P1 (deferred)

- Pre-compilación de condition expressions en `CartaPermit.WhenExpr` / `CartaDelegate.WhenExpr` (hoy re-parsean en runtime)
- SKILL blocks en runtime (hoy solo parsing estático)
- INVARIANT como predicados ABAC evaluables (hoy son string → policy rules)
- UI visual de edición de bloques Carta (Agent Studio, UC-A2)
- `on_exceed: degrade` (requiere model-switch en LLM adapter)
- `version:` header para migración de specs
