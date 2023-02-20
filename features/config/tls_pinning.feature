Feature: TLS pinning configuration
"""
  As an administrator, you can specify custom pinning of Mashery certificates. Where the certificate would not
  match your pinning configuration, the call would be rejected.
  """

  Scenario: Pinning leaf certificate with CN alone leads to refused call
    Given remounted secret engine
    Given leaf certificate is pinned with:
      | CommonName | non-existing |
    Given tls pinning set to custom
    Then mashery leaf certificate is pinned as "cn=non-existing"
    And effective tls pinning is custom
    Given role v3Role configured with:
      | AreaID   | a-b-c-d |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
    * reading /roles/v3Role/v3/services should fail due to: Post "https://api.mashery.com/v3/token": no matching chains

  Scenario: Pinning issuer certificate with CN alone leads to refused call
    Given remounted secret engine
    Given issuer certificate is pinned with:
      | CommonName | non-existing |
    Given tls pinning set to custom
    Then mashery issuer certificate is pinned as "cn=non-existing"
    And effective tls pinning is custom
    Given role v3Role configured with:
      | AreaID   | a-b-c-d |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
    * reading /roles/v3Role/v3/services should fail due to: Post "https://api.mashery.com/v3/token": no matching chains

  Scenario: Pinning root certificate with CN alone leads to refused call
    Given remounted secret engine
    Given root certificate is pinned with:
      | CommonName | non-existing |
    Given tls pinning set to custom
    Then mashery root certificate is pinned as "cn=non-existing"
    And effective tls pinning is custom
    Given role v3Role configured with:
      | AreaID   | a-b-c-d |
      | ApiKey   | key     |
      | Secret   | Secret  |
      | Username | user    |
      | Password | pwd     |
    * reading /roles/v3Role/v3/services should fail due to: Post "https://api.mashery.com/v3/token": no matching chains

