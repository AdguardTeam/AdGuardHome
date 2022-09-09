package websvc

import (
	"net"
	"sync"
)

// Wait Listener

// waitListener is a wrapper around a listener that also calls wg.Done() on the
// first call to Accept.  It is useful in situations where it is important to
// catch the precise moment of the first call to Accept, for example when
// starting an HTTP server.
//
// TODO(a.garipov): Move to aghnet?
type waitListener struct {
	net.Listener

	firstAcceptWG   *sync.WaitGroup
	firstAcceptOnce sync.Once
}

// type check
var _ net.Listener = (*waitListener)(nil)

// Accept implements the [net.Listener] interface for *waitListener.
func (l *waitListener) Accept() (conn net.Conn, err error) {
	l.firstAcceptOnce.Do(l.firstAcceptWG.Done)

	return l.Listener.Accept()
}
