@UC-A7 @stack-go
Feature: Human Override and Approval
  As a platform operator
  I want sensitive workflow actions to require human approval
  So that risky automation stays under explicit control

  @happy @FR-061 @FR-070 @TST-057 @behavior-human_override
  Scenario: Require approval before a sensitive workflow action
    Given a workflow action is classified as sensitive
    When the runtime requests human approval for the action
    Then an approval request is created and left pending
    And the approval requirement is recorded in the audit trail
