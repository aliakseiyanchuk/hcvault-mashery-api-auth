package mashery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/masherytypes"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/v3client"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/logical"
	"net/url"
	"strconv"
	"strings"
)

type V3TokenContext interface {
	RoleContext
	GetV3TokenResponse() *masherytypes.TimedAccessTokenResponse
	CarryV3TokenResponse(*masherytypes.TimedAccessTokenResponse)
}

type V3TokenContextContainer struct {
	RoleContainer

	token *masherytypes.TimedAccessTokenResponse
}

func (c *V3TokenContextContainer) GetV3TokenResponse() *masherytypes.TimedAccessTokenResponse {
	return c.token
}

func (c *V3TokenContextContainer) CarryV3TokenResponse(s *masherytypes.TimedAccessTokenResponse) {
	c.token = s
}

func fetchV3Resource(path string, qs url.Values) func(context.Context, *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
		resp, err := fetchWithErrorHandling(ctx, reqCtx, reqCtx.heap.GetRole(), func(ctx context.Context, client v3client.WildcardClient) (*transport.WrappedResponse, error) {
			return client.FetchAny(ctx, path, &qs)
		})

		reqCtx.heap.CarryAPIResponse(resp)
		return nil, err
	}
}

func deleteV3Resource(path string) func(context.Context, *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
		resp, err := fetchWithErrorHandling(ctx, reqCtx, reqCtx.heap.GetRole(), func(ctx context.Context, client v3client.WildcardClient) (*transport.WrappedResponse, error) {
			return client.DeleteAny(ctx, path)
		})

		reqCtx.heap.CarryAPIResponse(resp)
		return nil, err
	}
}

func writeToV3Resource(path string, meth int, data map[string]interface{}) func(context.Context, *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
		resp, err := fetchWithErrorHandling(ctx, reqCtx, reqCtx.heap.GetRole(), func(ctx context.Context, client v3client.WildcardClient) (*transport.WrappedResponse, error) {
			switch meth {
			case methodPOST:
				return client.PostAny(ctx, path, data)
			case methodPUT:
				return client.PutAny(ctx, path, data)
			default:
				return client.PostAny(ctx, path, data)
			}
		})

		reqCtx.heap.CarryAPIResponse(resp)
		return nil, err
	}
}

func bounceErrorCodes(_ context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	resp := reqCtx.heap.GetResponse()
	bodyStr, err := resp.Body()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 403 {
		return logical.ErrorResponse("access to this resource is denied by Mashery", errors.New(string(bodyStr))), nil
	} else if resp.StatusCode == 404 {
		return logical.ErrorResponse("requested object is not found in Mashery", errors.New(string(bodyStr))), nil
	} else if resp.StatusCode > 299 {
		return logical.ErrorResponse("unsupported status code %d", resp.StatusCode), nil
	}

	return nil, nil
}

type HttpFetchFunction func(context.Context, v3client.WildcardClient) (*transport.WrappedResponse, error)

func fetchWithErrorHandling(ctx context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext], sr *StoredRole, fetchFunc HttpFetchFunction) (*transport.WrappedResponse, error) {
	for i := 0; i < 3; i++ {
		client := reqCtx.plugin.GetMasheryV3Client(sr)
		if resp, err := fetchFunc(ctx, client); err != nil {
			return nil, err
		} else if errCode := resp.Header.Get("X-Mashery-Error-Code"); errCode == "ERR_403_DEVELOPER_INACTIVE" {
			sr.Usage.ResetToken()
			if _, err = ensureAccessTokenValid(ctx, reqCtx); err != nil {
				return nil, err
			} else {
				continue
			}
		} else {
			return resp, err
		}
	}

	// Unreachable code under normal operation; would only be reachable  if the access tokens
	// are continuously lost on the server side.
	return nil, errors.New("resource is impossible to retrieve")
}

