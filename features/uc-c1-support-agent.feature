@UC-C1 @stack-go
Feature: Support Agent
  As a support operator
  I want the Support Agent to expose the public outcomes we now ship
  So that customer cases remain grounded, safe, and auditable

  @happy @FR-092 @FR-061 @FR-070 @TST-038
  Scenario: Resolve a case with outcome completed
    Given a support case has grounded evidence and an allowed resolution path
    When the Support Agent is triggered for the case
    Then the support run outcome is completed
    And the support run is recorded in audit and usage

  @abstention @FR-092 @FR-210 @TST-039
  Scenario: Abstain with outcome abstained
    Given a support case only has medium-confidence evidence
    When the Support Agent is triggered for an abstention path
    Then the support run outcome is abstained
    And the run explains the lack of decisive evidence

  @handoff @FR-092 @FR-070 @TST-037
  Scenario: Hand off with outcome handed off
    Given a high-priority support case lacks grounding for autonomous resolution
    When the Support Agent triggers a human handoff
    Then the support run outcome is handed off
    And the handoff package preserves case context and evidence

  @approval @FR-061 @FR-070 @TST-044
  Scenario: Require approval with outcome awaiting approval
    Given a high-priority support case has a sensitive but grounded remediation
    When the Support Agent proposes the sensitive action
    Then the support run outcome is awaiting approval
    And the approval request is available to the operator
