package mashery

import (
	"context"
	"errors"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"strings"
)

const (
	pathField         = "path"
	offsetField       = "offset"
	limitField        = "limit"
	selectFieldsField = "fields"
	filterField       = "filter"
	sortField         = "sort"

	respUnparsableBodyField = "unparsed_body"
	respTotalCountField     = "total_count"
)

const (
	methodPOST = iota
	methodPUT
)

const (
	helpSynRoleV3  = "Execute CRUD operation on Mashery V3 resource"
	helpDescRoleV3 = `
Execute V3 CRUD operation using  Vault CLI and return the result to the administrator. The method will automatically
authenticate to Mashery API using the credentials of this role.

To execute V3 commands, the role must be V3-capable. Typically, the secret engine will disable write-type operations
from the CLI. To use the CLI to create/update/delete operations, the CLI write must be enabled.

** The output of this path will differ from Mashery. For programmatic access. use ./proxy/v2 path **
`
)

type HttpStatus struct {
	Status     string              `json:"status"`
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
}

var v3PathFields map[string]*framework.FieldSchema

func init() {
	v3PathFields = map[string]*framework.FieldSchema{
		roleName: {
			Type:        framework.TypeString,
			Description: "Role name",
			Required:    true,
		},
		pathField: {
			Type:        framework.TypeString,
			Description: "V3 path to be invoked",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Path",
			},
			Required: true,
		},
		offsetField: {
			Type:        framework.TypeInt,
			Description: "Query offset (for list operations)",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Query offset",
			},
			Required: false,
			Default:  0,
		},
		limitField: {
			Type:        framework.TypeInt,
			Description: "Query limit (for list operations)",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Query limit",
			},
			Required: false,
			Default:  100,
		},
		selectFieldsField: {
			Type:        framework.TypeCommaStringSlice,
			Description: "Object fields to be selected",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Fields",
			},
			Required: false,
		},
		filterField: {
			Type:        framework.TypeCommaStringSlice,
			Description: "Filter objects (for list operations)",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Filter",
			},
			Required: false,
		},
		sortField: {
			Type:        framework.TypeCommaStringSlice,
			Description: "Sort objects (for list operations)",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Sort",
			},
			Required: false,
		},
		assumeObjectExist: {
			Type:        framework.TypeBool,
			Description: "Assume that target object exists and trigger update (rather than create) operation",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Assume object exists",
			},
			Required: false,
		},
	}
}

func pathRoleV3(b *AuthPlugin) *framework.Path {
	return &framework.Path{
		Pattern: "roles/" + framework.GenericNameRegex(roleName) + "/v3/" + framework.MatchAllRegex(pathField),
		Fields:  v3PathFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ListOperation: &framework.PathOperation{
				Callback: b.executeV3List,
				Summary:  "Execute GET method on V3 API for this role with query semantics",
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.executeV3Get,
				Summary:  "Execute GET method on V3 API for this role with a direct object reference semantic",
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.DoIfCLIWriteEnabled(func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.executeV3Write(ctx, request, data, methodPOST)
				}),
				Summary: "Execute POST method on V3 API for this role with a direct object reference semantic",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.DoIfCLIWriteEnabled(func(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
					return b.executeV3Write(ctx, request, data, methodPUT)
				}),
				Summary: "Execute PUT method on V3 API for this role with a direct object reference semantic",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.DoIfCLIWriteEnabled(b.executeV3Delete),
				Summary:  "Execute DElETE method on V3 API for this role",
			},
		},

		ExistenceCheck:  b.v3ObjectExists,
		HelpSynopsis:    helpSynRoleV3,
		HelpDescription: helpDescRoleV3,
	}
}

