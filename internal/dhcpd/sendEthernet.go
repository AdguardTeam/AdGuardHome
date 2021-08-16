// Copyright 2018-present the CoreDHCP Authors. All rights reserved
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package dhcpd

import (
	"fmt"
	"net"
	"syscall"

	"github.com/AdguardTeam/golibs/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

//this function sends an unicast to the hardware address defined in resp.ClientHWAddr,
//the layer3 destination address is still the broadcast address;
//iface: the interface where the DHCP message should be sent;
//resp: DHCPv4 struct, which should be sent;
func sendEthernet(iface net.Interface, resp *dhcpv4.DHCPv4) error {
	eth := layers.Ethernet{
		EthernetType: layers.EthernetTypeIPv4,
		SrcMAC:       iface.HardwareAddr,
		DstMAC:       resp.ClientHWAddr,
	}
	ip := layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    resp.ServerIPAddr,
		DstIP:    resp.YourIPAddr,
		Protocol: layers.IPProtocolUDP,
		Flags:    layers.IPv4DontFragment,
	}
	udp := layers.UDP{
		SrcPort: dhcpv4.ServerPort,
		DstPort: dhcpv4.ClientPort,
	}

	err := udp.SetNetworkLayerForChecksum(&ip)
	if err != nil {
		return fmt.Errorf("Send Ethernet: Couldn't set network layer: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	// Decode a packet
	packet := gopacket.NewPacket(resp.ToBytes(), layers.LayerTypeDHCPv4, gopacket.NoCopy)
	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	dhcp, ok := dhcpLayer.(gopacket.SerializableLayer)
	if !ok {
		return fmt.Errorf("Layer %s is not serializable", dhcpLayer.LayerType().String())
	}
	err = gopacket.SerializeLayers(buf, opts, &eth, &ip, &udp, dhcp)
	if err != nil {
		return fmt.Errorf("Cannot serialize layer: %v", err)
	}
	data := buf.Bytes()

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, 0)
	if err != nil {
		return fmt.Errorf("Send Ethernet: Cannot open socket: %v", err)
	}
	defer func() {
		err = syscall.Close(fd)
		if err != nil {
			log.Error("Send Ethernet: Cannot close socket: %v", err)
		}
	}()

	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		log.Error("Send Ethernet: Cannot set option for socket: %v", err)
	}

	var hwAddr [8]byte
	copy(hwAddr[0:6], resp.ClientHWAddr[0:6])
	ethAddr := syscall.SockaddrLinklayer{
		Protocol: 0,
		Ifindex:  iface.Index,
		Halen:    6,
		Addr:     hwAddr, //not used
	}
	err = syscall.Sendto(fd, data, 0, &ethAddr)
	if err != nil {
		return fmt.Errorf("Cannot send frame via socket: %v", err)
	}
	return nil
}
