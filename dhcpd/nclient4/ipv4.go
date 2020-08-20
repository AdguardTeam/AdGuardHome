// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This file contains code taken from gVisor.

// +build go1.12

package nclient4

import (
	"encoding/binary"
	"net"

	"github.com/u-root/u-root/pkg/uio"
)

const (
	versIHL  = 0
	tos      = 1
	totalLen = 2
	id       = 4
	flagsFO  = 6
	ttl      = 8
	protocol = 9
	checksum = 10
	srcAddr  = 12
	dstAddr  = 16
)

// TransportProtocolNumber is the number of a transport protocol.
type TransportProtocolNumber uint32

// IPv4Fields contains the fields of an IPv4 packet. It is used to describe the
// fields of a packet that needs to be encoded.
type IPv4Fields struct {
	// IHL is the "internet header length" field of an IPv4 packet.
	IHL uint8

	// TOS is the "type of service" field of an IPv4 packet.
	TOS uint8

	// TotalLength is the "total length" field of an IPv4 packet.
	TotalLength uint16

	// ID is the "identification" field of an IPv4 packet.
	ID uint16

	// Flags is the "flags" field of an IPv4 packet.
	Flags uint8

	// FragmentOffset is the "fragment offset" field of an IPv4 packet.
	FragmentOffset uint16

	// TTL is the "time to live" field of an IPv4 packet.
	TTL uint8

	// Protocol is the "protocol" field of an IPv4 packet.
	Protocol uint8

	// Checksum is the "checksum" field of an IPv4 packet.
	Checksum uint16

	// SrcAddr is the "source ip address" of an IPv4 packet.
	SrcAddr net.IP

	// DstAddr is the "destination ip address" of an IPv4 packet.
	DstAddr net.IP
}

// IPv4 represents an ipv4 header stored in a byte array.
// Most of the methods of IPv4 access to the underlying slice without
// checking the boundaries and could panic because of 'index out of range'.
// Always call IsValid() to validate an instance of IPv4 before using other methods.
type IPv4 []byte

const (
	// IPv4MinimumSize is the minimum size of a valid IPv4 packet.
	IPv4MinimumSize = 20

	// IPv4MaximumHeaderSize is the maximum size of an IPv4 header. Given
	// that there are only 4 bits to represents the header length in 32-bit
	// units, the header cannot exceed 15*4 = 60 bytes.
	IPv4MaximumHeaderSize = 60

	// IPv4AddressSize is the size, in bytes, of an IPv4 address.
	IPv4AddressSize = 4

	// IPv4Version is the version of the ipv4 protocol.
	IPv4Version = 4
)

var (
	// IPv4Broadcast is the broadcast address of the IPv4 protocol.
	IPv4Broadcast = net.IP{0xff, 0xff, 0xff, 0xff}

	// IPv4Any is the non-routable IPv4 "any" meta address.
	IPv4Any = net.IP{0, 0, 0, 0}
)

// Flags that may be set in an IPv4 packet.
const (
	IPv4FlagMoreFragments = 1 << iota
	IPv4FlagDontFragment
)

// HeaderLength returns the value of the "header length" field of the ipv4
// header.
func (b IPv4) HeaderLength() uint8 {
	return (b[versIHL] & 0xf) * 4
}

// Protocol returns the value of the protocol field of the ipv4 header.
func (b IPv4) Protocol() uint8 {
	return b[protocol]
}

// SourceAddress returns the "source address" field of the ipv4 header.
func (b IPv4) SourceAddress() net.IP {
	return net.IP(b[srcAddr : srcAddr+IPv4AddressSize])
}

// DestinationAddress returns the "destination address" field of the ipv4
// header.
func (b IPv4) DestinationAddress() net.IP {
	return net.IP(b[dstAddr : dstAddr+IPv4AddressSize])
}

// TransportProtocol implements Network.TransportProtocol.
func (b IPv4) TransportProtocol() TransportProtocolNumber {
	return TransportProtocolNumber(b.Protocol())
}

// Payload implements Network.Payload.
func (b IPv4) Payload() []byte {
	return b[b.HeaderLength():][:b.PayloadLength()]
}

// PayloadLength returns the length of the payload portion of the ipv4 packet.
func (b IPv4) PayloadLength() uint16 {
	return b.TotalLength() - uint16(b.HeaderLength())
}

