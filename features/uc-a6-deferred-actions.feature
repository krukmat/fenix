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
