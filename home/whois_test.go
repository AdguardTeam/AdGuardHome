package home

import (
	"strings"
	"testing"

	whois "github.com/likexian/whois-go"
	"github.com/stretchr/testify/assert"
)

func TestWhois(t *testing.T) {
	resp, err := whois.Whois("8.8.8.8")
	assert.True(t, err == nil)
	assert.True(t, strings.Index(resp, "OrgName:        Google LLC") != -1)
	assert.True(t, strings.Index(resp, "City:           Mountain View") != -1)
	assert.True(t, strings.Index(resp, "Country:        US") != -1)
	m := whoisParse(resp)
	assert.True(t, m["orgname"] == "Google LLC")
	assert.True(t, m["country"] == "US")
	assert.True(t, m["city"] == "Mountain View")
}
