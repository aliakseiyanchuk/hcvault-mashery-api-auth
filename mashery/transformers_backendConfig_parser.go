package mashery

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/logical"
	"net/url"
	"strings"
	"time"
)

func parseCertificatePinConfiguration(_ context.Context, reqCtx *RequestHandlerContext[TLSPinningOperations]) (*logical.Response, error) {
	d := reqCtx.data
	var parseErrors []error

	pinCfg := reqCtx.heap.GetPinning()
	if pinCfg == nil {
		return nil, errors.New("tls pin hasn't been initialized")
	}

	copyStringFieldIfDefined(d, certCommonNameField, &pinCfg.CommonName)
	if err := consumeStringFieldIfDefined(d, certSerialNumberField, pinCfg.SerialNumberFromHex); err != nil {
		parseErrors = append(parseErrors, err)
	}
	if err := consumeStringFieldIfDefined(d, certFingerprintField, pinCfg.FingerprintFrom); err != nil {
		parseErrors = append(parseErrors, err)
	}

	if len(parseErrors) > 0 {
		return logical.ErrorResponse("invalid request due to %d parse errors, first error %s", len(parseErrors), parseErrors[0].Error()), nil
	} else {
		return nil, nil
	}
}

func parseBackEndConfigurationFunc(_ context.Context, reqCtx *RequestHandlerContext[BackendConfigurationContext]) (*logical.Response, error) {
	d := reqCtx.data

	be := reqCtx.heap.GetBackendConfiguration()

	if v, ok := d.GetOk(oaepLabelField); ok {
		be.OAEPLabel = []byte(v.(string))
	}

	var parseErrors []error

	if v, ok := d.GetOk(proxyServerField); ok {
		cfgStr := v.(string)
		if proxySrv, err := url.Parse(cfgStr); err != nil {
			parseErrors = append(parseErrors, errwrap.Wrapf("illegal value for proxy URL: {{err}}", err))
		} else if len(proxySrv.Hostname()) == 0 {
			parseErrors = append(parseErrors, errors.New("proxy server URL does not contain a host name"))
		} else {
			be.ProxyServer = cfgStr
		}
	}
	copyStringFieldIfDefined(d, proxyServerAuthField, &be.ProxyServerAuth)
	copyStringFieldIfDefined(d, proxyServerCredsField, &be.ProxyServerCreds)
	copyStringFieldIfDefined(d, rootCAField, &be.TLSCerts)

	if v, ok := d.GetOk(tlsPinningField); ok {
		switch strings.ToLower(v.(string)) {
		case tlsPinningDefaultOpt:
			be.TLSPinning = TLSPinningDefault
			break
		case tlsPinningSystemOpt:
			be.TLSPinning = TLSPinningSystem
			break
		case tlsPinningCustomOpt:
			be.TLSPinning = TLSPinningCustom
			break
		case tlsPinningInsecureOpt:
			be.TLSPinning = TLSPinningInsecure
			break
		default:
			parseErrors = append(parseErrors, errors.New(fmt.Sprintf("unsupported tls pinning: %s", v)))
		}
	}

	copyBooleanFieldIfDefined(d, cliWriteField, &be.CLIWriteEnabled)

	if v, ok := d.GetOk(netLatencyField); ok {
		latExp := v.(string)
		if dur, err := time.ParseDuration(latExp); err != nil {
			parseErrors = append(parseErrors, err)
		} else {
			be.NetworkLatency = int(dur.Milliseconds())
		}
	}

	if len(parseErrors) > 0 {
		errMsgs := make([]string, len(parseErrors))
		for k, v := range parseErrors {
			errMsgs[k] = v.Error()
		}

		return logical.ErrorResponse("incorrect input: %s", strings.Join(errMsgs, ", ")), nil
	} else {
		return nil, nil
	}
}
