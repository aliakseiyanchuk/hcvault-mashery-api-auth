package mashery_test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
	"yanchuk.nl/hcvault-mashery-api-auth/mashery"
)

func TestV3NeedsRenewOnEmpty(t *testing.T) {
	role := mashery.StoredRole{}
	assert.True(t, role.Usage.V3TokenNeedsRenew())
}

func TestV3TokenExpiry(t *testing.T) {
	role := mashery.StoredRole{
		Usage: mashery.StoredRoleUsage{
			V3Token:       "Boo",
			V3TokenExpiry: 1,
		},
	}

	assert.True(t, role.Usage.V3TokenExpired())
	assert.True(t, role.Usage.V3TokenNeedsRenew())

	role.Usage.V3TokenExpiry = time.Now().Unix() + 5
	assert.False(t, role.Usage.V3TokenExpired())
	assert.True(t, role.Usage.V3TokenNeedsRenew())

	role.Usage.V3TokenExpiry = time.Now().Unix() + 3600
	assert.False(t, role.Usage.V3TokenExpired())
	assert.False(t, role.Usage.V3TokenNeedsRenew())
}

func TestGettingExportableRole(t *testing.T) {
	role := mashery.StoredRole{
		Keys: mashery.RoleKeys{
			ApiKey: "key",
		},
	}

	dur, err := time.ParseDuration("72h")
	assert.Nil(t, err)

	exp := role.CreateRoleDataExchange(dur)
	assert.True(t, exp.UsageTerm.ExplicitTerm > 0)

	exp = role.CreateRoleDataExchange(0)
	assert.True(t, exp.UsageTerm.ExplicitTerm == 0)
}

func TestStoredRoleWillImportExportedData(t *testing.T) {
	sr := mashery.StoredRole{}

	exp := mashery.RoleDataExchange{
		RoleData: mashery.RoleKeys{
			ApiKey:    "apiKey",
			KeySecret: "keySecret",
			MaxQPS:    45,
			Username:  "User",
			Password:  "Pwd",
		},
		UsageTerm: &mashery.RoleUsageTerm{
			ExplicitTerm:    345,
			ExplicitNumUses: 500,
		},
	}

	sr.Import(exp)
	assert.Equal(t, "apiKey", sr.Keys.ApiKey)
	assert.Equal(t, "keySecret", sr.Keys.KeySecret)
	assert.Equal(t, 45, sr.Keys.MaxQPS)
	assert.Equal(t, "User", sr.Keys.Username)
	assert.Equal(t, "Pwd", sr.Keys.Password)
	assert.Equal(t, int64(345), sr.Usage.ExplicitTerm)
	assert.Equal(t, int64(500), sr.Usage.ExplicitNumUses)
	assert.Equal(t, int64(500), sr.Usage.RemainingNumUses)
}
