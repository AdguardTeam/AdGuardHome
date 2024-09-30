package dnsforward

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/ipset"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/miekg/dns"
)

// ipsetHandler is the ipset context.  ipsetMgr can be nil.
type ipsetHandler struct {
	ipsetMgr ipset.Manager
	logger   *slog.Logger
}

// newIpsetHandler returns a new initialized [ipsetHandler].  It is not safe for
// concurrent use.
func newIpsetHandler(
	ctx context.Context,
	logger *slog.Logger,
	ipsetList []string,
) (h *ipsetHandler, err error) {
	h = &ipsetHandler{
		logger: logger,
	}
	conf := &ipset.Config{
		Logger: logger,
		Lines:  ipsetList,
	}
	h.ipsetMgr, err = ipset.NewManager(ctx, conf)
	if errors.Is(err, os.ErrInvalid) ||
		errors.Is(err, os.ErrPermission) ||
		errors.Is(err, errors.ErrUnsupported) {
		// ipset cannot currently be initialized if the server was installed
		// from Snap or when the user or the binary doesn't have the required
		// permissions, or when the kernel doesn't support netfilter.
		//
		// Log and go on.
		//
		// TODO(a.garipov): The Snap problem can probably be solved if we add
		// the netlink-connector interface plug.
		logger.WarnContext(ctx, "cannot initialize", slogutil.KeyError, err)

		return h, nil
	} else if err != nil {
		return nil, fmt.Errorf("initializing ipset: %w", err)
	}

	return h, nil
}

// close closes the Linux Netfilter connections.  close can be called on a nil
// handler.
func (h *ipsetHandler) close() (err error) {
	if h != nil && h.ipsetMgr != nil {
		return h.ipsetMgr.Close()
	}

	return nil
}

// dctxIsFilled returns true if dctx has enough information to process.
func dctxIsFilled(dctx *dnsContext) (ok bool) {
	return dctx != nil &&
		dctx.responseFromUpstream &&
		dctx.proxyCtx != nil &&
		dctx.proxyCtx.Res != nil &&
		dctx.proxyCtx.Req != nil &&
		len(dctx.proxyCtx.Req.Question) > 0
}

// skipIpsetProcessing returns true when the ipset processing can be skipped for
// this request.
func (h *ipsetHandler) skipIpsetProcessing(dctx *dnsContext) (ok bool) {
	if h == nil || h.ipsetMgr == nil || !dctxIsFilled(dctx) {
		return true
	}

	qtype := dctx.proxyCtx.Req.Question[0].Qtype

	return qtype != dns.TypeA && qtype != dns.TypeAAAA && qtype != dns.TypeANY
}

// ipFromRR returns an IP address from a DNS resource record.
func ipFromRR(rr dns.RR) (ip net.IP) {
	switch a := rr.(type) {
	case *dns.A:
		return a.A
	case *dns.AAAA:
		return a.AAAA
	default:
		return nil
	}
}

// ipsFromAnswer returns IPv4 and IPv6 addresses from a DNS answer.
func ipsFromAnswer(ans []dns.RR) (ip4s, ip6s []net.IP) {
	for _, rr := range ans {
		ip := ipFromRR(rr)
		if ip == nil {
			continue
		}

		if ip.To4() == nil {
			ip6s = append(ip6s, ip)

			continue
		}

		ip4s = append(ip4s, ip)
	}

	return ip4s, ip6s
}

// process adds the resolved IP addresses to the domain's ipsets, if any.
func (h *ipsetHandler) process(dctx *dnsContext) (rc resultCode) {
	// TODO(s.chzhen):  Use passed context.
	ctx := context.TODO()
	h.logger.DebugContext(ctx, "started processing")
	defer h.logger.DebugContext(ctx, "finished processing")

	if h.skipIpsetProcessing(dctx) {
		return resultCodeSuccess
	}

	req := dctx.proxyCtx.Req
	host := req.Question[0].Name
	host = strings.TrimSuffix(host, ".")
	host = strings.ToLower(host)

	ip4s, ip6s := ipsFromAnswer(dctx.proxyCtx.Res.Answer)
	n, err := h.ipsetMgr.Add(ctx, host, ip4s, ip6s)
	if err != nil {
		// Consider ipset errors non-critical to the request.
		h.logger.ErrorContext(ctx, "adding host ips", slogutil.KeyError, err)

		return resultCodeSuccess
	}

	h.logger.DebugContext(ctx, "added new ipset entries", "num", n)

	return resultCodeSuccess
}
