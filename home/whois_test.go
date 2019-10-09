package home

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhois(t *testing.T) {
	w := Whois{timeoutMsec: 5000}
	resp, err := w.queryAll("8.8.8.8")
	assert.True(t, err == nil)
	m := whoisParse(resp)
	assert.True(t, m["orgname"] == "Google LLC")
	assert.True(t, m["country"] == "US")
	assert.True(t, m["city"] == "Mountain View")
}
