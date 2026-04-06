@UC-S1 @stack-mobile
Feature: Sales Copilot Mobile Smoke
  As a mobile user
  I want to open Sales Copilot from account and deal detail screens
  So that the mobile app keeps a smoke-level entrypoint check without claiming canonical contract coverage

  @smoke @FR-001 @FR-202 @TST-011
  Scenario: Open Sales Copilot from the account detail screen
    Given an authenticated workspace user opens an account detail screen
    And the account has CRM timeline data and linked evidence
    When the user opens Sales Copilot from the account detail
    Then Sales Copilot shows the current account context

  @smoke @FR-202 @TST-046
  Scenario: Open Sales Copilot from the deal detail screen
    Given an authenticated workspace user opens a deal detail screen
    And the deal has stage, owner, and evidence-backed activity history
    When the user opens Sales Copilot from the deal detail
    Then Sales Copilot shows the current deal context
