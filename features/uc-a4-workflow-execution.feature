@UC-A4 @stack-go
Feature: Workflow Execution
  As a platform operator
  I want active workflows to execute through the runtime
  So that events trigger controlled, traceable automation

  @happy @FR-202 @FR-070 @TST-054 @behavior-execute_workflow
  Scenario: Execute an active workflow for a matching event
    Given an active workflow matches an incoming event
    And the workflow runtime has registered tools available
    When the workflow runtime executes the matching workflow
    Then the workflow run completes successfully
    And the workflow steps are recorded in the audit trail

  @error @FR-202 @FR-070 @TST-060 @behavior-execute_workflow_condition_false
  Scenario: Skip a conditional branch when the workflow condition is false
    Given an active workflow contains a conditional branch
    When the workflow runtime executes the workflow with a non-matching condition
    Then the conditional step is recorded as skipped
    And the workflow run still completes successfully

  @error @FR-202 @FR-070 @TST-061 @behavior-execute_workflow_tool_failure
  Scenario: Fail workflow execution when a mapped tool call errors
    Given an active workflow maps a statement to a failing tool
    When the workflow runtime executes the matching workflow
    Then the workflow run fails
    And the failing workflow step is recorded in the audit trail

  @approval @FR-202 @FR-070 @TST-062 @behavior-execute_workflow_approval_required
  Scenario: Leave the workflow pending when a nested action requires approval
    Given an active workflow delegates a nested action that requires approval
    When the workflow runtime executes the matching workflow
    Then the workflow run remains pending approval
    And the pending approval is recorded in the runtime trace
