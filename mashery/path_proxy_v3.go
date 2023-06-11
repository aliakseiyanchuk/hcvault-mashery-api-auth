package mashery

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"strings"
)

const (
	ProxyMethodGet = iota
	ProxyMethodPost
	ProxyMethodPut
	ProxyMethodDelete
)

const targetMethod = "target_method"

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

var optionalTargetMethodRegex string

func init() {
	optionalTargetMethodRegex = fmt.Sprintf("(;target-method-(?P<%s>.+))?", targetMethod)
}

// Proxy configuration for direct connection. In this scenario, the clients will have to supply an explicit
// suffix ;target-method-put to call the PUT-type operation
func pathProxyV3(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/proxy/v3/" + framework.MatchAllRegex(pathField) +
			optionalTargetMethodRegex,
		Fields: v3PathFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.proxyV3Request(ctx, request, data, ProxyMethodGet)
				},
				Summary: "Execute GET method on V3 API in proxy mode",
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.proxyV3Request(ctx, request, data, b.targetUpdateMethod(data))
				},
				Summary: "Execute POST method on V3 API in proxy mode",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.proxyV3Request(ctx, request, data, b.targetUpdateMethod(data))
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

		//ExistenceCheck: b.proxyExistenceCheck,

		HelpSynopsis:    helpSynProxyV3,
		HelpDescription: helpDescProxyV3,
	}
}

// Proxy path method suitable for deployments behind a TLS termination proxy that is capable of re-writing the
// path transparently from the client
func pathProxyV3WithExplicitMethod(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/proxy/v3-method/" + framework.GenericNameRegex(targetMethod) + "/" + framework.MatchAllRegex(pathField),
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
					return b.proxyV3Request(ctx, request, data, b.targetUpdateMethod(data))
				},
				Summary: "Execute POST method on V3 API in proxy mode",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.proxyV3Request(ctx, request, data, b.targetUpdateMethod(data))
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

		//ExistenceCheck: b.proxyExistenceCheck,

		HelpSynopsis:    helpSynProxyV3,
		HelpDescription: helpDescProxyV3,
	}
}

func (b *AuthPlugin) targetUpdateMethod(fd *framework.FieldData) int {
	if flg, ok := fd.GetOk(targetMethod); ok {
		rv := flg.(string)
		switch strings.ToLower(rv) {
		case "put":
			return ProxyMethodPut
		case "post":
			return ProxyMethodPost
		default:
			return ProxyMethodPost
		}
	} else {
		return ProxyMethodPost
	}
}

func (b *AuthPlugin) proxyV3Request(ctx context.Context, req *logical.Request, d *framework.FieldData, methSwitch int) (*logical.Response, error) {

	b.Logger().Trace("Executing V3 proxy with method switch", "method", methSwitch)

	path := "/" + d.Get(pathField).(string)
	vals := buildQueryString(d, offsetField, limitField, selectFieldsField, filterField, sortField)

	b.Logger().Trace("Executing V3 proxy with method switch",
		"path", path,
		"query string", vals,
		"method", methSwitch)

	var fetchFunc TransformerFunc[WildcardAPIResponseContext]

	switch methSwitch {
	case ProxyMethodGet:
		b.Logger().Trace("Executing V3 GET proxy")
		fetchFunc = b.fetchV3Resource(path, vals)
	case ProxyMethodPost:
		b.Logger().Trace("Executing V3 POST proxy")
		fetchFunc = b.writeToV3Resource(path, methodPOST, req.Data)
	case ProxyMethodPut:
		b.Logger().Trace("Executing V3 PUT proxy")
		fetchFunc = b.writeToV3Resource(path, methodPUT, req.Data)
	case ProxyMethodDelete:
		b.Logger().Trace("Executing V3 DELETE proxy")
		fetchFunc = b.deleteV3Resource(path)
	default:
		b.Logger().Error("cannot establish how to proxy this request: unsupported method switch")
		return nil, errors.New(fmt.Sprintf("unrecognized method swtich: %d", methSwitch))
	}

	sr := b.makeBaseV3InvocationChain()
	sr.Append(
		fetchFunc,
		// No error bouncing in proxy mode as the vault is performing only the authentication.
		renderV3ProxiedResponse,
	)

	return handleWildcardAPIRoleBoundOperation(ctx, b, req, d, sr.Run)
}
