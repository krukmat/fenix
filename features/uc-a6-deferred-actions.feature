@UC-A6 @stack-go
Feature: Deferred Actions
  As a platform operator
  I want workflow actions to pause and resume safely
  So that delayed execution stays controlled and traceable

  @happy @FR-070 @TST-056 @behavior-defer_action
  Scenario: Schedule and resume a deferred workflow action
    Given a workflow run reaches a wait step that must resume later
    When the runtime schedules the deferred action
    Then the deferred job is stored under scheduler control
    And the workflow can resume from the deferred action state

  @error @FR-070 @TST-063 @behavior-defer_action_workflow_archived
  Scenario: Reject a deferred resume when the workflow is no longer active
    Given a scheduled workflow resume targets an archived workflow
    When the resume handler processes the archived workflow job
    Then the deferred resume is rejected
    And the workflow run records the archived workflow error

  @error @FR-070 @TST-064 @behavior-defer_action_resume_failure
  Scenario: Fail a deferred resume when resumed execution errors
    Given a scheduled workflow resume points to a failing step
    When the resume handler processes the failing workflow job
    Then the deferred resume fails safely
    And the workflow run records the resume execution error
