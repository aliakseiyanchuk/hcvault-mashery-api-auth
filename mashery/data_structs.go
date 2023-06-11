package mashery

import (
	"context"
	"crypto/tls"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/v3client"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"net/url"
	"time"
)

const (
	roleAreaIdField   = "area_id"
	roleAreaNidField  = "area_nid"
	roleApiKeField    = "api_key"
	roleSecretField   = "secret"
	roleUsernameField = "username"
	rolePasswordField = "password"
	roleQpsField      = "qps"

	secretAccessToken           = "access_token"
	secretAccessTokenExpiryTime = "expiry"
	secretSignedSecretField     = "sig"

	secretInternalRoleStoragePath = "roleStoragePath"
	secretInternalRefreshToken    = "refresh_token"
	// Token expiry time in Epoch seconds
	secretInternalTokenExpiryTime = "token_expiry_time"
)

const (
	TLSPinningDefault = iota
	TLSPinningSystem
	TLSPinningCustom
)

type BackendConfiguration struct {
	OAEPLabel        []byte `json:"_oaep_label"`
	ProxyServer      string `json:"_proxy_s"`
	ProxyServerAuth  string `json:"_proxy_t"`
	ProxyServerCreds string `json:"_proxy_c"`

	CLIWriteEnabled bool `json:"_cli_rw"`

	NetworkLatency int `json:"_net_l"`

	TLSPinning int `json:"_tls_pinning"`

	// Pinning options
	LeafCertPin   transport.TLSCertChainPin `json:"_leaf_pin"`
	IssuerCertPin transport.TLSCertChainPin `json:"_int_pin"`
	RootCertPin   transport.TLSCertChainPin `json:"_root_pin"`
}

func (b *AuthPlugin) DoIfCLIWriteEnabled(cb framework.OperationFunc) framework.OperationFunc {
	return func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		if b.cfg.CLIWriteEnabled {
			return cb(ctx, request, data)
		} else {
			return logical.ErrorResponse("cli write operations are disabled for V3 API"), nil
		}
	}
}

func (bc *BackendConfiguration) ProxyServerURL() *url.URL {
	if len(bc.ProxyServer) == 0 {
		return nil
	}

	if proxyURL, err := url.Parse(bc.ProxyServer); err == nil {
		return proxyURL
	} else {
		panic(err)
	}
}

func (bc *BackendConfiguration) EffectiveOAEPLabel() []byte {
	return bc.OAEPLabel
}

func (bc *BackendConfiguration) EffectiveNetworkLatency() time.Duration {
	if bc.NetworkLatency > 0 {
		return time.Millisecond * time.Duration(bc.NetworkLatency)
	} else {
		return time.Millisecond * 147
	}
}

func (bc *BackendConfiguration) EffectiveTLSPinning() int {
	if bc.TLSPinning == TLSPinningCustom {
		if bc.LeafCertPin.IsEmpty() && bc.IssuerCertPin.IsEmpty() && bc.RootCertPin.IsEmpty() {
			return TLSPinningDefault
		}
	}
	return bc.TLSPinning
}

func (bc *BackendConfiguration) EffectiveTLSConfiguration() *tls.Config {
	switch bc.EffectiveTLSPinning() {
	case TLSPinningDefault:
		return transport.DefaultTLSConfig()
	case TLSPinningSystem:
		return nil
	case TLSPinningCustom:
		pinner := transport.TLSPinner{}

		if !bc.LeafCertPin.IsEmpty() {
			pinner.Add(bc.LeafCertPin)
		}
		if !bc.IssuerCertPin.IsEmpty() {
			pinner.Add(bc.IssuerCertPin)
		}
		if !bc.RootCertPin.IsEmpty() {
			pinner.Add(bc.RootCertPin)
		}

		return pinner.CreateTLSConfig()
	default:
		return transport.DefaultTLSConfig()
	}
}

// RoleKeys Keys of Mashery authentication role.
type RoleKeys struct {
	AreaId         string `json:"aid,omitempty"`
	AreaNid        int    `json:"nid,omitempty"`
	ApiKey         string `json:"key,omitempty"`
	KeySecret      string `json:"srt,omitempty"`
	Username       string `json:"usr,omitempty"`
	Password       string `json:"pwd,omitempty"`
	MaxQPS         int    `json:"qps,omitempty"`
	ForceProxyMode bool   `json:"fpm,omitempty"`
	Imported       bool   `json:"_imp"`
	Exportable     bool   `json:"_exp"`
}

type RoleUsageTerm struct {
	ExplicitTerm    int64 `json:"etm,omitempty"`
	ExplicitNumUses int64 `json:"enu,omitempty"`
}

// RoleDataExchange data struct used in the import/export operations
type RoleDataExchange struct {
	RoleData  RoleKeys       `json:"d"`
	UsageTerm *RoleUsageTerm `json:"u,omitempty"`
}

// StoredRolePrivateKey stored role private key. Private Keys are written in a separate struct, as these tend to
// be big and used infrequently
type StoredRolePrivateKey struct {
	PrivateKey string `json:"_pk"`
}

// StoredRoleUsage stores mutable information about role Usage, such as acquired token and remaining
// number of calls / time limit
type StoredRoleUsage struct {
	V3Token          string `json:"_v3t"`
	V3TokenExpiry    int64  `json:"_v3te"`
	ExplicitNumUses  int64  `json:"enu,omitempty"`
	RemainingNumUses int64  `json:"_urm"`
	ExplicitTerm     int64  `json:"etm,omitempty"`
}

