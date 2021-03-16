// +build amd64 386 arm arm64 mipsle mips64le ppc64le

// This file is an adapted version of github.com/josharian/native.

package aghos

import "encoding/binary"

// NativeEndian is the native endianness of this system.
var NativeEndian = binary.LittleEndian
