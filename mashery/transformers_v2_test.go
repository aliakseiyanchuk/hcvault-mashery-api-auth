package mashery

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerifyVaultV2OperationRequest_WillRejectNoParams(t *testing.T) {
	_, reqCtx := setupRoleRequestMockWithFields(pathRoleV2Fields)
	_, err := verifyVaultV2OperationRequest(reqCtx.request, reqCtx.data)
	assert.NotNil(t, err)
	assert.Equal(t, "input message does not contain `params` or 'query' key", err.Error())
}

func TestVerifyVaultV2OperationRequest_WhenReceivingQueryONly(t *testing.T) {
	_, reqCtx := setupRoleRequestMockWithData(map[string]interface{}{
		v2Query: "select * from applications",
	}, pathRoleV2Fields)
	rv, err := verifyVaultV2OperationRequest(reqCtx.request, reqCtx.data)
	assert.Nil(t, err)
	assert.Equal(t, "object.query", rv.Method)

	params := rv.Params.([]interface{})
	assert.Equal(t, 1, len(params))
	assert.Equal(t, "select * from applications", params[0])
}

func TestVerifyVaultV2OperationRequest_WhenPassingParams(t *testing.T) {
	_, reqCtx := setupRoleRequestMockWithData(map[string]interface{}{
		v2Params:      "select * from applications",
		v2MethodField: "applicaitons.list",
	}, pathRoleV2Fields)
	rv, err := verifyVaultV2OperationRequest(reqCtx.request, reqCtx.data)
	assert.Nil(t, err)
	assert.Equal(t, "applicaitons.list", rv.Method)

	assert.Equal(t, "select * from applications", rv.Params)
}
