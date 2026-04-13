@UC-K1 @stack-mobile
Feature: KB Agent Mobile
  As a mobile support operator
  I want to trigger KB draft generation from a resolved case
  So that reusable knowledge can be created from grounded support outcomes

  @happy @FR-092 @TST-035
  Scenario: Show the KB trigger on a resolved support case
    Given an authenticated workspace user opens a resolved support case detail
    When the case detail screen is displayed
    Then the mobile app shows the Generate KB Article trigger

  @guardrail @FR-092 @TST-035
  Scenario: Hide the KB trigger on a non-resolved support case
    Given an authenticated workspace user opens a non-resolved support case detail
    When the case detail screen is displayed
    Then the mobile app does not show the Generate KB Article trigger

  @pending @FR-092 @TST-035
  Scenario: Prevent duplicate KB submissions while a run is pending
    Given an authenticated workspace user opens a resolved support case detail
    And the KB trigger request is pending
    When the case detail screen is displayed
    Then the Generate KB Article trigger is disabled
    And the trigger shows a running state

  @error @FR-092 @TST-035
  Scenario: Keep the case detail stable when KB generation fails
    Given an authenticated workspace user opens a resolved support case detail
    And the KB trigger request fails
    When the user runs the KB Agent from the case detail
    Then the mobile app stays on the case detail screen
    And no activity run detail is opened

