Feature: Role CRUD API
"""
  The administrator can configure the role using a combination of fields. These feilds are write-only by design.
  The administrator still needs to see a basic confirmation that the role is capable of certain operations.
  """

  Scenario: creating and empty role
    Given empty role emptyRole
    Then role emptyRole current state:
    * - is exportable
    * - does not force proxy mode
    * - allows 2 queries per second
    * - is not V2-capable
    * - is not V3-capable
    * - has indefinite term
    * - has indefinite term remaining
    * - has indefinite use remaining


  Scenario: create a V2-capable role
    Given role v2Role configured with:
      | AreaNID | 34     |
      | ApiKey  | key    |
      | Secret  | Secret |
    Then role v2Role current state:
    * - is exportable
    * - does not force proxy mode
    * - allows 2 queries per second
    * - is V2-capable
    * - is not V3-capable
    * - has indefinite term
    * - has indefinite term remaining
    * - has indefinite use remaining

  Scenario: create a V3-capable role
    Given role v3Role configured with:
      | AreaID   | a-b-c-d |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
    Then role v3Role current state:
    * - is exportable
    * - does not force proxy mode
    * - allows 2 queries per second
    * - is not V2-capable
    * - is V3-capable
    * - has indefinite term
    * - has indefinite term remaining
    * - has indefinite use remaining

  Scenario: create full role
    Given role fullRole configured with:
      | AreaID   | a-b-c-d |
      | AreaNID  | 123     |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
      | QPS      | 65      |
    Then role fullRole current state:
    * - is exportable
    * - does not force proxy mode
    * - allows 65 queries per second
    * - is V2-capable
    * - is V3-capable
    * - has indefinite term
    * - has indefinite term remaining
    * - has indefinite use remaining




