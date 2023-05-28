package mashery

import (
	"context"
	"fmt"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	oaepLabelField        = "oaep_label"
	proxyServerField      = "proxy_server"
	proxyServerAuthField  = "proxy_server_auth"
	proxyServerCredsField = "proxy_server_creds"
	cliWriteField         = "enable_cli_v3_write"
	netLatencyField       = "net_latency"
	tlsPinningField       = "tls_pinning"

	tlsPinningDefaultOpt = "default"
	tlsPinningSystemOpt  = "system"
	tlsPinningCustomOpt  = "custom"

	certCommonNameField   = "cn"
	certSerialNumberField = "sn"
	certFingerprintField  = "fp"

	helpSynConfig  = "Configure connectivity to TIBCO Cloud Mashery"
	helpDescConfig = `
A customized configuration is may be required where a direct connection between Vault
and TIBCO Cloud Mashery is not possible, such as where Vault needs to connect via a
non-transparent proxy server.
`
)

var pathBackendConfigFields = map[string]*framework.FieldSchema{
	oaepLabelField: {
		Type:        framework.TypeString,
		Description: "Custom OAEP Label; used to ensure",
		Required:    false,
	},
	proxyServerField: {
		Type:        framework.TypeString,
		Description: "Proxy server",
		Required:    false,
	},
	proxyServerAuthField: {
		Type:        framework.TypeString,
		Description: "Proxy server authentication type",
		Required:    false,
	},
	proxyServerCredsField: {
		Type:        framework.TypeString,
		Description: "Proxy server authentication credentials",
		Required:    false,
	},
	cliWriteField: {
		Type:        framework.TypeBool,
		Description: "Whether to enable CLI write for V3 write-type commands",
		Required:    false,
	},
	netLatencyField: {
		Type:        framework.TypeString,
		Description: "Network latency between Vault and Mashery",
	},
	tlsPinningField: {
		Type:        framework.TypeString,
		Description: fmt.Sprintf("TLS pinning options: %s, %s, or %s", tlsPinningDefaultOpt, tlsPinningSystemOpt, tlsPinningCustomOpt),
	},
}

func pathConfig(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "config/?",
		Fields:  pathBackendConfigFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.storeConfiguration,
				Summary:  "Store back-end configuration",
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.readConfiguration,
				Summary:  "Read effective vs desired back-end configuration",
			},
		},

		ExistenceCheck: alwaysExist,

		HelpSynopsis:    helpSynConfig,
		HelpDescription: helpDescConfig,
	}
}

var pathCertConfigFields = map[string]*framework.FieldSchema{
	certCommonNameField: {
		Type:        framework.TypeString,
		Description: "Common name to pin",
		Required:    false,
	},
	certSerialNumberField: {
		Type:        framework.TypeString,
		Description: "Serial number in hexadecimal notation",
		Required:    false,
	},
	certFingerprintField: {
		Type:        framework.TypeString,
		Description: "SHA-256 fingerprint in hexadecimal notation",
		Required:    false,
	},
}

func pathCertConfig(b *AuthPlugin, suffix string, pinner targetCertPinningSelector) *framework.Path {
	return &framework.Path{
		Pattern: "config/certs/" + suffix,
		Fields:  pathCertConfigFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.updateCert(ctx, request, data, pinner)
				},
				Summary: "Update certificate pinning information",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.resetCert(ctx, request, pinner)
				},
				Summary: "Reset pinned certificate to empty state",
			},
		},

		ExistenceCheck: alwaysExist,
	}
}

func (b *AuthPlugin) readConfiguration(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	return &logical.Response{
		Data: map[string]interface{}{
			"build version":                  "0.3.1",
			oaepLabelField + " (effective)":  formatOptionalSecretValue(b.cfg.EffectiveOAEPLabel()),
			proxyServerField:                 b.cfg.ProxyServer,
			proxyServerAuthField:             b.cfg.ProxyServerAuth,
			proxyServerCredsField:            b.cfg.ProxyServerCreds,
			cliWriteField:                    b.cfg.CLIWriteEnabled,
			netLatencyField + " (effective)": b.cfg.EffectiveNetworkLatency().String(),
			tlsPinningField + " (effective)": formatTLSPinningOption(b.cfg.EffectiveTLSPinning()),
			tlsPinningField + " (desired)":   formatTLSPinningOption(b.cfg.TLSPinning),
			"mashery leaf cert":              formatCertPin(b.cfg.LeafCertPin),
			"mashery issuer cert":            formatCertPin(b.cfg.IssuerCertPin),
			"mashery root cert":              formatCertPin(b.cfg.RootCertPin),
		},
	}, nil
}

type targetCertPinningSelector func(cfg *BackendConfiguration) *transport.TLSCertChainPin

func wrapTargetCertPinSelector(f targetCertPinningSelector) func(ctx context.Context, reqCtx *RequestHandlerContext[TLSPinningOperations]) (*logical.Response, error) {
	return func(_ context.Context, reqCtx *RequestHandlerContext[TLSPinningOperations]) (*logical.Response, error) {
		reqCtx.heap.CarryPinning(f(reqCtx.heap.GetBackendConfiguration()))
		return nil, nil
	}
}

func (b *AuthPlugin) updateCert(ctx context.Context,
	req *logical.Request,
	d *framework.FieldData,
	pinSelector targetCertPinningSelector) (*logical.Response, error) {

	var container TLSPinningOperations
	container = &TLSPinningContainer{}

	pinChain := SimpleChain(
		readBackEndConfig[TLSPinningOperations],
		wrapTargetCertPinSelector(pinSelector),
		parseCertificatePinConfiguration,
		saveBackEndConfigFunc[TLSPinningOperations],
		acceptBackendConfigurationFunc[TLSPinningOperations],
	)

	return handleOperationWithContainer(ctx, b, req, d, container, b.configPath(), pinChain)
}

func (b *AuthPlugin) resetCert(ctx context.Context, req *logical.Request,
	pinSelector targetCertPinningSelector) (*logical.Response, error) {

	var container TLSPinningOperations
	container = &TLSPinningContainer{}

	pinChain := SimpleChain(
		readBackEndConfig[TLSPinningOperations],
		wrapTargetCertPinSelector(pinSelector),
		resetCertificatePin,
		saveBackEndConfigFunc[TLSPinningOperations],
		acceptBackendConfigurationFunc[TLSPinningOperations],
	)

	return handleOperationWithContainer(ctx, b, req, nil, container, b.configPath(), pinChain)
}

func (b *AuthPlugin) storeConfiguration(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {

	var container BackendConfigurationContext
	container = &BackendConfigurationContainer{}

	chain := SimpleChain(
		readBackEndConfig[BackendConfigurationContext],
		parseBackEndConfigurationFunc,
		saveBackEndConfigFunc[BackendConfigurationContext],
		acceptBackendConfigurationFunc[BackendConfigurationContext],
	)

	return handleOperationWithContainer(ctx, b, req, d, container, b.configPath(), chain)
}
