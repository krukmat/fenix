@UC-S3 @stack-go
Feature: Deal Risk Agent
  As a sales operator
  I want deal risk assessments to be grounded in evidence
  So that mitigation actions are credible and safe

  @happy @FR-092 @TST-033
  Scenario: Detect deal risk with grounded evidence
    Given a deal has evidence of stalled progress and negative signals
    When the Deal Risk Agent evaluates the deal
    Then the Deal Risk Agent flags the deal as at risk
    And the Deal Risk Agent explains the grounded evidence
