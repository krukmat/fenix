---
doc_type: summary
title: Workflow Activation and Conformance Story
status: complete
created: 2026-05-02
---

# Workflow Activation and Conformance Story

## Why This Story Exists

A workflow in FenixCRM does not execute just because somebody authored it.

Two separate domain concepts control what happens next:

- `workflow.Status` in `internal/domain/workflow/repository.go`
- `agent.ConformanceProfile` in `internal/domain/agent/conformance.go`

They are related, but they are not the same thing.

## The Two Models

| Concept | Source | Values | Meaning |
|---|---|---|---|
| Workflow FSM | `internal/domain/workflow/repository.go` | `draft`, `testing`, `active`, `archived` | Lifecycle state of the stored workflow version |
| Conformance profile | `internal/domain/agent/conformance.go` | `safe`, `extended`, `invalid` | Tooling/validation classification of the workflow definition |

The important rule is simple:

> `invalid` is not a workflow status.  
> It is a conformance result that explains why a workflow version should not be promoted.

Operationally, `active` is the only state that is executable by the runtime. Drafts, testing versions, and archived versions can exist in storage without being runnable.

## Case Study: `resolve_support_case`

Use the UC-A3 flow as the anchor:

- a platform admin authors a support workflow
- verification is requested
- the system decides whether the version is ready for the activation path

The workflow begins in `draft`.

### Step 1: Authored, but not executable

The admin saves a workflow version:

- name: `resolve_support_case`
- status: `draft`
- inputs: `dsl_source` + `spec_source`

At this point the workflow exists, but existence is not permission to run. It is still an editable draft.

### Step 2: Verification produces a conformance result

Verification evaluates the authored definition and returns machine-readable conformance data:

- `safe`
- `extended`
- `invalid`

This is where the system answers a different question from the FSM.

- The FSM asks: "Where is this version in its lifecycle?"
- Conformance asks: "What kind of workflow definition is this, and is it compatible with the governed tooling contract?"

### Step 3: The activation path diverges by conformance profile

#### Path A: `safe`

If the workflow verifies as `safe`, it is a clean candidate for the normal path:

`draft -> testing -> active`

This is the happy-path story for public demos:

- the workflow was authored
- it passed verification
- it moved into `testing`
- it was promoted to `active`
- the runtime may now execute it

#### Path B: `extended`

If the workflow verifies as `extended`, the system is saying:

- the workflow is still structurally understandable
- but its semantic graph contains nodes outside the stable tooling contract

In current domain terms, `extended` is typically produced by the detail code:

- `unsupported_semantic_node`

This is not the same as a broken workflow. It is a warning that the workflow goes beyond the stable supported slice.

The practical reading for demos and product narrative is:

- `extended` is not a lifecycle state
- `extended` is not a successful "safe automation" story
- it belongs in review/testing until the team decides whether to narrow the workflow or intentionally support the extension

#### Path C: `invalid`

If the workflow verifies as `invalid`, the system returns a blocking conformance result with error details, such as:

- `invalid_dsl`
- `invalid_carta`

Those details can also carry `line` and `column`, which means the failure is explainable, not hand-wavy.

This is the core activation story:

`draft -> verification -> conformance invalid -> no promotion to active`

The workflow can exist as a stored record, but it does not become runnable. The system has a concrete reason for refusal, and that reason is part of the product story.

## What "Blocked Activation" Means In Practice

Blocked activation does not require inventing a fifth workflow state.

The block is expressed by the combination of:

- the workflow remaining outside `active`
- a conformance profile of `invalid`
- machine-readable details explaining why promotion was refused

That is stronger than a vague "something failed" message because it preserves both:

- lifecycle truth from the FSM
- validation truth from conformance

## Feature Slice

This wave can be explained as one narrow governed feature slice:

1. Author a workflow version in `draft`.
2. Request verification for the draft.
3. Produce a conformance result using existing domain types.
4. If conformance is acceptable, move the version into `testing`.
5. Promote to `active` only when the version is on the verified path.
6. Run only `active` workflows.
7. Archive old active versions when replaced.

The key product value is not "we have statuses."
The value is that the platform can explain why a workflow is:

- still an editable draft
- under testing review
- safely active
- or no longer runnable because it was archived

And independently from that, it can explain whether the authored definition is:

- `safe`
- `extended`
- `invalid`

## Demo Notes

For a live or recorded demo, the shortest convincing story is:

1. Show a newly authored workflow in `draft`.
2. Run verification and show the conformance result.
3. First show a passing example: `safe`, then transition to `testing`, then `active`.
4. Immediately contrast it with a failing example: the workflow still exists, but conformance returns `invalid_dsl` or `invalid_carta`.
5. Emphasize that the second workflow is not "missing" and not "deleted" — it is deliberately blocked from activation.
6. Close with the distinction: lifecycle state tells you where the version is; conformance tells you whether the definition deserves promotion.

## Public Positioning

This story supports the governance narrative well because it avoids magical automation claims.

The message is:

> Workflows are not trusted because they exist.  
> They are promoted only when their lifecycle state and their conformance evidence both justify execution.

That is the difference between "workflow storage" and "governed workflow activation."
