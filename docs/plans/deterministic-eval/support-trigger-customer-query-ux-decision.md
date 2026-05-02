# Support Trigger Customer Query UX Decision

## Status

Accepted for implementation planning.

## Context

The canonical support trigger contract is already decided as:

```json
{
  "case_id": "<case-id>",
  "customer_query": "<operator prompt or customer issue>",
  "language": "es",
  "priority": "low|medium|high"
}
```

The remaining open question is where `customer_query` should come from in a real human-driven demo.

Relevant live surfaces today:

- support case detail already exists at `/support/[id]`;
- support case detail already links to `/support/[id]/copilot`;
- the support Copilot surface already has a visible text input and explicit send action;
- the current `Run Support Agent` button on case detail is a blind one-tap action with no text capture step.

References:

- [support-trigger-contract-decision.md](./support-trigger-contract-decision.md)
- [demo-execution-surfaces.md](./demo-execution-surfaces.md)

## Decision

`customer_query` will come from the support Copilot text input on `/support/[id]/copilot`.

The live support demo should trigger the governed support run from a Copilot-originated prompt, not from hidden derivation and not from the current one-tap button on case detail.

## Chosen Operator Interaction

The intended operator flow is:

1. Open `Support`.
2. Open the target support case.
3. Review visible case context: priority, account, signals, and recent activity.
4. Tap `Open Copilot`.
5. Enter the operator-facing support request in the Copilot text input.
6. Submit that text from the Copilot surface.
7. The app triggers the support run with:
   - `case_id` from the current case route/context
   - `customer_query` from the Copilot input text
   - optional `priority` from current case data
   - optional `language` from default/app context if needed
8. The operator then follows the resulting run through Activity and approvals as part of the demo.

## Why This UX Is Chosen

### 1. It uses an existing explicit human input surface

The support Copilot screen already has:

- case context;
- a text input;
- a send action that a demo operator can explain naturally.

That makes the source of `customer_query` visible and non-magical.

### 2. It avoids fake intelligence in the trigger contract

Deriving `customer_query` from case description or other stored fields would blur the difference between:

- the customer record context;
- the operator's present intent;
- the actual instruction that triggers the run.

For demo reliability, the operator should explicitly provide the text that launches the governed action.

### 3. It preserves support case detail as a context screen

The support case detail page is good for:

- selecting the case;
- showing severity and surrounding context;
- explaining why the case matters.

It is not currently a good source of trigger text because the page has no explicit prompt input.

### 4. It keeps the demo narration coherent

The story becomes easy to explain:

- “Here is the case.”
- “Here is the operator asking Copilot for help on this case.”
- “That exact text becomes the governed support trigger.”

This is stronger than asking the audience to assume the system inferred the support request from metadata.

## Rejected UX Alternatives

### Rejected: derive `customer_query` from case description

Why rejected:

- it hides operator intent;
- it makes the run source ambiguous;
- it weakens the explanation of why this run started now;
- it creates magical behavior that is harder to validate in the demo.

### Rejected: keep the current one-tap `Run Support Agent` button as the live trigger

Why rejected:

- it has no visible text capture step;
- it cannot satisfy the canonical contract by itself;
- it would require hidden defaulting or a second invisible source of `customer_query`.

### Rejected: add a pre-trigger modal on case detail as the primary demo path

Why rejected for the demo path:

- it is workable, but it duplicates a prompt-entry interaction that Copilot already provides;
- it creates a second operator mental model for “ask the system something about this case”;
- the Copilot surface is already more natural for typed support intent.

A modal remains a valid fallback implementation option if Copilot coupling becomes undesirable later, but it is not the chosen demo UX.

## Validation Expectations

The UX should enforce these basic rules before triggering:

- trimmed `customer_query` must not be empty;
- whitespace-only input is invalid;
- the operator must remain on the input surface if validation fails;
- the validation message should make clear that a support request is required.

Suggested minimum validation copy:

- `Support request is required`

## Edge Cases

### Empty input

- Do not trigger the support run.
- Keep the operator on the Copilot screen.
- Show a clear validation message.

### Very short but non-empty input

- Allow it unless product wants a minimum length later.
- Do not block the demo on speculative length rules.

### Existing multi-turn Copilot conversation

- The trigger should use the currently submitted text as `customer_query`.
- It should not silently concatenate the whole prior chat history into the support trigger contract unless that behavior is later specified explicitly.

### Case detail still shows `Run Support Agent`

- Until implementation catches up, that control should not be presented as the canonical live demo path.
- The canonical narrative should instruct the operator to use `Open Copilot` and submit the support request there.

## Implementation Consequence

The next implementation slice should assume:

- support case detail remains the entry/context screen;
- support Copilot becomes the canonical source of `customer_query`;
- the support trigger payload should be assembled from case context plus the submitted Copilot text;
- the one-tap support trigger on case detail should be demoted, replaced, or reworked so it does not compete with the chosen live path.

## Demo Script Impact

The live demo instruction can now be concrete:

1. Open the support case.
2. Tap `Open Copilot`.
3. Type the support request.
4. Submit it.
5. Show how the governed run proceeds to approval, activity, and audit.

That is specific enough for another engineer or presenter to implement and rehearse without inventing undocumented behavior.
