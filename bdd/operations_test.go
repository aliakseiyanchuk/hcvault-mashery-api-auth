package bdd_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cucumber/godog"
	"github.com/hashicorp/vault/api"
	"reflect"
	"regexp"
	"strings"
	"yanchuk.nl/hcvault-mashery-api-auth/mashery"
)

func mountSecretEngineForScenario(s *godog.ScenarioContext, scenarioMountPoint string) {
	s.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		uCtx := context.WithValue(ctx, ctxKeyMountPointPath{}, scenarioMountPoint)

		// Clear any remaining points.
		_ = unmount(scenarioMountPoint)
		return uCtx, mount(scenarioMountPoint)
	})

	s.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		return ctx, unmount(scenarioMountPoint)
	})
}

func setupScenarioSteps(s *godog.ScenarioContext) {
	setupConfigSteps(s)

	setupRoleCRUDSteps(s)
	setupImportExportSteps(s)

	setupIOSteps(s)
}

func setupIOSteps(s *godog.ScenarioContext) {
	s.Step("^reading (.+) with query should fail due to:\\s{1,}(.+)$", cannotReadSecret)

	s.Step("^reading (.+) should fail due to:\\s{1,}(.+)$", func(ctx context.Context, path string, expl string) error {
		return cannotReadSecret(ctx, path, expl, nil)
	})

	s.Step("^writing to (.+) should fail due to: (.+)$", cannotWriteSecret)
	s.Step("^deleting (.+) should fail due to: (.+)$", cannotDeleteSecret)

	s.Step("^after reading (.+) role (\\d+) times$", func(ctx context.Context, roleName string, rpt int) error {
		for i := 0; i < rpt; i++ {
			if _, err := readRoleState(ctx, roleName); err != nil {
				return errors.New(fmt.Sprintf("failure %s at iteration %d", err.Error(), (i + 1)))
			}
		}
		return nil
	})

	s.Step("^invoking v2 (.+) \"(.+)\" for role (.+) should fail due to: (.+)$", func(ctx context.Context, method string, query string, roleName string, errMsg string) error {
		if _, err := invokeV2Query(ctx, method, query, roleName); err != nil {
			return checkAPIErrorContainsExplanation(err, errMsg)
		} else {
			return errors.New("invocation is successful, whereas it should have failed")
		}
	})
}

func invokeV2Query(ctx context.Context, method string, query string, role string) (context.Context, error) {
	path := mountPoint(ctx) + "/roles/" + role + "/v2"

	req := mashery.APIV2QueryRequest{
		Method: method,
		Query:  query,
	}

	if sec, err := vcl.Logical().Write(path, vaultAPIMap(req)); err != nil {
		return ctx, err
	} else {
		return context.WithValue(ctx, ctxKeyCurrentSecret{}, sec), nil
	}
}

func setupConfigSteps(s *godog.ScenarioContext) {
	s.Step("^remounted secret engine$", func(ctx context.Context) error {
		_ = unmount(mountPoint(ctx))
		return mount(mountPoint(ctx))
	})

	s.Step("^remounted secret engine configured with$", func(ctx context.Context, d *godog.Table) error {
		_ = unmount(mountPoint(ctx))
		if err := mount(mountPoint(ctx)); err != nil {
			return err
		}

		if apiCall, err := assist.CreateInstance(&mashery.APIConfigRequest{}, d); err != nil {
			return err
		} else if _, err := vcl.Logical().Write(mountPoint(ctx)+"/config", vaultAPIMap(apiCall)); err != nil {
			return err
		}

		return nil
	})

	s.Step("^configuration property (.+) reads (.+)$", assertConfigurationKey)
	s.Step("^configuration property (.+) matches (.+)$", assertConfigurationKeyMatchesPattern)
	s.Step("^configuration property (.+) is empty$", func(ctx context.Context, key string) error {
		return assertConfigurationKey(ctx, key, "")
	})
	s.Step("^network latency is (\\d+)$", func(ctx context.Context, val int64) error {
		return assertConfigurationIntKey(ctx, "net_latency", val)
	})
	s.Step("^cli write is enabled$", func(ctx context.Context) error {
		return assertConfigurationBoolKey(ctx, "enable_cli_v3_write", true)
	})
	s.Step("^cli write is disabled", func(ctx context.Context) error {
		return assertConfigurationBoolKey(ctx, "enable_cli_v3_write", false)
	})

	s.Step("^root CA is Google$", setRootCAToGoogle)
	s.Step("^(leaf|issuer|root) certificate is pinned with:$", pinCertificate)
	s.Step("^tls pinning set to (default|system|custom|insecure)$", setTLSPinning)
	s.Step("^effective tls pinning is (default|system|custom|insecure)$", func(ctx context.Context, expVal string) error {
		return assertConfigurationKey(ctx, "tls_pinning (effective)", expVal)
	})
	s.Step("^mashery (leaf|issuer|root) certificate is pinned as \"(.+)\"$", func(ctx context.Context, certType string, expVal string) error {
		return assertConfigurationKey(ctx, fmt.Sprintf("mashery %s cert", certType), expVal)
	})
	s.Step("^mashery (leaf|issuer|root) certificate is not pinned$", func(ctx context.Context, certType string) error {
		return assertConfigurationKey(ctx, fmt.Sprintf("mashery %s cert", certType), "")
	})

	s.Step("^after oaep label has been changed to (.+)$", changeOAEPLabel)
}

