package mashery

import (
	"context"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	helpSynProxyV2  = "Proxy V2 requests"
	helpDescProxyV2 = `
Execute V2 request against Mashery V2 API, and return back the results to the calling application. The output
of the execution is idem Mashery V2 API response.

** This path is NOT compatible with Vault CLI command **

The path allows the organization/administrator to apply customized application authentication using 
Vault-provided auth methods and authorization using Vault policies.
`
)

func pathProxyV2(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/proxy/v2",
		Fields: map[string]*framework.FieldSchema{
			roleName: {
				Type:        framework.TypeString,
				Description: "Role name",
				Required:    true,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.proxyV2Request,
				Summary:  "Execute GET method on V3 API for this role",
			},
		},

		ExistenceCheck: doesNotExist,

		HelpSynopsis:    helpSynProxyV2,
		HelpDescription: helpDescProxyV2,
	}
}

func (b *AuthPlugin) proxyV2Request(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	v2Request, err := verifyVaultV2OperationRequest(req, d)
	if err != nil {
		return logical.ErrorResponse("invalid V2 request: %s", err), nil
	}

	sr := SimpleRunner[RoleContext]{}
	sr.Append(
		readRole[RoleContext](true),
		allowOnlyV2CapableRole[RoleContext],
		blockUsageExceedingLimits[RoleContext],
		decreaseRemainingUsageQuota[RoleContext],
	)

	mr := MappingRunner[RoleContext, WildcardAPIResponseContext]{
		parent:   &sr,
		exporter: func(WildcardAPIResponseContext) RoleContext { return &RoleContainer{} },
		importer: func(in RoleContext, out WildcardAPIResponseContext) {
			out.CarryRole(in.GetRole())
		},
	}

	mr.Append(
		executeV2CallToRawResponseUsing(v2Request),
		renderV2ProxiedResponse,
	)

	return handleWildcardAPIRoleBoundOperation(ctx, b, req, d, mr.Run)
}
