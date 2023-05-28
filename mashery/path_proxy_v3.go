package mashery

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	ProxyMethodGet = iota
	ProxyMethodPost
	ProxyMethodPut
	ProxyMethodDelete
)

const assumeObjectExist = "assume_object_exists"

const proxyModeIndicatorHeader = "X-Proxy-Mode"
const proxyModeServerDateHeader = "X-Server-Date"

const (
	helpSynProxyV3  = "Proxy V3 requests"
	helpDescProxyV3 = `
Execute V3 request against Mashery V3 API, and return back the results to the calling application.

** This path is NOT compatible with Vault CLI command **

The path allows the organization/administrator to apply customized application authentication using 
Vault-provided auth methods and authorization using Vault policies.
`
)

func pathProxyV3(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/proxy/v3/" + framework.MatchAllRegex(pathField),
		Fields:  v3PathFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.proxyV3Request(ctx, request, data, ProxyMethodGet)
				},
				Summary: "Execute GET method on V3 API in proxy mode",
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.proxyV3Request(ctx, request, data, ProxyMethodPost)
				},
				Summary: "Execute POST method on V3 API in proxy mode",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.proxyV3Request(ctx, request, data, ProxyMethodPut)
				},
				Summary: "Execute PUT method on V3 API in proxy mode",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.proxyV3Request(ctx, request, data, ProxyMethodDelete)
				},
				Summary: "Execute DELETE method on V3 API in proxy mode",
			},
		},

		ExistenceCheck: b.proxyExistenceCheck,

		HelpSynopsis:    helpSynProxyV3,
		HelpDescription: helpDescProxyV3,
	}
}

func (b *AuthPlugin) proxyExistenceCheck(_ context.Context, req *logical.Request, fd *framework.FieldData) (bool, error) {
	if flg, ok := fd.GetOk(assumeObjectExist); ok {
		rv := flg.(bool)
		b.Logger().Trace("V3 proxy existence check: %b on explicit field request")
		return rv, nil
	} else if logical.UpdateOperation == req.Operation {
		b.Logger().Trace("V3 proxy existence check is true on update-type operation")
		return true, nil
	} else {
		b.Logger().Trace("V3 proxy existence check is false")
		return false, nil
	}
}

func (b *AuthPlugin) proxyV3Request(ctx context.Context, req *logical.Request, d *framework.FieldData, methSwitch int) (*logical.Response, error) {

	b.Logger().Trace("Executing V3 proxy with method switch", "method", methSwitch)

	path := "/" + d.Get(pathField).(string)
	vals := buildQueryString(d, offsetField, limitField, selectFieldsField, filterField, sortField)

	var fetchFunc TransformerFunc[WildcardAPIResponseContext]

	switch methSwitch {
	case ProxyMethodGet:
		b.Logger().Trace("Executing V3 GET proxy with")
		fetchFunc = fetchV3Resource(path, vals)
	case ProxyMethodPost:
		b.Logger().Trace("Executing V3 POST proxy with")
		fetchFunc = writeToV3Resource(path, methodPOST, req.Data)
	case ProxyMethodPut:
		b.Logger().Trace("Executing V3 PUT proxy with")
		fetchFunc = writeToV3Resource(path, methodPUT, req.Data)
	case ProxyMethodDelete:
		b.Logger().Trace("Executing V3 DELETE proxy with")
		fetchFunc = deleteV3Resource(path)
	default:
		return nil, errors.New(fmt.Sprintf("unrecognized method swtich: %d", methSwitch))
	}

	sr := makeBaseV3InvocationChain()
	sr.Append(
		fetchFunc,
		bounceErrorCodes,
		renderV3ProxiedResponse,
	)

	return handleWildcardAPIRoleBoundOperation(ctx, b, req, d, sr.Run)
}
