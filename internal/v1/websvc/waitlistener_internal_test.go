package websvc

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
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

	done := make(chan struct{})
	a := time.After(testTimeout)
	go func() {
		select {
		case <-a:
			panic(fmt.Errorf("did not finish after %s", testTimeout))
		case <-done:
			// Success.
		}
	}()

	go func() {
		var wrapper net.Listener = &waitListener{
			Listener:      l,
			firstAcceptWG: wg,
		}

		_, _ = wrapper.Accept()
	}()

	wg.Wait()
	close(done)

	assert.Equal(t, uint32(1), atomic.LoadUint32(&numAcceptCalls))
}
