package upstream

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/mholt/caddy"
	"log"
)

func init() {
	caddy.RegisterPlugin("upstream", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

// Read the configuration and initialize upstreams
func setup(c *caddy.Controller) error {

	p, err := setupPlugin(c)
	if err != nil {
		return err
	}
	config := dnsserver.GetConfig(c)
	config.AddPlugin(func(next plugin.Handler) plugin.Handler {
		p.Next = next
		return p
	})

	c.OnShutdown(p.onShutdown)
	return nil
}

// Read the configuration
func setupPlugin(c *caddy.Controller) (*UpstreamPlugin, error) {

	p := New()

	log.Println("Initializing the Upstream plugin")

	bootstrap := ""
	upstreamUrls := []string{}
	for c.Next() {
		args := c.RemainingArgs()
		if len(args) > 0 {
			upstreamUrls = append(upstreamUrls, args...)
		}
		for c.NextBlock() {
			switch c.Val() {
			case "bootstrap":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				bootstrap = c.Val()
			}
		}
	}

	for _, url := range upstreamUrls {
		u, err := NewUpstream(url, bootstrap)
		if err != nil {
			log.Printf("Cannot initialize upstream %s", url)
			return nil, err
		}

		p.Upstreams = append(p.Upstreams, u)
	}

	return p, nil
}

func (p *UpstreamPlugin) onShutdown() error {
	for i := range p.Upstreams {

		u := p.Upstreams[i]
		err := u.Close()
		if err != nil {
			log.Printf("Error while closing the upstream: %s", err)
		}
	}

	return nil
}