func (b *AuthPlugin) v3ObjectExists(ctx context.Context, req *logical.Request, d *framework.FieldData) (bool, error) {
	path := d.Get(pathField).(string)

	sr := SimpleRunner[RoleContext]{}
	sr.Append(
		readRole[RoleContext](true),
		blockUsageExceedingLimits[RoleContext],
		allowOnlyV3CapableRole[RoleContext],

		// Counter is not decreased at this point
	)

	mr := MappingRunner[RoleContext, WildcardAPIResponseContext]{
		parent:   &sr,
		exporter: func(responseContext WildcardAPIResponseContext) RoleContext { return &RoleContainer{} },
		importer: func(in RoleContext, out WildcardAPIResponseContext) { out.CarryRole(in.GetRole()) },
	}

	mr.Append(
		ensureAccessTokenValid,
		fetchV3Resource(path, nil),
		bounceErrorCodes,
	)

	var container WildcardAPIResponseContext
	container = &APIResponseContainer[*transport.WrappedResponse]{}

	if lr, err := handleOperationWithContainer(ctx, b, req, d, container, b.storagePathForRole(d), mr.Run); err != nil {
		return false, err
	} else if lr != nil {
		return false, errwrap.Wrapf("query rejected: %s", lr.Error())
	} else {
		return container.GetResponse().StatusCode <= 299, nil
	}
}

func (b *AuthPlugin) executeV3Write(ctx context.Context, req *logical.Request, d *framework.FieldData, meth int) (*logical.Response, error) {
	path := "/" + d.Get(pathField).(string)
	postObj := req.Data

	if postObj == nil || len(postObj) == 0 {
		return nil, errors.New("you need to post a JSON object on this path")
	}

	sr := makeBaseV3InvocationChain()
	sr.Append(
		writeToV3Resource(path, meth, postObj),
		bounceErrorCodes,
		renderV3SingleObjectResponse,
	)

	return handleWildcardAPIRoleBoundOperation(ctx, b, req, d, sr.Run)
}

func (b *AuthPlugin) executeV3Delete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	path := "/" + d.Get(pathField).(string)

	sr := makeBaseV3InvocationChain()
	sr.Append(
		deleteV3Resource(path),
		bounceErrorCodes,
		renderV3ResponseToEmpty,
	)

	return handleWildcardAPIRoleBoundOperation(ctx, b, req, d, sr.Run)
}

func (b *AuthPlugin) executeV3Get(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	path := "/" + d.Get(pathField).(string)
	vals := buildQueryString(d, selectFieldsField)

	renderingFunc := renderV3SingleObjectResponse

	if strings.HasSuffix(path, ";list") {
		path = strings.TrimSuffix(path, ";list")
		vals = buildQueryString(d, offsetField, limitField, selectFieldsField, filterField, sortField)
		renderingFunc = renderV3ArrayOfObjects
	} else if strings.HasSuffix(path, ";count") {
		path = strings.TrimSuffix(path, ";count")
		vals = buildQueryString(d, offsetField, filterField)
		renderingFunc = renderV3ObjectCountResponse
	}

	sr := makeBaseV3InvocationChain()
	sr.Append(
		fetchV3Resource(path, vals),
		bounceErrorCodes,
		renderingFunc,
	)

	return handleWildcardAPIRoleBoundOperation(ctx, b, req, d, sr.Run)
}

func makeBaseV3InvocationChain() SimpleRunner[WildcardAPIResponseContext] {
	rv := SimpleRunner[WildcardAPIResponseContext]{}
	rv.Append(
		readRole[WildcardAPIResponseContext](true),
		blockUsageExceedingLimits[WildcardAPIResponseContext],
		allowOnlyV3CapableRole[WildcardAPIResponseContext],
		ensureAccessTokenValid,
		decreaseRemainingUsageQuota[WildcardAPIResponseContext],
	)

	return rv
}

func (b *AuthPlugin) executeV3List(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	path := "/" + d.Get(pathField).(string)

	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	vals := buildQueryString(d, offsetField, limitField, selectFieldsField, filterField, sortField)

	sr := makeBaseV3InvocationChain()
	sr.Append(
		fetchV3Resource(path, vals),
		bounceErrorCodes,
		renderV3ListResponse,
	)

	return handleWildcardAPIRoleBoundOperation(ctx, b, req, d, sr.Run)
}
