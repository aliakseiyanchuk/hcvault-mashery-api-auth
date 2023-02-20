package mashery

import (
	"context"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	roleNamePEMHeader = "Role"

	pemCommonNameField   = "cn"
	pemContainerField    = "pem"
	explicitTermField    = "explicit_term"
	explicitNumUsesField = "explicit_num_uses"
	explicitQpsField     = "explicit_qps"
	onlyV2Field          = "v2_only"
	onlyV3Field          = "v3_only"
	forceProxyModeField  = "force_proxy_mode"
	exportableField      = "exportable"

	masheryRoleRecipientPEMBlockName = "MASHERY ROLE RECIPIENT"
	masheryRoleDataPEMBlockName      = "MASHERY ROLE DATA"

	helpSynRolePEM  = "Export role certificate"
	helpDescRolePEM = `
Retrieve the role certificate for encrypting the data exchanges of Mashery credentials in transit.
`
	helpSynRoleExport  = "Export encrypted role data"
	helpDescRoleExport = `
Export role data encrypted using the submitted public key. The output of this method can be transferred to
the original requester.
`
	helpSynRoleImport  = "Import encrypted role data"
	helpDescRoleImport = `
Import role data encrypted using the public key. 
`
)

var pathRolePemReadFields = map[string]*framework.FieldSchema{
	roleName: {
		Type:        framework.TypeString,
		Description: "Mashery area logical name",
	},
	pemCommonNameField: {
		Type:        framework.TypeString,
		Description: "Common name to specify in the request",
		Required:    false,
		Default:     "Bearer",
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "Common Name",
		},
	},
}

var pathRolePemImportFields = map[string]*framework.FieldSchema{
	roleName: {
		Type:        framework.TypeString,
		Description: "Role name",
		Required:    true,
	},
	pemContainerField: {
		Type:        framework.TypeString,
		Description: "PEM-encoded encrypted role data, obtained using ./export path",
		Required:    true,
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "PEM-encoded data intended for this role",
		},
	},
}

var pathRoleExportFields = map[string]*framework.FieldSchema{
	roleName: {
		Type:        framework.TypeString,
		Description: "Role name",
		Required:    true,
	},
	pemContainerField: {
		Type:        framework.TypeString,
		Description: "PEM-encoded certificate data, obtained from ./pem path",
		Required:    true,
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "PEM-encoded certificate of the recipient",
		},
	},
	explicitTermField: {
		Type:        framework.TypeString,
		Description: "The term that the recipient can use the exported data",
		Required:    false,
		DisplayAttrs: &framework.DisplayAttributes{
			Name: "Explicit term recipient can use this data",
		},
	},
	explicitNumUsesField: {
		Type:        framework.TypeInt,
		Description: "Number of times the recipient can use the exported data",
		Required:    false,
	},
	onlyV2Field: {
		Type:        framework.TypeBool,
		Description: "Disable V2 for the recipient",
		Required:    false,
	},
	onlyV3Field: {
		Type:        framework.TypeBool,
		Description: "Disable V3 for the recipient",
		Required:    false,
	},
	explicitQpsField: {
		Type:        framework.TypeInt,
		Description: "Specifies QPS to be granted at export, if needs to differ from the QPS stored with the role",
		Required:    false,
	},
	forceProxyModeField: {
		Type: framework.TypeBool,
		Description: `If set to true, will require recipient to work only in proxy mode. The recipient will not
							be able to retrieve the access credentials by value.`,
		Required: false,
	},
	exportableField: {
		Type:        framework.TypeBool,
		Description: "Allows the recipient to re-export the role further",
		Required:    false,
	},
}

func pathRoleImpExpGetPEM(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/pem",
		Fields:  pathRolePemReadFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathRolePEMRead,
				Summary:  "Retrieves the PEM certificate",
			},
		},

		ExistenceCheck: b.roleExistenceCheck,

		HelpSynopsis:    helpSynRolePEM,
		HelpDescription: helpDescRolePEM,
	}
}

func pathRoleImpExpExport(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/export",
		Fields:  pathRoleExportFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathRoleExport,
				Summary:  "Retrieves encrypted settings for the supplied certificate",
			},
		},

		ExistenceCheck: b.roleExistenceCheck,

		HelpSynopsis:    helpSynRoleExport,
		HelpDescription: helpDescRoleExport,
	}
}

func pathRoleImpExpImport(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/import",
		Fields:  pathRolePemImportFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: func(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
					return logical.ErrorResponse("importing data into non-existing role cannot possibly work"), nil
				},
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathRoleImport,
				Summary:  "Imports exported settings",
			},
		},

		ExistenceCheck: b.roleExistenceCheck,

		HelpSynopsis:    helpSynRoleImport,
		HelpDescription: helpDescRoleImport,
	}
}

// pathRoleExport Exports settings for secure exchange
func (b *AuthPlugin) pathRoleExport(ctx context.Context, req *logical.Request,
	d *framework.FieldData) (*logical.Response, error) {

	// Read the certificate and role
	readChain := SimpleChain(
		readRole[RoleExportContext](true),
		blockNonExportableRole,
		readRecipientCertificate,
		renderEncryptedRoleData,
	)

	var container RoleExportContext = &RoleExportContainer{}
	return handleOperationWithContainer(ctx, b, req, d, container, b.storagePathForRole(d), readChain)
}

func (b *AuthPlugin) pathRoleImport(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if pemBlock, pemErr := retrieveImportPEMBlockFromRequest(d); pemErr != nil {
		return logical.ErrorResponse("input does not contain a valid PEM block (%s)", pemErr.Error()), nil
	} else {
		chain := SimpleChain(
			readRole[RoleContext](true),
			retrievePrivateKey[RoleContext],
			importPEMEncodedExchangeData(pemBlock),
			saveRoleKeys[RoleContext],
			saveRoleUsage[RoleContext],
		)

		return handleRoleBoundOperation(ctx, b, req, d, chain)
	}
}

func (b *AuthPlugin) pathRolePEMRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	chain := SimpleChain(
		readRole[RoleContext](true),
		retrievePrivateKey[RoleContext],
		renderRoleCertificate,
	)

	return handleRoleBoundOperation(ctx, b, req, d, chain)
}