func assertConfigurationKey(ctx context.Context, key string, valExp string) error {
	if sec, err := vcl.Logical().Read(mountPoint(ctx) + "/config"); err != nil {
		return err
	} else if sec == nil {
		return errors.New("nil secret not expected at this point")
	} else {
		val := sec.Data[key]

		if val == nil {
			return errors.New(fmt.Sprintf("no such key in secret data: %s", key))
		} else if valTyped, ok := val.(string); !ok {
			return errors.New(fmt.Sprintf("key %s contains %s where string es expected", key, reflect.TypeOf(val).Name()))
		} else if valTyped != valExp {
			return errors.New(fmt.Sprintf("key %s contains unexpected value %s, expected was %s", key, valTyped, valExp))
		} else {
			return nil
		}
	}
}

func assertConfigurationKeyMatchesPattern(ctx context.Context, key string, valExpStr string) error {
	valExp := regexp.MustCompile(valExpStr)

	if sec, err := vcl.Logical().Read(mountPoint(ctx) + "/config"); err != nil {
		return err
	} else if sec == nil {
		return errors.New("nil secret not expected at this point")
	} else {
		val := sec.Data[key]

		if val == nil {
			return errors.New(fmt.Sprintf("no such key in secret data: %s", key))
		} else if valTyped, ok := val.(string); !ok {
			return errors.New(fmt.Sprintf("key %s contains %s where string es expected", key, reflect.TypeOf(val).Name()))
		} else if !valExp.MatchString(valTyped) {
			return errors.New(fmt.Sprintf("key %s contains unexpected value %s, expected was to match pattern %s", key, valTyped, valExp.String()))
		} else {
			return nil
		}
	}
}

func assertConfigurationIntKey(ctx context.Context, key string, exp int64) error {
	if sec, err := vcl.Logical().Read(mountPoint(ctx) + "/config"); err != nil {
		return err
	} else if sec == nil {
		return errors.New("nil secret not expected at this point")
	} else {
		val := sec.Data[key]

		if val == nil {
			return errors.New(fmt.Sprintf("no such key in secret data: %s", key))
		} else if valTyped, ok := val.(json.Number); !ok {
			return errors.New(fmt.Sprintf("key %s contains %s where json.Number es expected", key, reflect.TypeOf(val).Name()))
		} else {
			if n, err := valTyped.Int64(); err != nil {
				return err
			} else if n != exp {
				return errors.New(fmt.Sprintf("key %s contains unexpected value %d, expected was %d", key, n, exp))
			} else {
				return nil
			}
		}
	}
}

func assertConfigurationBoolKey(ctx context.Context, key string, exp bool) error {
	if sec, err := vcl.Logical().Read(mountPoint(ctx) + "/config"); err != nil {
		return err
	} else if sec == nil {
		return errors.New("nil secret not expected at this point")
	} else {
		val := sec.Data[key]

		if val == nil {
			return errors.New(fmt.Sprintf("no such key in secret data: %s", key))
		} else if valTyped, ok := val.(bool); !ok {
			return errors.New(fmt.Sprintf("key %s contains %s where bool es expected", key, reflect.TypeOf(val).Name()))
		} else {
			if valTyped != exp {
				return errors.New(fmt.Sprintf("key %s contains unexpected value %t, expected was %t", key, valTyped, exp))
			} else {
				return nil
			}
		}
	}
}

