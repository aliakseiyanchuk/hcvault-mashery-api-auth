package mashery

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"testing"
)
import "github.com/stretchr/testify/mock"

type MockedVaultStorageWrapper struct {
	logical.Storage
	mock.Mock
}

type dummyStruct struct {
	A string `json:"a"`
}

type malformedStruct struct {
	A int `json:"a"`
}

func (w *MockedVaultStorageWrapper) Get(ctx context.Context, path string) (*logical.StorageEntry, error) {
	args := w.Called(ctx, path)

	var se *logical.StorageEntry
	var err error

	if len(args) > 0 {
		if seArg := args.Get(0); seArg != nil {
			se = seArg.(*logical.StorageEntry)
		}
		if errArg := args.Get(1); errArg != nil {
			err = errArg.(error)
		}
	}

	return se, err
}

func (w *MockedVaultStorageWrapper) Put(ctx context.Context, se *logical.StorageEntry) error {
	args := w.Called(ctx, se)

	var err error = nil
	if len(args) > 0 {
		if errArg := args.Get(0); errArg != nil {
			err = errArg.(error)
		}
	}

	return err
}

func TestVaultStorageImpl_ReadFailsOnStorageIOFailure(t *testing.T) {
	emulStore := new(MockedVaultStorageWrapper)
	emulStore.On("Get", mock.Anything, "/storPath").Return(nil, errors.New("sample read error"))

	vs := VaultStorageImpl{}
	s := dummyStruct{}

	exist, err := vs.Read(context.TODO(), emulStore, "/storPath", &s)
	emulStore.AssertExpectations(t)

	assert.False(t, exist)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to read object from storage: sample read error", err.Error())
}

func TestVaultStorageImpl_ReadNonExistingObject(t *testing.T) {
	emulStore := new(MockedVaultStorageWrapper)
	emulStore.On("Get", mock.Anything, "/storPath").Return(nil, nil)

	vs := VaultStorageImpl{}
	s := dummyStruct{}

	exist, err := vs.Read(context.TODO(), emulStore, "/storPath", &s)
	emulStore.AssertExpectations(t)

	assert.False(t, exist)
	assert.Nil(t, err)
}

func createJsonStorageEntryFrom(t *testing.T, path string, json interface{}) *logical.StorageEntry {
	rv, err := logical.StorageEntryJSON(path, json)
	assert.Nil(t, err)
	return rv
}

func createBinaryStorageEntryFrom(t *testing.T, path string, data []byte) *logical.StorageEntry {
	assert.True(t, len(data) > 0)

	return &logical.StorageEntry{
		Key:   path,
		Value: data,
	}
}

func TestVaultStorageImpl_ReadExistingObject(t *testing.T) {
	refObj := dummyStruct{
		A: "vvv",
	}

	emulStore := new(MockedVaultStorageWrapper)
	emulStore.On("Get", mock.Anything, "/storPath").Return(createJsonStorageEntryFrom(t, "/storPath", &refObj), nil)

	vs := VaultStorageImpl{}
	s := dummyStruct{}

	exist, err := vs.Read(context.TODO(), emulStore, "/storPath", &s)
	emulStore.AssertExpectations(t)

	assert.True(t, exist)
	assert.Nil(t, err)
	assert.Equal(t, refObj.A, s.A)
}

func TestVaultStorageImpl_ReadMalformedObject(t *testing.T) {
	refObj := malformedStruct{
		A: 35,
	}

	emulStore := new(MockedVaultStorageWrapper)
	emulStore.On("Get", mock.Anything, "/storPath").Return(createJsonStorageEntryFrom(t, "/storPath", &refObj), nil)

	vs := VaultStorageImpl{}
	s := dummyStruct{}

	exist, err := vs.Read(context.TODO(), emulStore, "/storPath", &s)
	emulStore.AssertExpectations(t)

	assert.True(t, exist)
	assert.NotNil(t, err)
	assert.Equal(t, "cannot unmarshal data structure (json: cannot unmarshal number into Go struct field dummyStruct.a of type string)", err.Error())
}

func TestVaultStorageImpl_Persist_WithNilObject(t *testing.T) {
	emulStore := new(MockedVaultStorageWrapper)
	emulStore.On("Put", mock.Anything, mock.Anything).Return(nil).Maybe()

	vs := VaultStorageImpl{}
	e := vs.Persist(context.TODO(), emulStore, "/storPath", nil)
	emulStore.AssertExpectations(t)

	assert.NotNil(t, e)
	assert.Equal(t, "failed to create config storage entry: failed to encode storage entry: input for encoding is nil", e.Error())
}

func TestVaultStorageImpl_Persist_WithObject(t *testing.T) {
	refObj := dummyStruct{
		A: "vvv",
	}

	emulStore := new(MockedVaultStorageWrapper)
	emulStore.On("Put", mock.Anything, mock.MatchedBy(func(req *logical.StorageEntry) bool {

		var jsonIn dummyStruct
		json.Unmarshal(req.Value, &jsonIn)
		return jsonIn.A == refObj.A
	})).Return(nil)

	vs := VaultStorageImpl{}
	e := vs.Persist(context.TODO(), emulStore, "/storPath", &refObj)
	emulStore.AssertExpectations(t)

	assert.Nil(t, e)
}
