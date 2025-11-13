//go:build linux

package ossvc

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/kardianos/service"
)

// configureServiceOptions defines additional settings of the service
// configuration on Linux.  conf must not be nil.
func configureOSOptions(conf *service.Config) {
	conf.Option["LogOutput"] = true
	conf.Option["Restart"] = "always"

	conf.Dependencies = []string{
		"After=syslog.target network-online.target",
	}

	conf.Option["SystemdScript"] = systemdScript

	// Use different scripts on OpenWrt.
	if aghos.IsOpenWrt() {
		conf.Option["SysvScript"] = openWrtScript
	} else {
		conf.Option["SysvScript"] = sysvScript
	}
}

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
