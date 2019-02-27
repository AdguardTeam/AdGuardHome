package main

import (
	"testing"

	"github.com/AdguardTeam/golibs/log"
)

func TestGetValidNetInterfacesForWeb(t *testing.T) {
	ifaces, err := getValidNetInterfacesForWeb()
	if err != nil {
		t.Fatalf("Cannot get net interfaces: %s", err)
	}
	if len(ifaces) == 0 {
		t.Fatalf("No net interfaces found")
	}

	for _, iface := range ifaces {
		if len(iface.Addresses) == 0 {
			t.Fatalf("No addresses found for %s", iface.Name)
		}

		log.Printf("%v", iface)
	}
}
