// +build integration

package dnsforward

import (
	"errors"
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
	"github.com/vishvananda/netns"
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

const useNetns bool = true
const netnsName string = "aghTest"
const netnsBindMountPath string = "/run/netns" // should be exported by netns...
const netnsPath string = netnsBindMountPath + "/" + netnsName

func runCommandNetns(args ...string) error {
	netnsCmd := make([]string, len(args), len(args)+2)
	if useNetns {
		netnsCmd[0] = "nsenter"
		netnsCmd[1] = "-n" + netnsPath
		netnsCmd = append(append(netnsCmd, ""), "")
		copy(netnsCmd[2:], args)
	} else {
		copy(netnsCmd, args)
	}
	code, _, err := util.RunCommand(netnsCmd[0], netnsCmd[1:]...)
	if err != nil {
		return err
	}
	if code != 0 {
		return errors.New(fmt.Sprintf("exit code %d", code))
	}
	return nil
}

func makeNlConfigMaybeInNetns() (*netlink.Config, error) {
	if useNetns {
		newns, err := netns.NewNamed(netnsName)
		if err != nil {
			return nil, err
		} else {
			return &netlink.Config{NetNS: int(newns)}, nil
		}
	} else {
		return &netlink.Config{}, nil
	}
}

func (s *state) doIpsetCreate(ipsetName string, ipv6 bool) {
	family := "inet"
	if ipv6 {
		family = "inet6"
	}
	err := runCommandNetns("ipset", "create", ipsetName, "hash:ip", "family", family)
	if err != nil {
		panic(err)
	}
	s.activeIpsets = append(s.activeIpsets, ipsetName)
}

func (s *state) doIpsetFlush() {
	for _, ipsetName := range s.activeIpsets {
		err := runCommandNetns("ipset", "flush", ipsetName)
		if err != nil {
			panic(err)
		}
	}
}

var ipsetConfigs = []string{
		"HOST.com/aghTestHost",
		"host2.com,host3.com/aghTestHost23",
		"host4.com/aghTestHost4,aghTestHost4-6",
		"sub.host4.com/aghTestSubhost4",
}

func withSetup(testFn func(*state)) {
	s := &state{}
	s.activeIpsets = make([]string, 0, 5)
	s.server.conf.IPSETList = ipsetConfigs

	nlConfig, err := makeNlConfigMaybeInNetns()
	if err != nil {
		panic(err)
	}

	// make sure we (try to) clean up the netns and/or any ipsets
	defer func() {
		errs := []error{}
		fails := []string{}
		for _, ipsetName := range s.activeIpsets {
			err := runCommandNetns("ipset", "destroy", ipsetName)
			if err != nil {
				errs = append(errs, err)
				fails = append(fails, ipsetName)
			}
		}

		if useNetns {
			err := netns.DeleteNamed(netnsName)
			if err != nil {
				errs = append(errs, err)
			} else {
				errs = []error{}
				fails = []string{}
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

	s.doIpsetCreate("aghTestHost", false)
	s.doIpsetCreate("aghTestHost23", false)
	s.doIpsetCreate("aghTestHost4", false)
	s.doIpsetCreate("aghTestHost4-6", true)
	s.doIpsetCreate("aghTestSubhost4", false)

	err = s.c.init(s.server.conf.IPSETList, nlConfig)
	if err != nil {
		panic(err)
	}

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
		err := runCommandNetns("ipset", "add", set.name, ip.String())
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
	var cmd *exec.Cmd
	if useNetns {
		cmdArgs = append([]string{"-n" + netnsPath}, cmdArgs...)
		cmd = exec.Command("nsenter", cmdArgs...)
	} else {
		cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	}
	cmd.Run()
	return cmd.ProcessState.ExitCode() == 0
}

func ipsetV4(name string) ipsetProps {
	return ipsetProps{name, netfilter.ProtoIPv4}
}

func ipsetV6(name string) ipsetProps {
	return ipsetProps{name, netfilter.ProtoIPv6}
}

func TestIpsetParsing(t *testing.T) {
	withSetup(func(s *state) {
		assert.Equal(t, ipsetV4("aghTestHost"), s.c.domainMap["host.com"][0])
		assert.Equal(t, ipsetV4("aghTestHost23"), s.c.domainMap["host2.com"][0])
		assert.Equal(t, ipsetV4("aghTestHost23"), s.c.domainMap["host3.com"][0])
		assert.Equal(t, ipsetV4("aghTestHost4"), s.c.domainMap["host4.com"][0])
		assert.Equal(t, ipsetV6("aghTestHost4-6"), s.c.domainMap["host4.com"][1])

		_, ok := s.c.domainMap["host0.com"]
		assert.False(t, ok)
	})
}

func TestIpsetNoQuestion(t *testing.T) {
	withSetup(func(s *state) {
		b := map[binding]int{}
		s.doProcess(t, b)
		assert.Equal(t, 0, len(b))
	})
}

func TestIpsetNoAnswer(t *testing.T) {
	withSetup(func(s *state) {
		s.ctx.proxyCtx.Req = makeReqA("HOST4.COM.")

		b := map[binding]int{}
		s.doProcess(t, b)
		assert.Equal(t, 0, len(b))
	})
}

func TestIpsetCache(t *testing.T) {
	withSetup(func(s *state) {
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
	withSetup(func(s *state) {
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
	withSetup(func(s *state) {
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
	withSetup(func(s *state) {
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
	withSetup(func(s *state) {
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

func generateIpv4Addrs(n int) []net.IP {
	addrs := make([]net.IP, n)
	for i := 0; i < n; i++ {
		addrs[i] = net.IPv4(1, 2, 3, byte(i))
	}
	return addrs
}

func makeDomainWithSubs(root string, subCount int) string {
	domain := root
	for i := 0; i < subCount; i++ {
		domain = "x." + domain
	}
	return domain
}

func (s *state) setupCtxForBenchmark(addrCount int, subCount int) {
	rrs := make([]dns.RR, addrCount)
	domain := makeDomainWithSubs("host.com.", subCount)
	for i, ip := range generateIpv4Addrs(addrCount) {
		rrs[i] = makeA(domain, ip)
	}

	s.ctx.proxyCtx.Req = makeReqA(domain)
	s.ctx.proxyCtx.Res = &dns.Msg{
		Answer: rrs,
	}
}

func benchmarkIpset(b *testing.B, addrCount int, subCount int,
	addEntries func(*ipsetCtx, string, ipsetProps, []net.IP), reset func(*state)) {
	b.StopTimer()
	b.ResetTimer()

	withSetup(func(s *state) {
		s.setupCtxForBenchmark(addrCount, subCount)

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
	benchmarkIpset(b, n, 0, addWithIpsetCmd, resetIpsetContent)
}

func benchmarkIpsetNf(b *testing.B, n int) {
	benchmarkIpset(b, n, 0, addToIpset, resetIpsetContent)
}

func benchmarkIpsetCache(b *testing.B, n int) {
	benchmarkIpset(b, n, 0, addToBindings(map[binding]int{}), func(s *state) {})
}

func benchmarkIpsetLookup(b *testing.B, n int) {
	benchmarkIpset(b, n, n, addToBindings(map[binding]int{}), func(s *state) {})
}

func BenchmarkIpsetCmd1(b *testing.B)     { benchmarkIpsetCmd(b, 1) }
func BenchmarkIpsetCmd10(b *testing.B)    { benchmarkIpsetCmd(b, 10) }
func BenchmarkIpsetNf1(b *testing.B)      { benchmarkIpsetNf(b, 1) }
func BenchmarkIpsetNf10(b *testing.B)     { benchmarkIpsetNf(b, 10) }
func BenchmarkIpsetCache10(b *testing.B)  { benchmarkIpsetCache(b, 10) }
func BenchmarkIpsetLookup10(b *testing.B) { benchmarkIpsetLookup(b, 10) }
