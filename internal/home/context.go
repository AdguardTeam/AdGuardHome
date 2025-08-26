package home

import (
	"context"
	"fmt"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/errors"
)

// ctxKey is the type for context keys within this package.
type ctxKey uint8

const (
	ctxKeyWebUser ctxKey = iota
)

// type check
var _ fmt.Stringer = ctxKey(0)

// String implements the [fmt.Stringer] interface for ctxKey.
func (k ctxKey) String() (s string) {
	switch k {
	case ctxKeyWebUser:
		return "ctxKeyWebUser"
	default:
		panic(fmt.Errorf("ctx key: %w: %d", errors.ErrBadEnumValue, k))
	}
}

// panicBadType is a helper that panics with a message about the context key and
// the expected type.
func panicBadType(key ctxKey, v any) {
	panic(fmt.Errorf("bad type for %s: %T(%[2]v)", key, v))
}

// withWebUser returns a copy of the parent context with the web user added.
func withWebUser(ctx context.Context, u *aghuser.User) (withUser context.Context) {
	return context.WithValue(ctx, ctxKeyWebUser, u)
}

// webUserFromContext returns the web user from the context, if any.
func webUserFromContext(ctx context.Context) (u *aghuser.User, ok bool) {
	const key = ctxKeyWebUser
	v := ctx.Value(key)
	if v == nil {
		return nil, false
	}

	u, ok = v.(*aghuser.User)
	if !ok {
		panicBadType(key, v)
	}

	return u, true
}
