// Package configmodifier provides a way to update the global configuration.
package configmodifier

import (
	"context"
)

// Interface defines a method for updating the global configuration.
type Interface interface {
	Apply(ctx context.Context)
}

// Empty is an empty [Interface] implementation that does nothing.
type Empty struct{}

// Apply implements the [Interface] for Empty.
func (em Empty) Apply(ctx context.Context) {}

// type check
var _ Interface = Empty{}

// Mock is a fake [Interface] implementation for tests.
//
// TODO(s.chzhen): !! Move to aghtest.
type Mock struct {
	OnApply func(ctx context.Context)
}

// type check
var _ Interface = (*Mock)(nil)

// Apply implements the [Interface] interface for *Mock.
func (m *Mock) Apply(ctx context.Context) {
	m.OnApply(ctx)
}
