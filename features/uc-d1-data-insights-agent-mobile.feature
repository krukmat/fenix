@UC-D1 @stack-mobile
Feature: Data Insights Agent Mobile
  As a mobile operator
  I want to submit analytical questions from the Activity area
  So that grounded answers and safe rejections remain traceable in the same workflow

  @happy @FR-090 @TST-031
  Scenario: Run an Insights query from the mobile form
    Given an authenticated workspace user opens the Insights screen from Activity
    And the user enters a non-empty analytical query
    When the user runs the Insights Agent
    Then the mobile app routes to the created activity run detail

  @guardrail @FR-090 @TST-031
  Scenario: Keep the Insights trigger disabled when the query is empty
    Given an authenticated workspace user opens the Insights screen from Activity
    When the user has not entered a query
    Then the Run Insights trigger is disabled

  @pending @FR-090 @TST-031
  Scenario: Prevent duplicate Insights submissions while a run is pending
    Given an authenticated workspace user opens the Insights screen from Activity
    And the user enters a non-empty analytical query
    And the Insights trigger request is pending
    When the screen is displayed
    Then the Run Insights trigger is disabled
    And the trigger shows a running state

  @safe @FR-091 @TST-041
  Scenario: Route to activity detail when the Insights Agent safely rejects a conclusion
    Given an authenticated workspace user opens the Insights screen from Activity
    And the submitted query leads to a safe rejection outcome
    When the user runs the Insights Agent
    Then the mobile app routes to the created activity run detail

  @error @FR-091 @TST-041
  Scenario: Keep the Insights screen stable when the trigger request fails
    Given an authenticated workspace user opens the Insights screen from Activity
    And the user enters a non-empty analytical query
    And the Insights trigger request fails
    When the user runs the Insights Agent
    Then the mobile app stays on the Insights screen
    And no activity run detail is opened
