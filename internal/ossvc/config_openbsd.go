//go:build openbsd

package ossvc

import (
	"github.com/kardianos/service"
)

// configureServiceOptions defines additional settings of the service
// configuration on OpenBSD.  conf must not be nil.
func configureOSOptions(conf *service.Config) {
	conf.Option["RunComScript"] = openBSDScript
}

// openBSDScript is the source of the daemon script for OpenBSD.
//
//lint:ignore U1000 TODO(e.burkov): Use.
const openBSDScript = `#!/bin/ksh
#
# $OpenBSD: {{ .SvcInfo }}

daemon="{{.Path}}"
daemon_flags={{ .Arguments | args }}
daemon_logger="daemon.info"

. /etc/rc.d/rc.subr

rc_bg=YES

rc_cmd $1
`
