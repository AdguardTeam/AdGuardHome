// +build linux
// +build integration

package dnsforward

import (
	"fmt"
	"net"
	"os/exec"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/mdlayher/netlink"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/ti-mo/netfilter"
)

type binding struct {
	host  string
	ipset string
	ipStr string
}

type state struct {
	server Server
	c      ipsetCtx
	ctx    *dnsContext

	activeIpsets []string
}

func (s *state) doIpsetCreate(ipsetName string, ipv6 bool, comment bool) {
	family := "inet"
	if ipv6 {
		family = "inet6"
	}

	args := []string{"create", ipsetName, "hash:ip", "family", family}

	if comment {
		args = append(args, "comment")
	}

	_, _, err := util.RunCommand("ipset", args...)
	if err != nil {
		panic(err)
	}
	s.activeIpsets = append(s.activeIpsets, ipsetName)
}

func (s *state) doIpsetFlush() {
	for _, ipsetName := range s.activeIpsets {
		_, _, err := util.RunCommand("ipset", "flush", ipsetName)
		if err != nil {
			panic(err)
		}
	}
}

func (s *state) doIpsetGetComment(ipsetName string, addr net.IP) string {
	sets, err := s.c.ipv4Conn.ListAll()
	if err != nil {
		panic(err)
	}
	for _, set := range sets {
		if set.Name.Get() == ipsetName {
			for _, entry := range set.Entries {
				if entry.IP.Get().Equal(addr) {
					return entry.Comment.Get()
				}
			}
		}
	}
	return ""
}

var ipsetConfigs = []string{
	"HOST.com/aghTestHost",
	"host2.com,host3.com/aghTestHost23",
	"host4.com/aghTestHost4,aghTestHost4-6",
	"sub.host4.com/aghTestSubhost4",
}

func withSetup(configs []string, testFn func(*state)) {
	if configs == nil {
		configs = ipsetConfigs
	}

	s := &state{}
	s.activeIpsets = make([]string, 0, 5)
	s.server.conf.IPSETList = configs

	// make sure we (try to) clean up the test ipsets
	defer func() {
		errs := []error{}
		fails := []string{}
		for _, ipsetName := range s.activeIpsets {
			_, _, err := util.RunCommand("ipset", "destroy", ipsetName)
			if err != nil {
				errs = append(errs, err)
				fails = append(fails, ipsetName)
			}
		}

		if len(errs) != 0 {
			msg := ""
			for _, err := range errs {
				msg += fmt.Sprintf("%s\n", err)
			}
			if len(fails) != 0 {
				msg += fmt.Sprintf("leaked ipsets: %v", fails)
			}
			panic(msg)
		}
	}()

	s.doIpsetCreate("aghTestHost", false, true)
	s.doIpsetCreate("aghTestHost23", false, false)
	s.doIpsetCreate("aghTestHost4", false, false)
	s.doIpsetCreate("aghTestHost4-6", true, false)
	s.doIpsetCreate("aghTestSubhost4", false, false)

	err := s.c.init(s.server.conf.IPSETList, &netlink.Config{})
	if err != nil {
		panic(err)
	}
	defer s.c.Uninit()

	s.ctx = &dnsContext{
		srv: &s.server,
	}
	s.ctx.responseFromUpstream = true
	s.ctx.proxyCtx = &proxy.DNSContext{}

	testFn(s)
}

func makeReq(fqdn string, qtype uint16) *dns.Msg {
	return &dns.Msg{
		Question: []dns.Question{
			{
				Name:  fqdn,
				Qtype: qtype,
			},
		},
	}
}

func makeReqA(fqdn string) *dns.Msg {
	return makeReq(fqdn, dns.TypeA)
}

func makeReqAAAA(fqdn string) *dns.Msg {
	return makeReq(fqdn, dns.TypeAAAA)
}

func makeA(fqdn string, ip net.IP) *dns.A {
	return &dns.A{
		Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
		A:   ip,
	}
}

