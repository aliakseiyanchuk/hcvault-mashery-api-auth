package mashery

import (
	"context"
	"errors"
	"fmt"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/v2client"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"time"
)

type V2SignatureContext interface {
	RoleContext
	GetV2Signature() string
	CarryV2Signature(string)
}

type V2SignatureContainer struct {
	RoleContainer
	sig string
}

func (c *V2SignatureContainer) GetV2Signature() string {
	return c.sig
}

func (c *V2SignatureContainer) CarryV2Signature(s string) {
	c.sig = s
}

func retrieveV2Signature(_ context.Context, reqCtx *RequestHandlerContext[V2SignatureContext]) (*logical.Response, error) {
	reqCtx.heap.CarryV2Signature(reqCtx.plugin.v2SignatureFor(reqCtx.heap.GetRole()))
	return nil, nil
}

func renderV2LeaseResponse(_ context.Context, reqCtx *RequestHandlerContext[V2SignatureContext]) (*logical.Response, error) {
	role := reqCtx.heap.GetRole()
	signature := reqCtx.heap.GetV2Signature()

	resp := reqCtx.plugin.Secret(secretMasheryV2Access).Response(map[string]interface{}{
		roleAreaNidField:        role.Keys.AreaNid,
		roleQpsField:            role.Keys.MaxQPS,
		roleApiKeField:          role.Keys.ApiKey,
		secretSignedSecretField: signature,
	}, map[string]interface{}{})

	resp.Secret.TTL = time.Minute * 1
	resp.Secret.MaxTTL = time.Minute * 2

	return resp, nil
}

func renderV2PlainResponse(_ context.Context, reqCtx *RequestHandlerContext[V2SignatureContext]) (*logical.Response, error) {
	role := reqCtx.heap.GetRole()
	signature := reqCtx.heap.GetV2Signature()

	resp := &logical.Response{
		Data: map[string]interface{}{
			roleAreaNidField:        role.Keys.AreaNid,
			roleQpsField:            role.Keys.MaxQPS,
			roleApiKeField:          role.Keys.ApiKey,
			secretSignedSecretField: signature,
		},
	}

	return resp, nil
}

func verifyVaultV2OperationRequest(req *logical.Request, d *framework.FieldData) (v2client.V2Request, error) {
	rv := v2client.V2Request{
		Id:      1,
		Method:  "object.query",
		Version: "2.0",
	}

	if v, ok := d.GetOk(v2MethodField); ok {
		rv.Method = v.(string)
	}

	if v, ok := d.GetOk(v2Query); ok {
		rv.Params = []interface{}{v}
	}

	dat := req.Data

	if dat[v2Params] != nil {
		rv.Params = dat[v2Params]
	}

	copyStringFieldIfDefined(d, "jsonrpc", &rv.Version)
	copyStringFieldIfDefined(d, "method", &rv.Method)
	copyIntFieldIfDefined(d, "id", &rv.Id)

	if rv.Params == nil {
		return rv, errors.New(fmt.Sprintf("input message does not contain `%s` or '%s' key", v2Params, v2Query))
	} else {
		return rv, nil
	}
}

type APIResponseContext[T any] interface {
	RoleContext
	GetResponse() T
	CarryAPIResponse(t T)
	GetMethod() string
	CarryMethod(t string)
}

type APIResponseContainer[T any] struct {
	RoleContainer
	response T
	method   string
}

func (arc *APIResponseContainer[T]) GetResponse() T {
	return arc.response
}

func (arc *APIResponseContainer[T]) CarryAPIResponse(t T) {
	arc.response = t
}

func (arc *APIResponseContainer[T]) GetMethod() string {
	return arc.method
}

func (arc *APIResponseContainer[T]) CarryMethod(t string) {
	arc.method = t
}

type WildcardAPIResponseContext APIResponseContext[*transport.WrappedResponse]

func executeV2CallToRawResponseUsing(v2Request v2client.V2Request) func(context.Context, *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
		cl := reqCtx.plugin.GetMasheryV2Client(reqCtx.heap.GetRole())
		resp, err := cl.GetRawResponse(ctx, v2Request)

		reqCtx.heap.CarryAPIResponse(resp)
		return nil, err
	}
}

func executeV2CallUsing(v2Request v2client.V2Request) func(context.Context, *RequestHandlerContext[APIResponseContext[v2client.V2Result]]) (*logical.Response, error) {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[APIResponseContext[v2client.V2Result]]) (*logical.Response, error) {
		cl := reqCtx.plugin.GetMasheryV2Client(reqCtx.heap.GetRole())
		resp, err := cl.InvokeDirect(ctx, v2Request)
		reqCtx.heap.CarryAPIResponse(resp)

		return nil, err
	}
}

func renderV2ProxiedResponse(_ context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	resp := reqCtx.heap.GetResponse()
	body, _ := resp.Body()

	lr := logical.Response{
		Data: map[string]interface{}{
			logical.HTTPRawBody:     body,
			logical.HTTPContentType: "application/json",
			logical.HTTPStatusCode:  resp.StatusCode,
		},
		Headers: map[string][]string{
			proxyModeIndicatorHeader: {pluginVersionL},
		},
	}

	appendXHeadersToResponse(resp, &lr)

	return &lr, nil
}

func renderV2Response(_ context.Context, reqCtx *RequestHandlerContext[APIResponseContext[v2client.V2Result]]) (*logical.Response, error) {
	resp := reqCtx.heap.GetResponse()

	lr := logical.Response{
		Data: map[string]interface{}{
			"result": resp.Result,
			"error":  resp.Error,
		},
	}

	return &lr, nil
}
