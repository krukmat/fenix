@UC-S2 @stack-mobile
Feature: Prospecting Agent Mobile
  As a mobile sales operator
  I want to launch the Prospecting Agent from lead detail
  So that prospect research remains accessible and traceable in the mobile workflow

  @happy @FR-092 @TST-036
  Scenario: Launch the Prospecting Agent from lead detail
    Given an authenticated workspace user opens a lead detail screen
    And the lead is visible in the Sales leads tab
    When the user runs the Prospecting Agent from the lead detail
    Then the mobile app routes to the created activity run detail

  @guardrail @FR-202 @TST-045
  Scenario: Prevent duplicate Prospecting submissions while a run is pending
    Given an authenticated workspace user opens a lead detail screen
    And the Prospecting Agent trigger is already pending
    When the user views the Prospecting trigger
    Then the trigger is disabled
    And the trigger shows a running state

  @error @FR-202 @TST-045
  Scenario: Keep the lead detail stable when the Prospecting trigger fails
    Given an authenticated workspace user opens a lead detail screen
    And the Prospecting trigger request fails
    When the user runs the Prospecting Agent from the lead detail
    Then the mobile app stays on the lead detail screen
    And no activity run detail is opened

