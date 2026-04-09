package dnsforward

import (
	"context"
	"fmt"
)

// ctxKey is the type for context keys.
type ctxKey int

// Context key values.
const (
	ctxKeyClientID ctxKey = iota
)

// contextWithClientID returns a new context with the given ID.
func contextWithClientID(parent context.Context, id string) (ctx context.Context) {
	return context.WithValue(parent, ctxKeyClientID, id)
}

// clientIDFromContext returns ID for this request, if any.
func clientIDFromContext(ctx context.Context) (id string, ok bool) {
	v := ctx.Value(ctxKeyClientID)
	if v == nil {
		return id, false
	}

	id, ok = v.(string)
	if !ok {
		panic(fmt.Errorf("bad type for ctxKeyClientID: %T(%[1]v)", v))
	}

	return id, true
}
