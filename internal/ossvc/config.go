package ossvc

import (
	"fmt"
	"runtime"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/kardianos/service"
)

// Paths to stdout and stderr logs for Darwin service manager.
const (
	launchdStdoutPath = "/var/log/AdGuardHome.stdout.log"
	launchdStderrPath = "/var/log/AdGuardHome.stderr.log"
)

// configureServiceOptions defines additional settings of the service
// configuration.
//
// TODO(e.burkov):  Move into OS-specific files.
//
//lint:ignore U1000 TODO(e.burkov): Use.
func configureServiceOptions(c *service.Config, versionInfo string) {
	// macOS

	// Redefines the launchd config file template.
	// The purpose is to enable stdout/stderr redirect by default.
	c.Option["LaunchdConfig"] = launchdConfig
	// This key is used to start the job as soon as it has been loaded. For
	// daemons this means execution at boot time, for agents execution at login.
	c.Option["RunAtLoad"] = true

	// POSIX / systemd

	// Redirect stderr and stdout to files.  Make sure we always restart.
	c.Option["LogOutput"] = true
	c.Option["Restart"] = "always"

	// Start only once network is up on Linux/systemd.
	if runtime.GOOS == "linux" {
		c.Dependencies = []string{
			"After=syslog.target network-online.target",
		}
	}

	// Use the modified service file templates.
	c.Option["SystemdScript"] = systemdScript
	c.Option["SysvScript"] = sysvScript

	// Use different scripts on OpenWrt and FreeBSD.
	if aghos.IsOpenWrt() {
		c.Option["SysvScript"] = openWrtScript
	} else if runtime.GOOS == "freebsd" {
		c.Option["SysvScript"] = freeBSDScript
	}

	c.Option["RunComScript"] = openBSDScript
	c.Option["SvcInfo"] = fmt.Sprintf("%s %s", versionInfo, time.Now())
}

// Basically the same template as the one defined in
// github.com/kardianos/service but with two additional keys:
//   - StandardOutPath
//   - StandardErrorPath
//
//lint:ignore U1000 TODO(e.burkov): Use.
const launchdConfig = `<?xml version='1.0' encoding='UTF-8'?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN"
"http://www.apple.com/DTDs/PropertyList-1.0.dtd" >
<plist version='1.0'>
<dict>
<key>Label</key><string>{{html .Name}}</string>
<key>ProgramArguments</key>
<array>
        <string>{{html .Path}}</string>
{{range .Config.Arguments}}
        <string>{{html .}}</string>
{{end}}
</array>
{{if .UserName}}<key>UserName</key><string>{{html .UserName}}</string>{{end}}
{{if .ChRoot}}<key>RootDirectory</key><string>{{html .ChRoot}}</string>{{end}}
{{if .WorkingDirectory}}<key>WorkingDirectory</key><string>{{html .WorkingDirectory}}</string>{{end}}
<key>SessionCreate</key><{{bool .SessionCreate}}/>
<key>KeepAlive</key><{{bool .KeepAlive}}/>
<key>RunAtLoad</key><{{bool .RunAtLoad}}/>
<key>Disabled</key><false/>
<key>StandardOutPath</key>
<string>` + launchdStdoutPath + `</string>
<key>StandardErrorPath</key>
<string>` + launchdStderrPath + `</string>
</dict>
</plist>
`

// systemdScript is an improved version of the systemd script originally from
// the systemdScript constant in file service_systemd_linux.go in module
// github.com/kardianos/service.  The following changes have been made:
//
//  1. The RestartSec setting is set to a lower value of 10 to make sure we
//     always restart quickly.
//
//  2. The StandardOutput and StandardError settings are set to redirect the
//     output to the systemd journal, see
//     https://man7.org/linux/man-pages/man5/systemd.exec.5.html#LOGGING_AND_STANDARD_INPUT/OUTPUT.
//
//lint:ignore U1000 TODO(e.burkov): Use.
const systemdScript = `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}}
{{$dep}} {{end}}

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart={{.Path|cmdEscape}}{{range .Arguments}} {{.|cmd}}{{end}}
{{if .ChRoot}}RootDirectory={{.ChRoot|cmd}}{{end}}
{{if .WorkingDirectory}}WorkingDirectory={{.WorkingDirectory|cmdEscape}}{{end}}
{{if .UserName}}User={{.UserName}}{{end}}
{{if .ReloadSignal}}ExecReload=/bin/kill -{{.ReloadSignal}} "$MAINPID"{{end}}
{{if .PIDFile}}PIDFile={{.PIDFile|cmd}}{{end}}
{{if and .LogOutput .HasOutputFileSupport -}}
StandardOutput=journal
StandardError=journal
{{- end}}
{{if gt .LimitNOFILE -1 }}LimitNOFILE={{.LimitNOFILE}}{{end}}
{{if .Restart}}Restart={{.Restart}}{{end}}
{{if .SuccessExitStatus}}SuccessExitStatus={{.SuccessExitStatus}}{{end}}
RestartSec=10
EnvironmentFile=-/etc/sysconfig/{{.Name}}

[Install]
WantedBy=multi-user.target
`