// TotalLength returns the "total length" field of the ipv4 header.
func (b IPv4) TotalLength() uint16 {
	return binary.BigEndian.Uint16(b[totalLen:])
}

// SetTotalLength sets the "total length" field of the ipv4 header.
func (b IPv4) SetTotalLength(totalLength uint16) {
	binary.BigEndian.PutUint16(b[totalLen:], totalLength)
}

// SetChecksum sets the checksum field of the ipv4 header.
func (b IPv4) SetChecksum(v uint16) {
	binary.BigEndian.PutUint16(b[checksum:], v)
}

// SetFlagsFragmentOffset sets the "flags" and "fragment offset" fields of the
// ipv4 header.
func (b IPv4) SetFlagsFragmentOffset(flags uint8, offset uint16) {
	v := (uint16(flags) << 13) | (offset >> 3)
	binary.BigEndian.PutUint16(b[flagsFO:], v)
}

// SetSourceAddress sets the "source address" field of the ipv4 header.
func (b IPv4) SetSourceAddress(addr net.IP) {
	copy(b[srcAddr:srcAddr+IPv4AddressSize], addr.To4())
}

// SetDestinationAddress sets the "destination address" field of the ipv4
// header.
func (b IPv4) SetDestinationAddress(addr net.IP) {
	copy(b[dstAddr:dstAddr+IPv4AddressSize], addr.To4())
}

// CalculateChecksum calculates the checksum of the ipv4 header.
func (b IPv4) CalculateChecksum() uint16 {
	return Checksum(b[:b.HeaderLength()], 0)
}

// Encode encodes all the fields of the ipv4 header.
func (b IPv4) Encode(i *IPv4Fields) {
	b[versIHL] = (4 << 4) | ((i.IHL / 4) & 0xf)
	b[tos] = i.TOS
	b.SetTotalLength(i.TotalLength)
	binary.BigEndian.PutUint16(b[id:], i.ID)
	b.SetFlagsFragmentOffset(i.Flags, i.FragmentOffset)
	b[ttl] = i.TTL
	b[protocol] = i.Protocol
	b.SetChecksum(i.Checksum)
	copy(b[srcAddr:srcAddr+IPv4AddressSize], i.SrcAddr)
	copy(b[dstAddr:dstAddr+IPv4AddressSize], i.DstAddr)
}

const (
	udpSrcPort  = 0
	udpDstPort  = 2
	udpLength   = 4
	udpChecksum = 6
)

// UDPFields contains the fields of a UDP packet. It is used to describe the
// fields of a packet that needs to be encoded.
type UDPFields struct {
	// SrcPort is the "source port" field of a UDP packet.
	SrcPort uint16

	// DstPort is the "destination port" field of a UDP packet.
	DstPort uint16

	// Length is the "length" field of a UDP packet.
	Length uint16

	// Checksum is the "checksum" field of a UDP packet.
	Checksum uint16
}

// UDP represents a UDP header stored in a byte array.
type UDP []byte

const (
	// UDPMinimumSize is the minimum size of a valid UDP packet.
	UDPMinimumSize = 8

	// UDPProtocolNumber is UDP's transport protocol number.
	UDPProtocolNumber TransportProtocolNumber = 17
)

// SourcePort returns the "source port" field of the udp header.
func (b UDP) SourcePort() uint16 {
	return binary.BigEndian.Uint16(b[udpSrcPort:])
}

// DestinationPort returns the "destination port" field of the udp header.
func (b UDP) DestinationPort() uint16 {
	return binary.BigEndian.Uint16(b[udpDstPort:])
}

// Length returns the "length" field of the udp header.
func (b UDP) Length() uint16 {
	return binary.BigEndian.Uint16(b[udpLength:])
}

// SetSourcePort sets the "source port" field of the udp header.
func (b UDP) SetSourcePort(port uint16) {
	binary.BigEndian.PutUint16(b[udpSrcPort:], port)
}

// SetDestinationPort sets the "destination port" field of the udp header.
func (b UDP) SetDestinationPort(port uint16) {
	binary.BigEndian.PutUint16(b[udpDstPort:], port)
}

// SetChecksum sets the "checksum" field of the udp header.
func (b UDP) SetChecksum(checksum uint16) {
	binary.BigEndian.PutUint16(b[udpChecksum:], checksum)
}

