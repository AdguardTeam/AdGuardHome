package client_test

import (
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/stretchr/testify/assert"
)

func TestRuntimeIndex(t *testing.T) {
	const cliSrc = client.SourceARP

	var (
		ip1 = netip.MustParseAddr("1.1.1.1")
		ip2 = netip.MustParseAddr("2.2.2.2")
		ip3 = netip.MustParseAddr("3.3.3.3")
	)

	ri := client.NewRuntimeIndex()
	currentSize := 0

	testCases := []struct {
		ip    netip.Addr
		name  string
		hosts []string
		src   client.Source
	}{{
		src:   cliSrc,
		ip:    ip1,
		name:  "1",
		hosts: []string{"host1"},
	}, {
		src:   cliSrc,
		ip:    ip2,
		name:  "2",
		hosts: []string{"host2"},
	}, {
		src:   cliSrc,
		ip:    ip3,
		name:  "3",
		hosts: []string{"host3"},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rc := client.NewRuntime(tc.ip)
			rc.SetInfo(tc.src, tc.hosts)

			ri.Add(rc)
			currentSize++

			got := ri.Client(tc.ip)
			assert.Equal(t, rc, got)
		})
	}

	t.Run("size", func(t *testing.T) {
		assert.Equal(t, currentSize, ri.Size())
	})

	t.Run("range", func(t *testing.T) {
		s := 0

		ri.Range(func(rc *client.Runtime) (cont bool) {
			s++

			return true
		})

		assert.Equal(t, currentSize, s)
	})

	t.Run("delete", func(t *testing.T) {
		ri.Delete(ip1)
		currentSize--

		assert.Equal(t, currentSize, ri.Size())
	})

	t.Run("delete_by_src", func(t *testing.T) {
		assert.Equal(t, currentSize, ri.DeleteBySource(cliSrc))
		assert.Equal(t, 0, ri.Size())
	})
}
