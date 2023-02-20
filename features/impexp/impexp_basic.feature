Feature: Import-export behaviour

  Scenario: Importing data from another role makes it unexportable
    Given  role seedRole configured with:
      | AreaID   | a-b-c-d |
      | AreaNID  | 123     |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    Given empty role destProxy
    When  data export from role seedRole for role destProxy
    * is imported into role destProxy
    Then role destProxy current state:
    * - is not exportable


  Scenario: Attempt to export data from non-exportable function should fail
    Given  role seedRole configured with:
      | AreaID   | a-b-c-d |
      | AreaNID  | 123     |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    Given empty role destProxy
    When  data export from role seedRole for role destProxy
    * is imported into role destProxy
    Then role destProxy current state:
    * - is not exportable
    Given empty role destProxy2
    Then  data export from role destProxy for role destProxy2 fails due to: this role is not exportable