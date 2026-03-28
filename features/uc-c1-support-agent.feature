@UC-C1 @stack-go
Feature: Support Agent
  As a support operator
  I want the Support Agent to resolve, abstain, and escalate correctly
  So that customer cases remain grounded, safe, and auditable

  @happy @FR-092 @FR-061 @FR-070 @TST-038
  Scenario: Resolve a case with sufficient grounded evidence
    Given a support case has relevant history, evidence, and an allowed resolution action
    When the Support Agent proposes and executes the case resolution
    Then the case response is grounded in the available evidence
    And the case action is recorded in the audit trail

  @abstention @FR-092 @TST-039
  Scenario: Abstain when the evidence is insufficient
    Given a support case lacks sufficient grounded evidence
    When the Support Agent is asked to resolve the case
    Then the Support Agent abstains from taking a definitive action
    And the response explains the missing evidence

  @handoff @FR-092 @FR-070 @TST-037
  Scenario: Hand off to a human with preserved context
    Given a support case needs human review
    And the Support Agent has collected the case context and evidence
    When the Support Agent hands off the case
    Then the human handoff preserves the case context
    And the handoff is recorded in the audit trail

  @approval @FR-061 @FR-070 @TST-044
  Scenario: Require approval before a sensitive case action
    Given a support case requires a sensitive remediation action
    When the Support Agent proposes the sensitive action
    Then the action is blocked pending approval
    And the approval workflow is recorded in the audit trail
