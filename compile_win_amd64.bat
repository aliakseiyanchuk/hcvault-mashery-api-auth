set GOOS=windows
set GOARCH=amd64
go build -o vault/plugins/mashery-api-auth.exe cmd/main.go