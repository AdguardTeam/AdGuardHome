package websvc

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
)

func TestWaitListener_Accept(t *testing.T) {
	// TODO(a.garipov): use atomic.Bool in Go 1.19.
	var numAcceptCalls uint32
	var l net.Listener = &aghtest.Listener{
		OnAccept: func() (conn net.Conn, err error) {
			atomic.AddUint32(&numAcceptCalls, 1)

			return nil, nil
		},
		OnAddr: func() (addr net.Addr) {
			panic("not implemented")
		},
		OnClose: func() (err error) {
			panic("not implemented")
		},
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
}
