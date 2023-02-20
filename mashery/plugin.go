package mashery

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/v2client"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/v3client"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"strings"
	"sync"
	"time"
)

const (
	pluginVersionL = "TIBCO Cloud Mashery Secret Engine v 0.2a"

	httpIdle = time.Minute * -15
)

type V3ClientAndAuthorizer struct {
	client        v3client.WildcardClient
	tokenProvider v3client.FixedTokenProvider
	lastUsed      time.Time
}

type V2ClientAndAuthorizer struct {
	client        v2client.Client
	tokenProvider *v2client.V2Authorizer
	lastUsed      time.Time
}

type AuthPlugin struct {
	*framework.Backend

	cfg               BackendConfiguration
	v3OauthHelper     *v3client.V3OAuthHelper
	v3OAuthHelperInit sync.Once

	v3Clients   map[string]V3ClientAndAuthorizer
	v2Clients   map[string]V2ClientAndAuthorizer
	backendUUID string

	vaultStorage VaultStorage
}

func (b *AuthPlugin) GetOAuthHelper() *v3client.V3OAuthHelper {
	b.v3OAuthHelperInit.Do(func() {
		params := v3client.OAuthHelperParams{
			HTTPClientParams: transport.HTTPClientParams{
				TLSConfig:            b.cfg.EffectiveTLSConfiguration(),
				ProxyServer:          b.cfg.ProxyServerURL(),
				ProxyAuthType:        b.cfg.ProxyServerAuth,
				ProxyAuthCredentials: b.cfg.ProxyServerCreds,
			},
		}

		// Make sure the helper will delegate this to the system.
		if b.cfg.EffectiveTLSPinning() == TLSPinningSystem {
			params.HTTPClientParams.TLSConfigDelegateSystem = true
		}

		b.v3OauthHelper = v3client.NewOAuthHelper(params)
	})

	return b.v3OauthHelper
}

func (b *AuthPlugin) AcceptConfigurationUpdate(ctx context.Context, newCfg BackendConfiguration) {
	b.cfg = newCfg

	b.v3OAuthHelperInit = sync.Once{}
	for k := range b.v2Clients {
		cl := b.v2Clients[k]
		delete(b.v2Clients, k)

		cl.client.Close(ctx)
	}

	for k := range b.v3Clients {
		cl := b.v3Clients[k]
		delete(b.v3Clients, k)

		cl.client.Close(ctx)
	}
}

func (b *AuthPlugin) GetMasheryV3Client(role *StoredRole) v3client.WildcardClient {
	key := fmt.Sprintf("%s::%s", b.backendUUID, role.Name)

	if cl := b.v3Clients[key]; cl.client != nil {
		cl.lastUsed = time.Now()
		cl.tokenProvider.UpdateToken(role.Usage.V3Token)
		return cl.client
	} else {
		var provider = v3client.NewFixedTokenProvider(role.Usage.V3Token).(v3client.FixedTokenProvider)

		params := v3client.Params{
			// Mashery V3 configuration needs to be added!

			HTTPClientParams: transport.HTTPClientParams{
				TLSConfig:               b.cfg.EffectiveTLSConfiguration(),
				TLSConfigDelegateSystem: b.cfg.EffectiveTLSPinning() == TLSPinningSystem,
			},
			Authorizer:    provider,
			QPS:           int64(role.Keys.MaxQPS),
			AvgNetLatency: time.Millisecond * 172,
		}

		var clInst = v3client.NewWildcardClient(params)

		cl := V3ClientAndAuthorizer{
			tokenProvider: provider,
			client:        clInst,
			lastUsed:      time.Now(),
		}

		b.v3Clients[key] = cl
		return clInst
	}

}

