package dnsforward

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
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
	Address() string
}

//
// plain DNS
//
type plainDNS struct {
	address   string
	preferTCP bool
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

func (p *plainDNS) Address() string { return p.address }

func (p *plainDNS) Exchange(m *dns.Msg) (*dns.Msg, error) {
	if p.preferTCP {
		reply, _, err := defaultTCPClient.Exchange(m, p.address)
		return reply, err
	}

	reply, _, err := defaultUDPClient.Exchange(m, p.address)
	if err != nil && reply != nil && reply.Truncated {
		log.Printf("Truncated message was received, retrying over TCP, question: %s", m.Question[0].String())
		reply, _, err = defaultTCPClient.Exchange(m, p.address)
	}
	return reply, err
}

//
// DNS-over-TLS
//
type dnsOverTLS struct {
	address string
	pool    *TLSPool

	sync.RWMutex // protects pool
}

var defaultTLSClient = dns.Client{
	Net:       "tcp-tls",
	Timeout:   defaultTimeout,
	UDPSize:   dns.MaxMsgSize,
	TLSConfig: &tls.Config{},
}

func (p *dnsOverTLS) Address() string { return p.address }

func (p *dnsOverTLS) Exchange(m *dns.Msg) (*dns.Msg, error) {
	var pool *TLSPool
	p.RLock()
	pool = p.pool
	p.RUnlock()
	if pool == nil {
		p.Lock()
		// lazy initialize it
		p.pool = &TLSPool{Address: p.address}
		p.Unlock()
	}

	p.RLock()
	poolConn, err := p.pool.Get()
	p.RUnlock()
	if err != nil {
		return nil, errorx.Decorate(err, "Failed to get a connection from TLSPool to %s", p.address)
	}
	c := dns.Conn{Conn: poolConn}
	err = c.WriteMsg(m)
	if err != nil {
		poolConn.Close()
		return nil, errorx.Decorate(err, "Failed to send a request to %s", p.address)
	}

	reply, err := c.ReadMsg()
	if err != nil {
		poolConn.Close()
		return nil, errorx.Decorate(err, "Failed to read a request from %s", p.address)
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
	address string
}

var defaultHTTPSTransport = http.Transport{}

var defaultHTTPSClient = http.Client{
	Transport: &defaultHTTPSTransport,
	Timeout:   defaultTimeout,
}

func (p *dnsOverHTTPS) Address() string { return p.address }

func (p *dnsOverHTTPS) Exchange(m *dns.Msg) (*dns.Msg, error) {
	buf, err := m.Pack()
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't pack request msg")
	}
	bb := bytes.NewBuffer(buf)
	resp, err := http.Post(p.address, "application/dns-message", bb)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't do a POST request to '%s'", p.address)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't read body contents for '%s'", p.address)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Got an unexpected HTTP status code %d from '%s'", resp.StatusCode, p.address)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("Got an unexpected empty body from '%s'", p.address)
	}
	response := dns.Msg{}
	err = response.Unpack(body)
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't unpack DNS response from '%s': body is %s", p.address, string(body))
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
			if url.Port() == "" {
				url.Host += ":53"
			}
			return &plainDNS{address: url.Host}, nil
		case "tcp":
			if url.Port() == "" {
				url.Host += ":53"
			}
			return &plainDNS{address: url.Host, preferTCP: true}, nil
		case "tls":
			if url.Port() == "" {
				url.Host += ":853"
			}
			return &dnsOverTLS{address: url.String()}, nil
		case "https":
			return &dnsOverHTTPS{address: url.String()}, nil
		default:
			// assume it's plain DNS
			if url.Port() == "" {
				url.Host += ":53"
			}
			return &plainDNS{address: url.String()}, nil
		}
	}

	// we don't have scheme in the url, so it's just a plain DNS host:port
	_, _, err := net.SplitHostPort(address)
	if err != nil {
		// doesn't have port, default to 53
		address = net.JoinHostPort(address, "53")
	}
	return &plainDNS{address: address}, nil
}
