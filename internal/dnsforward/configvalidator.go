package dnsforward

import (
	"fmt"
	"strings"
	"sync"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
)

// upstreamConfigValidator parses the [*proxy.UpstreamConfig] and checks the
// actual DNS availability of each upstream.
type upstreamConfigValidator struct {
	// general is the general upstream configuration.
	general []*upstreamResult

	// fallback is the fallback upstream configuration.
	fallback []*upstreamResult

	// private is the private upstream configuration.
	private []*upstreamResult
}

// upstreamResult is a result of validation of an [upstream.Upstream] within an
// [proxy.UpstreamConfig].
type upstreamResult struct {
	// server is the parsed upstream.  It is nil when there was an error during
	// parsing.
	server upstream.Upstream

	// err is the error either from parsing or from checking the upstream.
	err error

	// original is the piece of configuration that have either been turned to an
	// upstream or caused an error.
	original string

	// isSpecific is true if the upstream is domain-specific.
	isSpecific bool
}

// compare compares two [upstreamResult]s.  It returns 0 if they are equal, -1
// if ur should be sorted before other, and 1 otherwise.
//
// TODO(e.burkov):  Perhaps it makes sense to sort the results with errors near
// the end.
func (ur *upstreamResult) compare(other *upstreamResult) (res int) {
	return strings.Compare(ur.original, other.original)
}

// newUpstreamConfigValidator parses the upstream configuration and returns a
// validator for it.  cv already contains the parsed upstreams along with errors
// related.
func newUpstreamConfigValidator(
	general []string,
	fallback []string,
	private []string,
	opts *upstream.Options,
) (cv *upstreamConfigValidator) {
	cv = &upstreamConfigValidator{}

	for _, line := range general {
		cv.general = cv.insertLineResults(cv.general, line, opts)
	}
	for _, line := range fallback {
		cv.fallback = cv.insertLineResults(cv.fallback, line, opts)
	}
	for _, line := range private {
		cv.private = cv.insertLineResults(cv.private, line, opts)
	}

	return cv
}

// insertLineResults parses line and inserts the result into s.  It can insert
// multiple results as well as none.
func (cv *upstreamConfigValidator) insertLineResults(
	s []*upstreamResult,
	line string,
	opts *upstream.Options,
) (result []*upstreamResult) {
	upstreams, isSpecific, err := splitUpstreamLine(line)
	if err != nil {
		return cv.insert(s, &upstreamResult{
			err:      err,
			original: line,
		})
	}

	for _, upstreamAddr := range upstreams {
		var res *upstreamResult
		if upstreamAddr != "#" {
			res = cv.parseUpstream(upstreamAddr, opts)
		} else if !isSpecific {
			res = &upstreamResult{
				err:      errNotDomainSpecific,
				original: upstreamAddr,
			}
		} else {
			continue
		}

		res.isSpecific = isSpecific
		s = cv.insert(s, res)
	}

	return s
}

// insert inserts r into slice in a sorted order, except duplicates.  slice must
// not be nil.
func (cv *upstreamConfigValidator) insert(
	s []*upstreamResult,
	r *upstreamResult,
) (result []*upstreamResult) {
	i, has := slices.BinarySearchFunc(s, r, (*upstreamResult).compare)
	if has {
		log.Debug("dnsforward: duplicate configuration %q", r.original)

		return s
	}

	return slices.Insert(s, i, r)
}

// parseUpstream parses addr and returns the result of parsing.  It returns nil
// if the specified server points at the default upstream server which is
// validated separately.
func (cv *upstreamConfigValidator) parseUpstream(
	addr string,
	opts *upstream.Options,
) (r *upstreamResult) {
	// Check if the upstream has a valid protocol prefix.
	//
	// TODO(e.burkov):  Validate the domain name.
	if proto, _, ok := strings.Cut(addr, "://"); ok {
		if !slices.Contains(protocols, proto) {
			return &upstreamResult{
				err:      fmt.Errorf("bad protocol %q", proto),
				original: addr,
			}
		}
	}

	ups, err := upstream.AddressToUpstream(addr, opts)

	return &upstreamResult{
		server:   ups,
		err:      err,
		original: addr,
	}
}

