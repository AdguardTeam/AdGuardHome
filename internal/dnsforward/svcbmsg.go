package dnsforward

import (
	"encoding/base64"
	"net"
	"strconv"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// genAnswerHTTPS returns a properly initialized HTTPS resource record.
//
// See the comment on genAnswerSVCB for a list of current restrictions on
// parameter values.
func (s *Server) genAnswerHTTPS(req *dns.Msg, svcb *rules.DNSSVCB) (ans *dns.HTTPS) {
	ans = &dns.HTTPS{
		SVCB: *s.genAnswerSVCB(req, svcb),
	}

	ans.Hdr.Rrtype = dns.TypeHTTPS

	return ans
}

// strToSVCBKey is the string-to-svcb-key mapping.
//
// See https://github.com/miekg/dns/blob/23c4faca9d32b0abbb6e179aa1aadc45ac53a916/svcb.go#L27.
//
// TODO(a.garipov): Propose exporting this API or something similar in the
// github.com/miekg/dns module.
var strToSVCBKey = map[string]dns.SVCBKey{
	"alpn":            dns.SVCB_ALPN,
	"ech":             dns.SVCB_ECHCONFIG,
	"ipv4hint":        dns.SVCB_IPV4HINT,
	"ipv6hint":        dns.SVCB_IPV6HINT,
	"mandatory":       dns.SVCB_MANDATORY,
	"no-default-alpn": dns.SVCB_NO_DEFAULT_ALPN,
	"port":            dns.SVCB_PORT,

	// TODO(a.garipov): This is the previous name for the parameter that has
	// since been changed.  Remove this in v0.109.0.
	"echconfig": dns.SVCB_ECHCONFIG,
}

// svcbKeyHandler is a handler for one SVCB parameter key.
type svcbKeyHandler func(valStr string) (val dns.SVCBKeyValue)

// svcbKeyHandlers are the supported SVCB parameters handlers.
var svcbKeyHandlers = map[string]svcbKeyHandler{
	"alpn": func(valStr string) (val dns.SVCBKeyValue) {
		return &dns.SVCBAlpn{
			Alpn: []string{valStr},
		}
	},

	"ech": func(valStr string) (val dns.SVCBKeyValue) {
		ech, err := base64.StdEncoding.DecodeString(valStr)
		if err != nil {
			log.Debug("can't parse svcb/https ech: %s; ignoring", err)

			return nil
		}

		return &dns.SVCBECHConfig{
			ECH: ech,
		}
	},

	"ipv4hint": func(valStr string) (val dns.SVCBKeyValue) {
		ip := net.ParseIP(valStr)
		if ip4 := ip.To4(); ip == nil || ip4 == nil {
			log.Debug("can't parse svcb/https ipv4 hint %q; ignoring", valStr)

			return nil
		}

		return &dns.SVCBIPv4Hint{
			Hint: []net.IP{ip},
		}
	},

	"ipv6hint": func(valStr string) (val dns.SVCBKeyValue) {
		ip := net.ParseIP(valStr)
		if ip == nil {
			log.Debug("can't parse svcb/https ipv6 hint %q; ignoring", valStr)

			return nil
		}

		return &dns.SVCBIPv6Hint{
			Hint: []net.IP{ip},
		}
	},

	"mandatory": func(valStr string) (val dns.SVCBKeyValue) {
		code, ok := strToSVCBKey[valStr]
		if !ok {
			log.Debug("unknown svcb/https mandatory key %q, ignoring", valStr)

			return nil
		}

		return &dns.SVCBMandatory{
			Code: []dns.SVCBKey{code},
		}
	},

	"no-default-alpn": func(_ string) (val dns.SVCBKeyValue) {
		return &dns.SVCBNoDefaultAlpn{}
	},

	"port": func(valStr string) (val dns.SVCBKeyValue) {
		port64, err := strconv.ParseUint(valStr, 10, 16)
		if err != nil {
			log.Debug("can't parse svcb/https port: %s; ignoring", err)

			return nil
		}

		return &dns.SVCBPort{
			Port: uint16(port64),
		}
	},

	// TODO(a.garipov): This is the previous name for the parameter that has
	// since been changed.  Remove this in v0.109.0.
	"echconfig": func(valStr string) (val dns.SVCBKeyValue) {
		log.Info(
			`warning: svcb/https record parameter name "echconfig" is deprecated; ` +
				`use "ech" instead`,
		)

		ech, err := base64.StdEncoding.DecodeString(valStr)
		if err != nil {
			log.Debug("can't parse svcb/https ech: %s; ignoring", err)

			return nil
		}

		return &dns.SVCBECHConfig{
			ECH: ech,
		}
	},

	"dohpath": func(valStr string) (val dns.SVCBKeyValue) {
		return &dns.SVCBDoHPath{
			Template: valStr,
		}
	},
}

// genAnswerSVCB returns a properly initialized SVCB resource record.
//
// Currently, there are several restrictions on how the parameters are parsed.
// Firstly, the parsing of non-contiguous values isn't supported.  Secondly, the
// parsing of value-lists is not supported either.
//
//	ipv4hint=127.0.0.1             // Supported.
//	ipv4hint="127.0.0.1"           // Unsupported.
//	ipv4hint=127.0.0.1,127.0.0.2   // Unsupported.
//	ipv4hint="127.0.0.1,127.0.0.2" // Unsupported.
//
// TODO(a.garipov): Support all of these.
func (s *Server) genAnswerSVCB(req *dns.Msg, svcb *rules.DNSSVCB) (ans *dns.SVCB) {
	ans = &dns.SVCB{
		Hdr:      s.hdr(req, dns.TypeSVCB),
		Priority: svcb.Priority,
		Target:   dns.Fqdn(svcb.Target),
	}
	if len(svcb.Params) == 0 {
		return ans
	}

	values := make([]dns.SVCBKeyValue, 0, len(svcb.Params))
	for k, valStr := range svcb.Params {
		handler, ok := svcbKeyHandlers[k]
		if !ok {
			log.Debug("unknown svcb/https key %q, ignoring", k)

			continue
		}

		val := handler(valStr)
		if val == nil {
			continue
		}

		values = append(values, val)
	}

	if len(values) > 0 {
		ans.Value = values
	}

	return ans
}