// sysvScript is the source of the daemon script for SysV-based Linux systems.
// Keep as close as possible to the https://github.com/kardianos/service/blob/29f8c79c511bc18422bb99992779f96e6bc33921/service_sysv_linux.go#L187.
//
// Use ps command instead of reading the procfs since it's a more
// implementation-independent approach.
//
//lint:ignore U1000 TODO(e.burkov): Use.
const sysvScript = `#!/bin/sh
# For RedHat and cousins:
# chkconfig: - 99 01
# description: {{.Description}}
# processname: {{.Path}}

### BEGIN INIT INFO
# Provides:          {{.Path}}
# Required-Start:
# Required-Stop:
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: {{.DisplayName}}
# Description:       {{.Description}}
### END INIT INFO

cmd="{{.Path}}{{range .Arguments}} {{.|cmd}}{{end}}"

name=$(basename $(readlink -f $0))
pid_file="/var/run/$name.pid"
stdout_log="/var/log/$name.log"
stderr_log="/var/log/$name.err"

[ -e /etc/sysconfig/$name ] && . /etc/sysconfig/$name

get_pid() {
    cat "$pid_file"
}

is_running() {
    [ -f "$pid_file" ] && ps -p "$(get_pid)" > /dev/null 2>&1
}

case "$1" in
    start)
        if is_running; then
            echo "Already started"
        else
            echo "Starting $name"
            {{if .WorkingDirectory}}cd '{{.WorkingDirectory}}'{{end}}
            $cmd >> "$stdout_log" 2>> "$stderr_log" &
            echo $! > "$pid_file"
            if ! is_running; then
                echo "Unable to start, see $stdout_log and $stderr_log"
                exit 1
            fi
        fi
    ;;
    stop)
        if is_running; then
            echo -n "Stopping $name.."
            kill $(get_pid)
            for i in $(seq 1 10)
            do
                if ! is_running; then
                    break
                fi
                echo -n "."
                sleep 1
            done
            echo
            if is_running; then
                echo "Not stopped; may still be shutting down or shutdown may have failed"
                exit 1
            else
                echo "Stopped"
                if [ -f "$pid_file" ]; then
                    rm "$pid_file"
                fi
            fi
        else
            echo "Not running"
        fi
    ;;
    restart)
        $0 stop
        if is_running; then
            echo "Unable to stop, will not attempt to start"
            exit 1
        fi
        $0 start
    ;;
    status)
        if is_running; then
            echo "Running"
        else
            echo "Stopped"
            exit 1
        fi
    ;;
    *)
    echo "Usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac
exit 0
`

// OpenWrt procd init script
// https://github.com/AdguardTeam/AdGuardHome/internal/issues/1386
//
//lint:ignore U1000 TODO(e.burkov): Use.
const openWrtScript = `#!/bin/sh /etc/rc.common

USE_PROCD=1

START=95
STOP=01

cmd="{{.Path}}{{range .Arguments}} {{.|cmd}}{{end}}"
name="{{.Name}}"
pid_file="/var/run/${name}.pid"

start_service() {
    echo "Starting ${name}"

    procd_open_instance
    procd_set_param command ${cmd}
    procd_set_param respawn             # respawn automatically if something died
    procd_set_param stdout 1            # forward stdout of the command to logd
    procd_set_param stderr 1            # same for stderr
    procd_set_param pidfile ${pid_file} # write a pid file on instance start and remove it on stop

    procd_close_instance
    echo "${name} has been started"
}

stop_service() {
    echo "Stopping ${name}"
}

EXTRA_COMMANDS="status"
EXTRA_HELP="        status  Print the service status"

get_pid() {
    cat "${pid_file}"
}

is_running() {
    [ -f "${pid_file}" ] && ps | grep -v grep | grep $(get_pid) >/dev/null 2>&1
}

status() {
    if is_running; then
        echo "Running"
    else
        echo "Stopped"
        exit 1
    fi
}
`

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
