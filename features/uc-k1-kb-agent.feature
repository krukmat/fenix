@UC-K1 @stack-go
Feature: KB Agent
  As a support knowledge manager
  I want grounded support outcomes turned into knowledge drafts
  So that reusable knowledge stays evidence-backed

  @happy @FR-092 @TST-035
  Scenario: Generate a knowledge base draft from grounded evidence
    Given a resolved support outcome has grounded evidence attached
    When the KB Agent generates a knowledge article draft
    Then the KB Agent produces a grounded knowledge draft
