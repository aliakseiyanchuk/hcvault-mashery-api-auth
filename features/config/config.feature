Feature: Supplying configuration

  We should be able to read and write values and handle expected errors gracefully.

  Scenario: default configuration options
    Given remounted secret engine
    Then cli write is disabled
    * mashery leaf certificate is not pinned
    * mashery issuer certificate is not pinned
    * mashery root certificate is not pinned
    * configuration property net_latency (effective) reads 147ms
    * configuration property oaep_label (effective) matches sha256:[a-z0-9]{10,}
    * configuration property proxy_server is empty
    * configuration property proxy_server_auth is empty
    * configuration property proxy_server_creds is empty


  Scenario: Custom TLS pinning should be enabled only if either of TLS certificates is pinned
    Given remounted secret engine
    Given tls pinning set to custom
    Then effective tls pinning is default

  Scenario: System TLS pinning can be enabled when desired
    Given remounted secret engine
    Given tls pinning set to system
    Then effective tls pinning is system

  Scenario: Configuration accepts all applicable parameters
    Given remounted secret engine configured with
      | OAEPLabel              | label                 |
      | ProxyServer            | http://proxy          |
      | ProxyServerAuth        | NotReallyBasic        |
      | ProxyServerCredentials | SuperSecretProxyCreds |
      | CLIWriteEnabled        | true                  |
      | NetworkLatency         | 500ms                 |
      | TLSPinning             | system                |
    Then cli write is enabled
    * mashery leaf certificate is not pinned
    * mashery issuer certificate is not pinned
    * mashery root certificate is not pinned
    * configuration property net_latency (effective) reads 500ms
    * configuration property oaep_label (effective) reads sha256:1aca80e8b55c802f7b43740da2990e1b5735bbb323d93eb5ebda8395b04025e2
    * configuration property proxy_server reads http://proxy
    * configuration property proxy_server_auth reads NotReallyBasic
    * configuration property proxy_server_creds reads SuperSecretProxyCreds
    * configuration property tls_pinning (desired) reads system
    * configuration property tls_pinning (effective) reads system