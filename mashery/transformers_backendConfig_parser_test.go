package mashery

import (
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"testing"
)

func setupConfigRequestMockWithData[T any](container T, data map[string]interface{}, schema map[string]*framework.FieldSchema) *RequestHandlerContext[T] {
	reqCtx := &RequestHandlerContext[T]{
		storagePath: "/backendUUID/testRole",
		request: &logical.Request{
			Storage: nil,
			Data:    data,
		},
		data: &framework.FieldData{
			Raw:    data,
			Schema: schema,
		},
		plugin: &AuthPlugin{
			vaultStorage: &VaultStorageImpl{},
		},
		heap: container,
	}
	return reqCtx
}

func TestParseBackEndConfigurationFunc_FullSettings(t *testing.T) {
	container := BackendConfigurationContainer{}

	var reqCtx = setupConfigRequestMockWithData[BackendConfigurationContext](&container,
		map[string]interface{}{
			oaepLabelField:        "abc",
			proxyServerField:      "http://proxy/",
			proxyServerAuthField:  "proxyuser",
			proxyServerCredsField: "proxycreds",
			tlsPinningField:       tlsPinningSystemOpt,
			cliWriteField:         "false",
			netLatencyField:       "157ms",
		}, pathBackendConfigFields)

	lr, err := parseBackEndConfigurationFunc(nil, reqCtx)

	assert.Nil(t, lr)
	assert.Nil(t, err)

	cfg := container.GetBackendConfiguration()

	assert.Equal(t, "abc", string(cfg.OAEPLabel))
	assert.Equal(t, "http://proxy/", cfg.ProxyServerURL().String())
	assert.Equal(t, "proxyuser", cfg.ProxyServerAuth)
	assert.Equal(t, "proxycreds", cfg.ProxyServerCreds)

	assert.Equal(t, TLSPinningSystem, cfg.TLSPinning)
	assert.False(t, cfg.CLIWriteEnabled)
	assert.Equal(t, 157, cfg.NetworkLatency)
}

func TestParseBackEndConfigurationFunc_MalformedProxyURL(t *testing.T) {
	container := BackendConfigurationContainer{}
	reqCtx := setupConfigRequestMockWithData[BackendConfigurationContext](&container,
		map[string]interface{}{
			proxyServerField: "./this is not a proxy server url",
		}, pathBackendConfigFields)

	lr, err := parseBackEndConfigurationFunc(nil, reqCtx)

	assert.NotNil(t, lr)
	assert.Equal(t, "incorrect input: proxy server URL does not contain a host name", lr.Error().Error())
	assert.Nil(t, err)
}

func TestParseBackEndConfigurationFunc_MalformedDuration(t *testing.T) {
	container := BackendConfigurationContainer{}
	reqCtx := setupConfigRequestMockWithData[BackendConfigurationContext](&container,
		map[string]interface{}{
			netLatencyField: "this is not a network duration",
		}, pathBackendConfigFields)

	lr, err := parseBackEndConfigurationFunc(nil, reqCtx)

	assert.NotNil(t, lr)
	assert.Equal(t, "incorrect input: time: invalid duration \"this is not a network duration\"", lr.Error().Error())
	assert.Nil(t, err)
}

func TestParseBackEndConfigurationFunc_DefaultPinning(t *testing.T) {
	container := BackendConfigurationContainer{}
	reqCtx := setupConfigRequestMockWithData[BackendConfigurationContext](&container,
		map[string]interface{}{
			tlsPinningField: tlsPinningDefaultOpt,
		}, pathBackendConfigFields)

	lr, err := parseBackEndConfigurationFunc(nil, reqCtx)

	assert.Nil(t, lr)
	assert.Equal(t, TLSPinningDefault, container.GetBackendConfiguration().TLSPinning)
	assert.Nil(t, err)
}

func TestParseBackEndConfigurationFunc_CustomPinning(t *testing.T) {
	container := BackendConfigurationContainer{}
	reqCtx := setupConfigRequestMockWithData[BackendConfigurationContext](&container,
		map[string]interface{}{
			tlsPinningField: tlsPinningCustomOpt,
		}, pathBackendConfigFields)

	lr, err := parseBackEndConfigurationFunc(nil, reqCtx)

	assert.Nil(t, lr)
	assert.Equal(t, TLSPinningCustom, container.GetBackendConfiguration().TLSPinning)
	assert.Nil(t, err)
}

func TestParseCertificatePinConfiguration(t *testing.T) {
	container := TLSPinningContainer{
		TLSCertificatePinningContainer: TLSCertificatePinningContainer{
			pin: &transport.TLSCertChainPin{},
		},
	}

	reqCtx := setupConfigRequestMockWithData[TLSPinningOperations](&container,
		map[string]interface{}{
			certCommonNameField:   "cn",
			certSerialNumberField: "aa:bb:cc",
			certFingerprintField:  "aa:bb:cc",
		}, pathCertConfigFields)

	lr, err := parseCertificatePinConfiguration(nil, reqCtx)

	assert.Nil(t, lr)

	pin := container.GetPinning()
	assert.Equal(t, "cn", pin.CommonName)
	assert.Equal(t, []byte{0xaa, 0xbb, 0xcc}, pin.SerialNumber)
	assert.Equal(t, []byte{0xaa, 0xbb, 0xcc}, pin.Fingerprint)

	assert.Nil(t, err)
}

func TestParseCertificatePinConfiguration_FailOnInvalidHex_InFingerprint(t *testing.T) {
	container := TLSPinningContainer{
		TLSCertificatePinningContainer: TLSCertificatePinningContainer{
			pin: &transport.TLSCertChainPin{},
		},
	}
	reqCtx := setupConfigRequestMockWithData[TLSPinningOperations](&container,
		map[string]interface{}{
			certCommonNameField:   "cn",
			certSerialNumberField: "aa:bb:cc",
			certFingerprintField:  "aa.bb.cc",
		}, pathCertConfigFields)

	lr, err := parseCertificatePinConfiguration(nil, reqCtx)

	assert.NotNil(t, lr)
	assert.Equal(t, "invalid request due to 1 parse errors, first error encoding/hex: invalid byte: U+002E '.'", lr.Error().Error())

	assert.Nil(t, err)
}

func TestParseCertificatePinConfiguration_FailOnInvalidHex_InSerialNumber(t *testing.T) {
	container := TLSPinningContainer{
		TLSCertificatePinningContainer: TLSCertificatePinningContainer{
			pin: &transport.TLSCertChainPin{},
		},
	}

	reqCtx := setupConfigRequestMockWithData[TLSPinningOperations](&container,
		map[string]interface{}{
			certCommonNameField:   "cn",
			certSerialNumberField: "aa.bb.cc",
			certFingerprintField:  "aa:bb:cc",
		}, pathCertConfigFields)

	lr, err := parseCertificatePinConfiguration(nil, reqCtx)

	assert.NotNil(t, lr)
	assert.Equal(t, "invalid request due to 1 parse errors, first error encoding/hex: invalid byte: U+002E '.'", lr.Error().Error())

	assert.Nil(t, err)
}
