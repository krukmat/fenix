@UC-G1 @stack-go
Feature: Governance
  As a governance operator
  I want to inspect, replay, and control agent activity safely
  So that the platform remains auditable and policy compliant

  @audit @FR-060 @FR-070 @TST-026
  Scenario: Inspect an agent run and its audit trace
    Given an agent run has executed in production
    When a governance operator inspects the run
    Then the operator can see the run decisions and audit trail
    And the audit trail shows the actor, action, and timestamp

  @happy @FR-060 @FR-071 @FR-070 @TST-042
  Scenario: Replay an agent run when replay is allowed
    Given an agent run is eligible for replay under policy
    When a governance operator requests a replay
    Then the replay is accepted
    And the replay decision is recorded in the audit trail

  @denial @FR-060 @FR-071 @FR-070 @TST-025
  Scenario: Reject replay or rollback when policy denies it
    Given an agent run is not eligible for replay or rollback under policy
    When a governance operator requests a replay or rollback
    Then the request is rejected
    And the denial reason is recorded in the audit trail
