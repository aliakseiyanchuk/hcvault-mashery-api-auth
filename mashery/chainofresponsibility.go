package mashery

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// Chain of responsibility pattern helper classes aiming to simplify the code and improve the readability of the
// overall code. The processing of the request required several steps, such as request validation, error handling,
// performing storage updates. The chain of responsibility breaks this into discreet, logical steps.
//
// Furthermore, the chain of responsibility support introduces the notion of operation template. The caller needs
// to supply the specific functions while the template will take care of the boilerplate construction.

type RequestHandlerContext[T any] struct {
	request *logical.Request
	data    *framework.FieldData
	plugin  *AuthPlugin

	// Storage path on which this request operates
	storagePath string

	// Heap of objects carried by this request.
	heap T
}

func MapRequestHandlerContext[From any, To any](from *RequestHandlerContext[From], heapMapper func(from From) To) *RequestHandlerContext[To] {
	rv := RequestHandlerContext[To]{
		from.request,
		from.data,
		from.plugin,
		from.storagePath,
		heapMapper(from.heap),
	}

	return &rv
}

type RoleContext interface {
	GetRole() *StoredRole
	CarryRole(role *StoredRole)
}

type RoleContainer struct {
	role *StoredRole
}

func (r *RoleContainer) GetRole() *StoredRole {
	return r.role
}

func (r *RoleContainer) CarryRole(role *StoredRole) {
	r.role = role
}

type PemContext interface {
	GetPEMBlock() *pem.Block
	CarryPEMBlock(block *pem.Block)
}

type RoleExportContext interface {
	RoleContext

	GetRecipientCertificate() *x509.Certificate
	GetRecipientName() string

	CarryRecipientCertificate(cert *x509.Certificate)
	CarryRecipientName(name string)
}

type RoleExportContainer struct {
	RoleContainer

	cert      *x509.Certificate
	recipeint string
}

func (cc *RoleExportContainer) GetRecipientCertificate() *x509.Certificate {
	return cc.cert
}

func (cc *RoleExportContainer) GetRecipientName() string {
	return cc.recipeint
}

func (cc *RoleExportContainer) CarryRecipientCertificate(cert *x509.Certificate) {
	cc.cert = cert
}
func (cc *RoleExportContainer) CarryRecipientName(name string) {
	cc.recipeint = name
}

type BackendConfigurationContext interface {
	GetBackendConfiguration() *BackendConfiguration
	CarryBackendConfiguration(cfg BackendConfiguration)
}

type BackendConfigurationContainer struct {
	cfg BackendConfiguration
}

func (b *BackendConfigurationContainer) GetBackendConfiguration() *BackendConfiguration {
	return &b.cfg
}

func (b *BackendConfigurationContainer) CarryBackendConfiguration(cfg BackendConfiguration) {
	b.cfg = cfg
}

type TLSCertificatePinningContext interface {
	GetPinning() *transport.TLSCertChainPin
	CarryPinning(cfg *transport.TLSCertChainPin)
}

type TLSCertificatePinningContainer struct {
	pin *transport.TLSCertChainPin
}

func (c *TLSCertificatePinningContainer) GetPinning() *transport.TLSCertChainPin {
	return c.pin
}

func (c *TLSCertificatePinningContainer) CarryPinning(cfg *transport.TLSCertChainPin) {
	c.pin = cfg
}

type TLSPinningOperations interface {
	BackendConfigurationContext
	TLSCertificatePinningContext
}

type TLSPinningContainer struct {
	BackendConfigurationContainer
	TLSCertificatePinningContainer
}

///////////////////////////
// Reusable types
//////////////////////////

// Read and de-serialize the object at the indicated path. returns:
// - boolean indicating if the object was read, and
// - error indicating if the error has occurred.
// The combination of (false, error) indicates that the object was absent. The calling code should decide
// if such condition warrants an error, or a default configuration coud be used.
func (reqCtx *RequestHandlerContext[T]) Read(ctx context.Context, json interface{}) (bool, error) {
	return reqCtx.ReadPath(ctx, reqCtx.storagePath, json)
}

func (reqCtx *RequestHandlerContext[T]) ReadPath(ctx context.Context, path string, json interface{}) (bool, error) {
	if len(reqCtx.storagePath) == 0 {
		return false, errors.New("storage path to read/write to/from hasn't been initialized")
	}

	return reqCtx.plugin.vaultStorage.Read(ctx, reqCtx.request.Storage, path, json)
}

func (reqCtx *RequestHandlerContext[T]) ReadBinaryPath(ctx context.Context, path string) (bool, []byte, error) {
	if len(reqCtx.storagePath) == 0 {
		return false, []byte{}, errors.New("storage path to read/write to/from hasn't been initialized")
	}

	return reqCtx.plugin.vaultStorage.ReadBinary(ctx, reqCtx.request.Storage, path)
}

