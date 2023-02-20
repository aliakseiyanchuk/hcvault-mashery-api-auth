package bdd_test

import (
	"encoding/json"
	vault "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"testing"
)

var vcl *vault.Client

func init() {
	vcl, _ = vault.NewClient(vault.DefaultConfig())
	vcl.SetAddress("http://localhost:8200/")
	vcl.SetToken("root")
}

func TestVaultNotNull(t *testing.T) {
	assert.NotNil(t, vcl)
}

func vaultAPIQueryMap(in map[string]string) map[string][]string {
	rv := map[string][]string{}
	for k, v := range in {
		rv[k] = []string{v}
	}

	return rv
}

func vaultAPIMap(i interface{}) map[string]interface{} {
	str, _ := json.Marshal(i)

	var rv map[string]interface{}
	_ = json.Unmarshal(str, &rv)

	return rv
}

func mount(path string) error {
	req := vault.MountInput{
		Type:        "mashery-api-auth.exe",
		Description: "Mount for unit testing",
	}
	_, err := vcl.Logical().Write("/sys/mounts/"+path, vaultAPIMap(req))
	return err
}

func unmount(path string) error {
	_, err := vcl.Logical().Delete("/sys/mounts/" + path)
	return err
}
