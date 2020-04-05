package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/golibs/log"
	"github.com/krolaw/dhcp4"
)

func main() {
	if len(os.Args) < 2 {
		log.Printf("Usage: %s <interface name>", os.Args[0])
		os.Exit(64)
	}

	ifaceName := os.Args[1]
	present, err := dhcpd.CheckIfOtherDHCPServersPresent(ifaceName)
	if err != nil {
		panic(err)
	}
	log.Printf("Found DHCP server? %v", present)
	if present {
		log.Printf("Will not start DHCP server because there's already running one on the network")
		os.Exit(1)
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		panic(err)
	}

	// get ipv4 address of an interface
	ifaceIPNet := getIfaceIPv4(iface)
	if ifaceIPNet == nil {
		panic(err)
	}

	// append 10 to server's IP address as start
	start := dhcp4.IPAdd(ifaceIPNet.IP, 10)
	// lease range is 100 IP's, but TODO: don't go beyond end of subnet mask
	stop := dhcp4.IPAdd(start, 100)

	server := dhcpd.Server{}
	config := dhcpd.ServerConfig{
		InterfaceName: ifaceName,
		RangeStart:    start.String(),
		RangeEnd:      stop.String(),
		SubnetMask:    "255.255.255.0",
		GatewayIP:     "192.168.7.1",
	}
	log.Printf("Starting DHCP server")
	err = server.Init(config)
	if err != nil {
		panic(err)
	}
	err = server.Start()
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)
	log.Printf("Stopping DHCP server")
	err = server.Stop()
	if err != nil {
		panic(err)
	}
	log.Printf("Starting DHCP server")
	err = server.Start()
	if err != nil {
		panic(err)
	}
	log.Printf("Starting DHCP server while it's already running")
	err = server.Start()
	if err != nil {
		panic(err)
	}
	log.Printf("Now serving DHCP")
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	<-signalChannel

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
