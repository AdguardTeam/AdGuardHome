package home

import (
	"github.com/joomcode/errorx"
)

func startDHCPServer() error {
	if !config.DHCP.Enabled {
		// not enabled, don't do anything
		return nil
	}

	err := config.dhcpServer.Init(config.DHCP)
	if err != nil {
		return errorx.Decorate(err, "Couldn't init DHCP server")
	}

	err = config.dhcpServer.Start()
	if err != nil {
		return errorx.Decorate(err, "Couldn't start DHCP server")
	}
	return nil
}

func stopDHCPServer() error {
	if !config.DHCP.Enabled {
		return nil
	}

	err := config.dhcpServer.Stop()
	if err != nil {
		return errorx.Decorate(err, "Couldn't stop DHCP server")
	}

	return nil
}