func (b *AuthPlugin) GetMasheryV2Client(role *StoredRole) v2client.Client {
	key := b.getClientLookupKey(role)

	if cl := b.v2Clients[key]; cl.client != nil {
		cl.lastUsed = time.Now()
		cl.tokenProvider.UpdateSignature(b.v2SignatureFor(role))
		return cl.client
	} else {
		b.Logger().Info("Constructing the V2 client", "token", role.Usage.V3Token)
		var provider = v2client.NewV2Authorizer(role.Keys.ApiKey)
		provider.UpdateSignature(b.v2SignatureFor(role))

		v2Params := v2client.Params{
			AreaNID:        role.Keys.AreaNid,
			Authorizer:     provider,
			QPS:            int64(role.Keys.MaxQPS),
			TravelTimeComp: time.Millisecond * 172,
		}
		var clInst = v2client.NewHTTPClient(v2Params)

		cl := V2ClientAndAuthorizer{
			tokenProvider: provider,
			client:        clInst,
			lastUsed:      time.Now(),
		}

		b.v2Clients[key] = cl
		return clInst
	}
}

func (b *AuthPlugin) getClientLookupKey(role *StoredRole) string {
	key := fmt.Sprintf("%s::%s", b.backendUUID, role.Name)
	return key
}

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, _ := makeNew(conf)
	setupErr := b.Setup(ctx, conf)
	if setupErr != nil {
		return nil, setupErr
	}

	return b, nil
}

func (b *AuthPlugin) initOnMount(ctx context.Context, request *logical.InitializationRequest) error {
	if cfg, err := b.loadBackendConfiguration(ctx, request.Storage); err != nil {
		return err
	} else {
		b.cfg = cfg
		return nil
	}
}

// Housekeeping performs a housekeeping, freeing the client objects that are not actively used.
func (b *AuthPlugin) Housekeeping(ctx context.Context, _ *logical.Request) error {
	lastUseCutover := time.Now().Add(httpIdle)
	for k := range b.v3Clients {
		if b.v3Clients[k].lastUsed.Before(lastUseCutover) {
			o := b.v3Clients[k]

			delete(b.v3Clients, k)

			o.client.Close(ctx)
		}
	}

	for k := range b.v2Clients {
		if b.v2Clients[k].lastUsed.Before(lastUseCutover) {
			o := b.v2Clients[k]
			delete(b.v2Clients, k)

			o.client.Close(ctx)
		}
	}

	return nil
}

func makeNew(conf *logical.BackendConfig) (*AuthPlugin, error) {

	vaultStorage := VaultStorageImpl{}

	retVal := AuthPlugin{
		v3OauthHelper:     nil,
		v3OAuthHelperInit: sync.Once{},
		backendUUID:       conf.BackendUUID,
		v2Clients:         map[string]V2ClientAndAuthorizer{},
		v3Clients:         map[string]V3ClientAndAuthorizer{},
		vaultStorage:      &vaultStorage,
	}

	retVal.Backend = &framework.Backend{
		Help:           strings.TrimSpace(pluginHelp),
		InitializeFunc: retVal.initOnMount,
		PeriodicFunc:   retVal.Housekeeping,

		BackendType: logical.TypeLogical,
		Paths: []*framework.Path{
			pathConfig(&retVal),
			pathCertConfig(&retVal, "leaf", func(cfg *BackendConfiguration) *transport.TLSCertChainPin {
				return &cfg.LeafCertPin
			}),
			pathCertConfig(&retVal, "issuer", func(cfg *BackendConfiguration) *transport.TLSCertChainPin {
				return &cfg.IssuerCertPin
			}),
			pathCertConfig(&retVal, "root", func(cfg *BackendConfiguration) *transport.TLSCertChainPin {
				return &cfg.RootCertPin
			}),

			pathRolesRoot(&retVal),
			pathRole(&retVal),
			pathRoleImpExpGetPEM(&retVal),
			pathRoleImpExpExport(&retVal),
			pathRoleImpExpImport(&retVal),
			pathRoleGrant(&retVal),
			pathRoleForgetToken(&retVal),

			// Support several flavours of accepting the V2 method.
			pathRoleV2(&retVal, "roles/"+framework.GenericNameRegex(roleName)+"/v2"),
			pathRoleV2(&retVal, "roles/"+framework.GenericNameRegex(roleName)+"/v2/"+framework.GenericNameRegex(v2MethodField)),

			pathRoleV3(&retVal),

			pathProxyV2(&retVal),
			pathProxyV3(&retVal),
		},
		Secrets: []*framework.Secret{
			v2AccessSecret(),
			v3AccessSecret(),
		},
	}

	retVal.Logger().Info("Mashery V2/V3 authentication plugin has been initialized")
	return &retVal, nil
}

