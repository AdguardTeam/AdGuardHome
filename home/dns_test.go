package home

import (
	"os"
	"testing"
)

func TestResolveRDNS(t *testing.T) {
	_ = os.RemoveAll(config.getDataDir())
	defer func() { _ = os.RemoveAll(config.getDataDir()) }()

	config.DNS.BindHost = "1.1.1.1"
	initDNSServer()
	if r := config.dnsctx.rdns.resolve("1.1.1.1"); r != "one.one.one.one" {
		t.Errorf("resolveRDNS(): %s", r)
	}
}
