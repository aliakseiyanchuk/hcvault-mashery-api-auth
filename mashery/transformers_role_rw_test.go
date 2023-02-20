package mashery

import (
	"context"
	"errors"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"reflect"
	"testing"
)

func TestRoleKeysPath(t *testing.T) {
	reqCtx := RequestHandlerContext[RoleContext]{
		storagePath: "/backendUUID/role",
	}
	assert.Equal(t, "/backendUUID/role/key", roleKeysPath(&reqCtx))
}

func TestRolePrivateKeyPath(t *testing.T) {
	reqCtx := RequestHandlerContext[RoleContext]{
		storagePath: "/backendUUID/role",
	}
	assert.Equal(t, "/backendUUID/role/pk", rolePrivateKeyPath(&reqCtx))
}

func TestRoleUsagePath(t *testing.T) {
	reqCtx := RequestHandlerContext[RoleContext]{
		storagePath: "/backendUUID/role",
	}
	assert.Equal(t, "/backendUUID/role/usage", roleUsagePath(&reqCtx))
}

func TestReadRole_FailOnKeyRead(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		fieldSchema: pathRoleFields,
	}
	emulStorage, reqCtx := mockBuilder.Build()
	emulStorage.On("Get", mock.Anything, "/backendUUID/testRole/key").Return(nil, errors.New("sample reject"))

	lr, err := readRole[RoleContext](true)(context.TODO(), reqCtx)
	emulStorage.AssertExpectations(t)
	assert.Nil(t, lr)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to read object from storage: sample reject", err.Error())
}

func TestReadRole_FailOnUsageRead(t *testing.T) {
	mockBider := RoleRequestMockBuilder[RoleContext]{
		fieldSchema: pathRoleFields,
	}
	emulStorage, reqCtx := mockBider.Build()
	emulStorage.
		On("Get", mock.Anything, reqCtx.storagePath+"/key").
		Return(createJsonStorageEntryFrom(t, reqCtx.storagePath+"/key", &RoleKeys{}), nil)

	emulStorage.
		On("Get", mock.Anything, "/backendUUID/testRole/usage").
		Return(nil, errors.New("sample reject"))

	lr, err := readRole[RoleContext](true)(context.TODO(), reqCtx)
	emulStorage.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to read object from storage: sample reject", err.Error())
}

func TestReadRole_WhenMissingRoleKeys_WillFailForRequired(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		fieldSchema: pathRoleFields,
	}

	emulStorage, reqCtx := mockBuilder.Build()
	emulStorage.
		On("Get", mock.Anything, reqCtx.storagePath+"/key").
		Return(nil, nil)

	lr, err := readRole[RoleContext](true)(context.TODO(), reqCtx)
	emulStorage.AssertExpectations(t)

	assert.Nil(t, err)
	assert.NotNil(t, lr)
	assert.Equal(t, "role is not found", lr.Error().Error())
}

func TestReadRole_WhenMissingRoleUsage_WillFailForRequired(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		fieldSchema: pathRoleFields,
	}

	emulStorage, reqCtx := mockBuilder.Build()
	emulStorage.
		On("Get", mock.Anything, reqCtx.storagePath+"/key").
		Return(createJsonStorageEntryFrom(t, reqCtx.storagePath+"/key", &RoleKeys{}), nil)

	emulStorage.
		On("Get", mock.Anything, reqCtx.storagePath+"/usage").
		Return(nil, nil)

	lr, err := readRole[RoleContext](true)(context.TODO(), reqCtx)
	emulStorage.AssertExpectations(t)

	assert.Nil(t, err)
	assert.NotNil(t, lr)
	assert.Equal(t, "role usage is not found", lr.Error().Error())
}