func makeAAAA(fqdn string, ip net.IP) *dns.AAAA {
	return &dns.AAAA{
		Hdr:  dns.RR_Header{Name: fqdn, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
		AAAA: ip,
	}
}

func makeCNAME(fqdn string, cnameFqdn string) *dns.CNAME {
	return &dns.CNAME{
		Hdr:    dns.RR_Header{Name: fqdn, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 0},
		Target: cnameFqdn,
	}
}

func addToBindings(b map[binding]int) func(*ipsetCtx, string, ipsetProps, []net.IP) {
	return func(_ *ipsetCtx, host string, set ipsetProps, ips []net.IP) {
		for _, ip := range ips {
			bind := binding{host, set.name, ip.String()}
			count := b[bind]
			b[bind] = count + 1
		}
	}
}

// This is only used for benchmarking as an alternate implementation comparison
func addWithIpsetCmd(_ *ipsetCtx, host string, set ipsetProps, ips []net.IP) {
	for _, ip := range ips {
		_, _, err := util.RunCommand("ipset", "add", set.name, ip.String())
		if err != nil {
			panic(err)
		}
	}
}

func (s *state) doProcess(t *testing.T, b map[binding]int) {
	assert.Equal(t, resultDone, s.c.processEntries(s.ctx, addToBindings(b)))
}

func (s *state) doSystem(t *testing.T) {
	assert.Equal(t, resultDone, s.c.process(s.ctx))
}

func isInIpset(t *testing.T, ipsetName string, ip net.IP) bool {
	cmdArgs := []string{"ipset", "test", ipsetName, ip.String()}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Run()
	return cmd.ProcessState.ExitCode() == 0
}

func ipsetV4(name string, comment bool) ipsetProps {
	return ipsetProps{name, netfilter.ProtoIPv4, comment}
}

func ipsetV6(name string, comment bool) ipsetProps {
	return ipsetProps{name, netfilter.ProtoIPv6, comment}
}

func TestIpsetParsing(t *testing.T) {
	withSetup(nil, func(s *state) {
		assert.Equal(t, ipsetV4("aghTestHost", true), s.c.domainMap["host.com"][0])
		assert.Equal(t, ipsetV4("aghTestHost23", false), s.c.domainMap["host2.com"][0])
		assert.Equal(t, ipsetV4("aghTestHost23", false), s.c.domainMap["host3.com"][0])
		assert.Equal(t, ipsetV4("aghTestHost4", false), s.c.domainMap["host4.com"][0])
		assert.Equal(t, ipsetV6("aghTestHost4-6", false), s.c.domainMap["host4.com"][1])

		_, ok := s.c.domainMap["host0.com"]
		assert.False(t, ok)
	})
}

func TestIpsetNoQuestion(t *testing.T) {
	withSetup(nil, func(s *state) {
		b := map[binding]int{}
		s.doProcess(t, b)
		assert.Equal(t, 0, len(b))
	})
}

func TestIpsetNoAnswer(t *testing.T) {
	withSetup(nil, func(s *state) {
		s.ctx.proxyCtx.Req = makeReqA("HOST4.COM.")

		b := map[binding]int{}
		s.doProcess(t, b)
		assert.Equal(t, 0, len(b))
	})
}

func TestIpsetCache(t *testing.T) {
	withSetup(nil, func(s *state) {
		s.ctx.proxyCtx.Req = makeReqA("HOST4.COM.")
		s.ctx.proxyCtx.Res = &dns.Msg{
			Answer: []dns.RR{
				makeA("HOST4.COM.", net.IPv4(127, 0, 0, 1)),
				makeAAAA("HOST4.COM.", net.IPv6loopback),
			},
		}

		b := map[binding]int{}
		s.doProcess(t, b)

		assert.Equal(t, 1, b[binding{"host4.com", "aghTestHost4", "127.0.0.1"}])
		assert.Equal(t, 1, b[binding{"host4.com", "aghTestHost4-6", net.IPv6loopback.String()}])
		assert.Equal(t, 2, len(b))

		s.doProcess(t, b)

		assert.Equal(t, 1, b[binding{"host4.com", "aghTestHost4", "127.0.0.1"}])
		assert.Equal(t, 1, b[binding{"host4.com", "aghTestHost4-6", net.IPv6loopback.String()}])
		assert.Equal(t, 2, len(b))
	})
}

func TestIpsetSubdomainOverride(t *testing.T) {
	withSetup(nil, func(s *state) {
		s.ctx.proxyCtx.Req = makeReqA("sub.host4.com.")
		s.ctx.proxyCtx.Res = &dns.Msg{
			Answer: []dns.RR{
				makeA("sub.host4.com.", net.IPv4(127, 0, 0, 1)),
			},
		}

		b := map[binding]int{}
		s.doProcess(t, b)

		assert.Equal(t, 1, b[binding{"sub.host4.com", "aghTestSubhost4", "127.0.0.1"}])
		assert.Equal(t, 1, len(b))
	})
}

func TestIpsetSubdomainWildcard(t *testing.T) {
	withSetup(nil, func(s *state) {
		s.ctx.proxyCtx.Req = makeReqA("sub.host.com.")
		s.ctx.proxyCtx.Res = &dns.Msg{
			Answer: []dns.RR{
				makeA("sub.host.com.", net.IPv4(127, 0, 0, 1)),
			},
		}

		b := map[binding]int{}
		s.doProcess(t, b)

		assert.Equal(t, 1, b[binding{"sub.host.com", "aghTestHost", "127.0.0.1"}])
		assert.Equal(t, 1, len(b))
	})
}

func TestIpsetCnameThirdParty(t *testing.T) {
	withSetup(nil, func(s *state) {
		s.ctx.proxyCtx.Req = makeReqA("host.com.")
		s.ctx.proxyCtx.Res = &dns.Msg{
			Answer: []dns.RR{
				makeCNAME("host.com.", "foo.bar.baz.elb.amazonaws.com."),
				makeA("foo.bar.baz.elb.amazonaws.com.", net.IPv4(8, 8, 8, 8)),
			},
		}

		b := map[binding]int{}
		s.doProcess(t, b)

		assert.Equal(t, 1, b[binding{"host.com", "aghTestHost", "8.8.8.8"}])
		assert.Equal(t, 1, len(b))
	})
}

func TestIpsetAdd(t *testing.T) {
	withSetup(nil, func(s *state) {
		ips := []net.IP{
			net.IPv4(1, 2, 3, 4),
			net.IPv4(5, 6, 7, 8),
			net.ParseIP("123:4567:89ab:cdef:fedc:ba98:7654:3210"),
		}
		rrs := []dns.RR{}
		for _, ip := range ips {
			if ip.To4() == nil {
				rrs = append(rrs, makeAAAA("host4.com.", ip))
			} else {
				rrs = append(rrs, makeA("host4.com.", ip))
			}
		}

		s.ctx.proxyCtx.Req = makeReqA("host4.com.")
		s.ctx.proxyCtx.Res = &dns.Msg{
			Answer: rrs,
		}

		for _, ip := range ips {
			if ip.To4() == nil {
				assert.False(t, isInIpset(t, "aghTestHost4-6", ip))
			} else {
				assert.False(t, isInIpset(t, "aghTestHost4", ip))
			}
		}
		s.doSystem(t)
		for _, ip := range ips {
			if ip.To4() == nil {
				assert.True(t, isInIpset(t, "aghTestHost4-6", ip))
			} else {
				assert.True(t, isInIpset(t, "aghTestHost4", ip))
			}
		}
	})
}

func TestIpsetComment(t *testing.T) {
	withSetup(nil, func(s *state) {
		domainName := "requested.subdomain.host.com"
		ip := net.IPv4(1, 2, 3, 5)
		s.ctx.proxyCtx.Req = makeReqA(domainName + ".")
		s.ctx.proxyCtx.Res = &dns.Msg{
			Answer: []dns.RR{
				makeA("a.subdomain.not.requested.host.com.", ip),
			},
		}

		s.doSystem(t)
		assert.Equal(t, domainName, s.doIpsetGetComment("aghTestHost", ip))
	})
}

func generateIpv4Addrs(n int) []net.IP {
	addrs := make([]net.IP, n)
	for i := 0; i < n; i++ {
		addrs[i] = net.IPv4(1, 2, 3, byte(i))
	}
	return addrs
}

func generateIpsetConfigStrings(n int) []string {
	configs := make([]string, n)
	for i := 0; i < n; i++ {
		configs[i] = fmt.Sprintf("domain-%d.com/aghTestHost", i)
	}
	return configs
}

func makeDomainWithSubs(root string, subCount int) string {
	domain := root
	for i := 0; i < subCount; i++ {
		domain = "x." + domain
	}
	return domain
}

func makeSetupBasicCtx(domain string, addrCount int, subCount int) func(*state) {
	return func(s *state) {
		rrs := make([]dns.RR, addrCount)
		domain := makeDomainWithSubs(domain, subCount)
		for i, ip := range generateIpv4Addrs(addrCount) {
			rrs[i] = makeA(domain, ip)
		}

		s.ctx.proxyCtx.Req = makeReqA(domain)
		s.ctx.proxyCtx.Res = &dns.Msg{
			Answer: rrs,
		}
	}
}

func makeSetupCachedCtx(addrCount int, subCount int) func(*state) {
	return func(s *state) {
		makeSetupBasicCtx("host.com.", addrCount, subCount)(s)
		s.c.processEntries(s.ctx, addToBindings(map[binding]int{}))
	}
}

func makeSetupUnboundCtx(addrCount int, subCount int) func(*state) {
	return func(s *state) {
		makeSetupBasicCtx("example.net.", addrCount, subCount)(s)
	}
}

func benchmarkIpset(b *testing.B, configs []string, setupCtx func(*state),
	addEntries func(*ipsetCtx, string, ipsetProps, []net.IP), reset func(*state)) {
	b.StopTimer()
	b.ResetTimer()

	withSetup(configs, func(s *state) {
		setupCtx(s)

		for i := 0; i < b.N; i++ {
			reset(s)
			b.StartTimer()
			s.c.processEntries(s.ctx, addEntries)
			b.StopTimer()
		}
	})
}

func resetIpsetContent(s *state) {
	s.doIpsetFlush()
	s.c.clearCache()
}

func benchmarkIpsetCmd(b *testing.B, n int) {
	benchmarkIpset(b, nil,
		makeSetupBasicCtx("host.com.", n, 0),
		addWithIpsetCmd,
		resetIpsetContent)
}

func benchmarkIpsetNf(b *testing.B, n int) {
	benchmarkIpset(b, nil,
		makeSetupBasicCtx("host.com.", n, 0),
		addToIpset,
		resetIpsetContent)
}

func benchmarkIpsetZero(b *testing.B, n int) {
	benchmarkIpset(b, []string{}, makeSetupBasicCtx("host.com.", n, 0), addToIpset, func(s *state) {})
}

func benchmarkIpsetCacheHit(b *testing.B, n int) {
	benchmarkIpset(b, nil, makeSetupCachedCtx(n, 0), addToBindings(map[binding]int{}), func(s *state) {})
}

func benchmarkIpsetUnbound(b *testing.B, n int, depth int) {
	benchmarkIpset(b, nil, makeSetupUnboundCtx(n, depth), addToBindings(map[binding]int{}), func(s *state) {})
}

func benchmarkIpsetUnboundBig(b *testing.B, n int, depth int) {
	benchmarkIpset(b,
		generateIpsetConfigStrings(1024),
		makeSetupUnboundCtx(n, depth),
		addToBindings(map[binding]int{}),
		func(s *state) {})
}

func BenchmarkIpsetCmd1(b *testing.B)  { benchmarkIpsetCmd(b, 1) }
func BenchmarkIpsetCmd10(b *testing.B) { benchmarkIpsetCmd(b, 10) }
func BenchmarkIpsetNf1(b *testing.B)   { benchmarkIpsetNf(b, 1) }
func BenchmarkIpsetNf10(b *testing.B)  { benchmarkIpsetNf(b, 10) }

func BenchmarkIpsetZero1(b *testing.B)              { benchmarkIpsetZero(b, 1) }
func BenchmarkIpsetCacheHit1(b *testing.B)          { benchmarkIpsetCacheHit(b, 1) }
func BenchmarkIpsetUnboundShallow1(b *testing.B)    { benchmarkIpsetUnbound(b, 1, 0) }
func BenchmarkIpsetUnboundDeep1(b *testing.B)       { benchmarkIpsetUnbound(b, 1, 10) }
func BenchmarkIpsetUnboundShallowBig1(b *testing.B) { benchmarkIpsetUnboundBig(b, 1, 0) }
func BenchmarkIpsetUnboundDeepBig1(b *testing.B)    { benchmarkIpsetUnboundBig(b, 1, 10) }
