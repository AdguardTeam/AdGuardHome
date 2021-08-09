package aghnet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetValidNetInterfacesForWeb(t *testing.T) {
	ifaces, err := GetValidNetInterfacesForWeb()
	require.NoErrorf(t, err, "cannot get net interfaces: %s", err)
	require.NotEmpty(t, ifaces, "no net interfaces found")
	for _, iface := range ifaces {
		require.NotEmptyf(t, iface.Addresses, "no addresses found for %s", iface.Name)
	}
}
