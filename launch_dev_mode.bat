@echo off
set VAULT_EXEC=D:\lib\vault\1.9.2\vault

start "Vault Dev Server" %VAULT_EXEC% server -dev -dev-root-token-id=root -dev-plugin-dir=./vault/plugins -log-level=trace
timeout 5
set VAULT_ADDR=http://localhost:8200/
%VAULT_EXEC% login root
%VAULT_EXEC% secrets enable -path=mash-auth^
    -allowed-response-headers="X-Total-Count"^
    -allowed-response-headers="X-Mashery-Responder"^
    -allowed-response-headers="X-Server-Date"^
    -allowed-response-headers="X-Proxy-Mode"^
    -allowed-response-headers="WWW-Authenticate"^
    -allowed-response-headers="X-Mashery-Error-Code"^
    -allowed-response-headers="X-Mashery-Responder"^
    mashery-api-auth.exe

%VAULT_EXEC% policy write agent-mcc ./samples/agent/grant_demoRole_policy.hcl
%VAULT_EXEC% auth enable approle

%VAULT_EXEC% write auth/approle/role/agent-demoRole token_policies=agent-mcc

mkdir .secrets
%VAULT_EXEC% read -format=json auth/approle/role/agent-demoRole/role-id | jq -r .data.role_id > ./.secrets/role-id.txt
%VAULT_EXEC% write -format=json -f auth/approle/role/agent-demoRole/secret-id | jq -r .data.secret_id > ./.secrets/secret-id.txt