func TestReadRole_WhenAllPreset_WillPass(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		fieldSchema: pathRoleFields,
		container:   &RoleContainer{},
	}
	emulStorage, reqCtx := mockBuilder.Build()
	emulStorage.
		On("Get", mock.Anything, reqCtx.storagePath+"/key").
		Return(createJsonStorageEntryFrom(t, reqCtx.storagePath+"/key", &RoleKeys{}), nil)

	emulStorage.
		On("Get", mock.Anything, reqCtx.storagePath+"/usage").
		Return(createJsonStorageEntryFrom(t, reqCtx.storagePath+"/usage", &StoredRoleUsage{}), nil)

	lr, err := readRole[RoleContext](true)(context.TODO(), reqCtx)
	emulStorage.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestReadRole_WhenMissingAll_WillPassForOptional(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		container:   &RoleContainer{},
		fieldSchema: pathRoleFields,
	}
	emulStorage, reqCtx := mockBuilder.Build()
	emulStorage.
		On("Get", mock.Anything, reqCtx.storagePath+"/key").
		Return(nil, nil)

	emulStorage.
		On("Get", mock.Anything, reqCtx.storagePath+"/usage").
		Return(nil, nil)

	lr, err := readRole[RoleContext](false)(context.TODO(), reqCtx)
	emulStorage.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.Nil(t, err)
	assert.NotNil(t, reqCtx.heap.GetRole())
}

func TestUpdateRoleKeysFromRequest_WillRejectNilRole(t *testing.T) {
	_, reqCtx := setupRoleRequestMock()
	lr, err := updateRoleKeysFromRequest(nil, reqCtx)
	assert.Nil(t, lr)
	assert.NotNil(t, err)

	assert.Equal(t, "updateRoleKeysFromRequest requires a non-nil role to operate", err.Error())
}

func TestUpdateRoleKeysFromRequest(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		container: &RoleContainer{
			role: &StoredRole{},
		},
		data: map[string]interface{}{
			roleAreaIdField:   "a-b-c-d",
			roleAreaNidField:  745,
			roleApiKeField:    "apiKey",
			roleSecretField:   "secret",
			roleUsernameField: "user",
			rolePasswordField: "pwd",
			roleQpsField:      15,
		},
		fieldSchema: pathRoleFields,
	}

	reqCtx := mockBuilder.Request()

	lr, err := updateRoleKeysFromRequest(context.TODO(), reqCtx)

	assert.Nil(t, lr)
	assert.Nil(t, err)

	sr := reqCtx.heap.GetRole()
	assert.Equal(t, "a-b-c-d", sr.Keys.AreaId)
	assert.Equal(t, 745, sr.Keys.AreaNid)
	assert.Equal(t, "apiKey", sr.Keys.ApiKey)
	assert.Equal(t, "secret", sr.Keys.KeySecret)
	assert.Equal(t, "user", sr.Keys.Username)
	assert.Equal(t, "pwd", sr.Keys.Password)
	assert.Equal(t, 15, sr.Keys.MaxQPS)
}