func setTLSPinning(ctx context.Context, vl string) error {
	req := mashery.APIConfigRequest{
		TLSPinning: vl,
	}
	_, err := vcl.Logical().Write(mountPoint(ctx)+"/config", vaultAPIMap(req))
	return err
}

func pinCertificate(ctx context.Context, certType string, d *godog.Table) error {
	if dRaw, err := assist.CreateInstance(&mashery.APICertificatePinnigRequest{}, d); err != nil {
		return err
	} else {
		_, err := vcl.Logical().Write(mountPoint(ctx)+"/config/certs/"+certType, vaultAPIMap(dRaw))
		return err
	}
}

func setRootCAToGoogle(ctx context.Context) error {
	param := map[string]interface{}{
		"root_ca": "-----BEGIN CERTIFICATE-----\nMIIFYjCCBEqgAwIBAgIQd70NbNs2+RrqIQ/E8FjTDTANBgkqhkiG9w0BAQsFADBX\nMQswCQYDVQQGEwJCRTEZMBcGA1UEChMQR2xvYmFsU2lnbiBudi1zYTEQMA4GA1UE\nCxMHUm9vdCBDQTEbMBkGA1UEAxMSR2xvYmFsU2lnbiBSb290IENBMB4XDTIwMDYx\nOTAwMDA0MloXDTI4MDEyODAwMDA0MlowRzELMAkGA1UEBhMCVVMxIjAgBgNVBAoT\nGUdvb2dsZSBUcnVzdCBTZXJ2aWNlcyBMTEMxFDASBgNVBAMTC0dUUyBSb290IFIx\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAthECix7joXebO9y/lD63\nladAPKH9gvl9MgaCcfb2jH/76Nu8ai6Xl6OMS/kr9rH5zoQdsfnFl97vufKj6bwS\niV6nqlKr+CMny6SxnGPb15l+8Ape62im9MZaRw1NEDPjTrETo8gYbEvs/AmQ351k\nKSUjB6G00j0uYODP0gmHu81I8E3CwnqIiru6z1kZ1q+PsAewnjHxgsHA3y6mbWwZ\nDrXYfiYaRQM9sHmklCitD38m5agI/pboPGiUU+6DOogrFZYJsuB6jC511pzrp1Zk\nj5ZPaK49l8KEj8C8QMALXL32h7M1bKwYUH+E4EzNktMg6TO8UpmvMrUpsyUqtEj5\ncuHKZPfmghCN6J3Cioj6OGaK/GP5Afl4/Xtcd/p2h/rs37EOeZVXtL0m79YB0esW\nCruOC7XFxYpVq9Os6pFLKcwZpDIlTirxZUTQAs6qzkm06p98g7BAe+dDq6dso499\niYH6TKX/1Y7DzkvgtdizjkXPdsDtQCv9Uw+wp9U7DbGKogPeMa3Md+pvez7W35Ei\nEua++tgy/BBjFFFy3l3WFpO9KWgz7zpm7AeKJt8T11dleCfeXkkUAKIAf5qoIbap\nsZWwpbkNFhHax2xIPEDgfg1azVY80ZcFuctL7TlLnMQ/0lUTbiSw1nH69MG6zO0b\n9f6BQdgAmD06yK56mDcYBZUCAwEAAaOCATgwggE0MA4GA1UdDwEB/wQEAwIBhjAP\nBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTkrysmcRorSCeFL1JmLO/wiRNxPjAf\nBgNVHSMEGDAWgBRge2YaRQ2XyolQL30EzTSo//z9SzBgBggrBgEFBQcBAQRUMFIw\nJQYIKwYBBQUHMAGGGWh0dHA6Ly9vY3NwLnBraS5nb29nL2dzcjEwKQYIKwYBBQUH\nMAKGHWh0dHA6Ly9wa2kuZ29vZy9nc3IxL2dzcjEuY3J0MDIGA1UdHwQrMCkwJ6Al\noCOGIWh0dHA6Ly9jcmwucGtpLmdvb2cvZ3NyMS9nc3IxLmNybDA7BgNVHSAENDAy\nMAgGBmeBDAECATAIBgZngQwBAgIwDQYLKwYBBAHWeQIFAwIwDQYLKwYBBAHWeQIF\nAwMwDQYJKoZIhvcNAQELBQADggEBADSkHrEoo9C0dhemMXoh6dFSPsjbdBZBiLg9\nNR3t5P+T4Vxfq7vqfM/b5A3Ri1fyJm9bvhdGaJQ3b2t6yMAYN/olUazsaL+yyEn9\nWprKASOshIArAoyZl+tJaox118fessmXn1hIVw41oeQa1v1vg4Fv74zPl6/AhSrw\n9U5pCZEt4Wi4wStz6dTZ/CLANx8LZh1J7QJVj2fhMtfTJr9w4z30Z209fOU0iOMy\n+qduBmpvvYuR7hZL6Dupszfnw0Skfths18dG9ZKb59UhvmaSGZRVbNQpsg3BZlvi\nd0lIKO2d1xozclOzgjXPYovJJIultzkMu34qQb9Sz/yilrbCgj8=\n-----END CERTIFICATE-----",
	}
	_, err := vcl.Logical().Write(mountPoint(ctx)+"/config", param)
	return err
}

