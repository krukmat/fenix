@UC-G1 @stack-go
Feature: Governance
  As a governance operator
  I want to inspect agent activity through the governance surfaces we actually ship
  So that the platform remains auditable and policy compliant

  @audit @FR-060 @FR-070 @TST-026
  Scenario: Inspect an agent run and its audit trace
    Given a completed governed support run exists
    When a governance operator inspects the run
    Then the operator can see the run outcome and runtime trace
    And the audit trail shows actor, action, and timestamp

  @approval @FR-060 @FR-071 @FR-070 @TST-042
  Scenario: Inspect and approve a pending approval request
    Given a governed support run is awaiting approval
    When the governance operator lists pending approvals
    And the governance operator can approve the request
    Then the approval decision is accepted

  @rejection @FR-060 @FR-071 @FR-070 @TST-025
  Scenario: Record a governance rejection decision in the audit trail
    Given a governance rejection decision has been applied to a pending approval
    When the governance operator inspects the audit trail for the rejection
    Then the rejection is recorded in the audit trail

  @usage @FR-060 @FR-070 @FR-071 @TST-043
  Scenario: Inspect usage and quota state for a governed run
    Given a governed run has emitted usage and a quota state exists
    When the governance operator inspects usage and quota state
    Then the operator can see usage events for the run
    And the operator can see the current quota state

  @deferred @FR-060 @FR-071 @FR-070 @TST-042
  Scenario: Replay an agent run when replay is allowed
    Given an agent run is eligible for replay under policy
    When a governance operator requests a replay
    Then the replay is accepted
    And the replay decision is recorded in the audit trail

  @deferred @FR-060 @FR-071 @FR-070 @TST-025
  Scenario: Reject replay or rollback when policy denies it
    Given an agent run is not eligible for replay or rollback under policy
    When a governance operator requests a replay or rollback
    Then the request is rejected
    And the denial reason is recorded in the audit trail
