package dhcpd

import "fmt"

func CheckIfOtherDHCPServersPresentV4(ifaceName string) (bool, error) {
	return false, fmt.Errorf("not supported")
}

func CheckIfOtherDHCPServersPresentV6(ifaceName string) (bool, error) {
	return false, fmt.Errorf("not supported")
}