func (reqCtx *RequestHandlerContext[T]) Write(ctx context.Context, json interface{}) error {
	return reqCtx.WritePath(ctx, reqCtx.storagePath, json)
}

// WritePath Writes an object into the vault storage at the specified path.
func (reqCtx *RequestHandlerContext[T]) WritePath(ctx context.Context, path string, json interface{}) error {
	if len(reqCtx.storagePath) == 0 {
		return errors.New("storage path to read/write to/from hasn't been initialized")
	}

	return reqCtx.plugin.vaultStorage.Persist(ctx, reqCtx.request.Storage, path, json)
}

func (reqCtx *RequestHandlerContext[T]) WriteBinaryPath(ctx context.Context, path string, data []byte) error {
	if len(reqCtx.storagePath) == 0 {
		return errors.New("storage path to read/write to/from hasn't been initialized")
	}

	return reqCtx.plugin.vaultStorage.PersistBinary(ctx, reqCtx.request.Storage, path, data)
}

// TransformerFunc function performing a transformation in the chain of responsibility. The function
// returns either response (and no further processing would be needed), or a data element for the next-in-chain
type TransformerFunc[T any] func(context.Context, *RequestHandlerContext[T]) (*logical.Response, error)

func SimpleChain[T any](subsequent ...TransformerFunc[T]) TransformerFunc[T] {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
		for _, f := range subsequent {
			response, err := f(ctx, reqCtx)
			if err != nil || response != nil {
				return response, err
			}
		}
		return nil, nil
	}
}

func Tail[T any](f1 TransformerFunc[T],
	subsequent ...TransformerFunc[T]) TransformerFunc[T] {

	return func(ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
		if resp, err := f1(ctx, reqCtx); resp != nil || err != nil {
			return resp, err
		} else {
			for _, f2 := range subsequent {
				if resp, err = f2(ctx, reqCtx); resp != nil || err != nil {
					break
				}
			}

			return resp, err
		}
	}
}

/*
func Join[Extension any, Previous any, Bump any](unfulfilledValue Previous,
	f1 TransformerFunc[Extension, Previous, Bump],
	f2 TransformerFunc[Extension, Bump, Previous],
	thenFuncs ...TransformerFunc[Extension, Previous, Previous],
) TransformerFunc[Extension, Previous, Previous] {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[Extension], v Previous) (*logical.Response, Previous, error) {
		if resp, bumpValue, err := f1(ctx, reqCtx, v); err != nil || resp != nil {
			return resp, unfulfilledValue, err
		} else if resp, backVal, err := f2(ctx, reqCtx, bumpValue); err != nil || resp != nil {
			return resp, backVal, err
		} else {
			// Iterate of thennable functions until first available response
			// or error
			for _, f3 := range thenFuncs {
				if resp, backVal, err = f3(ctx, reqCtx, backVal); err != nil || resp != nil {
					break
				}
			}

			return resp, backVal, err
		}
	}
}

// AndThen chains two functions together
func AndThen[Extension any, Previous any, Next any](unfulfilledValue Next,
	f1 TransformerJumpFunc[Extension, Previous],
	f2 TransformerFunc[Extension, Previous, Next],
	thenFuncs ...TransformerFunc[Extension, Next, Next],
) TransformerJumpFunc[Extension, Next] {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[Extension]) (*logical.Response, Next, error) {
		if resp, prevValue, err := f1(ctx, reqCtx); err != nil || resp != nil {
			return resp, unfulfilledValue, err
		} else {
			// Previous value is available for conversion into the
			// next value
			if lr, nextVal, err := f2(ctx, reqCtx, prevValue); err != nil || lr != nil {
				return lr, nextVal, err
			} else {
				// Next value is available. This value can be enhanced with further functions
				// if these were supplied. Iterate until the first logical response is available or
				// until an error is encountered
				for _, f3 := range thenFuncs {
					if lr, nextVal, err = f3(ctx, reqCtx, nextVal); lr != nil || err != nil {
						break
					}
				}

				return lr, nextVal, err
			}
		}
	}
}

func AndThenTo[Extension any, Previous any, Next any](unfulfilledValue Next, f1 TransformerJumpFunc[Extension, Previous], f2 TransformerJumpFunc[Extension, Next]) TransformerJumpFunc[Extension, Next] {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[Extension]) (*logical.Response, Next, error) {
		if resp, _, err := f1(ctx, reqCtx); err != nil {
			return nil, unfulfilledValue, err
		} else if resp != nil {
			return resp, unfulfilledValue, nil
		} else {
			return f2(ctx, reqCtx)
		}
	}
}
*/