func renderV3ListResponse(_ context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	response := reqCtx.heap.GetResponse()
	if bodyStr, err := response.Body(); err != nil {
		return nil, err
	} else {
		var parsedJson []map[string]interface{}
		if err = json.Unmarshal(bodyStr, &parsedJson); err != nil {
			return nil, err
		}

		var ids []string
		var idsInfo = make(map[string]interface{}, 1)
		var warnings []string

		if err == nil {
			unidentifiedObjects := 0

			for k := range parsedJson {
				if v := parsedJson[k]["id"]; v != nil {
					id := fmt.Sprint(v)
					ids = append(ids, id)
					idsInfo[id] = parsedJson[k]
				} else {
					unidentifiedObjects++
				}
			}

			if unidentifiedObjects > 0 {
				warnings = append(warnings, fmt.Sprintf("there were %d unidentified objects in this response", unidentifiedObjects))
			}

		} else {
			warnings = append(warnings,
				err.Error(),
				"returned object is not a json array. If it is an object, try running `vault read` on this path instead",
			)
		}

		lr := logical.ListResponseWithInfo(ids, idsInfo)
		if parsedJson != nil {
			checkNeedCountWarning(response, parsedJson, lr)
		} else {
			lr.Data[respUnparsableBodyField] = string(bodyStr)
		}

		if len(warnings) > 0 {
			lr.Warnings = append(lr.Warnings, warnings...)
		}

		return lr, nil
	}
}

func checkNeedCountWarning(resp *transport.WrappedResponse, parsedJson []map[string]interface{}, lr *logical.Response) {
	cntHeader := resp.Header.Get("X-Total-Count")
	if len(cntHeader) > 0 {
		if iCnt, err := strconv.Atoi(cntHeader); err == nil {
			if iCnt > len(parsedJson) {
				lr.AddWarning(fmt.Sprintf("there are total %d objects while only %d are returned", iCnt, len(parsedJson)))
			}
		}
	}
}

func identify(obj interface{}) (string, interface{}) {
	if typedMap, ok := obj.(map[string]interface{}); ok {
		rawKey := typedMap["id"]
		if rawKey != nil {
			if typedKey, ok := rawKey.(string); ok {
				rawKey = typedKey

				var rvMap = make(map[string]interface{}, len(typedMap)-1)
				for k, v := range typedMap {
					if k != "id" {
						rvMap[k] = v
					}
				}

				return typedKey, rvMap
			}
		}
	}

	return "", nil
}

// TODO where is this lost?
func renderV3ArrayOfObjects(_ context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	response := reqCtx.heap.GetResponse()
	bodyStr, _ := response.Body()

	lr := &logical.Response{
		Data: map[string]interface{}{},
	}

	var parsedJsonArr []map[string]interface{}
	if jsonError := json.Unmarshal(bodyStr, &parsedJsonArr); jsonError == nil {
		for k := range parsedJsonArr {
			if key, desc := identify(parsedJsonArr[k]); len(key) > 0 {
				lr.Data[key] = desc
			}
		}

		checkNeedCountWarning(response, parsedJsonArr, lr)
	} else {
		lr.Warnings = append(lr.Warnings,
			jsonError.Error(),
			"returned data structure does not match array",
		)
		lr.Data[respUnparsableBodyField] = string(bodyStr)
	}

	return lr, nil
}

func renderV3ProxiedResponse(_ context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	response := reqCtx.heap.GetResponse()

	cType := response.Header.Get("Content-Type")
	if len(cType) == 0 {
		cType = "text/plain"
	}

	respBody, _ := response.Body()

	lr := logical.Response{
		Data: map[string]interface{}{
			logical.HTTPStatusCode:  response.StatusCode,
			logical.HTTPContentType: cType,
			logical.HTTPRawBody:     respBody,
		},
		Headers: map[string][]string{
			proxyModeIndicatorHeader: {pluginVersionL},
		},
	}

	appendXHeadersToResponse(response, &lr)
	return &lr, nil
}

