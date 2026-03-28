@UC-A3 @stack-go
Feature: Workflow Verification and Activation
  As a platform admin
  I want workflow verification to guard activation
  So that only validated workflow versions progress safely

  @happy @FR-240 @FR-070 @TST-053 @behavior-verify_workflow
  Scenario: Verify a draft workflow before activation
    Given a workflow draft has DSL source and a behavior spec
    When the admin requests workflow verification
    Then the workflow passes verification and moves to testing status
    And the verification result is recorded for audit review
