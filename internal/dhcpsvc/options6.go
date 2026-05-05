package dhcpsvc

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/AdguardTeam/golibs/validate"
	"github.com/google/gopacket/layers"
)

// iaNAMinLen is the minimum length of an IA_NA option data field, in bytes.
//
// See RFC 9915 Section 21.4.
const iaNAMinLen = 12

// iaNAOption represents a parsed IA_NA (Identity Association for Non-temporary
// Addresses) option.
//
// See RFC 9915 Section 21.4.
type iaNAOption struct {
	// nested are the IA Address options nested within this IA_NA.
	nested []iaAddrOption

	// iaid is the Identity Association IDentifier, a 4-octet value uniquely
	// identifying this IA within the client.
	iaid uint32

	// t1 is the time after which the client must contact the same server to
	// extend the lifetimes of the addresses in this IA.
	t1 time.Duration

	// t2 is the time after which the client may contact any available server to
	// extend the lifetimes.
	t2 time.Duration
}

// type check
var _ encoding.BinaryUnmarshaler = (*iaNAOption)(nil)

// UnmarshalBinary implements the [encoding.BinaryUnmarshaler] interface for
// *iaNAOption.  data should have the following format:
//
//	 0                   1                   2                   3
//	 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                        IAID (4 octets)                        |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                              T1                               |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                              T2                               |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                                                               |
//	.                         IA_NA-options                         .
//	.                                                               .
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
func (opt *iaNAOption) UnmarshalBinary(data []byte) (err error) {
	err = validate.NoLessThan("data length", len(data), iaNAMinLen)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	opt.iaid = binary.BigEndian.Uint32(data[0:4])
	opt.t1 = time.Duration(binary.BigEndian.Uint32(data[4:8])) * time.Second
	opt.t2 = time.Duration(binary.BigEndian.Uint32(data[8:12])) * time.Second

	// Parse the nested options that follow the fixed fields.
	nested := data[iaNAMinLen:]
	for i := 0; len(nested) >= 4; i++ {
		code := layers.DHCPv6Opt(binary.BigEndian.Uint16(nested[0:2]))
		l := int(binary.BigEndian.Uint16(nested[2:4]))

		err = validate.NoGreaterThan("nested option length", l, len(nested)-4)
		if err != nil {
			return fmt.Errorf("nested option at index %d: %w", i, err)
		}

		if code == layers.DHCPv6OptIAAddr {
			addr := iaAddrOption{}
			err = addr.UnmarshalBinary(nested[4 : 4+l])
			if err != nil {
				return fmt.Errorf("nested ia_addr at index %d: %w", i, err)
			}

			opt.nested = append(opt.nested, addr)
		}

		nested = nested[4+l:]
	}

	return nil
}

// Encode serializes ia into a DHCPv6 IA_NA option.  Each contained
// [iaAddrOption] is encoded as a nested IA Address option.
//
// TODO(e.burkov):  Use.
func (opt iaNAOption) Encode() (iaOpt layers.DHCPv6Option) {
	// Each nested IA Address option: code (2) + length (2) + data (24).
	const nestedAddrSize = 2 + 2 + iaAddrDataLen

	data := make([]byte, 0, iaNAMinLen+len(opt.nested)*nestedAddrSize)

	data = binary.BigEndian.AppendUint32(data, opt.iaid)
	data = binary.BigEndian.AppendUint32(data, uint32(opt.t1.Seconds()))
	data = binary.BigEndian.AppendUint32(data, uint32(opt.t2.Seconds()))

	for _, addr := range opt.nested {
		data = addr.append(data)
	}

	return layers.NewDHCPv6Option(layers.DHCPv6OptIANA, data)
}

// iaAddrDataLen is the minimum length of an IA Address option data field, which
// is encoded [iaAddrOption], in bytes, excluding any nested options.  It
// consists of the IPv6 address (16 bytes) and the preferred and valid lifetimes
// (4 bytes each).
const iaAddrDataLen = 24

// iaAddrOption represents a parsed IA Address option.
//
// See RFC 9915 Section 21.6.
type iaAddrOption struct {
	// addr is the IPv6 address.
	addr netip.Addr

	// preferredLifetime is the preferred lifetime of the address.  When it is
	// zero, the address is deprecated.
	preferredLifetime time.Duration

	// validLifetime is the valid lifetime of the address.  When it is zero, the
	// address is no longer valid.
	validLifetime time.Duration
}

// type check
var _ encoding.BinaryUnmarshaler = (*iaAddrOption)(nil)

// UnmarshalBinary implements the [encoding.BinaryUnmarshaler] interface for
// *iaAddrOption.  Nested options within IA Address, if any, are
// ignored.  data should have the following format:
//
//	 0                   1                   2                   3
//	 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                                                               |
//	|                         IPv6-address                          |
//	|                                                               |
//	|                                                               |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                      preferred-lifetime                       |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                        valid-lifetime                         |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	.                                                               .
//	.                        IAaddr-options                         .
//	.                                                               .
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
func (ia *iaAddrOption) UnmarshalBinary(data []byte) (err error) {
	err = validate.NoLessThan("data length", len(data), iaAddrDataLen)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	var ok bool
	ia.addr, ok = netip.AddrFromSlice(data[0:16])
	if !ok {
		return fmt.Errorf("ia_addr: invalid ipv6 address bytes")
	}

	ia.preferredLifetime = time.Duration(binary.BigEndian.Uint32(data[16:20])) * time.Second
	ia.validLifetime = time.Duration(binary.BigEndian.Uint32(data[20:24])) * time.Second

	return nil
}

// append returns the data portion of the IA Address option encoding,
// suitable for use as a nested option inside an IA_NA.
func (ia iaAddrOption) append(orig []byte) (data []byte) {
	data = orig

	data = binary.BigEndian.AppendUint16(data, uint16(layers.DHCPv6OptIAAddr))
	data = binary.BigEndian.AppendUint16(data, uint16(iaAddrDataLen))

	// [netip.Addr.AppendBinary] never returns errors.
	data, _ = ia.addr.AppendBinary(data)

	data = binary.BigEndian.AppendUint32(data, uint32(ia.preferredLifetime.Seconds()))
	data = binary.BigEndian.AppendUint32(data, uint32(ia.validLifetime.Seconds()))

	return data
}

// newServerDUID creates a DUID-LL (Link-Layer Address) from the given MAC
// address per RFC 9915 §11.4.  The result is deterministic: the same MAC
// address always produces the same DUID, satisfying the stability requirement
// of §11.
func newServerDUID(mac net.HardwareAddr) (duid *layers.DHCPv6DUID) {
	return &layers.DHCPv6DUID{
		Type:             layers.DHCPv6DUIDTypeLL,
		HardwareType:     HardwareTypeEthernet,
		LinkLayerAddress: mac,
	}
}