func renderV3SingleObjectResponse(_ context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	response := reqCtx.heap.GetResponse()
	bodyStr, _ := response.Body()

	lr := &logical.Response{
		Data: map[string]interface{}{},
	}

	parsedJsonObj := map[string]interface{}{}
	if jsonError := json.Unmarshal(bodyStr, &parsedJsonObj); jsonError == nil {
		for k, v := range parsedJsonObj {
			lr.Data[k] = v
		}
	} else {
		lr.Warnings = append(lr.Warnings,
			jsonError.Error(),
			"returned data structure does not match object. Are you forgetting `;list` suffix",
		)
		lr.Data[respUnparsableBodyField] = string(bodyStr)
	}

	return lr, nil
}

func renderV3ObjectCountResponse(_ context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	response := reqCtx.heap.GetResponse()
	lr := &logical.Response{
		Data: map[string]interface{}{},
	}

	lr.Data[respTotalCountField] = -1

	cntHeader := response.Header.Get("X-Total-Count")
	if len(cntHeader) > 0 {
		if iCnt, err := strconv.Atoi(cntHeader); err == nil {
			lr.Data[respTotalCountField] = iCnt
		}
	}

	return lr, nil
}

func renderV3ResponseToEmpty(_ context.Context, _ *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	return nil, nil
}

func ensureAccessTokenValid(ctx context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) (*logical.Response, error) {
	role := reqCtx.heap.GetRole()
	if role.Usage.V3TokenNeedsRenew() {
		creds := role.asV3Credentials()

		if tkn, err := reqCtx.plugin.GetOAuthHelper().RetrieveAccessTokenFor(&creds); err != nil {
			return nil, err
		} else {
			role.Usage.V3Token = tkn.AccessToken
			role.Usage.V3TokenExpiry = tkn.ExpiryTime().Unix()

			if writeErr := reqCtx.WritePath(ctx, roleUsagePath(reqCtx), &role.Usage); writeErr != nil {
				return nil, writeErr
			}
		}
	}

	return nil, nil
}

func retrieveV3AccessToken(_ context.Context, reqCtx *RequestHandlerContext[V3TokenContext]) (*logical.Response, error) {
	role := reqCtx.heap.GetRole()

	v3Credentials := role.asV3Credentials()
	if tkn, err := reqCtx.plugin.GetOAuthHelper().RetrieveAccessTokenFor(&v3Credentials); err != nil {
		return nil, errwrap.Wrapf("access token was not granted by Mashery: {{err}}", err)
	} else if tkn.ServerTime.Unix() > 0 && role.Usage.AfterExpiryTerm(tkn.ServerTime) {
		return nil, errors.New("your system's clock is skewed. Mashery response is after expiry term of your role grant")
	} else {
		reqCtx.heap.CarryV3TokenResponse(tkn)
		return nil, nil
	}
}

func renderV3LeaseResponse(_ context.Context, reqCtx *RequestHandlerContext[V3TokenContext]) (*logical.Response, error) {
	return reqCtx.plugin.createV3LeasedResponse(reqCtx.heap.GetV3TokenResponse(), reqCtx.heap.GetRole()), nil
}

func renderV3PlainResponse(_ context.Context, reqCtx *RequestHandlerContext[V3TokenContext]) (*logical.Response, error) {
	token := reqCtx.heap.GetV3TokenResponse()
	if token == nil {
		return nil, errors.New("v3 response rendering called before the response is available")
	}
	rv := &logical.Response{
		Data: map[string]interface{}{
			secretAccessToken:           token.AccessToken,
			secretAccessTokenExpiryTime: token.ExpiryTime(),
			roleQpsField:                reqCtx.heap.GetRole().Keys.MaxQPS,
		},
	}

	return rv, nil
}

func appendXHeadersToResponse(resp *transport.WrappedResponse, lr *logical.Response) {
	if dateHdr := resp.Header.Get("Date"); len(dateHdr) > 0 {
		lr.Headers[proxyModeServerDateHeader] = []string{dateHdr}
	}

	for k, v := range resp.Header {
		if strings.HasPrefix(k, "X-") || strings.HasPrefix(k, "WWW-") {
			lr.Headers[k] = v
		}
	}
}
