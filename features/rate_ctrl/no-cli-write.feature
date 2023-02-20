Feature: CLI Write is disabled by default

  Scenario: attempt to invoke CLI write with default configuration should be rejected
    Given remounted secret engine
    Given role cliDisabled configured with:
      | AreaID   | a-b-c-d |
      | AreaNID  | 123     |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    * writing to /roles/cliDisabled/v3/services should fail due to: li write operations are disabled for V3 API
      | a | B |
    * deleting /roles/cliDisabled/v3/services/nonExistent should fail due to: cli write operations are disabled for V3 API