func setupRoleCRUDSteps(s *godog.ScenarioContext) {
	s.Step("^role (.+) configured with:$", setRoleConfiguration)
	s.Step("^empty role (.+)$", func(ctx context.Context, name string) error { return setRoleConfiguration(ctx, name, nil) })
	s.Step("^role (.+) current state:$", readRoleState)

	s.Step("^- is exportable$", func(ctx context.Context) error {
		return assertCurrentSecretBoolEntry(ctx, "exportable", true)
	})
	s.Step("^- is not exportable$", func(ctx context.Context) error {
		return assertCurrentSecretBoolEntry(ctx, "exportable", false)
	})

	s.Step("^- forces proxy mode$", func(ctx context.Context) error {
		return assertCurrentSecretBoolEntry(ctx, "forced_proxy_mode", true)
	})
	s.Step("^- does not force proxy mode$", func(ctx context.Context) error {
		return assertCurrentSecretBoolEntry(ctx, "forced_proxy_mode", false)
	})

	s.Step("^- is V2-capable$", func(ctx context.Context) error {
		return assertCurrentSecretBoolEntry(ctx, "v2_capable", true)
	})
	s.Step("^- is not V2-capable$", func(ctx context.Context) error {
		return assertCurrentSecretBoolEntry(ctx, "v2_capable", false)
	})

	s.Step("^- is V3-capable$", func(ctx context.Context) error {
		return assertCurrentSecretBoolEntry(ctx, "v3_capable", true)
	})
	s.Step("^- is not V3-capable$", func(ctx context.Context) error {
		return assertCurrentSecretBoolEntry(ctx, "v3_capable", false)
	})

	s.Step("^- has indefinite term$", func(ctx context.Context) error {
		return assertCurrentSecretStringEntry(ctx, "term", "∞")
	})
	s.Step("^- has indefinite term remaining$", func(ctx context.Context) error {
		return assertCurrentSecretStringEntry(ctx, "term_remaining", "∞")
	})
	s.Step("^- has indefinite use remaining$", func(ctx context.Context) error {
		return assertCurrentSecretStringEntry(ctx, "use_remaining", "∞")
	})
	s.Step("^- is use-depleted$", func(ctx context.Context) error {
		return assertCurrentSecretStringEntry(ctx, "use_remaining", "---DEPLETED---")
	})
	s.Step("^- is expired$", func(ctx context.Context) error {
		return assertCurrentSecretStringEntry(ctx, "term_remaining", "---EXPIRED---")
	})

	s.Step("^- allows (\\d+) queries per second$", func(ctx context.Context, qps int64) error {
		return assertCurrentSecretIntEntry(ctx, "qps", qps)
	})
}

func assertCurrentSecretBoolEntry(ctx context.Context, key string, expValue bool) error {
	if sec := ctx.Value(ctxKeyCurrentSecret{}); sec == nil {
		return errors.New("no current secret available for this step")
	} else {
		secTyped := sec.(*api.Secret)
		if valRaw := secTyped.Data[key]; valRaw == nil {
			return errors.New(fmt.Sprintf("secret data does not contain key %s", key))
		} else if valTyped, ok := valRaw.(bool); !ok {
			return errors.New(fmt.Sprintf("secret data contains %s type for key %s, where bool was expcted", reflect.TypeOf(valRaw).String(), key))
		} else {
			if valTyped != expValue {
				return errors.New(fmt.Sprintf("expected %t, but %t was returned", expValue, valTyped))
			}
		}

		return nil
	}
}