func TestUpdateRoleKeysFromRequest_IsResilientToTypeError(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		data: map[string]interface{}{
			roleAreaIdField:   43,
			roleAreaNidField:  "745",
			roleApiKeField:    44,
			roleSecretField:   45,
			roleUsernameField: 46,
			rolePasswordField: 47,
			roleQpsField:      "15",
		},
		fieldSchema: pathRoleFields,
		container: &RoleContainer{
			role: &StoredRole{},
		},
	}
	reqCtx := mockBuilder.Request()

	lr, err := updateRoleKeysFromRequest(context.TODO(), reqCtx)

	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestBlockOperationOnImportedRole(t *testing.T) {
	role := StoredRole{
		Keys: RoleKeys{
			Imported: false,
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(role)

	lr, err := blockOperationOnImportedRole(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	role.Keys.Imported = true
	_, reqCtx = setupRoleRequestMockHaving(role)
	lr, err = blockOperationOnImportedRole(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "operation is not permitted on an imported role", lr.Error().Error())
	assert.Nil(t, err)
}

func TestBlockOperationOnForceProxyRole(t *testing.T) {
	role := StoredRole{
		Keys: RoleKeys{
			ForceProxyMode: false,
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(role)

	lr, err := blockOperationOnForceProxyRole(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	role.Keys.ForceProxyMode = true

	_, reqCtx = setupRoleRequestMockHaving(role)
	lr, err = blockOperationOnForceProxyRole(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "operation is not permitted as this role requires proxy mode", lr.Error().Error())
	assert.Nil(t, err)
}

func TestBlockUsageExceedingLimits(t *testing.T) {
	role := StoredRole{
		Usage: StoredRoleUsage{
			RemainingNumUses: -1,
			ExplicitNumUses:  -1,
			ExplicitTerm:     -1,
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(role)

	lr, err := blockUsageExceedingLimits(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	role.Usage.ExplicitTerm = 1000

	_, reqCtx = setupRoleRequestMockHaving(role)
	lr, err = blockUsageExceedingLimits(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "this role has expired (granted until 01 Jan 70 01:16 CET)", lr.Error().Error())
	assert.Nil(t, err)

	role.Usage.ExplicitTerm = 0
	role.Usage.ExplicitNumUses = 20
	role.Usage.RemainingNumUses = 0

	_, reqCtx = setupRoleRequestMockHaving(role)
	lr, err = blockUsageExceedingLimits(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "this role has depleted its usage quota", lr.Error().Error())
	assert.Nil(t, err)
}

func TestDecreaseRemainingUsageQuota_DoesNothingOnUnlimitedRoles(t *testing.T) {
	role := StoredRole{
		Usage: StoredRoleUsage{
			RemainingNumUses: 40,
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(role)

	lr, err := decreaseRemainingUsageQuota(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestDecreaseRemainingUsageQuota_WillDecrementAndSaveChanges(t *testing.T) {
	const startUse = 19

	role := StoredRole{
		Usage: StoredRoleUsage{
			ExplicitNumUses:  20,
			RemainingNumUses: startUse,
		},
	}

	emulStorage, reqCtx := setupRoleRequestMockHaving(role)
	emulStorage.
		On("Put", mock.Anything, mock.MatchedBy(func(sr *logical.StorageEntry) bool {
			var passedUsage StoredRoleUsage
			sr.DecodeJSON(&passedUsage)
			return passedUsage.RemainingNumUses+1 == startUse
		})).
		Return(nil)

	lr, err := decreaseRemainingUsageQuota(context.TODO(), reqCtx)
	emulStorage.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestDecreaseRemainingUsageQuota_WillReturnErrorOnWriteFailure(t *testing.T) {
	const startUse = 19

	role := StoredRole{
		Usage: StoredRoleUsage{
			ExplicitNumUses:  20,
			RemainingNumUses: startUse,
		},
	}

	emulStorage, reqCtx := setupRoleRequestMockHaving(role)
	emulStorage.
		On("Put", mock.Anything, mock.Anything).
		Return(errors.New("sample reject"))

	lr, err := decreaseRemainingUsageQuota(context.TODO(), reqCtx)
	emulStorage.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to persist object: sample reject", err.Error())
}

func TestBlockRoleIncapableOf_V3(t *testing.T) {
	role := StoredRole{
		Keys: RoleKeys{
			AreaNid:   45,
			ApiKey:    "key",
			KeySecret: "secret",
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(role)

	lr, err := blockRoleIncapableOf[RoleContext](3)(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "role is not capable of api version 3", lr.Error().Error())
	assert.Nil(t, err)

	// Should allow V2
	lr, err = blockRoleIncapableOf[RoleContext](2)(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestBlockRoleIncapableOf_V2(t *testing.T) {
	role := StoredRole{
		Keys: RoleKeys{
			AreaId: "a-b-c-d",

			ApiKey:    "key",
			KeySecret: "secret",

			Username: "user",
			Password: "pwd",
		},
	}

	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		container: &RoleContainer{
			role: &role,
		},
	}

	reqCtx := mockBuilder.Request()
	lr, err := blockRoleIncapableOf[RoleContext](2)(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "role is not capable of api version 2", lr.Error().Error())
	assert.Nil(t, err)

	// Should allow V2
	reqCtx = mockBuilder.Request()
	lr, err = blockRoleIncapableOf[RoleContext](3)(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestBlockRoleIncapableOf_Unsupported(t *testing.T) {
	role := StoredRole{}

	_, reqCtx := setupRoleRequestMockHaving(role)
	lr, err := blockRoleIncapableOf[RoleContext](9)(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "unsupported api version: 9", lr.Error().Error())
	assert.Nil(t, err)
}

func TestBlockNonExportableRole(t *testing.T) {
	role := StoredRole{
		Keys: RoleKeys{
			Exportable: true,
		},
	}

	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		fieldSchema: pathRoleExportFields,
		container: &RoleExportContainer{
			RoleContainer: RoleContainer{
				role: &role,
			},
		},
	}

	reqCtx := mockBuilder.Request()

	lr, err := blockNonExportableRole(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	role.Keys.Exportable = false
	reqCtx = mockBuilder.Request()
	lr, err = blockNonExportableRole(nil, reqCtx)

	assert.NotNil(t, lr)
	assert.Equal(t, "this role is not exportable", lr.Error().Error())

	assert.Nil(t, err)
}

func TestAllowOnlyV2CapableRole(t *testing.T) {
	role := StoredRole{
		Keys: RoleKeys{
			AreaNid:   456,
			ApiKey:    "key",
			KeySecret: "secret",
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(role)

	lr, err := allowOnlyV2CapableRole(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	role.Keys.AreaNid = 0
	_, reqCtx = setupRoleRequestMockHaving(role)
	lr, err = allowOnlyV2CapableRole(nil, reqCtx)

	assert.NotNil(t, lr)
	assert.Equal(t, "this role is not V2 capable", lr.Error().Error())

	assert.Nil(t, err)
}

func TestAllowOnlyV3CapableRole(t *testing.T) {
	role := StoredRole{
		Keys: RoleKeys{
			AreaId:    "a-b-c-d",
			ApiKey:    "key",
			KeySecret: "secret",
			Username:  "user",
			Password:  "password",
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(role)
	lr, err := allowOnlyV3CapableRole(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	role.Keys.AreaId = ""
	_, reqCtx = setupRoleRequestMockHaving(role)
	lr, err = allowOnlyV3CapableRole(nil, reqCtx)

	assert.NotNil(t, lr)
	assert.Equal(t, "this role is not V3 capable", lr.Error().Error())

	assert.Nil(t, err)
}

func TestSaveRoleKeys_Succeeds(t *testing.T) {
	sr := StoredRole{
		Keys: RoleKeys{
			AreaNid: 500,
		},
	}

	// TODO: there may be a better way to go about createing a generic
	// method
	emulStore, reqCtx := setupRoleRequestMockHaving(sr)
	emulStore.
		On("Put", mock.Anything, mock.MatchedBy(storageEntryBearing(&sr.Keys, RoleKeys{}))).
		Return(nil)

	lr, err := saveRoleKeys(context.TODO(), reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestSaveRoleKeys_ReturnsErrorOnFailedWrite(t *testing.T) {
	sr := StoredRole{
		Keys: RoleKeys{
			AreaNid: 500,
		},
	}

	emulStore, reqCtx := setupRoleRequestMockHaving(sr)
	emulStore.
		On("Put", mock.Anything, mock.Anything).
		Return(errors.New("sample rejection"))

	lr, err := saveRoleKeys(context.TODO(), reqCtx)
	assert.Nil(t, lr)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to persist object: sample rejection", err.Error())
}

func TestSaveRoleUsage_Succeeds(t *testing.T) {
	sr := StoredRole{
		Usage: StoredRoleUsage{
			RemainingNumUses: 345,
		},
	}

	emulStore, reqCtx := setupRoleRequestMockHaving(sr)
	emulStore.
		On("Put", mock.Anything, mock.MatchedBy(storageEntryBearing(&sr.Usage, StoredRoleUsage{}))).
		Return(nil)

	lr, err := saveRoleUsage(context.TODO(), reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestSaveRoleUsage_ReturnsErrorOnFailedWrite(t *testing.T) {
	sr := StoredRole{
		Usage: StoredRoleUsage{
			RemainingNumUses: 345,
		},
	}

	emulStore, reqCtx := setupRoleRequestMockHaving(sr)
	emulStore.
		On("Put", mock.Anything, mock.Anything).
		Return(errors.New("sample rejection"))

	lr, err := saveRoleUsage(context.TODO(), reqCtx)
	assert.Nil(t, lr)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to persist object: sample rejection", err.Error())
}

func TestSetInitialRoleUsage(t *testing.T) {
	scenarioRole := StoredRole{
		Usage: StoredRoleUsage{
			V3Token:          "token",
			V3TokenExpiry:    500,
			ExplicitTerm:     500,
			ExplicitNumUses:  500,
			RemainingNumUses: 500,
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(scenarioRole)

	lr, err := setInitialRoleUsage(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	role := reqCtx.heap.GetRole()
	assert.Empty(t, role.Usage.V3Token)
	assert.Negative(t, role.Usage.V3TokenExpiry)

	assert.Negative(t, role.Usage.ExplicitTerm)
	assert.Negative(t, role.Usage.ExplicitNumUses)
	assert.Negative(t, role.Usage.RemainingNumUses)
}

func TestForgetToken(t *testing.T) {
	scenarioRole := StoredRole{
		Usage: StoredRoleUsage{
			V3Token:          "token",
			V3TokenExpiry:    500,
			ExplicitTerm:     500,
			ExplicitNumUses:  500,
			RemainingNumUses: 500,
		},
	}

	_, reqCtx := setupRoleRequestMockHaving(scenarioRole)
	lr, err := forgetUsedToken(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	outRole := reqCtx.heap.GetRole()
	assert.Empty(t, outRole.Usage.V3Token)
	assert.Negative(t, outRole.Usage.V3TokenExpiry)

	assert.Equal(t, int64(500), outRole.Usage.ExplicitTerm)
	assert.Equal(t, int64(500), outRole.Usage.ExplicitNumUses)
	assert.Equal(t, int64(500), outRole.Usage.RemainingNumUses)
}

//////////
// Helper methods
/////////

type RoleRequestMockBuilder[T any] struct {
	container   T
	data        map[string]interface{}
	fieldSchema map[string]*framework.FieldSchema
}

func (rrmb *RoleRequestMockBuilder[T]) Build() (*MockedVaultStorageWrapper, *RequestHandlerContext[T]) {
	emulStorage := new(MockedVaultStorageWrapper)

	reqCtx := rrmb.buildRequest(emulStorage)

	return emulStorage, reqCtx
}

func (rrmb *RoleRequestMockBuilder[T]) Request() *RequestHandlerContext[T] {
	emulStorage := new(MockedVaultStorageWrapper)

	reqCtx := rrmb.buildRequest(emulStorage)

	return reqCtx
}

func (rrmb *RoleRequestMockBuilder[T]) buildRequest(emulStorage *MockedVaultStorageWrapper) *RequestHandlerContext[T] {
	actualData := map[string]interface{}{
		"roleName": "testRole",
	}

	if rrmb.data != nil {
		for k, v := range rrmb.data {
			actualData[k] = v
		}
	}

	reqCtx := &RequestHandlerContext[T]{
		storagePath: "/backendUUID/testRole",
		request: &logical.Request{
			Storage: emulStorage,
			Data:    actualData,
		},
		data: &framework.FieldData{
			Raw:    actualData,
			Schema: rrmb.fieldSchema,
		},
		plugin: &AuthPlugin{
			vaultStorage: &VaultStorageImpl{},
		},
		heap: rrmb.container,
	}
	return reqCtx
}

func storageEntryBearing[T any](js *T, out T) func(entry *logical.StorageEntry) bool {
	return func(entry *logical.StorageEntry) bool {
		if e := entry.DecodeJSON(&out); e != nil {
			return false
		}

		return reflect.DeepEqual(*js, out)
	}
}

func storageEntryBearingData() interface{} {
	return mock.MatchedBy(func(entry *logical.StorageEntry) bool {
		return len(entry.Value) > 0
	})
}

func setupRoleRequestMock() (*MockedVaultStorageWrapper, *RequestHandlerContext[RoleContext]) {
	return setupRoleRequestMockWithData(nil, nil)
}

func setupRoleRequestMockWithFields(schema map[string]*framework.FieldSchema) (*MockedVaultStorageWrapper, *RequestHandlerContext[RoleContext]) {
	builder := RoleRequestMockBuilder[RoleContext]{
		container:   &RoleContainer{},
		fieldSchema: schema,
	}

	return builder.Build()
}

func setupRoleRequestMockHaving(role StoredRole) (*MockedVaultStorageWrapper, *RequestHandlerContext[RoleContext]) {
	builder := RoleRequestMockBuilder[RoleContext]{
		container: &RoleContainer{
			role: &role,
		},
	}

	return builder.Build()
}

func setupRoleRequestMockWithData(data map[string]interface{}, schema map[string]*framework.FieldSchema) (*MockedVaultStorageWrapper, *RequestHandlerContext[RoleContext]) {
	builder := RoleRequestMockBuilder[RoleContext]{
		container:   &RoleContainer{},
		data:        data,
		fieldSchema: schema,
	}

	return builder.Build()
}
