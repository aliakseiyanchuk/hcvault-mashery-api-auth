package mashery

import (
	"context"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/v2client"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	v2MethodField = "method"
	v2Query       = "query"
	v2Params      = "params"

	helpSynRoleV2  = "Execute V2 query"
	helpDescRoleV2 = `
Execute V2 query using Vault CLI and return the result to the administrator. The method will automatically
authenticate to Mashery API using the credentials of this role.

To execute V2 commands, the role must be V2-capable.

** The output of this path will differ from Mashery. For programmatic access. use ./proxy/v2 path **
`
)

var pathRoleV2Fields = map[string]*framework.FieldSchema{
	roleName: {
		Type:        framework.TypeString,
		Description: "Role name",
		Required:    true,
	},
	v2MethodField: {
		Type:        framework.TypeString,
		Description: "V2 Method",
		Required:    false,
	},
	v2Query: {
		Type:        framework.TypeString,
		Description: "Query string to be sent",
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "Query parameters",
		},
		Required: false,
	},
}

func pathRoleV2(b *AuthPlugin, pattern string) *framework.Path {
	return &framework.Path{
		Pattern: pattern,
		Fields:  pathRoleV2Fields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.executeV2Write,
				Summary:  "Execute GET method on V3 API for this role",
			},
		},

		ExistenceCheck:  doesNotExist,
		HelpSynopsis:    helpSynRoleV2,
		HelpDescription: helpDescRoleV2,
	}
}

func doesNotExist(ctx context.Context, request *logical.Request, data *framework.FieldData) (bool, error) {
	return false, nil
}

func alwaysExist(ctx context.Context, request *logical.Request, data *framework.FieldData) (bool, error) {
	return true, nil
}

func (b *AuthPlugin) executeV2Write(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	v2Request, err := verifyVaultV2OperationRequest(req, d)
	if err != nil {
		return logical.ErrorResponse("invalid V2 request: %s", err), nil
	}

	var container APIResponseContext[v2client.V2Result] = &APIResponseContainer[v2client.V2Result]{}

	baseChain := SimpleChain(
		readRole[APIResponseContext[v2client.V2Result]](true),
		blockUsageExceedingLimits[APIResponseContext[v2client.V2Result]],
		allowOnlyV2CapableRole[APIResponseContext[v2client.V2Result]],
		decreaseRemainingUsageQuota[APIResponseContext[v2client.V2Result]],
		executeV2CallUsing(v2Request),
		renderV2Response,
	)

	return handleOperationWithContainer(ctx, b, req, d, container, b.storagePathForRole(d), baseChain)
}
