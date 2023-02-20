package mashery

import (
	"context"
	"github.com/hashicorp/vault/sdk/logical"
)

type Runner[T any] interface {
	Run(context.Context, *RequestHandlerContext[T]) (*logical.Response, error)
}

type SimpleRunner[T any] struct {
	Runner[T]

	f []TransformerFunc[T]
}

func (sr *SimpleRunner[T]) Append(subsequent ...TransformerFunc[T]) {
	sr.f = append(sr.f, subsequent...)
}

func (sr *SimpleRunner[T]) Run(ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	for _, f := range sr.f {
		if lr, err := f(ctx, reqCtx); lr != nil || err != nil {
			return lr, err
		}
	}

	return nil, nil
}

type MappingRunner[From any, To any] struct {
	SimpleRunner[To]
	parent   Runner[From]
	exporter func(To) From
	importer func(From, To)
}

func (mr *MappingRunner[From, To]) Run(ctx context.Context, reqCtx *RequestHandlerContext[To]) (*logical.Response, error) {
	parentCtx := MapRequestHandlerContext(reqCtx, mr.exporter)

	if lr, err := mr.parent.Run(ctx, parentCtx); lr != nil || err != nil {
		return lr, err
	} else {
		if mr.importer != nil {
			mr.importer(parentCtx.heap, reqCtx.heap)
		}

		return mr.SimpleRunner.Run(ctx, reqCtx)
	}
}
