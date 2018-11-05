package upstream

import "github.com/miekg/dns"

// Performs a simple health-check of the specified upstream
func IsAlive(u Upstream) (bool, error) {

	// Using ipv4only.arpa. domain as it is a part of DNS64 RFC and it should exist everywhere
	ping := new(dns.Msg)
	ping.SetQuestion("ipv4only.arpa.", dns.TypeA)

	resp, err := u.Exchange(nil, ping)

	// If we got a header, we're alright, basically only care about I/O errors 'n stuff.
	if err != nil && resp != nil {
		// Silly check, something sane came back.
		if resp.Response || resp.Opcode == dns.OpcodeQuery {
			err = nil
		}
	}

	return err == nil, err
}
