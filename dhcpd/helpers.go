package dhcpd

import (
	"fmt"
	"net"

	"github.com/hmage/golibs/log"
	"github.com/joomcode/errorx"
)

func isTimeout(err error) bool {
	operr, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	return operr.Timeout()
}

// return first IPv4 address of an interface, if there is any
func getIfaceIPv4(iface *net.Interface) *net.IPNet {
	ifaceAddrs, err := iface.Addrs()
	if err != nil {
		panic(err)
	}

	for _, addr := range ifaceAddrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			// not an IPNet, should not happen
			log.Fatalf("SHOULD NOT HAPPEN: got iface.Addrs() element %s that is not net.IPNet", addr)
		}

		if ipnet.IP.To4() == nil {
			log.Printf("Got IP that is not IPv4: %v", ipnet.IP)
			continue
		}

		log.Printf("Got IP that is IPv4: %v", ipnet.IP)
		return &net.IPNet{
			IP:   ipnet.IP.To4(),
			Mask: ipnet.Mask,
		}
	}
	return nil
}

func wrapErrPrint(err error, message string, args ...interface{}) error {
	var errx error
	if err == nil {
		errx = fmt.Errorf(message, args...)
	} else {
		errx = errorx.Decorate(err, message, args...)
	}
	log.Println(errx.Error())
	return errx
}

func parseIPv4(text string) (net.IP, error) {
	result := net.ParseIP(text)
	if result == nil {
		return nil, fmt.Errorf("%s is not an IP address", text)
	}
	if result.To4() == nil {
		return nil, fmt.Errorf("%s is not an IPv4 address", text)
	}
	return result.To4(), nil
}
