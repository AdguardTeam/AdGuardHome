package dnsforward

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBlockedIPAllowed(t *testing.T) {
	a := &accessCtx{}
	assert.True(t, a.Init([]string{"1.1.1.1", "2.2.0.0/16"}, nil, nil) == nil)

	disallowed, disallowedRule := a.IsBlockedIP("1.1.1.1")
	assert.False(t, disallowed)
	assert.Equal(t, "", disallowedRule)

	disallowed, disallowedRule = a.IsBlockedIP("1.1.1.2")
	assert.True(t, disallowed)
	assert.Equal(t, "", disallowedRule)

	disallowed, disallowedRule = a.IsBlockedIP("2.2.1.1")
	assert.False(t, disallowed)
	assert.Equal(t, "", disallowedRule)

	disallowed, disallowedRule = a.IsBlockedIP("2.3.1.1")
	assert.True(t, disallowed)
	assert.Equal(t, "", disallowedRule)
}

func TestIsBlockedIPDisallowed(t *testing.T) {
	a := &accessCtx{}
	assert.True(t, a.Init(nil, []string{"1.1.1.1", "2.2.0.0/16"}, nil) == nil)

	disallowed, disallowedRule := a.IsBlockedIP("1.1.1.1")
	assert.True(t, disallowed)
	assert.Equal(t, "1.1.1.1", disallowedRule)

	disallowed, disallowedRule = a.IsBlockedIP("1.1.1.2")
	assert.False(t, disallowed)
	assert.Equal(t, "", disallowedRule)

	disallowed, disallowedRule = a.IsBlockedIP("2.2.1.1")
	assert.True(t, disallowed)
	assert.Equal(t, "2.2.0.0/16", disallowedRule)

	disallowed, disallowedRule = a.IsBlockedIP("2.3.1.1")
	assert.False(t, disallowed)
	assert.Equal(t, "", disallowedRule)
}

func TestIsBlockedIPBlockedDomain(t *testing.T) {
	a := &accessCtx{}
	assert.True(t, a.Init(nil, nil, []string{"host1",
		"host2",
		"*.host.com",
		"||host3.com^",
	}) == nil)

	// match by "host2.com"
	assert.True(t, a.IsBlockedDomain("host1"))
	assert.True(t, a.IsBlockedDomain("host2"))
	assert.True(t, !a.IsBlockedDomain("host3"))

	// match by wildcard "*.host.com"
	assert.True(t, !a.IsBlockedDomain("host.com"))
	assert.True(t, a.IsBlockedDomain("asdf.host.com"))
	assert.True(t, a.IsBlockedDomain("qwer.asdf.host.com"))
	assert.True(t, !a.IsBlockedDomain("asdf.zhost.com"))

	// match by wildcard "||host3.com^"
	assert.True(t, a.IsBlockedDomain("host3.com"))
	assert.True(t, a.IsBlockedDomain("asdf.host3.com"))
}
