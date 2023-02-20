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

		ExistenceCheck: doesNotExist,

		HelpSynopsis:    helpSynProxyV3,
		HelpDescription: helpDescProxyV3,
	}
}

func (b *AuthPlugin) proxyV3Request(ctx context.Context, req *logical.Request, d *framework.FieldData, methSwitch int) (*logical.Response, error) {
	path := "/" + d.Get(pathField).(string)
	vals := buildQueryString(d, offsetField, limitField, selectFieldsField, filterField, sortField)

	var fetchFunc TransformerFunc[WildcardAPIResponseContext]

	switch methSwitch {
	case ProxyMethodGet:
		fetchFunc = fetchV3Resource(path, vals)
	case ProxyMethodPost:
		fetchFunc = writeToV3Resource(path, methodPOST, req.Data)
	case ProxyMethodPut:
		fetchFunc = writeToV3Resource(path, methodPUT, req.Data)
	case ProxyMethodDelete:
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
