TEST?=$$(go list ./... | grep -v 'vendor')
HOSTNAME=github.com
NAMESPACE=aliakseiyanchuk
VERSION=0.3.1
BINARY_PREFIX=hcvault-mashery-api-auth
BINARY=${BINARY_PREFIX}_v${VERSION}
DEV_PLUGINS_DIR=./vault/plugins
MASH_AUTH_DEV_BINARY=${BINARY}

default: install

build: vendor
	go build -o ${BINARY} cmd/main.go

launch_dev_mode: kill_dev_vault
	mkdir -p ${DEV_PLUGINS_DIR}
	find ${DEV_PLUGINS_DIR} -type f -exec /bin/rm {} \;
	go build -o ${DEV_PLUGINS_DIR}/${MASH_AUTH_DEV_BINARY} cmd/main.go
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=${DEV_PLUGINS_DIR} -log-level=trace > ./vault/dev-server.log 2>&1 &
	# Let the server start-up before proceeding with the mount
	sleep 5
	echo root | vault login -address=http://localhost:8200/ -
	vault secrets enable -address=http://localhost:8200/ -path=mash-auth \
              -allowed-response-headers="X-Total-Count" \
              -allowed-response-headers="X-Mashery-Responder" \
              -allowed-response-headers="X-Server-Date" \
              -allowed-response-headers="X-Proxy-Mode" \
              -allowed-response-headers="WWW-Authenticate" \
              -allowed-response-headers="X-Mashery-Error-Code" \
              -allowed-response-headers="X-Mashery-Responder" \
              ${MASH_AUTH_DEV_BINARY}

	vault write -address=http://localhost:8200/ mash-auth/roles/demoRole area_id=abc area_nid=10 username=user password=password api_key=apiKey secret=secret

	vault policy write -address=http://localhost:8200/ agent-mcc ./samples/agent/grant_demoRole_policy.hcl
	vault auth enable -address=http://localhost:8200/ approle

	vault write -address=http://localhost:8200/ auth/approle/role/agent-demoRole token_policies=agent-mcc
	if [ ! -d ./.secrets ]; then mkdir .secrets > /dev/null; fi
	vault read -address=http://localhost:8200/ -format=json auth/approle/role/agent-demoRole/role-id | jq -r .data.role_id > ./.secrets/role-id.txt
	vault write -address=http://localhost:8200/ -format=json -f auth/approle/role/agent-demoRole/secret-id | jq -r .data.secret_id > ./.secrets/secret-id.txt
# Do some testing, then execute `make kill_dev_vault` to clean-up

kill_dev_vault:
	./scripts/killDevVault.sh


launch_docker:
	GOOS=linux GOARCH=amd64 go build -o ./docker/local-no-tls/${BINARY} 					cmd/main.go
	sudo docker-compose -f ./docker/local-no-tls/docker-compose.yaml build
	sudo docker-compose -f ./docker/local-no-tls/docker-compose.yaml up

build_base_container_amd64:
	GOOS=linux GOARCH=amd64 go build -o ./docker/base-image/${BINARY} 						cmd/main.go
	openssl dgst -sha256 ./docker/base-image/${BINARY} > ./docker/base-image/${BINARY}.sha256
	DOCKER_DEFAULT_PLATFORM=linux/amd64 docker build --progress=plain ./docker/base-image -t mash-auth-base-v${VERSION}

build_tls_enabled_container_amd64:
	GOOS=linux GOARCH=amd64 go build -o ./docker/tls-enabled/${BINARY} 						cmd/main.go
	openssl dgst -sha256 ./docker/base-image/${BINARY} > ./docker/base-image/${BINARY}.sha256
	docker build ./docker/tls-enabled -t lspwd2/hcvault-mashery-api-auth:${VERSION} -t lspwd2/hcvault-mashery-api-auth:latest

run_tls_enabled_container_amd64: build_tls_enabled_container_amd64
	docker run --rm --cap-add=IPC_LOCK -p 127.0.0.1:8200:8200 lspwd2/hcvault-mashery-api-auth:latest

build_tls_enabled_container_arm:
	GOOS=linux GOARCH=arm64 go build -o ./docker/tls-enabled/${BINARY} 						cmd/main.go
	openssl dgst -sha256 ./docker/base-image/${BINARY} > ./docker/base-image/${BINARY}.sha256
	docker build ./docker/tls-enabled -t lspwd2/hcvault-mashery-api-auth:${VERSION} -t lspwd2/hcvault-mashery-api-auth-arm:latest

run_tls_enabled_container_arm: build_tls_enabled_container_arm
	docker run --rm --cap-add=IPC_LOCK -p 127.0.0.1:8200:8200 lspwd2/hcvault-mashery-api-auth-arm:latest

release:
	GOOS=darwin GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_darwin_amd64 			cmd/main.go
	GOOS=freebsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_freebsd_386 				cmd/main.go
	GOOS=freebsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_freebsd_amd64			cmd/main.go
	GOOS=freebsd GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_freebsd_arm 				cmd/main.go
	GOOS=linux GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_linux_386 					cmd/main.go
	GOOS=linux GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_linux_amd64 				cmd/main.go
	GOOS=linux GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_linux_arm 					cmd/main.go
	GOOS=openbsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_openbsd_386 				cmd/main.go
	GOOS=openbsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_openbsd_amd64 			cmd/main.go
	GOOS=solaris GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_solaris_amd64 			cmd/main.go
	GOOS=windows GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_windows_386.exe 			cmd/main.go
	GOOS=windows GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_windows_amd64.exe 		cmd/main.go

install: build
	mkdir -p ./vault/plugins
	mv ${BINARY} ./vault/plugins

test: FORCE
	go test ./mashery

testacc: kill_dev_vault install launch_dev_mode
	go test ./bdd -v

FORCE:

vendor:
	go mod tidy
	go mod vendor

