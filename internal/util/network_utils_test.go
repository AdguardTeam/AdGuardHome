package util

import (
	"log"
	"testing"
)

func TestGetValidNetInterfacesForWeb(t *testing.T) {
	ifaces, err := GetValidNetInterfacesForWeb()
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
