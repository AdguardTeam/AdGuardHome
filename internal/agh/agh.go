// Package agh contains common entities and interfaces of AdGuard Home.
package agh

import (
	"context"
)

// ConfigModifier defines an interface for updating the global configuration.
type ConfigModifier interface {
	// Apply applies changes to the global configuration.
	Apply(ctx context.Context)
}

// EmptyConfigModifier is an empty [ConfigModifier] implementation that does
// nothing.
type EmptyConfigModifier struct{}

// Apply implements the [ConfigModifier] for EmptyConfigModifier.
func (em EmptyConfigModifier) Apply(ctx context.Context) {}

// type check
var _ ConfigModifier = EmptyConfigModifier{}
