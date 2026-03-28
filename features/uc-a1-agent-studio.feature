@UC-A1 @stack-go
Feature: Agent Studio
  As a platform operator
  I want to create and validate agent configuration changes safely
  So that promotion to production stays controlled and traceable

  @happy @FR-202 @FR-070 @TST-051
  Scenario: Validate an agent studio configuration before promotion
    Given an agent studio draft includes a tool-enabled configuration
    And governance checks are required before promotion
    When the operator validates the draft for promotion
    Then the agent studio draft passes validation
    And the validation outcome is recorded for governance review
