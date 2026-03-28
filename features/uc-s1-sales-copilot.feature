@UC-S1 @stack-mobile
Feature: Sales Copilot
  As a CRM user
  I want to launch Sales Copilot from account and deal contexts
  So that I can get grounded guidance without leaving the flow

  @happy @FR-001 @FR-202 @FR-092 @TST-011
  Scenario: Launch Sales Copilot from account detail with grounded context
    Given an authenticated workspace user opens an account detail screen
    And the account has CRM timeline data and linked evidence
    When the user opens Sales Copilot from the account detail
    Then Sales Copilot shows the current account context
    And the response includes evidence-backed guidance for the next step

  @happy @FR-001 @FR-202 @FR-092 @TST-047
  Scenario: Launch Sales Copilot from deal detail with grounded context
    Given an authenticated workspace user opens a deal detail screen
    And the deal has stage, owner, and evidence-backed activity history
    When the user opens Sales Copilot from the deal detail
    Then Sales Copilot shows the current deal context
    And the response includes evidence-backed guidance for the next step

  @fallback @FR-202 @FR-092 @TST-046
  Scenario: Fall back safely when context grounding is insufficient
    Given an authenticated workspace user opens Sales Copilot from a CRM record
    And the record does not have enough grounded evidence for a recommendation
    When the user asks for the next best action
    Then Sales Copilot does not fabricate a recommendation
    And Sales Copilot explains that more evidence is required
