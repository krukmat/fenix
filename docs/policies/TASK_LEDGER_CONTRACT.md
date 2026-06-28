---
doc_type: policy
id: TASK_LEDGER_CONTRACT
title: "Task Ledger Behavioral Coverage Contract"
status: active
created: 2026-06-28
---

# Task Ledger Behavioral Coverage Contract

This policy defines the optional `unit-v1` behavioral coverage contract that task ledger files can adopt to certify that their implementation is covered by unit tests.

## Opt-in sentinel

A task file opts into enforcement by including the following line anywhere in its body (outside frontmatter):

```
Behavioral coverage contract: unit-v1
```

Task files without this sentinel are silently skipped by `scripts/check-task-unit-coverage.sh`.

## Required sections (per completed development `##` section)

A `##` section is checked if and only if it satisfies both:

- `Status: [x] Done` (or `DONE`)
- `Type: development`

Such sections must include:

### 1. Happy paths considered

List at least one stable case ID of the form `HP-N`:

```
### Happy paths considered
- HP-1: description of the happy path
- HP-2: another happy path
```

### 2. Edge cases considered

List at least one stable case ID of the form `EC-N`:

```
### Edge cases considered
- EC-1: description of the edge case
```

### 3. Unit coverage certification table

A markdown table with at least one row per HP and EC case ID. The second-to-last column must contain at least one backtick-wrapped test reference; the last column must be `passed`.

```markdown
### Unit coverage certification

| Case | Description | File | Test | Evidence | Result |
|---|---|---|---|---|---|
| HP-1 | happy path | internal/domain/foo_test.go | TestFoo_HappyPath | `internal/domain/foo_test.go::TestFoo_HappyPath` | passed |
| EC-1 | edge case  | internal/domain/foo_test.go | TestFoo_EdgeCase  | `internal/domain/foo_test.go::TestFoo_EdgeCase`  | passed |
```

### 4. Owner final verification block

```
### Owner final verification
- Owner: <name>
- Date: YYYY-MM-DD
- Statement: I verified every happy path and edge case has unit test evidence
- Commands run: <exact commands>
```

## Test reference formats

| Stack | Format | Validation |
|---|---|---|
| Go | `` `path/to/file_test.go::TestFuncName` `` | File exists; `func TestFuncName(` present |
| React Native / TypeScript | `` `path/to/file.test.ts::describe > test name` `` | File exists; `it('test name'` or `test('test name'` present |

## Enforcement

`scripts/check-task-unit-coverage.sh` is run as part of `make qa-task-coverage` and composed into `make qa-docs`. It exits 1 with a violation list if any opted-in completed development section fails the contract.

## RRI band rules

| RRI band | Additional requirement |
|---|---|
| Low (0–25) | HP/EC ids + certification table + owner verification |
| Medium (26–50) | same (Reflection log recommended but not enforced here) |
| High (51+) | same (full approval packet via HITL policy) |

Gemma reviewer evidence (from DubBridge Phase E) is deferred and not enforced by this validator.