// ReplaceAccessToken replace access token and expiry time used in this struct.
func (sru *StoredRoleUsage) ReplaceAccessToken(tkn string, expiry int64) {
	sru.V3Token = tkn
	sru.V3TokenExpiry = expiry
}

// StoredRole Authentication role data that is stored within Vault encrypted storage
type StoredRole struct {
	Keys       RoleKeys
	Usage      StoredRoleUsage
	PrivateKey []byte

	Name        string
	StoragePath string
}

func (sr *StoredRoleUsage) HasUsageQuota() bool {
	return sr.ExplicitNumUses > 0
}

func (sr *StoredRoleUsage) ReduceRemainingQuota() {
	if sr.RemainingNumUses > 0 {
		sr.RemainingNumUses--
	}
}

func (sr *StoredRoleUsage) V3TokenExpired() bool {
	return sr.V3TokenExpiry > 0 && time.Now().Unix() > sr.V3TokenExpiry
}

func (sr *StoredRoleUsage) UnboundedUsage() {
	sr.ExplicitNumUses = -1
	sr.ExplicitTerm = -1
	sr.RemainingNumUses = -1
}

func (sr *StoredRoleUsage) IsUnboundedUsage() bool {
	return sr.ExplicitNumUses <= 0 &&
		sr.ExplicitTerm <= 0
}

func (sr *StoredRoleUsage) ResetToken() {
	sr.V3Token = ""
	sr.V3TokenExpiry = -1
}

func (sr *StoredRoleUsage) V3TokenNeedsRenew() bool {
	return len(sr.V3Token) == 0 || time.Now().Unix() > sr.V3TokenExpiry-300
}

func (ar *StoredRoleUsage) ExpiryTimeString() string {
	if ar.ExplicitTerm > 0 {
		return time.Unix(ar.ExplicitTerm, 0).Format(time.RFC822)
	} else {
		return "∞"
	}
}

func (ar *RoleKeys) IsV2Capable() bool {
	return ar.AreaNid > 0 && ar.SuppliesKeyAndSecret()
}

func (ar *RoleKeys) SuppliesKeyAndSecret() bool {
	return len(ar.ApiKey) > 0 && len(ar.KeySecret) > 0
}

func (ar *RoleKeys) IsV3Capable() bool {
	return len(ar.AreaId) > 0 &&
		ar.SuppliesKeyAndSecret() &&
		len(ar.Username) > 0 && len(ar.Password) > 0
}

func (ar *StoredRoleUsage) AfterExpiryTerm(t time.Time) bool {
	return ar.ExplicitTerm > 0 && t.Unix() > ar.ExplicitTerm
}

func (ar *StoredRoleUsage) HasNotExpired() bool {
	return ar.ExplicitTerm == 0 || time.Now().Unix() < ar.ExplicitTerm
}

func (ar *StoredRoleUsage) Expired() bool {
	return ar.ExplicitTerm > 0 && time.Now().Unix() > ar.ExplicitTerm
}

func (ar *StoredRoleUsage) Depleted() bool {
	return ar.ExplicitNumUses > 0 && ar.RemainingNumUses <= 0
}

func (ar *StoredRole) CreateRoleDataExchange(term time.Duration) RoleDataExchange {
	var exp int64 = 0
	if term != 0 {
		exp = time.Now().Add(term).Unix()
	}

	return RoleDataExchange{
		RoleData: RoleKeys{AreaId: ar.Keys.AreaId,
			AreaNid:   ar.Keys.AreaNid,
			ApiKey:    ar.Keys.ApiKey,
			KeySecret: ar.Keys.KeySecret,
			Username:  ar.Keys.Username,
			Password:  ar.Keys.Password,
			MaxQPS:    ar.Keys.MaxQPS,
		},
		UsageTerm: &RoleUsageTerm{
			ExplicitTerm: exp,
		},
	}
}

func (ar *StoredRole) Import(role RoleDataExchange) {
	ar.Keys.AreaId = role.RoleData.AreaId
	ar.Keys.AreaNid = role.RoleData.AreaNid
	ar.Keys.ApiKey = role.RoleData.ApiKey
	ar.Keys.KeySecret = role.RoleData.KeySecret
	ar.Keys.Username = role.RoleData.Username
	ar.Keys.Password = role.RoleData.Password
	ar.Keys.MaxQPS = role.RoleData.MaxQPS
	ar.Keys.ForceProxyMode = role.RoleData.ForceProxyMode
	ar.Keys.Imported = true
	ar.Keys.Exportable = role.RoleData.Exportable

	if role.UsageTerm != nil {
		ar.Usage.ExplicitTerm = role.UsageTerm.ExplicitTerm
		ar.Usage.ExplicitNumUses = role.UsageTerm.ExplicitNumUses

		if ar.Usage.ExplicitNumUses > 0 {
			ar.Usage.RemainingNumUses = ar.Usage.ExplicitNumUses
		}
	} else {
		ar.Usage.UnboundedUsage()
	}
}

func (ar *StoredRole) asV3Credentials() v3client.MasheryV3Credentials {
	return v3client.MasheryV3Credentials{
		AreaId:   ar.Keys.AreaId,
		ApiKey:   ar.Keys.ApiKey,
		Secret:   ar.Keys.KeySecret,
		Username: ar.Keys.Username,
		Password: ar.Keys.Password,
	}
}