// check tries to exchange with each successfully parsed upstream and enriches
// the results with the healthcheck errors.  It should not be called after the
// [upsConfValidator.close] method, since it makes no sense to check the closed
// upstreams.
func (cv *upstreamConfigValidator) check() {
	const (
		// testTLD is the special-use fully-qualified domain name for testing
		// the DNS server reachability.
		//
		// See https://datatracker.ietf.org/doc/html/rfc6761#section-6.2.
		testTLD = "test."

		// inAddrARPATLD is the special-use fully-qualified domain name for PTR
		// IP address resolution.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1035#section-3.5.
		inAddrARPATLD = "in-addr.arpa."
	)

	commonChecker := &healthchecker{
		hostname: testTLD,
		qtype:    dns.TypeA,
		ansEmpty: true,
	}

	arpaChecker := &healthchecker{
		hostname: inAddrARPATLD,
		qtype:    dns.TypePTR,
		ansEmpty: false,
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(cv.general) + len(cv.fallback) + len(cv.private))

	for _, res := range cv.general {
		go cv.checkSrv(res, wg, commonChecker)
	}
	for _, res := range cv.fallback {
		go cv.checkSrv(res, wg, commonChecker)
	}
	for _, res := range cv.private {
		go cv.checkSrv(res, wg, arpaChecker)
	}

	wg.Wait()
}

// checkSrv runs hc on the server from res, if any, and stores any occurred
// error in res.  wg is always marked done in the end.  It used to be called in
// a separate goroutine.
func (cv *upstreamConfigValidator) checkSrv(
	res *upstreamResult,
	wg *sync.WaitGroup,
	hc *healthchecker,
) {
	defer wg.Done()

	if res.server == nil {
		return
	}

	res.err = hc.check(res.server)
	if res.err != nil && res.isSpecific {
		res.err = domainSpecificTestError{Err: res.err}
	}
}

// close closes all the upstreams that were successfully parsed.  It enriches
// the results with deferred closing errors.
func (cv *upstreamConfigValidator) close() {
	for _, slice := range [][]*upstreamResult{cv.general, cv.fallback, cv.private} {
		for _, r := range slice {
			if r.server != nil {
				r.err = errors.WithDeferred(r.err, r.server.Close())
			}
		}
	}
}

// status returns all the data collected during parsing, healthcheck, and
// closing of the upstreams.  The returned map is keyed by the original upstream
// configuration piece and contains the corresponding error or "OK" if there was
// no error.
func (cv *upstreamConfigValidator) status() (results map[string]string) {
	result := map[string]string{}

	for _, res := range cv.general {
		resultToStatus("general", res, result)
	}
	for _, res := range cv.fallback {
		resultToStatus("fallback", res, result)
	}
	for _, res := range cv.private {
		resultToStatus("private", res, result)
	}

	return result
}

// resultToStatus puts "OK" or an error message from res into resMap.  section
// is the name of the upstream configuration section, i.e. "general",
// "fallback", or "private", and only used for logging.
//
// TODO(e.burkov):  Currently, the HTTP handler expects that all the results are
// put together in a single map, which may lead to collisions, see AG-27539.
// Improve the results compilation.
func resultToStatus(section string, res *upstreamResult, resMap map[string]string) {
	val := "OK"
	if res.err != nil {
		val = res.err.Error()
	}

	prevVal := resMap[res.original]
	switch prevVal {
	case "":
		resMap[res.original] = val
	case val:
		log.Debug("dnsforward: duplicating %s config line %q", section, res.original)
	default:
		log.Debug(
			"dnsforward: warning: %s config line %q (%v) had different result %v",
			section,
			val,
			res.original,
			prevVal,
		)
	}
}

// domainSpecificTestError is a wrapper for errors returned by checkDNS to mark
// the tested upstream domain-specific and therefore consider its errors
// non-critical.
//
// TODO(a.garipov):  Some common mechanism of distinguishing between errors and
// warnings (non-critical errors) is desired.
type domainSpecificTestError struct {
	// Err is the actual error occurred during healthcheck test.
	Err error
}

// type check
var _ error = domainSpecificTestError{}

// Error implements the [error] interface for domainSpecificTestError.
func (err domainSpecificTestError) Error() (msg string) {
	return fmt.Sprintf("WARNING: %s", err.Err)
}

// type check
var _ errors.Wrapper = domainSpecificTestError{}

// Unwrap implements the [errors.Wrapper] interface for domainSpecificTestError.
func (err domainSpecificTestError) Unwrap() (wrapped error) {
	return err.Err
}

// healthchecker checks the upstream's status by exchanging with it.
type healthchecker struct {
	// hostname is the name of the host to put into healthcheck DNS request.
	hostname string

	// qtype is the type of DNS request to use for healthcheck.
	qtype uint16

	// ansEmpty defines if the answer section within the response is expected to
	// be empty.
	ansEmpty bool
}

// check exchanges with u and validates the response.
func (h *healthchecker) check(u upstream.Upstream) (err error) {
	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   h.hostname,
			Qtype:  h.qtype,
			Qclass: dns.ClassINET,
		}},
	}

	reply, err := u.Exchange(req)
	if err != nil {
		return fmt.Errorf("couldn't communicate with upstream: %w", err)
	} else if h.ansEmpty && len(reply.Answer) > 0 {
		return errWrongResponse
	}

	return nil
}
