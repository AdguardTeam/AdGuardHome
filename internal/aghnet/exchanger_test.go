package aghnet

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiAddrExchanger(t *testing.T) {
	var e Exchanger
	var err error

	t.Run("empty", func(t *testing.T) {
		e, err = NewMultiAddrExchanger([]string{}, 0)
		require.NoError(t, err)
		assert.NotNil(t, e)
	})

	t.Run("successful", func(t *testing.T) {
		e, err = NewMultiAddrExchanger([]string{"www.example.com"}, 0)
		require.NoError(t, err)
		assert.NotNil(t, e)
	})

	t.Run("unsuccessful", func(t *testing.T) {
		e, err = NewMultiAddrExchanger([]string{"invalid-proto://www.example.com"}, 0)
		require.Error(t, err)
		assert.Nil(t, e)
	})
}

func TestMultiAddrExchanger_Exchange(t *testing.T) {
	e := &multiAddrExchanger{}

	t.Run("error", func(t *testing.T) {
		e.ups = []upstream.Upstream{&aghtest.TestErrUpstream{}}

		resp, err := e.Exchange(nil)
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("success", func(t *testing.T) {
		e.ups = []upstream.Upstream{&aghtest.TestUpstream{
			Reverse: map[string][]string{
				"abc": {"cba"},
			},
		}}

		resp, err := e.Exchange(&dns.Msg{
			Question: []dns.Question{{
				Name:  "abc",
				Qtype: dns.TypePTR,
			}},
		})
		require.NoError(t, err)
		require.Len(t, resp.Answer, 1)
		assert.Equal(t, "cba", resp.Answer[0].Header().Name)
	})
}
