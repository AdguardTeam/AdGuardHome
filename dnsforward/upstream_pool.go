package dnsforward

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sync"

	"github.com/joomcode/errorx"
)

// upstream TLS pool.
//
// Example:
// pool := TLSPool{Address: "tls://1.1.1.1:853"}
// netConn, err := pool.Get()
// if err != nil {panic(err)}
// c := dns.Conn{Conn: netConn}
// q := dns.Msg{}
// q.SetQuestion("google.com.", dns.TypeA)
// log.Println(q)
// err = c.WriteMsg(&q)
// if err != nil {panic(err)}
// r, err := c.ReadMsg()
// if err != nil {panic(err)}
// log.Println(r)
// pool.Put(c.Conn)
type TLSPool struct {
	Address            string
	parsedAddress      *url.URL
	parsedAddressMutex sync.RWMutex

	conns      []net.Conn
	sync.Mutex // protects conns
}

func (n *TLSPool) getHost() (string, error) {
	n.parsedAddressMutex.RLock()
	if n.parsedAddress != nil {
		n.parsedAddressMutex.RUnlock()
		return n.parsedAddress.Host, nil
	}
	n.parsedAddressMutex.RUnlock()

	n.parsedAddressMutex.Lock()
	defer n.parsedAddressMutex.Unlock()
	url, err := url.Parse(n.Address)
	if err != nil {
		return "", errorx.Decorate(err, "Failed to parse %s", n.Address)
	}
	if url.Scheme != "tls" {
		return "", fmt.Errorf("TLSPool only supports TLS")
	}
	n.parsedAddress = url
	return n.parsedAddress.Host, nil
}

func (n *TLSPool) Get() (net.Conn, error) {
	host, err := n.getHost()
	if err != nil {
		return nil, err
	}

	// get the connection from the slice inside the lock
	var c net.Conn
	n.Lock()
	num := len(n.conns)
	if num > 0 {
		last := num - 1
		c = n.conns[last]
		n.conns = n.conns[:last]
	}
	n.Unlock()

	// if we got connection from the slice, return it
	if c != nil {
		// log.Printf("Returning existing connection to %s", host)
		return c, nil
	}

	// we'll need a new connection, dial now
	// log.Printf("Dialing to %s", host)
	conn, err := tls.Dial("tcp", host, nil)
	if err != nil {
		return nil, errorx.Decorate(err, "Failed to connect to %s", host)
	}
	return conn, nil
}

func (n *TLSPool) Put(c net.Conn) {
	if c == nil {
		return
	}
	n.Lock()
	n.conns = append(n.conns, c)
	n.Unlock()
}
