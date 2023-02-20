package mashery

import (
	"context"
	"github.com/hashicorp/vault/sdk/logical"
)

// Initialize reading of the back-end configuration

func readBackEndConfig[T BackendConfigurationContext](ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	val := reqCtx.plugin.DefaultBackendConfiguration()

	_, err := reqCtx.Read(ctx, &val)

	reqCtx.heap.CarryBackendConfiguration(val)
	return nil, err
}

func resetCertificatePin(_ context.Context, reqCtx *RequestHandlerContext[TLSPinningOperations]) (*logical.Response, error) {
	trgt := reqCtx.heap.GetPinning()

	trgt.CommonName = ""
	trgt.SerialNumber = []byte{}
	trgt.Fingerprint = []byte{}

	return nil, nil
}

func saveBackEndConfigFunc[T BackendConfigurationContext](ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	return nil, reqCtx.Write(ctx, reqCtx.heap.GetBackendConfiguration())
}

func acceptBackendConfigurationFunc[T BackendConfigurationContext](ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	reqCtx.plugin.AcceptConfigurationUpdate(ctx, *reqCtx.heap.GetBackendConfiguration())
	return nil, nil
}
