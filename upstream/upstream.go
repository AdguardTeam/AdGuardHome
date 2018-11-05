package upstream

import (
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"log"
	"runtime"
	"time"
)

const (
	defaultTimeout = 5 * time.Second
)

// TODO: Add a helper method for health-checking an upstream (see health.go in coredns)

// Upstream is a simplified interface for proxy destination
type Upstream interface {
	Exchange(ctx context.Context, query *dns.Msg) (*dns.Msg, error)
	Close() error
}

// UpstreamPlugin is a simplified DNS proxy using a generic upstream interface
type UpstreamPlugin struct {
	Upstreams []Upstream
	Next      plugin.Handler
}

// Initialize the upstream plugin
func New() *UpstreamPlugin {
	p := &UpstreamPlugin{}

	// Make sure all resources are cleaned up
	runtime.SetFinalizer(p, (*UpstreamPlugin).finalizer)
	return p
}

// ServeDNS implements interface for CoreDNS plugin
func (p *UpstreamPlugin) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	var reply *dns.Msg
	var backendErr error

	for i := range p.Upstreams {
		upstream := p.Upstreams[i]
		reply, backendErr = upstream.Exchange(ctx, r)
		if backendErr == nil {
			w.WriteMsg(reply)
			return 0, nil
		}
	}

	return dns.RcodeServerFailure, errors.Wrap(backendErr, "failed to contact any of the upstreams")
}

// Name implements interface for CoreDNS plugin
func (p *UpstreamPlugin) Name() string { return "upstream" }

func (p *UpstreamPlugin) finalizer() {

	for i := range p.Upstreams {

		u := p.Upstreams[i]
		err := u.Close()
		if err != nil {
			log.Printf("Error while closing the upstream: %s", err)
		}
	}
}