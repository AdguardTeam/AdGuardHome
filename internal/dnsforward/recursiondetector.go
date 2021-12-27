package dnsforward

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/miekg/dns"
)

// uint* sizes in bytes to improve readability.
//
// TODO(e.burkov): Remove when there will be a more regardful way to define
// those.  See https://github.com/golang/go/issues/29982.
const (
	uint16sz = 2
	uint64sz = 8
)

// recursionDetector detects recursion in DNS forwarding.
type recursionDetector struct {
	recentRequests cache.Cache
	ttl            time.Duration
}

// check checks if the passed req was already sent by the server.
func (rd *recursionDetector) check(msg dns.Msg) (ok bool) {
	if len(msg.Question) == 0 {
		return false
	}

	key := msgToSignature(msg)
	expireData := rd.recentRequests.Get(key)
	if expireData == nil {
		return false
	}

	expire := time.Unix(0, int64(binary.BigEndian.Uint64(expireData)))

	return time.Now().Before(expire)
}

// add caches the msg if it has anything in the questions section.
func (rd *recursionDetector) add(msg dns.Msg) {
	now := time.Now()

	if len(msg.Question) == 0 {
		return
	}

	key := msgToSignature(msg)
	expire64 := uint64(now.Add(rd.ttl).UnixNano())
	expire := make([]byte, uint64sz)
	binary.BigEndian.PutUint64(expire, expire64)

	rd.recentRequests.Set(key, expire)
}

// clear clears the recent requests cache.
func (rd *recursionDetector) clear() {
	rd.recentRequests.Clear()
}

// newRecursionDetector returns the initialized *recursionDetector.
func newRecursionDetector(ttl time.Duration, suspectsNum uint) (rd *recursionDetector) {
	return &recursionDetector{
		recentRequests: cache.New(cache.Config{
			EnableLRU: true,
			MaxCount:  suspectsNum,
		}),
		ttl: ttl,
	}
}

// msgToSignature converts msg into it's signature represented in bytes.
func msgToSignature(msg dns.Msg) (sig []byte) {
	sig = make([]byte, uint16sz*2+netutil.MaxDomainNameLen)
	// The binary.BigEndian byte order is used everywhere except when the real
	// machine's endianness is needed.
	byteOrder := binary.BigEndian
	byteOrder.PutUint16(sig[0:], msg.Id)
	q := msg.Question[0]
	byteOrder.PutUint16(sig[uint16sz:], q.Qtype)
	copy(sig[2*uint16sz:], []byte(q.Name))

	return sig
}

// msgToSignatureSlow converts msg into it's signature represented in bytes in
// the less efficient way.
//
// See BenchmarkMsgToSignature.
func msgToSignatureSlow(msg dns.Msg) (sig []byte) {
	type msgSignature struct {
		name  [netutil.MaxDomainNameLen]byte
		id    uint16
		qtype uint16
	}

	b := bytes.NewBuffer(sig)
	q := msg.Question[0]
	signature := msgSignature{
		id:    msg.Id,
		qtype: q.Qtype,
	}
	copy(signature.name[:], q.Name)
	if err := binary.Write(b, binary.BigEndian, signature); err != nil {
		log.Debug("writing message signature: %s", err)
	}

	return b.Bytes()
}
