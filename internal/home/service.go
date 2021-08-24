package home

import (
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/kardianos/service"
)

// TODO(a.garipov): Move shell templates into actual files.  Either during the
// v0.106.0 cycle using packr or during the following cycle using go:embed.

const (
	launchdStdoutPath  = "/var/log/AdGuardHome.stdout.log"
	launchdStderrPath  = "/var/log/AdGuardHome.stderr.log"
	serviceName        = "AdGuardHome"
	serviceDisplayName = "AdGuard Home service"
	serviceDescription = "AdGuard Home: Network-level blocker"
)

// program represents the program that will be launched by as a service or a
// daemon.
type program struct {
	clientBuildFS fs.FS
	opts          options
}

// Start implements service.Interface interface for *program.
func (p *program) Start(_ service.Service) (err error) {
	// Start should not block.  Do the actual work async.
	args := p.opts
	args.runningAsService = true

	go run(args, p.clientBuildFS)

	return nil
}

// Stop implements service.Interface interface for *program.
func (p *program) Stop(_ service.Service) error {
	// Stop should not block.  Return with a few seconds.
	if Context.appSignalChannel == nil {
		os.Exit(0)
	}

	Context.appSignalChannel <- syscall.SIGINT

	return nil
}

// svcStatus returns the service's status.
//
// On OpenWrt, the service utility may not exist.  We use our service script
// directly in this case.
func svcStatus(s service.Service) (status service.Status, err error) {
	status, err = s.Status()
	if err != nil && service.Platform() == "unix-systemv" {
		var code int
		code, err = runInitdCommand("status")
		if err != nil || code != 0 {
			return service.StatusStopped, nil
		}

		return service.StatusRunning, nil
	}

	return status, err
}

// svcAction performs the action on the service.
//
// On OpenWrt, the service utility may not exist.  We use our service script
// directly in this case.
func svcAction(s service.Service, action string) (err error) {
	err = service.Control(s, action)
	if err != nil && service.Platform() == "unix-systemv" &&
		(action == "start" || action == "stop" || action == "restart") {
		_, err = runInitdCommand(action)

		return err
	}

	return err
}

// Send SIGHUP to a process with PID taken from our .pid file.  If it doesn't
// exist, find our PID using 'ps' command.
func sendSigReload() {
	if runtime.GOOS == "windows" {
		log.Error("not implemented on windows")

		return
	}

	pidfile := fmt.Sprintf("/var/run/%s.pid", serviceName)
	var pid int
	data, err := os.ReadFile(pidfile)
	if errors.Is(err, os.ErrNotExist) {
		if pid, err = aghos.PIDByCommand(serviceName, os.Getpid()); err != nil {
			log.Error("finding AdGuardHome process: %s", err)

			return
		}
	} else if err != nil {
		log.Error("reading pid file %s: %s", pidfile, err)

		return

	} else {
		parts := strings.SplitN(string(data), "\n", 2)
		if len(parts) == 0 {
			log.Error("can't read pid file %s: bad value", pidfile)

			return
		}

		if pid, err = strconv.Atoi(strings.TrimSpace(parts[0])); err != nil {
			log.Error("can't read pid file %s: %s", pidfile, err)

			return
		}
	}

	var proc *os.Process
	if proc, err = os.FindProcess(pid); err != nil {
		log.Error("can't send signal to pid %d: %s", pid, err)

		return
	}

	if err = proc.Signal(syscall.SIGHUP); err != nil {
		log.Error("Can't send signal to pid %d: %s", pid, err)

		return
	}

	log.Debug("sent signal to PID %d", pid)
}

// handleServiceControlAction one of the possible control actions:
// install -- installs a service/daemon
// uninstall -- uninstalls it
// status -- prints the service status
// start -- starts the previously installed service
// stop -- stops the previously installed service
// restart - restarts the previously installed service
// run - this is a special command that is not supposed to be used directly
// it is specified when we register a service, and it indicates to the app
// that it is being run as a service/daemon.
func handleServiceControlAction(opts options, clientBuildFS fs.FS) {
	// Call chooseSystem expicitly to introduce OpenBSD support for service
	// package.  It's a noop for other GOOS values.
	chooseSystem()

	action := opts.serviceControlAction
	log.Printf("Service control action: %s", action)

	if action == "reload" {
		sendSigReload()

		return
	}

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Unable to find the path to the current directory")
	}

	runOpts := opts
	runOpts.serviceControlAction = "run"
	svcConfig := &service.Config{
		Name:             serviceName,
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		WorkingDirectory: pwd,
		Arguments:        serialize(runOpts),
	}
	configureService(svcConfig)

	prg := &program{
		clientBuildFS: clientBuildFS,
		opts:          runOpts,
	}
	var s service.Service
	if s, err = service.New(prg, svcConfig); err != nil {
		log.Fatal(err)
	}

	switch action {
	case "status":
		handleServiceStatusCommand(s)
	case "run":
		if err = s.Run(); err != nil {
			log.Fatalf("Failed to run service: %s", err)
		}
	case "install":
		initConfigFilename(opts)
		initWorkingDir(opts)
		handleServiceInstallCommand(s)
	case "uninstall":
		handleServiceUninstallCommand(s)
	default:
		if err = svcAction(s, action); err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("action %s has been done successfully on %s", action, service.ChosenSystem())
}

