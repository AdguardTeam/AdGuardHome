//go:build linux

package ossvc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemdScript_syntax(t *testing.T) {
	// kardianos/service v1.3 rejects text/template field syntax.  The range
	// element {{.}} is valid; dotted names such as {{.Name}} are not.
	assert.NotContains(t, systemdScript, "{{.Name")
	assert.NotContains(t, systemdScript, "{{.Path")
	assert.NotContains(t, systemdScript, "{{.Description")
	assert.NotContains(t, systemdScript, "{{.WorkingDirectory")
	assert.NotContains(t, systemdScript, "{{if and")
	assert.NotContains(t, systemdScript, "{{if gt")

	assert.Contains(t, systemdScript, "{{Description}}")
	assert.Contains(t, systemdScript, "RestartSec=10")
	assert.Contains(t, systemdScript, "StandardOutput=journal")
	assert.Contains(t, systemdScript, "StandardError=journal")
}
