Feature: Error handling during import/export operations

  Scenario: import into non-existing role should fail
    Given  role seedRole configured with:
      | AreaID   | a-b-c-d |
      | AreaNID  | 123     |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    Given empty role destProxy
    Then  data export from role seedRole for role destProxy
    * cannot be imported for role notYetCreated explained as "importing data into non-existing role cannot possibly work"
