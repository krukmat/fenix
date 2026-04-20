@UC-P2 @stack-mobile
Feature: CRM List Centralized CRUD and Bulk Delete — Mobile screens
  As a sales or support rep
  I want to create, edit, and delete CRM entities from the list screens
  So that the list is the single operational surface and detail screens remain read-only

  @happy @FR-304 @TST-067 @behavior-row_multi_select
  Scenario: Select and deselect rows with always-visible checkboxes
    Given the user is on a CRM entity list screen with at least one row
    When the user taps the checkbox on a row
    Then that row is marked as selected and the selected count increases
    When the user taps the checkbox again
    Then the row is deselected and the selected count decreases

  @happy @FR-304 @TST-068 @behavior-select_all_visible
  Scenario: Select all visible filtered rows
    Given the user is on a CRM entity list screen with multiple rows
    When the user taps "Select all"
    Then all currently visible rows are selected
    And the selected count matches the visible row count

  @happy @FR-304 @TST-069 @behavior-clear_selection
  Scenario: Clear all selected rows
    Given the user has selected one or more rows on a CRM list screen
    When the user taps "Clear"
    Then all rows are deselected and the selected count is zero

  @happy @FR-304 @TST-070 @behavior-row_edit_navigation
  Scenario: Edit action per row navigates to edit screen
    Given the user is on a CRM entity list screen with at least one row
    When the user taps the edit button on a row
    Then the app navigates to the edit screen for that entity

  @happy @FR-304 @TST-071 @behavior-row_body_navigation
  Scenario: Row body navigation goes to read-only detail
    Given the user is on a CRM entity list screen with at least one row
    When the user taps the row body (not the checkbox or edit button)
    Then the app navigates to the read-only detail screen for that entity

  @happy @FR-304 @TST-072 @behavior-bulk_delete_hidden_no_selection
  Scenario: Delete selected button is hidden with no selection
    Given the user is on a CRM entity list screen with no rows selected
    Then the "Delete selected" button is not visible

  @happy @FR-304 @TST-073 @behavior-bulk_delete_visible_with_selection
  Scenario: Delete selected button is visible when rows are selected
    Given the user has selected one or more rows on a CRM list screen
    Then the "Delete selected" button is visible

  @happy @FR-304 @TST-074 @behavior-bulk_delete_confirm
  Scenario: Confirming bulk delete calls delete mutation for each selected entity
    Given the user has selected one or more rows on a CRM list screen
    When the user taps "Delete selected" and confirms the alert
    Then the delete mutation is called for each selected entity id
    And the selection is cleared after successful deletion

  @sad @FR-304 @TST-075 @behavior-bulk_delete_cancel
  Scenario: Cancelling bulk delete does not call delete mutations
    Given the user has selected one or more rows on a CRM list screen
    When the user taps "Delete selected" and cancels the alert
    Then no delete mutations are called
    And the selection remains unchanged

  @happy @FR-304 @TST-076 @behavior-detail_read_only
  Scenario: CRM detail screens are read-only with no primary edit action
    Given the user navigates to any CRM entity detail screen
    Then no primary edit action button is visible on the detail screen
