@UC-S1 @stack-go
Feature: Sales Copilot
  As a CRM user
  I want Sales Copilot to return grounded briefs for account and deal contexts
  So that the commercial wedge is validated against the backend contract that now exists

  @account @FR-001 @FR-092 @FR-202 @TST-047
  Scenario: Generate a grounded sales brief for an account
    Given an account record has grounded CRM timeline evidence
    When the workspace user requests a sales brief for the account
    Then the sales brief outcome is completed
    And the sales brief summary reflects the account context
    And the sales brief includes evidence-backed next best actions

  @deal @FR-001 @FR-092 @FR-202 @TST-048
  Scenario: Generate a grounded sales brief for a deal
    Given a deal record has grounded stage, owner, and activity evidence
    When the workspace user requests a sales brief for the deal
    Then the sales brief outcome is completed
    And the sales brief summary reflects the deal context
    And the sales brief includes evidence-backed next best actions

  @abstention @FR-092 @FR-202 @TST-049
  Scenario: Abstain when grounded evidence is insufficient for a sales brief
    Given a CRM record lacks enough grounded evidence for a sales brief
    When the workspace user requests a sales brief for the record
    Then the sales brief outcome is abstained
    And the sales brief explains that more evidence is required

  @contract @FR-092 @FR-202 @TST-050
  Scenario: Expose evidence pack and next best actions in the sales brief response
    Given a completed sales brief is requested for a CRM record
    Then the response exposes the evidence pack contract
    And the response exposes the proposed next best actions
