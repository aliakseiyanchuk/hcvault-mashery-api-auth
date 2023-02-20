package mashery

import (
	"context"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	helpSyncRolesRoot = "List configured roles"
	helpDescRolesRoot = `
The path supplies the list of roles that are configured within this secret engine. 
`
)

func pathRolesRoot(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/?",

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ListOperation: &framework.PathOperation{
				Callback: b.listRoles,
				Summary:  "List roles configured for this plugin",
			},
		},

		HelpSynopsis:    helpSyncRolesRoot,
		HelpDescription: helpDescRolesRoot,
	}
}

func (b *AuthPlugin) listRoles(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	if entities, err := req.Storage.List(ctx, b.rolesStorageRoot()); err != nil {
		return nil, err
	} else {
		return logical.ListResponse(entities), nil
	}
}
