package home

import (
	"testing"

	"github.com/AdguardTeam/golibs/log"
	"github.com/stretchr/testify/assert"
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

func TestSplitNext(t *testing.T) {
	s := " a,b , c "
	assert.True(t, SplitNext(&s, ',') == "a")
	assert.True(t, SplitNext(&s, ',') == "b")
	assert.True(t, SplitNext(&s, ',') == "c" && len(s) == 0)
}