func assertCurrentSecretStringEntry(ctx context.Context, key string, expValue string) error {
	if sec := ctx.Value(ctxKeyCurrentSecret{}); sec == nil {
		return errors.New("no current secret available for this step")
	} else {
		secTyped := sec.(*api.Secret)
		if valRaw := secTyped.Data[key]; valRaw == nil {
			return errors.New(fmt.Sprintf("secret data does not contain key %s", key))
		} else if valTyped, ok := valRaw.(string); !ok {
			return errors.New(fmt.Sprintf("secret data contains %s type for key %s, where string was expcted", reflect.TypeOf(valRaw).String(), key))
		} else {
			if valTyped != expValue {
				return errors.New(fmt.Sprintf("expected %s, but %s was returned", expValue, valTyped))
			}
		}

		return nil
	}
}

func assertCurrentSecretIntEntry(ctx context.Context, key string, expValue int64) error {
	if sec := ctx.Value(ctxKeyCurrentSecret{}); sec == nil {
		return errors.New("no current secret available for this step")
	} else {
		secTyped := sec.(*api.Secret)
		if valRaw := secTyped.Data[key]; valRaw == nil {
			return errors.New(fmt.Sprintf("secret data does not contain key %s", key))
		} else if valTyped, ok := valRaw.(json.Number); !ok {
			return errors.New(fmt.Sprintf("secret data contains %s type for key %s, where int was expcted", reflect.TypeOf(valRaw).String(), key))
		} else {
			i, _ := valTyped.Int64()
			if i != expValue {
				return errors.New(fmt.Sprintf("expected %d, but %d was returned", expValue, i))
			}
		}

		return nil
	}
}

func setupImportExportSteps(s *godog.ScenarioContext) {
	s.Step("^data export from role (.+) for role (.+) fails due to: (.+)$", func(ctx context.Context, from string, to string, expl string) error {
		if _, err := obtainRoleDataExport(ctx, from, to, nil); err != nil {
			return checkAPIErrorContainsExplanation(err, expl)
		} else {
			return errors.New("export succeeded where it should have failed")
		}
	})

	s.Step("^data export from role (.+) for role (.+) with:$", func(ctx context.Context, from string, to string, tbl *godog.Table) (context.Context, error) {
		return obtainRoleDataExport(ctx, from, to, tbl)
	})

	s.Step("^data export from role (.+) for role (.+)$", func(ctx context.Context, from string, to string) (context.Context, error) {
		return obtainRoleDataExport(ctx, from, to, nil)
	})

	s.Step("^is imported into role (.+)$", importExchangedData)

	s.Step("^cannot be imported for role (.+) explained as \"(.+)\"$", func(ctx context.Context, roleName string, expl string) error {
		err := importExchangedData(ctx, roleName)
		if err == nil {
			return errors.New("import succeeded where it should fail")
		} else {
			return checkAPIErrorContainsExplanation(err, expl)
		}
	})
}

func checkAPIErrorContainsExplanation(err error, expl string) error {
	apiErr := err.(*api.ResponseError)

	for _, chkErr := range apiErr.Errors {
		if strings.Index(chkErr, expl) >= 0 {
			return nil
		}
	}

	fmt.Println(fmt.Sprintf("no such message: %s", expl))
	for _, chkErr := range apiErr.Errors {
		fmt.Println(fmt.Sprintf("> %s", chkErr))
	}

	return errors.New("required error explanation was not found")
}

func changeOAEPLabel(ctx context.Context, newLabel string) error {
	upd := mashery.APIConfigRequest{
		OAEPLabel: newLabel,
	}
	_, err := vcl.Logical().Write(mountPoint(ctx)+"/config", vaultAPIMap(upd))
	return err
}

func importExchangedData(ctx context.Context, role string) error {
	req := mashery.APIRoleDataImportRequest{
		PEM: ctx.Value(ctxKeyExchangedPem{}).(string),
	}

	_, err := vcl.Logical().Write(mountPoint(ctx)+"/roles/"+role+"/import", vaultAPIMap(req))
	return err
}

