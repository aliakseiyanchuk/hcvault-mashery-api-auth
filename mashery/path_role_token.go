package mashery

import (
	"context"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/masherytypes"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"time"
)

func pathRoleToken(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/token",
		Fields: map[string]*framework.FieldSchema{
			roleName: {
				Type:        framework.TypeString,
				Description: "Role name",
				Required:    true,
			},
			grantAsLeaseFieldName: {
				Type:        framework.TypeBool,
				Description: "Whether to return this value as a lease",
				Default:     false,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.readRoleToken,
				Summary:  "Read a currently valid V3 access token",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.deleteToken,
				Summary:  "Deletes stored V3 access token",
			},
		},

		HelpSynopsis:    helpSynRoleToken,
		HelpDescription: helpDescRoleToken,

		ExistenceCheck: b.roleExistenceCheck,
	}
}

func (b *AuthPlugin) readRoleToken(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	lr, params := readGrantRequestParams(data)
	if lr != nil {
		return lr, nil
	}

	var container V3TokenContext
	container = &V3TokenContextContainer{}

	baseChecks := SimpleRunner[RoleContext]{}
	baseChecks.Append(
		readRole[RoleContext](true),
		blockOperationOnForceProxyRole[RoleContext],
		blockUsageExceedingLimits[RoleContext],
		blockRoleIncapableOf[RoleContext](3),
		decreaseRemainingUsageQuota[RoleContext],
		b.ensureAccessTokenValidWithRoleContext,
	)

	mr := mapRoleContextToV3TokenContext(&baseChecks)

	mr.Append(
		rehydrateV3AccessToken,
		params.selectV3RenderingFunc(),
	)

	return handleOperationWithContainer(ctx, b, request, data, container, b.storagePathForRole(data), mr.Run)
}

func rehydrateV3AccessToken(_ context.Context, reqCtx *RequestHandlerContext[V3TokenContext]) (*logical.Response, error) {
	role := reqCtx.heap.GetRole()

	now := time.Now()
	tkn := &masherytypes.TimedAccessTokenResponse{
		Obtained:   time.Unix(role.Usage.V3TokenObtained, 0),
		ServerTime: now,
		QPS:        role.Keys.MaxQPS,
		AccessTokenResponse: masherytypes.AccessTokenResponse{
			TokenType:   "Bearer",
			AccessToken: role.Usage.V3Token,
			ExpiresIn:   int(role.Usage.V3TokenExpiry - role.Usage.V3TokenObtained),
		},
	}
	reqCtx.heap.CarryV3TokenResponse(tkn)

	return nil, nil
}

func (b *AuthPlugin) deleteToken(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	chain := SimpleChain(
		readRole[RoleContext](true),
		forgetUsedToken,
		saveRoleUsage[RoleContext],
	)

	return handleRoleBoundOperation(ctx, b, request, data, chain)
}
