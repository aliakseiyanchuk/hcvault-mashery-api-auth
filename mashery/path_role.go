package mashery

import (
	"context"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	defaultQPSValue   = 2
	roleName          = "roleName"
	pathAreasHelpSyn  = "Store/see Mashery credentials"
	pathAreasHelpDesc = `
The path is write-only storage of Mashery credentials required to obtain the V2/V3 authentication tokens. This path 
is used first before authentication tokens can be retrieved. That path accepts configuration for both V2 and V3
Mashery API. The user is recommended to always follow the least-required principle and supply only fields that are 
require for intended authentication methods.

An organization may operate multiple Mashery package Keys that would be used for various purposes. Typically, these
are:
- Keys for testing purpose. These Keys have relatively low qps and daily quota;
- Deployment pipeline Keys. These Keys would have relatively low qps and rather high daily quota; and
- OAuth server Keys. These Keys would have high qps as well as high daily quota that is commensurate with the number
  of access tokens created by the OAuth server.

Mashery credential logic names should be descriptive, e.g. test, production, or test-oauth-server, 
prod-ci_cd-pipeline, etc. The actual tooling will need thus to refer only to the logical name of this site to retrieve 
access credentials.`
)

var pathRoleFields = map[string]*framework.FieldSchema{
	roleName: {
		Type:        framework.TypeString,
		Description: "Role name",
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "Area's logical name",
		},
		Required: true,
	},
	roleAreaIdField: {
		Type:        framework.TypeString,
		Description: "Mashery Area UUID. Required for V3 credentials",
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "Area UUID",
		},
	},
	roleAreaNidField: {
		Type:        framework.TypeInt,
		Description: "Mashery Area Numeric ID. Required for V2 credentials",
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "Area NID",
		},
	},
	roleApiKeField: {
		Type:        framework.TypeString,
		Description: "Mashery API Key. Required for both V2 and V3 credentials",
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "API Key",
			Sensitive: true,
		},
	},
	roleSecretField: {
		Type:        framework.TypeString,
		Description: "Mashery API Key Secret. Required for both V2 and V3 credentials",
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "API Key Secret",
			Sensitive: true,
		},
	},
	roleUsernameField: {
		Type:        framework.TypeString,
		Description: "Mashery V3 API User. Required for V3 credentials",
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Mashery user",
			Sensitive: true,
		},
	},
	rolePasswordField: {
		Type:        framework.TypeString,
		Description: "Mashery V3 API password. Required for V3 credentials",
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Mashery user password",
			Sensitive: true,
		},
	},
	roleQpsField: {
		Type:        framework.TypeInt,
		Description: "Maximum QPS this key can make. Recommended for all methods; defaults to 2",
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "Maximum V3 QPS",
		},
		Default: 2,
	},
}

// pathRole creates the process for the roles/{roleName} path supporting the "push"-mode credentials storage
func pathRole(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName),
		Fields:  pathRoleFields,

		ExistenceCheck: b.roleExistenceCheck,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.handleWriteRoleKeys,
				Summary:  "Store Mashery area authentication data",
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.handleReadRoleData,
				Summary:  "Read Mashery area authentication data",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleUpdateRoleData,
				Summary:  "Update Mashery area authentication data",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.handleDeleteRoleData,
				Summary:  "Delete area authentication data",
			},
		},
		HelpSynopsis:    pathAreasHelpSyn,
		HelpDescription: pathAreasHelpDesc,
	}
}

func (b *AuthPlugin) roleExistenceCheck(ctx context.Context,
	req *logical.Request,
	data *framework.FieldData) (bool, error) {

	return b.checkObjectExistsInStorage(ctx,
		req,
		b.storagePathForRole(data)+storedRoleKeyPathSuffix)
}

func (b *AuthPlugin) handleWriteRoleKeys(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	chain := SimpleChain(
		readRole[RoleContext](false),
		blockOperationOnImportedRole[RoleContext],
		updateRoleKeysFromRequest,
		saveRoleKeys[RoleContext],
		setInitialRoleUsage[RoleContext],
		saveRoleUsage[RoleContext],
	)

	return handleRoleBoundOperation(ctx, b, req, data, chain)
}

func (b *AuthPlugin) handleReadRoleData(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	chain := SimpleChain(
		readRole[RoleContext](true),
		renderRole)

	return handleRoleBoundOperation(ctx, b, req, data, chain)
}

func (b *AuthPlugin) handleUpdateRoleData(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	chain := SimpleChain(
		readRole[RoleContext](true),
		blockOperationOnImportedRole[RoleContext],
		updateRoleKeysFromRequest,
		saveRoleKeys[RoleContext],
		// no Usage reset
	)

	return handleRoleBoundOperation(ctx, b, req, data, chain)
}

func (b *AuthPlugin) handleDeleteRoleData(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleRoot := b.storagePathForRole(data)
	if err := req.Storage.Delete(ctx, roleRoot+storedRoleUsageKeyPathSuffix); err != nil {
		return nil, errwrap.Wrapf("failed to delete role Usage: {{err}}", err)
	} else if err := req.Storage.Delete(ctx, roleRoot+storedRolePrivateKeyPathSuffix); err != nil {
		return nil, errwrap.Wrapf("failed to delete role private key: {{err}}", err)
	} else if err := req.Storage.Delete(ctx, roleRoot+storedRoleKeyPathSuffix); err != nil {
		return nil, errwrap.Wrapf("failed to delete role key data: {{err}}", err)
	}

	return nil, nil
}
