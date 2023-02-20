package bdd_test

import (
	"encoding/json"
	"fmt"
	vault "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"os"
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
	execName := "hcvault-mashery-api-auth"

	if files, err := os.ReadDir("../vault/plugins"); err == nil {
		if len(files) == 1 {
			execName = files[0].Name()
		} else {
			fmt.Println("WARN: ambiguous plugin name. Mount can fail. Leave a single file in vault plugins directory")
		}
	} else {
		fmt.Println(err.Error())
	}

	req := vault.MountInput{
		Type:        execName,
		Description: "Mount for unit testing",
	}
	_, err := vcl.Logical().Write("/sys/mounts/"+path, vaultAPIMap(req))
	return err
}

func unmount(path string) error {
	_, err := vcl.Logical().Delete("/sys/mounts/" + path)
	return err
}
