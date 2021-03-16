package aghnet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetValidNetInterfacesForWeb(t *testing.T) {
	ifaces, err := GetValidNetInterfacesForWeb()
	require.Nilf(t, err, "Cannot get net interfaces: %s", err)
	require.NotEmpty(t, ifaces, "No net interfaces found")
	for _, iface := range ifaces {
		require.NotEmptyf(t, iface.Addresses, "No addresses found for %s", iface.Name)
	}
}
