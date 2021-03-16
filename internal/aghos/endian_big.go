// +build mips mips64

// This file is an adapted version of github.com/josharian/native.

package aghos

import "encoding/binary"

// NativeEndian is the native endianness of this system.
var NativeEndian = binary.BigEndian
