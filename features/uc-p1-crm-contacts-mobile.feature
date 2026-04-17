@UC-P1 @stack-mobile
Feature: CRM Contacts — Mobile screens
  As a sales or support rep
  I want to browse and inspect contacts from the mobile app
  So that I can access contact context during active calls or cases

  @happy @FR-300 @TST-mobile-contacts-01 @behavior-list_contacts
  Scenario: Browse contacts list
    Given the user is authenticated in the mobile app
    When the user navigates to the contacts screen
    Then a list of contacts is displayed with name and email
    And the user can search contacts by name

  @happy @FR-300 @TST-mobile-contacts-02 @behavior-view_contact_detail
  Scenario: View contact detail with agent activity
    Given the user is on the contacts list screen
    When the user taps a contact
    Then the contact detail screen shows name, email, phone, and account name
    And the agent activity section is visible for that contact
    And the signals section is visible for that contact
