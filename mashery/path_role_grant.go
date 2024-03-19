package mashery

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/masherytypes"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"math"
	"time"
)

const (
	grantApiVersionFieldName = "api"
	grantAsLeaseFieldName    = "lease"

	secretMasheryV3Access = "v3_access"
	secretMasheryV2Access = "v2_access"

	helpSyncRoleGrant = "Retrieve and share Mashery API access credentials by value"
	helpDescRoleGrant = `
The path allows extracting the direct value of the Mashery V2 and/or V3 credentials that can be used in the
consuming applications.
`
	helpSynRoleToken  = "Retrieve a valid V3 access token that can be used for an immediate V3 API call"
	helpDescRoleToken = `
The path retrieves the V3 access token, requesting the new token if required. The principle difference with the
/grant API method is that the token obtained here is stored and reused for the subsequent calls, provided that 
the token is valid. The /grant API always returns a new token.
`
)

func v2AccessSecret() *framework.Secret {
	return &framework.Secret{
		Type: secretMasheryV2Access,
		Fields: map[string]*framework.FieldSchema{
			roleAreaNidField: {
				Type:        framework.TypeInt,
				Description: "Mashery Area Numeric Id",
			},
			roleApiKeField: {
				Type:        framework.TypeString,
				Description: "Mashery V2 API Key",
			},
			secretSignedSecretField: {
				Type:        framework.TypeString,
				Description: "Salted signed secret",
			},
			roleQpsField: {
				Type:        framework.TypeInt,
				Description: "Maximum QPS this key can achieve",
			},
		},
		DefaultDuration: time.Minute,
		Renew:           noopRenewRevoke,
		Revoke:          noopRenewRevoke,
	}
}

func v3AccessSecret() *framework.Secret {
	return &framework.Secret{
		Type: secretMasheryV3Access,
		Fields: map[string]*framework.FieldSchema{
			secretAccessToken: {
				Type:        framework.TypeString,
				Description: "Mashery V3 access token",
			},
			roleQpsField: {
				Type:        framework.TypeInt,
				Description: "Maximum QPS this token is granted",
			},
		},

		DefaultDuration: time.Minute * 15,

		// The access token lease is not revocable and not renewable, since the access token is cannot be revoked.
		// The noop revoke is supplied nevertheless to avoid excessive error logging
		Revoke: noopRenewRevoke,
		Renew:  noopRenewRevoke,
	}
}

var pathRoleGrantFields = map[string]*framework.FieldSchema{
	roleName: {
		Type:        framework.TypeString,
		Description: "Role name",
		Required:    true,
	},
	grantApiVersionFieldName: {
		Type: framework.TypeInt,
		Description: `Mashery API version, for which access credentials are needed. 
Possible values are 2 or 3 for V2 and V3 respectively`,
		Default:       3,
		AllowedValues: []interface{}{2, 3},
	},
	grantAsLeaseFieldName: {
		Type:        framework.TypeBool,
		Description: "Whether to return this value as a lease",
		Default:     false,
	},
}

func pathRoleGrant(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/grant",
		Fields:  pathRoleGrantFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.issueGrant,
				Summary:  "Retrieve credentials for Mashery API",
			},
		},

		ExistenceCheck: b.roleExistenceCheck,

		HelpSynopsis:    helpSyncRoleGrant,
		HelpDescription: helpDescRoleGrant,
	}
}

func mapRoleContextToV3TokenContext(baseChecks *SimpleRunner[RoleContext]) MappingRunner[RoleContext, V3TokenContext] {
	mr := MappingRunner[RoleContext, V3TokenContext]{
		parent:   baseChecks,
		exporter: func(in V3TokenContext) RoleContext { return &RoleContainer{} },
		importer: func(in RoleContext, out V3TokenContext) {
			out.CarryRole(in.GetRole())
		},
	}
	return mr
}

func (b *AuthPlugin) issueGrant(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {

	lr, params := readGrantRequestParams(data)
	if lr != nil {
		return lr, nil
	}

	baseChecks := SimpleRunner[RoleContext]{}
	baseChecks.Append(
		readRole[RoleContext](true),
		blockOperationOnForceProxyRole[RoleContext],
		blockUsageExceedingLimits[RoleContext],
		blockRoleIncapableOf[RoleContext](params.apiVersion),
		decreaseRemainingUsageQuota[RoleContext],
	)

	switch params.apiVersion {
	case 2:
		var container V2SignatureContext
		container = &V2SignatureContainer{}

		mr := MappingRunner[RoleContext, V2SignatureContext]{
			parent:   &baseChecks,
			exporter: func(in V2SignatureContext) RoleContext { return &RoleContainer{} },
			importer: func(in RoleContext, out V2SignatureContext) {
				out.CarryRole(in.GetRole())
			},
		}

		mr.Append(
			retrieveV2Signature,
			params.selectV2RenderingFunc(),
		)

		return handleOperationWithContainer(ctx, b, request, data, container, b.storagePathForRole(data), mr.Run)
	case 3:
		var container V3TokenContext
		container = &V3TokenContextContainer{}

		mr := mapRoleContextToV3TokenContext(&baseChecks)

		mr.Append(
			retrieveV3AccessToken,
			params.selectV3RenderingFunc(),
		)

		return handleOperationWithContainer(ctx, b, request, data, container, b.storagePathForRole(data), mr.Run)
	default:
		return logical.ErrorResponse("unsupported api version"), nil
	}
}

func (b *AuthPlugin) createV3LeasedResponse(tkn *masherytypes.TimedAccessTokenResponse, v3Rec *StoredRole) *logical.Response {
	exp := time.Now().Add(time.Second * time.Duration(tkn.ExpiresIn))

	b.Logger().Info("Maximum token expiry time", "exp", exp.Unix())

	v3Secret := b.Secret(secretMasheryV3Access)
	response := v3Secret.Response(map[string]interface{}{
		secretAccessToken:            tkn.AccessToken,
		secretAccessTokenExpiryTime:  tkn.ExpiryTime(),
		secretAccessTokenExpiryEpoch: tkn.ExpiryTime().Unix(),
		roleQpsField:                 v3Rec.Keys.MaxQPS,
	}, map[string]interface{}{
		secretInternalRoleStoragePath: v3Rec.StoragePath,
		secretInternalRefreshToken:    tkn.RefreshToken,
		secretInternalTokenExpiryTime: exp.Unix(),
	})

	b.Logger().Info(fmt.Sprintf("Usable token time in seconds: %d, based on %d seconds before exipry time", tkn.ExpiresIn, tkn.ExpiresIn))

	response.Secret.LeaseOptions.MaxTTL = time.Duration(math.Round(0.9 * float64(time.Second) * float64(time.Duration(tkn.ExpiresIn))))
	response.Secret.LeaseOptions.Increment = time.Minute * 15
	response.Secret.LeaseOptions.Renewable = true

	b.Logger().Info(fmt.Sprintf("Response TTL %s", response.Secret.LeaseOptions.TTL))
	b.Logger().Info(fmt.Sprintf("Response Max TTL %s", response.Secret.LeaseOptions.MaxTTL))

	return response
}

func (b *AuthPlugin) v2SignatureFor(role *StoredRole) string {
	now := time.Now()
	rawSig := fmt.Sprintf("%s%s%d", role.Keys.ApiKey, role.Keys.KeySecret, now.Unix())

	hash := md5.New()
	hash.Write([]byte(rawSig))
	signature := hex.EncodeToString(hash.Sum(nil))

	return signature
}