func (b *AuthPlugin) storagePathForRole(data *framework.FieldData) string {
	return b.rolesStorageRoot() + b.roleName(data)
}

func (b *AuthPlugin) roleName(data *framework.FieldData) string {
	return data.Get(roleName).(string)
}

func (b *AuthPlugin) rolesStorageRoot() string {
	return b.backendUUID + "/role/"
}

func (b *AuthPlugin) checkObjectExistsInStorage(ctx context.Context,
	req *logical.Request,
	rolePath string) (bool, error) {

	if out, err := req.Storage.Get(ctx, rolePath); err != nil {
		return false, errwrap.Wrapf("existence check failed: {{err}}", err)
	} else {
		return out != nil, nil
	}
}

// noopRenewRevoke revocation of the secret that was issued
func noopRenewRevoke(context.Context, *logical.Request, *framework.FieldData) (*logical.Response, error) {
	return nil, nil
}

func (b *AuthPlugin) configPath() string {
	return b.backendUUID + "/config"
}

func (b *AuthPlugin) loadBackendConfiguration(ctx context.Context, storage logical.Storage) (BackendConfiguration, error) {
	cfg := b.DefaultBackendConfiguration()

	cfgFound, err := b.vaultStorage.Read(ctx, storage, b.configPath(), &cfg)

	if err == nil && !cfgFound {
		err = b.vaultStorage.Persist(ctx, storage, b.configPath(), &cfg)
	}
	return cfg, err
}

func (b *AuthPlugin) DefaultBackendConfiguration() BackendConfiguration {
	rv := BackendConfiguration{
		CLIWriteEnabled: false,
		NetworkLatency:  147,
		OAEPLabel:       make([]byte, 32),
	}

	_, _ = rand.Read(rv.OAEPLabel)

	return rv
}

func (b *AuthPlugin) InitialRole(data *framework.FieldData) StoredRole {
	return StoredRole{
		Keys: RoleKeys{
			MaxQPS:         defaultQPSValue,
			ForceProxyMode: false,
			Exportable:     true,
			Imported:       false,
		},
		Usage:       StoredRoleUsage{},
		Name:        b.roleName(data),
		StoragePath: b.storagePathForRole(data),
	}
}

// handleOperationWithContainer generic implementation where calling cole has to supply all parameters
func handleOperationWithContainer[Container any](ctx context.Context,
	b *AuthPlugin,
	req *logical.Request,
	d *framework.FieldData,
	c Container,
	path string,
	f TransformerFunc[Container]) (*logical.Response, error) {
	reqCtx := RequestHandlerContext[Container]{
		req,
		d,
		b,
		path,
		c,
	}

	return f(ctx, &reqCtx)
}

func handleRoleBoundOperation(ctx context.Context, b *AuthPlugin, req *logical.Request, data *framework.FieldData, f TransformerFunc[RoleContext]) (*logical.Response, error) {
	var container RoleContext
	container = &RoleContainer{}

	return handleOperationWithContainer(ctx, b, req, data, container, b.storagePathForRole(data), f)
}

func handleWildcardAPIRoleBoundOperation(ctx context.Context, b *AuthPlugin, req *logical.Request, data *framework.FieldData, f TransformerFunc[WildcardAPIResponseContext]) (*logical.Response, error) {
	var container WildcardAPIResponseContext
	container = &APIResponseContainer[*transport.WrappedResponse]{}

	return handleOperationWithContainer(ctx, b, req, data, container, b.storagePathForRole(data), f)
}

const pluginHelp = "Mashery V2/V3 Authentication plugin used to generate V2 signatures and V3 access tokens"
