@UC-A2 @stack-go
Feature: Workflow Authoring
  As a platform admin
  I want to save workflow drafts safely
  So that automation logic can evolve before verification and activation

  @happy @FR-240 @TST-052 @behavior-define_workflow
  Scenario: Create a workflow draft with DSL source
    Given an admin writes a workflow definition in DSL
    When the admin saves the workflow as a new draft
    Then the workflow is stored as version 1 in draft status
    And the admin can continue editing the draft
