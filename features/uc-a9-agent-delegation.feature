@UC-A9 @stack-go
Feature: Agent Delegation
  As a platform operator
  I want workflow execution to delegate safely to another agent
  So that cross-agent coordination preserves traceability and control

  @happy @FR-202 @FR-070 @TST-059 @behavior-delegate_workflow
  Scenario: Delegate workflow execution to another agent
    Given a workflow dispatch step targets another agent
    When the runtime delegates the workflow execution
    Then the delegated run is accepted with trace metadata
    And the delegation decision is recorded in the audit trail
