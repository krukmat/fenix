@UC-S3 @stack-mobile @deferred
Feature: Deal Risk Agent Mobile Placeholder
  As a mobile sales operator
  I want the future Deal Risk entry point to be visible
  So that the product surface stays stable while the backend runner remains deferred

  @placeholder @FR-092 @TST-033
  Scenario: Show the Deal Risk placeholder on deal detail
    Given an authenticated workspace user opens a deal detail screen
    When the deal detail screen is displayed
    Then the mobile app shows the Deal Risk trigger placeholder
    And the placeholder is disabled