// handleServiceStatusCommand handles service "status" command.
func handleServiceStatusCommand(s service.Service) {
	status, errSt := svcStatus(s)
	if errSt != nil {
		log.Fatalf("failed to get service status: %s", errSt)
	}

	switch status {
	case service.StatusUnknown:
		log.Printf("Service status is unknown")
	case service.StatusStopped:
		log.Printf("Service is stopped")
	case service.StatusRunning:
		log.Printf("Service is running")
	}
}

// handleServiceStatusCommand handles service "install" command
func handleServiceInstallCommand(s service.Service) {
	err := svcAction(s, "install")
	if err != nil {
		log.Fatal(err)
	}

	if aghos.IsOpenWrt() {
		// On OpenWrt it is important to run enable after the service
		// installation Otherwise, the service won't start on the system
		// startup.
		_, err = runInitdCommand("enable")
		if err != nil {
			log.Fatal(err)
		}
	}

	// Start automatically after install.
	err = svcAction(s, "start")
	if err != nil {
		log.Fatalf("Failed to start the service: %s", err)
	}
	log.Printf("Service has been started")

	if detectFirstRun() {
		log.Printf(`Almost ready!
AdGuard Home is successfully installed and will automatically start on boot.
There are a few more things that must be configured before you can use it.
Click on the link below and follow the Installation Wizard steps to finish setup.
AdGuard Home is now available at the following addresses:`)
		printHTTPAddresses(schemeHTTP)
	}
}

// handleServiceStatusCommand handles service "uninstall" command
func handleServiceUninstallCommand(s service.Service) {
	if aghos.IsOpenWrt() {
		// On OpenWrt it is important to run disable command first
		// as it will remove the symlink
		_, err := runInitdCommand("disable")
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := svcAction(s, "uninstall"); err != nil {
		log.Fatal(err)
	}

	if runtime.GOOS == "darwin" {
		// Remove log files on cleanup and log errors.
		err := os.Remove(launchdStdoutPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("removing stdout file: %s", err)
		}

		err = os.Remove(launchdStderrPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("removing stderr file: %s", err)
		}
	}
}

// configureService defines additional settings of the service
func configureService(c *service.Config) {
	c.Option = service.KeyValue{}

	// macOS

	// Redefines the launchd config file template
	// The purpose is to enable stdout/stderr redirect by default
	c.Option["LaunchdConfig"] = launchdConfig
	// This key is used to start the job as soon as it has been loaded. For daemons this means execution at boot time, for agents execution at login.
	c.Option["RunAtLoad"] = true

	// POSIX

	// Redirect StdErr & StdOut to files.
	c.Option["LogOutput"] = true

	// Use modified service file templates.
	c.Option["SystemdScript"] = systemdScript
	c.Option["SysvScript"] = sysvScript

	// On OpenWrt we're using a different type of sysvScript.
	if aghos.IsOpenWrt() {
		c.Option["SysvScript"] = openWrtScript
	} else if runtime.GOOS == "freebsd" {
		c.Option["SysvScript"] = freeBSDScript
	}

	c.Option["RunComScript"] = openBSDScript
	c.Option["SvcInfo"] = fmt.Sprintf("%s %s", version.Full(), time.Now())
}

// runInitdCommand runs init.d service command
// returns command code or error if any
func runInitdCommand(action string) (int, error) {
	confPath := "/etc/init.d/" + serviceName
	code, _, err := aghos.RunCommand("sh", "-c", confPath+" "+action)
	return code, err
}

// Basically the same template as the one defined in github.com/kardianos/service
// but with two additional keys - StandardOutPath and StandardErrorPath
var launchdConfig = `<?xml version='1.0' encoding='UTF-8'?>
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

// Note: we should keep it in sync with the template from service_systemd_linux.go file
// Add "After=" setting for systemd service file, because we must be started only after network is online
// Set "RestartSec" to 10
const systemdScript = `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
After=syslog.target network-online.target

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
StandardOutput=file:/var/log/{{.Name}}.out
StandardError=file:/var/log/{{.Name}}.err
{{- end}}
Restart=always
RestartSec=10
EnvironmentFile=-/etc/sysconfig/{{.Name}}

[Install]
WantedBy=multi-user.target
`

// Note: we should keep it in sync with the template from service_sysv_linux.go file
// Use "ps | grep -v grep | grep $(get_pid)" because "ps PID" may not work on OpenWrt
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
    [ -f "$pid_file" ] && ps | grep -v grep | grep $(get_pid) > /dev/null 2>&1
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

// TODO(a.garipov): Don't use .WorkingDirectory here.  There are currently no
// guarantees that it will actually be the required directory.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/2614.
const freeBSDScript = `#!/bin/sh
# PROVIDE: {{.Name}}
# REQUIRE: networking
# KEYWORD: shutdown
. /etc/rc.subr
name="{{.Name}}"
{{.Name}}_env="IS_DAEMON=1"
{{.Name}}_user="root"
pidfile="/var/run/${name}.pid"
command="/usr/sbin/daemon"
command_args="-p ${pidfile} -f -r {{.WorkingDirectory}}/{{.Name}}"
run_rc_command "$1"
`

const openBSDScript = `#!/bin/sh
#
# $OpenBSD: {{ .SvcInfo }}

daemon="{{.Path}}"
daemon_flags={{ .Arguments | args }}

. /etc/rc.d/rc.subr

rc_bg=YES

rc_cmd $1
`
