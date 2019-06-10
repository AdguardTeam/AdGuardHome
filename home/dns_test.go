package home

import (
	"testing"
)

func TestResolveRDNS(t *testing.T) {
	config.DNS.BindHost = "1.1.1.1"
	initDNSServer(".")
	if r := resolveRDNS("1.1.1.1"); r != "one.one.one.one" {
		t.Errorf("resolveRDNS(): %s", r)
	}
}
