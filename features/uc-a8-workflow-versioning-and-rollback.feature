@UC-A8 @stack-go
Feature: Workflow Versioning and Rollback
  As a platform operator
  I want workflow versions to evolve safely
  So that new drafts and rollback paths remain controlled and traceable

  @happy @FR-240 @FR-070 @TST-058 @behavior-version_workflow
  Scenario: Create a new draft version from an active workflow
    Given an active workflow is eligible for a new version
    When the operator creates a new workflow version
    Then a new draft workflow version is created from the active source
    And the versioning action is recorded in the audit trail
