package dnsforward

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/joomcode/errorx"
	"github.com/miekg/dns"
)

const defaultTimeout = time.Second * 10

type Upstream interface {
	Exchange(m *dns.Msg) (*dns.Msg, error)
}

//
// plain DNS
//
type plainDNS struct {
	Address string
}

var defaultUDPClient = dns.Client{
	Timeout: defaultTimeout,
	UDPSize: dns.MaxMsgSize,
}

var defaultTCPClient = dns.Client{
	Net:     "tcp",
	UDPSize: dns.MaxMsgSize,
	Timeout: defaultTimeout,
}

func (p *plainDNS) Exchange(m *dns.Msg) (*dns.Msg, error) {
	reply, _, err := defaultUDPClient.Exchange(m, p.Address)
	if err != nil && reply != nil && reply.Truncated {
		log.Printf("Truncated message was received, retrying over TCP, question: %s", m.Question[0].String())
		reply, _, err = defaultTCPClient.Exchange(m, p.Address)
	}
	return reply, err
}

//
// DNS-over-TLS
//
type dnsOverTLS struct {
	Address string
	pool    *TLSPool

	sync.RWMutex // protects pool
}

var defaultTLSClient = dns.Client{
	Net:       "tcp-tls",
	Timeout:   defaultTimeout,
	UDPSize:   dns.MaxMsgSize,
	TLSConfig: &tls.Config{},
}

func (p *dnsOverTLS) Exchange(m *dns.Msg) (*dns.Msg, error) {
	var pool *TLSPool
	p.RLock()
	pool = p.pool
	p.RUnlock()
	if pool == nil {
		p.Lock()
		// lazy initialize it
		p.pool = &TLSPool{Address: p.Address}
		p.Unlock()
	}

	p.RLock()
	poolConn, err := p.pool.Get()
	p.RUnlock()
	if err != nil {
		return nil, errorx.Decorate(err, "Failed to get a connection from TLSPool to %s", p.Address)
	}
	c := dns.Conn{Conn: poolConn}
	err = c.WriteMsg(m)
	if err != nil {
		poolConn.Close()
		return nil, errorx.Decorate(err, "Failed to send a request to %s", p.Address)
	}

	reply, err := c.ReadMsg()
	if err != nil {
		poolConn.Close()
		return nil, errorx.Decorate(err, "Failed to read a request from %s", p.Address)
	}
	p.RLock()
	p.pool.Put(poolConn)
	p.RUnlock()
	return reply, nil
}

//
// DNS-over-https
//
type dnsOverHTTPS struct {
	Address string
}

var defaultHTTPSTransport = http.Transport{}

var defaultHTTPSClient = http.Client{
	Transport: &defaultHTTPSTransport,
	Timeout:   defaultTimeout,
}

func (p *dnsOverHTTPS) Exchange(m *dns.Msg) (*dns.Msg, error) {
	buf, err := m.Pack()
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't pack request msg")
	}
	bb := bytes.NewBuffer(buf)
	resp, err := http.Post(p.Address, "application/dns-message", bb)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't do a POST request to '%s'", p.Address)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't read body contents for '%s'", p.Address)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Got an unexpected HTTP status code %d from '%s'", resp.StatusCode, p.Address)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("Got an unexpected empty body from '%s'", p.Address)
	}
	response := dns.Msg{}
	err = response.Unpack(body)
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't unpack DNS response from '%s': body is %s", p.Address, string(body))
	}
	return &response, nil
}

func (s *Server) chooseUpstream() Upstream {
	upstreams := s.Upstreams
	if upstreams == nil {
		upstreams = defaultValues.Upstreams
	}
	if len(upstreams) == 0 {
		panic("SHOULD NOT HAPPEN: no default upstreams specified")
	}
	if len(upstreams) == 1 {
		return upstreams[0]
	}
	n := rand.Intn(len(upstreams))
	upstream := upstreams[n]
	return upstream
}

func GetUpstream(address string) (Upstream, error) {
	if strings.Contains(address, "://") {
		url, err := url.Parse(address)
		if err != nil {
			return nil, errorx.Decorate(err, "Failed to parse %s", address)
		}
		switch url.Scheme {
		case "dns":
			return &plainDNS{Address: address}, nil
		case "tls":
			return &dnsOverTLS{Address: address}, nil
		case "https":
			return &dnsOverHTTPS{Address: address}, nil
		default:
			return &plainDNS{Address: address}, nil
		}
	}

	// we don't have scheme in the url, so it's just a plain DNS host:port
	return &plainDNS{Address: address}, nil
}
