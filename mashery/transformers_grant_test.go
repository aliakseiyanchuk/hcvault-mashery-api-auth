package mashery

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadGrantRequestParams_SimpleV2(t *testing.T) {
	for i := 2; i <= 3; i++ {
		_, reqCtx := setupRoleRequestMockWithData(map[string]interface{}{
			grantApiVersionFieldName: i,
		}, pathRoleGrantFields)
		lr, gr := readGrantRequestParams(reqCtx.data)
		assert.Nil(t, lr)
		assert.Equal(t, i, gr.apiVersion)
		assert.False(t, gr.asLease)

		_, reqCtx = setupRoleRequestMockWithData(map[string]interface{}{
			grantApiVersionFieldName: i,
			grantAsLeaseFieldName:    true,
		}, pathRoleGrantFields)
		lr, gr = readGrantRequestParams(reqCtx.data)
		assert.Nil(t, lr)
		assert.Equal(t, i, gr.apiVersion)
		assert.True(t, gr.asLease)

		_, reqCtx = setupRoleRequestMockWithData(map[string]interface{}{
			grantApiVersionFieldName: i,
			grantAsLeaseFieldName:    false,
		}, pathRoleGrantFields)
		lr, gr = readGrantRequestParams(reqCtx.data)
		assert.Nil(t, lr)
		assert.Equal(t, i, gr.apiVersion)
		assert.False(t, gr.asLease)
	}
}

func TestReadGrantRequestParams_WillRejectUnkonwnAPIVersion(t *testing.T) {
	_, reqCtx := setupRoleRequestMockWithData(map[string]interface{}{
		grantApiVersionFieldName: 4,
	}, pathRoleGrantFields)
	lr, gr := readGrantRequestParams(reqCtx.data)
	assert.NotNil(t, lr)
	assert.Equal(t, "unsupported api version: 4", lr.Error().Error())
	assert.Equal(t, 3, gr.apiVersion)
	assert.False(t, gr.asLease)
}
