package configmigrate

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/AdguardTeam/golibs/netutil"
)

// migrateTo10 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 9
//	'dns':
//	  'upstream_dns':
//	   - 'quic://some-upstream.com'
//	  'local_ptr_upstreams':
//	   - 'quic://some-upstream.com'
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 10
//	'dns':
//	  'upstream_dns':
//	   - 'quic://some-upstream.com:784'
//	  'local_ptr_upstreams':
//	   - 'quic://some-upstream.com:784'
//	  # …
//	# …
func migrateTo10(diskConf yobj) (err error) {
	diskConf["schema_version"] = 10

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	const quicPort = 784

	ups, ok, err := fieldVal[yarr](dns, "upstream_dns")
	if err != nil {
		return err
	} else if ok {
		if err = addQUICPorts(ups, quicPort); err != nil {
			return err
		}

		dns["upstream_dns"] = ups
	}

	ups, ok, err = fieldVal[yarr](dns, "local_ptr_upstreams")
	if err != nil {
		return err
	} else if ok {
		if err = addQUICPorts(ups, quicPort); err != nil {
			return err
		}

		dns["local_ptr_upstreams"] = ups
	}

	return nil
}

// addQUICPorts inserts a port into each QUIC upstream's hostname in ups if
// those are missing.
func addQUICPorts(ups yarr, port int) (err error) {
	for i, uVal := range ups {
		u, ok := uVal.(string)
		if !ok {
			return fmt.Errorf("unexpected type of upstream field: %T", uVal)
		}

		ups[i] = addQUICPort(u, port)
	}

	return nil
}

// addQUICPort inserts a port into QUIC upstream's hostname if it is missing.
func addQUICPort(ups string, port int) (withPort string) {
	if ups == "" || ups[0] == '#' {
		return ups
	}

	var doms string
	withPort = ups
	if strings.HasPrefix(ups, "[/") {
		domsAndUps := strings.Split(strings.TrimPrefix(ups, "[/"), "/]")
		if len(domsAndUps) != 2 {
			return ups
		}

		doms, withPort = "[/"+domsAndUps[0]+"/]", domsAndUps[1]
	}

	if !strings.Contains(withPort, "://") {
		return ups
	}

	upsURL, err := url.Parse(withPort)
	if err != nil || upsURL.Scheme != "quic" {
		return ups
	}

	var host string
	host, err = netutil.SplitHost(upsURL.Host)
	if err != nil || host != upsURL.Host {
		return ups
	}

	upsURL.Host = strings.Join([]string{host, strconv.Itoa(port)}, ":")

	return doms + upsURL.String()
}
