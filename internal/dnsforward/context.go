package dnsforward

import (
	"context"
	"fmt"
	"net/netip"
)

// ctxKey is the type for context keys.
type ctxKey int

// Context key values.
const (
	ctxKeyClientID ctxKey = iota
	ctxKeyECSClientAddr
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

// contextWithECSClientAddr returns a new context with the ECS client address.
func contextWithECSClientAddr(parent context.Context, addr netip.Addr) (ctx context.Context) {
	return context.WithValue(parent, ctxKeyECSClientAddr, addr)
}

// ecsClientAddrFromContext returns the ECS client address for this request.
func ecsClientAddrFromContext(ctx context.Context) (addr netip.Addr, ok bool) {
	v := ctx.Value(ctxKeyECSClientAddr)
	if v == nil {
		return addr, false
	}

	addr, ok = v.(netip.Addr)
	if !ok {
		panic(fmt.Errorf("bad type for ctxKeyECSClientAddr: %T(%[1]v)", v))
	}

	return addr, true
}
