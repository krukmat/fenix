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
