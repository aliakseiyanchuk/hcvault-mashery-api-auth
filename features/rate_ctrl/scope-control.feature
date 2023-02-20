Feature: Checking role's capabilities before invoking

  Scenario: attempt to invoke v3-incapable role will fail
    Given  role v3Miss configured with:
      | AreaNID  | 123    |
      | ApiKey   | key    |
      | Secret   | Secret |
      | Username | user   |
      | Password | pwd    |
      | QPS      | 65     |
    Then role v3Miss current state:
    * - is V2-capable
    * - is not V3-capable
    * reading /roles/v3Miss/v3/services should fail due to: this role is not V3-capable
    * reading /roles/v3Miss/grant should fail due to: this role doesn't bear required elements to issue V3 credentials
    * reading /roles/v3Miss/grant with query should fail due to: this role doesn't bear required elements to issue V3 credentials
      | api | 3 |

  Scenario: attempt to invoke v2-incapable role will fail
    Given  role v2Miss configured with:
      | AreaID   | a-b-c-d |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    Then role v2Miss current state:
    * - is not V2-capable
    * - is V3-capable
    * invoking v2 object.query "select * from applications" for role v2Miss should fail due to: this role is not V2-capable
    * reading /roles/v2Miss/grant with query should fail due to: this role doesn't bear required elements to issue V2 credentials
      | api | 2 |
