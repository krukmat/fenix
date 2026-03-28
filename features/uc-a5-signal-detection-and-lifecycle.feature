@UC-A5 @stack-go
Feature: Signal Detection and Lifecycle
  As a platform operator
  I want signals to be created only from grounded evidence
  So that surfaced opportunities remain actionable and trustworthy

  @happy @FR-091 @FR-092 @TST-055 @behavior-detect_signal
  Scenario: Detect a grounded signal from evidence
    Given a workflow evaluation produces grounded signal evidence
    When the system creates a new signal from the grounded evidence
    Then the signal is stored as an active actionable item
    And the signal remains linked to its evidence sources