// Payload returns the data contained in the UDP datagram.
func (b UDP) Payload() []byte {
	return b[UDPMinimumSize:]
}

// Checksum returns the "checksum" field of the udp header.
func (b UDP) Checksum() uint16 {
	return binary.BigEndian.Uint16(b[udpChecksum:])
}

// CalculateChecksum calculates the checksum of the udp packet, given the total
// length of the packet and the checksum of the network-layer pseudo-header
// (excluding the total length) and the checksum of the payload.
func (b UDP) CalculateChecksum(partialChecksum uint16, totalLen uint16) uint16 {
	// Add the length portion of the checksum to the pseudo-checksum.
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, totalLen)
	checksum := Checksum(tmp, partialChecksum)

	// Calculate the rest of the checksum.
	return Checksum(b[:UDPMinimumSize], checksum)
}

// Encode encodes all the fields of the udp header.
func (b UDP) Encode(u *UDPFields) {
	binary.BigEndian.PutUint16(b[udpSrcPort:], u.SrcPort)
	binary.BigEndian.PutUint16(b[udpDstPort:], u.DstPort)
	binary.BigEndian.PutUint16(b[udpLength:], u.Length)
	binary.BigEndian.PutUint16(b[udpChecksum:], u.Checksum)
}

func calculateChecksum(buf []byte, initial uint32) uint16 {
	v := initial

	l := len(buf)
	if l&1 != 0 {
		l--
		v += uint32(buf[l]) << 8
	}

	for i := 0; i < l; i += 2 {
		v += (uint32(buf[i]) << 8) + uint32(buf[i+1])
	}

	return ChecksumCombine(uint16(v), uint16(v>>16))
}

// Checksum calculates the checksum (as defined in RFC 1071) of the bytes in the
// given byte array.
//
// The initial checksum must have been computed on an even number of bytes.
func Checksum(buf []byte, initial uint16) uint16 {
	return calculateChecksum(buf, uint32(initial))
}

// ChecksumCombine combines the two uint16 to form their checksum. This is done
// by adding them and the carry.
//
// Note that checksum a must have been computed on an even number of bytes.
func ChecksumCombine(a, b uint16) uint16 {
	v := uint32(a) + uint32(b)
	return uint16(v + v>>16)
}

// PseudoHeaderChecksum calculates the pseudo-header checksum for the
// given destination protocol and network address, ignoring the length
// field. Pseudo-headers are needed by transport layers when calculating
// their own checksum.
func PseudoHeaderChecksum(protocol TransportProtocolNumber, srcAddr net.IP, dstAddr net.IP) uint16 {
	xsum := Checksum([]byte(srcAddr), 0)
	xsum = Checksum([]byte(dstAddr), xsum)
	return Checksum([]byte{0, uint8(protocol)}, xsum)
}

func udp4pkt(packet []byte, dest *net.UDPAddr, src *net.UDPAddr) []byte {
	ipLen := IPv4MinimumSize
	udpLen := UDPMinimumSize

	h := make([]byte, 0, ipLen+udpLen+len(packet))
	hdr := uio.NewBigEndianBuffer(h)

	ipv4fields := &IPv4Fields{
		IHL:         IPv4MinimumSize,
		TotalLength: uint16(ipLen + udpLen + len(packet)),
		TTL:         64, // Per RFC 1700's recommendation for IP time to live
		Protocol:    uint8(UDPProtocolNumber),
		SrcAddr:     src.IP.To4(),
		DstAddr:     dest.IP.To4(),
	}
	ipv4hdr := IPv4(hdr.WriteN(ipLen))
	ipv4hdr.Encode(ipv4fields)
	ipv4hdr.SetChecksum(^ipv4hdr.CalculateChecksum())

	udphdr := UDP(hdr.WriteN(udpLen))
	udphdr.Encode(&UDPFields{
		SrcPort: uint16(src.Port),
		DstPort: uint16(dest.Port),
		Length:  uint16(udpLen + len(packet)),
	})

	xsum := Checksum(packet, PseudoHeaderChecksum(
		ipv4hdr.TransportProtocol(), ipv4fields.SrcAddr, ipv4fields.DstAddr))
	udphdr.SetChecksum(^udphdr.CalculateChecksum(xsum, udphdr.Length()))

	hdr.WriteBytes(packet)
	return hdr.Data()
}
