Feature: Limiting number of times a role could be used

  Scenario: Use-based export
    Given  role seedRole configured with:
      | AreaID   | a-b-c-d |
      | AreaNID  | 123     |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    Given empty role dest
    Then data export from role seedRole for role dest with:
      | ExplicitNumUses | 3 |
    * is imported into role dest
    Then after reading dest role 3 times
    * role dest current state:
    * - is V2-capable
    * - is V3-capable
    * - is use-depleted
    * reading /roles/dest/v3/services should fail due to: this role has depleted its usage quota
    * invoking v2 object.query "select * from applications" for role dest should fail due to: this role has depleted its usage quota
    * reading /roles/dest/grant should fail due to: this role has depleted its usage quota
    * reading /roles/dest/grant with query should fail due to: this role has depleted its usage quota
      | api | 3 |
    * reading /roles/dest/grant with query should fail due to: this role has depleted its usage quota
      | api | 2 |


  Scenario: Time-based export
    Given  role seedRole configured with:
      | AreaID   | a-b-c-d |
      | AreaNID  | 123     |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    Given empty role dest
    Then data export from role seedRole for role dest with:
      | ExplicitTerm | 2020-01-01 |
    * is imported into role dest
    * role dest current state:
    * - is V2-capable
    * - is V3-capable
    * - is expired
    * reading /roles/dest/v3/services should fail due to: this role has expired
    * invoking v2 object.query "select * from applications" for role dest should fail due to: this role has expired
    * reading /roles/dest/grant should fail due to: this role has expired
    * reading /roles/dest/grant with query should fail due to: this role has expired
      | api | 2 |
    * reading /roles/dest/grant with query should fail due to: this role has expired
      | api | 3 |

