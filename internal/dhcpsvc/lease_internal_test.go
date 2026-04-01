package dhcpsvc

// BlockedHardwareAddr is the hardware address used to mark a lease as blocked.
// It's exported for testing purposes.
var BlockedHardwareAddr = blockedHardwareAddr

// CompareLeases is a helper function that sorts a slice of leases according to
// their IP.
func CompareLeases(a, b *Lease) (res int) {
	return a.IP.Compare(b.IP)
}
