package main

import "testing"

func TestResolveRDNS(t *testing.T) {
	if r := resolveRDNS("1.1.1.1", "1.1.1.1"); r != "one.one.one.one" {
		t.Errorf("resolveRDNS(): %s", r)
	}
}
