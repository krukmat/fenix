@UC-S2 @stack-go
Feature: Prospecting Agent
  As a sales operator
  I want prospect research and outreach drafting to stay grounded
  So that follow-up actions remain useful and safe

  @happy @FR-092 @TST-036
  Scenario: Research prospect context with grounded evidence
    Given a prospect record has grounded evidence in the knowledge base
    When the Prospecting Agent researches the prospect context
    Then the Prospecting Agent returns grounded prospect insights

  @happy @FR-202 @TST-045
  Scenario: Generate an outreach draft with approved tools
    Given a prospect research result is available
    And the required drafting tool is registered and allowed
    When the Prospecting Agent drafts an outreach message
    Then the Prospecting Agent returns an outreach draft
