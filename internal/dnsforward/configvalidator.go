package dnsforward

import (
	"fmt"
	"sync"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// upstreamConfigValidator parses each section of an upstream configuration into
// a corresponding [*proxy.UpstreamConfig] and checks the actual DNS
// availability of each upstream.
type upstreamConfigValidator struct {
	// generalUpstreamResults contains upstream results of a general section.
	generalUpstreamResults map[string]*upstreamResult

	// fallbackUpstreamResults contains upstream results of a fallback section.
	fallbackUpstreamResults map[string]*upstreamResult

	// privateUpstreamResults contains upstream results of a private section.
	privateUpstreamResults map[string]*upstreamResult

	// generalParseResults contains parsing results of a general section.
	generalParseResults []*parseResult

	// fallbackParseResults contains parsing results of a fallback section.
	fallbackParseResults []*parseResult

	// privateParseResults contains parsing results of a private section.
	privateParseResults []*parseResult
}

// upstreamResult is a result of parsing of an [upstream.Upstream] within an
// [proxy.UpstreamConfig].
type upstreamResult struct {
	// server is the parsed upstream.
	server upstream.Upstream

	// err is the upstream check error.
	err error

	// isSpecific is true if the upstream is domain-specific.
	isSpecific bool
}

// parseResult contains a original piece of upstream configuration and a
// corresponding error.
type parseResult struct {
	err      *proxy.ParseError
	original string
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
	cv = &upstreamConfigValidator{
		generalUpstreamResults:  map[string]*upstreamResult{},
		fallbackUpstreamResults: map[string]*upstreamResult{},
		privateUpstreamResults:  map[string]*upstreamResult{},
	}

	conf, err := proxy.ParseUpstreamsConfig(general, opts)
	cv.generalParseResults = collectErrResults(general, err)
	insertConfResults(conf, cv.generalUpstreamResults)

	conf, err = proxy.ParseUpstreamsConfig(fallback, opts)
	cv.fallbackParseResults = collectErrResults(fallback, err)
	insertConfResults(conf, cv.fallbackUpstreamResults)

	conf, err = proxy.ParseUpstreamsConfig(private, opts)
	cv.privateParseResults = collectErrResults(private, err)
	insertConfResults(conf, cv.privateUpstreamResults)

	return cv
}

// collectErrResults parses err and returns parsing results containing the
// original upstream configuration line and the corresponding error.  err can be
// nil.
func collectErrResults(lines []string, err error) (results []*parseResult) {
	if err == nil {
		return nil
	}

	// limit is a maximum length for upstream configuration lines.
	const limit = 80

	wrapper, ok := err.(errors.WrapperSlice)
	if !ok {
		log.Debug("dnsforward: configvalidator: unwrapping: %s", err)

		return nil
	}

	errs := wrapper.Unwrap()
	results = make([]*parseResult, 0, len(errs))
	for i, e := range errs {
		var parseErr *proxy.ParseError
		if !errors.As(e, &parseErr) {
			log.Debug("dnsforward: configvalidator: inserting unexpected error %d: %s", i, err)

			continue
		}

		idx := parseErr.Idx
		line := []rune(lines[idx])
		if len(line) > limit {
			line = line[:limit]
			line[limit-1] = '…'
		}

		results = append(results, &parseResult{
			original: string(line),
			err:      parseErr,
		})
	}

	return results
}

// insertConfResults parses conf and inserts the upstream result into results.
// It can insert multiple results as well as none.
func insertConfResults(conf *proxy.UpstreamConfig, results map[string]*upstreamResult) {
	insertListResults(conf.Upstreams, results, false)

	for _, ups := range conf.DomainReservedUpstreams {
		insertListResults(ups, results, true)
	}

	for _, ups := range conf.SpecifiedDomainUpstreams {
		insertListResults(ups, results, true)
	}
}

// insertListResults constructs upstream results from the upstream list and
// inserts them into results.  It can insert multiple results as well as none.
func insertListResults(ups []upstream.Upstream, results map[string]*upstreamResult, specific bool) {
	for _, u := range ups {
		addr := u.Address()
		_, ok := results[addr]
		if ok {
			continue
		}

		results[addr] = &upstreamResult{
			server:     u,
			isSpecific: specific,
		}
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
	wg.Add(len(cv.generalUpstreamResults) +
		len(cv.fallbackUpstreamResults) +
		len(cv.privateUpstreamResults))

	for _, res := range cv.generalUpstreamResults {
		go checkSrv(res, wg, commonChecker)
	}
	for _, res := range cv.fallbackUpstreamResults {
		go checkSrv(res, wg, commonChecker)
	}
	for _, res := range cv.privateUpstreamResults {
		go checkSrv(res, wg, arpaChecker)
	}

	wg.Wait()
}

// checkSrv runs hc on the server from res, if any, and stores any occurred
// error in res.  wg is always marked done in the end.  It is intended to be
// used as a goroutine.
func checkSrv(res *upstreamResult, wg *sync.WaitGroup, hc *healthchecker) {
	defer log.OnPanic(fmt.Sprintf("dnsforward: checking upstream %s", res.server.Address()))
	defer wg.Done()

	res.err = hc.check(res.server)
	if res.err != nil && res.isSpecific {
		res.err = domainSpecificTestError{Err: res.err}
	}
}

// close closes all the upstreams that were successfully parsed.  It enriches
// the results with deferred closing errors.
func (cv *upstreamConfigValidator) close() {
	all := []map[string]*upstreamResult{
		cv.generalUpstreamResults,
		cv.fallbackUpstreamResults,
		cv.privateUpstreamResults,
	}

	for _, m := range all {
		for _, r := range m {
			r.err = errors.WithDeferred(r.err, r.server.Close())
		}
	}
}

// sections of the upstream configuration according to the text label of the
// localization.
//
// Keep in sync with client/src/__locales/en.json.
//
// TODO(s.chzhen):  Refactor.
const (
	generalTextLabel  = "upstream_dns"
	fallbackTextLabel = "fallback_dns_title"
	privateTextLabel  = "local_ptr_title"
)

// status returns all the data collected during parsing, healthcheck, and
// closing of the upstreams.  The returned map is keyed by the original upstream
// configuration piece and contains the corresponding error or "OK" if there was
// no error.
func (cv *upstreamConfigValidator) status() (results map[string]string) {
	// Names of the upstream configuration sections for logging.
	const (
		generalSection  = "general"
		fallbackSection = "fallback"
		privateSection  = "private"
	)

	results = map[string]string{}

	for original, res := range cv.generalUpstreamResults {
		upstreamResultToStatus(generalSection, string(original), res, results)
	}
	for original, res := range cv.fallbackUpstreamResults {
		upstreamResultToStatus(fallbackSection, string(original), res, results)
	}
	for original, res := range cv.privateUpstreamResults {
		upstreamResultToStatus(privateSection, string(original), res, results)
	}

	parseResultToStatus(generalTextLabel, generalSection, cv.generalParseResults, results)
	parseResultToStatus(fallbackTextLabel, fallbackSection, cv.fallbackParseResults, results)
	parseResultToStatus(privateTextLabel, privateSection, cv.privateParseResults, results)

	return results
}

// upstreamResultToStatus puts "OK" or an error message from res into resMap.
// section is the name of the upstream configuration section, i.e. "general",
// "fallback", or "private", and only used for logging.
//
// TODO(e.burkov):  Currently, the HTTP handler expects that all the results are
// put together in a single map, which may lead to collisions, see AG-27539.
// Improve the results compilation.
func upstreamResultToStatus(
	section string,
	original string,
	res *upstreamResult,
	resMap map[string]string,
) {
	val := "OK"
	if res.err != nil {
		val = res.err.Error()
	}

	prevVal := resMap[original]
	switch prevVal {
	case "":
		resMap[original] = val
	case val:
		log.Debug("dnsforward: duplicating %s config line %q", section, original)
	default:
		log.Debug(
			"dnsforward: warning: %s config line %q (%v) had different result %v",
			section,
			val,
			original,
			prevVal,
		)
	}
}

// parseResultToStatus puts parsing error messages from results into resMap.
// section is the name of the upstream configuration section, i.e. "general",
// "fallback", or "private", and only used for logging.
//
// Parsing error message has the following format:
//
//	sectionTextLabel line: parsing error
//
// Where sectionTextLabel is a section text label of a localization and line is
// a line number.
func parseResultToStatus(
	textLabel string,
	section string,
	results []*parseResult,
	resMap map[string]string,
) {
	for _, res := range results {
		original := res.original
		_, ok := resMap[original]
		if ok {
			log.Debug("dnsforward: duplicating %s parsing error %q", section, original)

			continue
		}

		resMap[original] = fmt.Sprintf("%s %d: parsing error", textLabel, res.err.Idx+1)
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
		return errors.Error("wrong response")
	}

	return nil
}
