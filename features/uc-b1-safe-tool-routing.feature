@UC-C1 @stack-go
Feature: Safe Tool Routing
  As a platform operator
  I want tool calls to be restricted to allowlisted tools with validated parameters
  So that agents can only execute safe, auditable actions

  @happy @FR-211
  Scenario: Execute an allowlisted tool with valid parameters
    Given an agent has a registered allowlisted tool
    When the runtime validates a tool request with allowed parameters
    Then the tool execution is permitted
    And the tool decision is recorded in the audit trail

  @guardrail @FR-211
  Scenario: Deny a tool request with dangerous parameters
    Given an agent attempts a tool call with disallowed parameters
    When the runtime validates the tool request
    Then the tool execution is denied
    And the denial reason is recorded in the audit trail
