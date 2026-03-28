@UC-S1 @stack-mobile
Feature: Sales Copilot

  @happy @FR-TEST1 @TST-TEST1
  Scenario: Launch Sales Copilot from account detail with grounded context
    Given a grounded account context
    When the user opens Sales Copilot
    Then the user sees a grounded recommendation
