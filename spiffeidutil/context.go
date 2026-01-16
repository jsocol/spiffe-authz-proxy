package spiffeidutil

import (
	"context"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type ctxKey string

const spidKey ctxKey = "spiffeid"

func WithSPIFFEID(ctx context.Context, spID spiffeid.ID) context.Context {
	return context.WithValue(ctx, spidKey, spID)
}

func FromContext(ctx context.Context) spiffeid.ID {
	val := ctx.Value(spidKey)
	if spID, ok := val.(spiffeid.ID); ok {
		return spID
	}

	return spiffeid.ID{}
}
