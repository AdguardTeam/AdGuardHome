package dnsforward

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestRecursionDetector_Check(t *testing.T) {
	rd := newRecursionDetector(0, 2)

	const (
		recID  = 1234
		recTTL = time.Hour * 100
	)

	const nonRecID = recID * 2

	sampleQuestion := dns.Question{
		Name:  "some.domain",
		Qtype: dns.TypeAAAA,
	}
	sampleMsg := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id: recID,
		},
		Question: []dns.Question{sampleQuestion},
	}

	// Manually add the message with big ttl.
	key := msgToSignature(sampleMsg)
	expire := make([]byte, uint64sz)
	binary.BigEndian.PutUint64(expire, uint64(time.Now().Add(recTTL).UnixNano()))
	rd.recentRequests.Set(key, expire)

	// Add an expired message.
	sampleMsg.Id = nonRecID
	rd.add(sampleMsg)

	testCases := []struct {
		name      string
		questions []dns.Question
		id        uint16
		want      bool
	}{{
		name:      "recurrent",
		questions: []dns.Question{sampleQuestion},
		id:        recID,
		want:      true,
	}, {
		name:      "not_suspected",
		questions: []dns.Question{sampleQuestion},
		id:        recID + 1,
		want:      false,
	}, {
		name:      "expired",
		questions: []dns.Question{sampleQuestion},
		id:        nonRecID,
		want:      false,
	}, {
		name:      "empty",
		questions: []dns.Question{},
		id:        nonRecID,
		want:      false,
	}}

	for _, tc := range testCases {
		sampleMsg.Id = tc.id
		sampleMsg.Question = tc.questions
		t.Run(tc.name, func(t *testing.T) {
			detected := rd.check(sampleMsg)
			assert.Equal(t, tc.want, detected)
		})
	}
}

func TestRecursionDetector_Suspect(t *testing.T) {
	rd := newRecursionDetector(0, 1)

	testCases := []struct {
		name string
		msg  dns.Msg
		want int
	}{{
		name: "simple",
		msg: dns.Msg{
			MsgHdr: dns.MsgHdr{
				Id: 1234,
			},
			Question: []dns.Question{{
				Name:  "some.domain",
				Qtype: dns.TypeA,
			}},
		},
		want: 1,
	}, {
		name: "unencumbered",
		msg:  dns.Msg{},
		want: 0,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(rd.clear)
			rd.add(tc.msg)
			assert.Equal(t, tc.want, rd.recentRequests.Stats().Count)
		})
	}
}

var sink []byte

func BenchmarkMsgToSignature(b *testing.B) {
	const name = "some.not.very.long.host.name"

	msg := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id: 1234,
		},
		Question: []dns.Question{{
			Name:  name,
			Qtype: dns.TypeAAAA,
		}},
	}

	b.Run("efficient", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			sink = msgToSignature(msg)
		}

		assert.NotEmpty(b, sink)
	})

	b.Run("inefficient", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			sink = msgToSignatureSlow(msg)
		}

		assert.NotEmpty(b, sink)
	})
}
