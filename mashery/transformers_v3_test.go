package mashery

import (
	"context"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/v3client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type FetchWithErrorHandlingMock struct {
	mock.Mock
}

func (dm *FetchWithErrorHandlingMock) MockTokenRefresh(ctx context.Context, reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) error {
	args := dm.Called(ctx, reqCtx)
	if args == nil {
		return nil
	}
	return args.Error(0)
}

func (dm *FetchWithErrorHandlingMock) FetchFunctionMock(ctx context.Context, client v3client.WildcardClient) (*transport.WrappedResponse, error) {
	args := dm.Called(ctx, client)
	if args == nil {
		return nil, nil
	}

	return args.Get(0).(*transport.WrappedResponse), args.Error(1)
}

func TestFetchWithErrorHandlingWillRetryAfterBadRequest(t *testing.T) {
	reqCtx := mockWildcardAPIRequestContext()

	dm := FetchWithErrorHandlingMock{}
	dm.On("MockTokenRefresh", mock.Anything, mock.Anything).
		Run(refreshAccessTokenTo("abc")).Return(nil).Once()

	dm.On("FetchFunctionMock", mock.MatchedBy(func(ctx context.Context) bool {
		return "abc" == v3client.AccessTokenFromContext(ctx)
	}), mock.Anything).Return(badRequestResponse(), nil).Once()

	dm.On("MockTokenRefresh", mock.Anything, mock.Anything).Return(nil).Once()

	dm.On("FetchFunctionMock", mock.MatchedBy(func(ctx context.Context) bool {
		return "abc" == v3client.AccessTokenFromContext(ctx)
	}), mock.Anything).Return(okRequestResponse(), nil).Once()

	b := AuthPlugin{
		v3Clients: map[string]V3ClientAndAuthorizer{},
	}
	wr, err := b.doFetchWithErrorHandling(context.TODO(), reqCtx, dm.MockTokenRefresh, dm.FetchFunctionMock)

	assert.Nil(t, err)
	assert.NotNil(t, wr)
	assert.Equal(t, 200, wr.StatusCode)

	dm.AssertExpectations(t)
}

func TestFetchWithErrorHandlingWillResetAccessTokenAfter403(t *testing.T) {
	reqCtx := mockWildcardAPIRequestContext()

	dm := FetchWithErrorHandlingMock{}
	dm.On("MockTokenRefresh", mock.Anything, mock.Anything).
		Run(refreshAccessTokenTo("abc")).
		Return(nil).Once()

	dm.On("FetchFunctionMock", mock.MatchedBy(func(ctx context.Context) bool {
		return "abc" == v3client.AccessTokenFromContext(ctx)
	}), mock.Anything).Return(invalidAccessTokenRequestResponse(), nil).Once()

	dm.On("MockTokenRefresh", mock.Anything, mock.MatchedBy(func(reqCtx *RequestHandlerContext[WildcardAPIResponseContext]) bool {
		return len(reqCtx.heap.GetRole().Usage.V3Token) == 0
	})).
		Run(refreshAccessTokenTo("def")).
		Return(nil).Once()

	dm.On("FetchFunctionMock", mock.MatchedBy(func(ctx context.Context) bool {
		return "def" == v3client.AccessTokenFromContext(ctx)
	}), mock.Anything).Return(okRequestResponse(), nil).Once()

	b := AuthPlugin{
		v3Clients: map[string]V3ClientAndAuthorizer{},
	}
	wr, err := b.doFetchWithErrorHandling(context.TODO(), reqCtx, dm.MockTokenRefresh, dm.FetchFunctionMock)

	assert.Nil(t, err)
	assert.NotNil(t, wr)
	assert.Equal(t, 200, wr.StatusCode)

	dm.AssertExpectations(t)
}

func TestFetchWithErrorHandlingWillReturnErrorOnDeniedAccess(t *testing.T) {
	reqCtx := mockWildcardAPIRequestContext()

	dm := FetchWithErrorHandlingMock{}
	dm.On("MockTokenRefresh", mock.Anything, mock.Anything).
		Run(refreshAccessTokenTo("abc")).
		Return(nil).Once()

	dm.On("FetchFunctionMock", mock.MatchedBy(func(ctx context.Context) bool {
		return "abc" == v3client.AccessTokenFromContext(ctx)
	}), mock.Anything).Return(accessDeniedAccessTokenRequestResponse(), nil).Once()

	b := AuthPlugin{
		v3Clients: map[string]V3ClientAndAuthorizer{},
	}
	wr, err := b.doFetchWithErrorHandling(context.TODO(), reqCtx, dm.MockTokenRefresh, dm.FetchFunctionMock)

	assert.NotNil(t, err)
	assert.Equal(t, "mashery denies access to the selected resource", err.Error())
	assert.Nil(t, wr)

	dm.AssertExpectations(t)
}

func refreshAccessTokenTo(val string) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		ctx := args.Get(1).(*RequestHandlerContext[WildcardAPIResponseContext])
		ctx.heap.GetRole().Usage.V3Token = val
		ctx.heap.GetRole().Usage.V3TokenExpiry = time.Now().Add(time.Hour).Unix()
	}
}

func badRequestResponse() *transport.WrappedResponse {
	rv := &transport.WrappedResponse{
		StatusCode: 400,
	}
	return rv
}

func invalidAccessTokenRequestResponse() *transport.WrappedResponse {
	rv := &transport.WrappedResponse{
		StatusCode: 403,
		Header: map[string][]string{
			"X-Mashery-Error-Code": []string{"ERR_403_DEVELOPER_INACTIVE"},
		},
	}
	return rv
}

func accessDeniedAccessTokenRequestResponse() *transport.WrappedResponse {
	rv := &transport.WrappedResponse{
		StatusCode: 403,
		Header: map[string][]string{
			"X-Mashery-Error-Code": []string{"ERR_403_NOT_AUTHORIZED"},
		},
	}
	return rv
}

func okRequestResponse() *transport.WrappedResponse {
	rv := &transport.WrappedResponse{
		StatusCode: 200,
	}
	return rv
}

func mockWildcardAPIRequestContext() *RequestHandlerContext[WildcardAPIResponseContext] {
	var container WildcardAPIResponseContext
	container = &APIResponseContainer[*transport.WrappedResponse]{
		RoleContainer: RoleContainer{
			role: &StoredRole{
				Name:  "Unit Test Role",
				Usage: StoredRoleUsage{},
			},
		},
	}

	reqCtx := &RequestHandlerContext[WildcardAPIResponseContext]{
		heap: container,
	}
	return reqCtx
}
