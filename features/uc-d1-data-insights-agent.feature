@UC-D1 @stack-go
Feature: Data Insights Agent
  As an operator
  I want analytical answers to stay grounded in available data
  So that unsupported claims are rejected safely

  @happy @FR-090 @TST-031
  Scenario: Answer an analytical query with grounded data
    Given a grounded analytical dataset is available
    When the Data Insights Agent answers an analytical query
    Then the Data Insights Agent returns a grounded analytical answer

  @safe @FR-091 @TST-041
  Scenario: Reject an unsupported analytical conclusion
    Given the available data does not support a requested conclusion
    When the Data Insights Agent evaluates the unsupported conclusion
    Then the Data Insights Agent rejects the unsupported conclusion
