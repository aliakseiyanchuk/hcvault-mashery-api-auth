package mashery

import (
	"context"
	"fmt"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/logical"
)

// Vault storage

type VaultStorage interface {
	Persist(ctx context.Context, storage logical.Storage, path string, json interface{}) error
	PersistBinary(ctx context.Context, storage logical.Storage, path string, data []byte) error

	Read(ctx context.Context, storage logical.Storage, path string, receiver interface{}) (bool, error)
	ReadBinary(ctx context.Context, storage logical.Storage, path string) (bool, []byte, error)
}

type VaultStorageImpl struct {
	VaultStorage
}

func (vsi *VaultStorageImpl) Persist(ctx context.Context, storage logical.Storage,
	path string, json interface{}) error {

	if se, err := logical.StorageEntryJSON(path, json); err != nil {
		return errwrap.Wrapf("failed to create config storage entry: {{err}}", err)
	} else {
		if e := storage.Put(ctx, se); e != nil {
			return errwrap.Wrapf(fmt.Sprintf("failed to persist object: %s", e.Error()), e)
		} else {
			return nil
		}
	}
}

func (vsi *VaultStorageImpl) PersistBinary(ctx context.Context, storage logical.Storage,
	path string, data []byte) error {

	se := logical.StorageEntry{
		Key:   path,
		Value: data,
	}
	return storage.Put(ctx, &se)
}

func (vsi *VaultStorageImpl) Read(ctx context.Context, storage logical.Storage,
	path string, receiver interface{}) (bool, error) {

	if entry, err := storage.Get(ctx, path); err != nil {
		return false, errwrap.Wrapf("failed to read object from storage: {{err}}", err)
	} else if entry == nil {
		return false, nil
	} else {
		if err := entry.DecodeJSON(receiver); err != nil {
			return true, errwrap.Wrapf("cannot unmarshal data structure ({{err}})", err)
		} else {
			return true, nil
		}
	}
}

func (vsi *VaultStorageImpl) ReadBinary(ctx context.Context, storage logical.Storage,
	path string) (bool, []byte, error) {
	if entry, err := storage.Get(ctx, path); err != nil {
		return false, []byte{}, errwrap.Wrapf("failed to read data from storage: {{err}}", err)
	} else if entry == nil {
		return false, []byte{}, nil
	} else {
		return true, entry.Value, nil
	}
}