func obtainRoleDataExport(ctx context.Context, sourceRole string, targetRole string, tbl *godog.Table) (context.Context, error) {
	sec, err := vcl.Logical().Read(mountPoint(ctx) + "/roles/" + targetRole + "/pem")
	if err != nil {
		return ctx, err
	} else if sec == nil {
		return ctx, errors.New("empty secret returned where it should contain data")
	}

	pem := sec.Data["pem"]
	if pem == nil {
		return ctx, errors.New("response from target does not contain PEM-encoded data")
	}

	var baseReq *mashery.APIRoleDataExportRequest
	if tbl != nil {
		if conv, err := assist.CreateInstance(&mashery.APIRoleDataExportRequest{}, tbl); err != nil {
			return ctx, err
		} else {
			baseReq = conv.(*mashery.APIRoleDataExportRequest)
		}
	} else {
		baseReq = &mashery.APIRoleDataExportRequest{}
	}
	baseReq.PEM = pem.(string)

	expSec, err := vcl.Logical().Write(mountPoint(ctx)+"/roles/"+sourceRole+"/export", vaultAPIMap(baseReq))
	if err != nil {
		return ctx, err
	} else {
		return context.WithValue(ctx, ctxKeyExchangedPem{}, expSec.Data["pem"]), nil
	}
}

func mountPoint(ctx context.Context) string {
	rv := "mash-auth"

	if val := ctx.Value(ctxKeyMountPointPath{}); val != nil {
		rv = val.(string)
	}

	return rv
}

func readRoleState(ctx context.Context, name string) (context.Context, error) {
	return readSecret(ctx, "/roles/"+name, nil)
}

func cannotWriteSecret(ctx context.Context, path string, expl string, d *godog.Table) error {
	_, err := writeSecret(ctx, path, d)
	if err == nil {
		return errors.New("write succeeded where it should have failed")
	} else {
		return checkAPIErrorContainsExplanation(err, expl)
	}
}

func cannotReadSecret(ctx context.Context, path string, expl string, d *godog.Table) error {
	_, err := readSecret(ctx, path, d)
	if err == nil {
		return errors.New("read succeeded where it should have failed")
	} else {
		return checkAPIErrorContainsExplanation(err, expl)
	}
}

func readSecret(ctx context.Context, path string, d *godog.Table) (context.Context, error) {
	var sec *api.Secret
	var err error

	if d == nil {
		actualPath := mountPoint(ctx) + path
		sec, err = vcl.Logical().Read(actualPath)
	} else {
		if conv, pErr := assist.ParseMap(d); pErr != nil {
			return ctx, pErr
		} else {
			actualPath := mountPoint(ctx) + path
			sec, err = vcl.Logical().ReadWithData(actualPath, vaultAPIQueryMap(conv))
		}
	}

	if err != nil {
		return nil, err
	} else {
		return context.WithValue(ctx, ctxKeyCurrentSecret{}, sec), nil
	}
}

func writeSecret(ctx context.Context, path string, d *godog.Table) (context.Context, error) {
	if writeData, err := assist.ParseMap(d); err != nil {
		return ctx, err
	} else {
		_, err := vcl.Logical().Write(mountPoint(ctx)+path, vaultAPIMap(writeData))
		return ctx, err
	}
}

func cannotDeleteSecret(ctx context.Context, path string, expl string) error {
	err := deleteSecret(ctx, path)
	if err == nil {
		return errors.New("delete succeeded where it should have failed")
	} else {
		return checkAPIErrorContainsExplanation(err, expl)
	}
}

func deleteSecret(ctx context.Context, path string) error {
	_, err := vcl.Logical().Delete(mountPoint(ctx) + path)
	return err
}

func setRoleConfiguration(ctx context.Context, name string, m *godog.Table) error {
	var data map[string]interface{}

	if m != nil {
		t := mashery.APICreateRoleRequest{}
		if apiCallData, err := assist.CreateInstance(&t, m); err != nil {
			return err
		} else {
			data = vaultAPIMap(apiCallData)
		}
	}

	_, err := vcl.Logical().Write(mountPoint(ctx)+"/roles/"+name, data)
	return err
}
