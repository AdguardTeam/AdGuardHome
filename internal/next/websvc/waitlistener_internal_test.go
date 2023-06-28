package websvc

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
)

func TestWaitListener_Accept(t *testing.T) {
	var accepted atomic.Bool
	var l net.Listener = &aghtest.Listener{
		OnAccept: func() (conn net.Conn, err error) {
			accepted.Store(true)

			return nil, nil
		},
		OnAddr:  func() (addr net.Addr) { panic("not implemented") },
		OnClose: func() (err error) { panic("not implemented") },
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		var wrapper net.Listener = &waitListener{
			Listener:      l,
			firstAcceptWG: wg,
		}

		_, _ = wrapper.Accept()
	}()

	wg.Wait()

	assert.Eventually(t, accepted.Load, testTimeout, testTimeout/10)
}
