Feature: Limiting use to proxy mode

  Scenario: Prohibiting external grant
    Given  role seedRole configured with:
      | AreaID   | a-b-c-d |
      | AreaNID  | 123     |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    Given empty role destProxy
    Then data export from role seedRole for role destProxy with:
      | ForceProxyMode | "true" |
    * is imported into role destProxy
    Then role destProxy current state:
    * - is V2-capable
    * - is V3-capable
    * - forces proxy mode
    * reading /roles/destProxy/grant should fail due to: operation is not permitted as this role requires proxy mode
    * reading /roles/destProxy/grant with query should fail due to: operation is not permitted as this role requires proxy mode
      | api | 2 |
    * reading /roles/destProxy/grant with query should fail due to: operation is not permitted as this role requires proxy mode
      | api | 3 |


