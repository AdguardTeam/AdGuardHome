//go:build freebsd

package ossvc

import (
	"github.com/kardianos/service"
)

// configureServiceOptions defines additional settings of the service
// configuration on FreeBSD.  conf must not be nil.
func configureOSOptions(conf *service.Config) {
	conf.Option["SysvScript"] = freeBSDScript
}

// freeBSDScript is the source of the daemon script for FreeBSD.  Keep as close
// as possible to the https://github.com/kardianos/service/blob/18c957a3dc1120a2efe77beb401d476bade9e577/service_freebsd.go#L204.
//
//lint:ignore U1000 TODO(e.burkov): Use.
const freeBSDScript = `#!/bin/sh
# PROVIDE: {{.Name}}
# REQUIRE: networking
# KEYWORD: shutdown

. /etc/rc.subr

name="{{.Name}}"
{{.Name}}_env="IS_DAEMON=1"
{{.Name}}_user="root"
pidfile_child="/var/run/${name}.pid"
pidfile="/var/run/${name}_daemon.pid"
command="/usr/sbin/daemon"
daemon_args="-P ${pidfile} -p ${pidfile_child} -r -t ${name}"
command_args="${daemon_args} {{.Path}}{{range .Arguments}} {{.}}{{end}}"

run_rc_command "$1"
`
