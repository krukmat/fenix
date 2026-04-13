@UC-S3 @stack-mobile
Feature: Deal Risk Agent Mobile
  As a mobile sales operator
  I want to trigger deal risk analysis from the deal detail screen
  So that I can inspect the generated run immediately

  @active @FR-092 @TST-033
  Scenario: Trigger Deal Risk Agent from deal detail and navigate to run
    Given an authenticated workspace user opens a deal with stalled progress
    When the user taps the Deal Risk trigger button
    Then the app navigates to the agent run detail screen
