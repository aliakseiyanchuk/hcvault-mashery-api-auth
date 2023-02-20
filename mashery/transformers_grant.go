package mashery

import (
	"context"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type GrantRequest struct {
	apiVersion int
	asLease    bool
}

func (gr GrantRequest) selectV2RenderingFunc() func(context.Context, *RequestHandlerContext[V2SignatureContext]) (*logical.Response, error) {
	if gr.asLease {
		return renderV2LeaseResponse
	} else {
		return renderV2PlainResponse
	}
}

func (gr GrantRequest) selectV3RenderingFunc() func(context.Context, *RequestHandlerContext[V3TokenContext]) (*logical.Response, error) {
	if gr.asLease {
		return renderV3LeaseResponse
	} else {
		return renderV3PlainResponse
	}
}

func readGrantRequestParams(data *framework.FieldData) (*logical.Response, GrantRequest) {
	rv := GrantRequest{
		apiVersion: 3,
		asLease:    false,
	}

	if v, ok := data.GetOk(grantApiVersionFieldName); ok {
		vApi := v.(int)
		if vApi >= 2 && vApi <= 3 {
			rv.apiVersion = vApi
		} else {
			return logical.ErrorResponse("unsupported api version: %d", vApi), rv
		}
	}

	if v, ok := data.GetOk(grantAsLeaseFieldName); ok {
		rv.asLease = v.(bool)
	}

	return nil, rv
}
