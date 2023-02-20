Feature: OAEP label protects encryption

  Scenario: OAEP label changed between import and export, leading to import error
    Given role source configured with:
      | AreaNID | 345   |
      | AreaID | a-b-c-d   |
    Given empty role target
    Then  data export from role source for role target
    * after oaep label has been changed to a0:a1:a2:a3
    * cannot be imported for role target explained as "was unable to decrypt the Mashery role data (crypto/rsa: decryption error)"
