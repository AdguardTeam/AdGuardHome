package home

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/log"
	"github.com/kardianos/service"
)

const (
	launchdStdoutPath  = "/var/log/AdGuardHome.stdout.log"
	launchdStderrPath  = "/var/log/AdGuardHome.stderr.log"
	serviceName        = "AdGuardHome"
	serviceDisplayName = "AdGuard Home service"
	serviceDescription = "AdGuard Home: Network-level blocker"
)

// Represents the program that will be launched by a service or daemon
type program struct {
	opts options
}

// Start should quickly start the program
func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	args := p.opts
	args.runningAsService = true
	go run(args)
	return nil
}

// Stop stops the program
func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	if Context.appSignalChannel == nil {
		os.Exit(0)
	}
	Context.appSignalChannel <- syscall.SIGINT
	return nil
}

// Check the service's status
// Note: on OpenWrt 'service' utility may not exist - we use our service script directly in this case.
func svcStatus(s service.Service) (service.Status, error) {
	status, err := s.Status()
	if err != nil && service.Platform() == "unix-systemv" {
		code, err := runInitdCommand("status")
		if err != nil {
			return service.StatusStopped, nil
		}
		if code != 0 {
			return service.StatusStopped, nil
		}
		return service.StatusRunning, nil
	}
	return status, err
}

// Perform an action on the service
// Note: on OpenWrt 'service' utility may not exist - we use our service script directly in this case.
func svcAction(s service.Service, action string) error {
	err := service.Control(s, action)
	if err != nil && service.Platform() == "unix-systemv" &&
		(action == "start" || action == "stop" || action == "restart") {
		_, err := runInitdCommand(action)
		return err
	}
	return err
}

// Send SIGHUP to a process with ID taken from our pid-file
// If pid-file doesn't exist, find our PID using 'ps' command
func sendSigReload() {
	if runtime.GOOS == "windows" {
		log.Error("Not implemented on Windows")
		return
	}

	pidfile := fmt.Sprintf("/var/run/%s.pid", serviceName)
	data, err := ioutil.ReadFile(pidfile)
	if os.IsNotExist(err) {
		code, psdata, err := util.RunCommand("ps", "-C", serviceName, "-o", "pid=")
		if err != nil || code != 0 {
			log.Error("Can't find AdGuardHome process: %s  code:%d", err, code)
			return
		}
		data = []byte(psdata)

	} else if err != nil {
		log.Error("Can't read PID file %s: %s", pidfile, err)
		return
	}

	parts := strings.SplitN(string(data), "\n", 2)
	if len(parts) == 0 {
		log.Error("Can't read PID file %s: bad value", pidfile)
		return
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Error("Can't read PID file %s: %s", pidfile, err)
		return
	}
	err = util.SendProcessSignal(pid, syscall.SIGHUP)
	if err != nil {
		log.Error("Can't send signal to PID %d: %s", pid, err)
		return
	}
	log.Debug("Sent signal to PID %d", pid)
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
func handleServiceControlAction(opts options) {
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
	prg := &program{runOpts}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if action == "status" {
		handleServiceStatusCommand(s)
	} else if action == "run" {
		err = s.Run()
		if err != nil {
			log.Fatalf("Failed to run service: %s", err)
		}
	} else if action == "install" {
		initConfigFilename(opts)
		initWorkingDir(opts)
		handleServiceInstallCommand(s)
	} else if action == "uninstall" {
		handleServiceUninstallCommand(s)
	} else {
		err = svcAction(s, action)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("Action %s has been done successfully on %s", action, service.ChosenSystem().String())
}

// handleServiceStatusCommand handles service "status" command
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

	if util.IsOpenWrt() {
		// On OpenWrt it is important to run enable after the service installation
		// Otherwise, the service won't start on the system startup
		_, err := runInitdCommand("enable")
		if err != nil {
			log.Fatal(err)
		}
	}

	// Start automatically after install
	err = svcAction(s, "start")
	if err != nil {
		log.Fatalf("Failed to start the service: %s", err)
	}
	log.Printf("Service has been started")

	if detectFirstRun() {
		log.Printf(`Almost ready!
AdGuard Home is successfully installed and will automatically start on boot.
There are a few more things that must be configured before you can use it.
Click on the link below and follow the Installation Wizard steps to finish setup.`)
		printHTTPAddresses("http")
	}
}

// handleServiceStatusCommand handles service "uninstall" command
func handleServiceUninstallCommand(s service.Service) {
	if util.IsOpenWrt() {
		// On OpenWrt it is important to run disable command first
		// as it will remove the symlink
		_, err := runInitdCommand("disable")
		if err != nil {
			log.Fatal(err)
		}
	}

	err := svcAction(s, "uninstall")
	if err != nil {
		log.Fatal(err)
	}

	if runtime.GOOS == "darwin" {
		// Removing log files on cleanup and ignore errors
		err := os.Remove(launchdStdoutPath)
		if err != nil && !os.IsNotExist(err) {
			log.Printf("cannot remove %s", launchdStdoutPath)
		}
		err = os.Remove(launchdStderrPath)
		if err != nil && !os.IsNotExist(err) {
			log.Printf("cannot remove %s", launchdStderrPath)
		}
	}
}

// configureService defines additional settings of the service
func configureService(c *service.Config) {
	c.Option = service.KeyValue{}

	// OS X
	// Redefines the launchd config file template
	// The purpose is to enable stdout/stderr redirect by default
	c.Option["LaunchdConfig"] = launchdConfig
	// This key is used to start the job as soon as it has been loaded. For daemons this means execution at boot time, for agents execution at login.
	c.Option["RunAtLoad"] = true

	// POSIX
	// Redirect StdErr & StdOut to files.
	c.Option["LogOutput"] = true

	// Use modified service file templates
	c.Option["SystemdScript"] = systemdScript
	c.Option["SysvScript"] = sysvScript

	// On OpenWrt we're using a different type of sysvScript
	if util.IsOpenWrt() {
		c.Option["SysvScript"] = openWrtScript
	} else if util.IsFreeBSD() {
		c.Option["SysvScript"] = freeBSDScript
	}
}

// runInitdCommand runs init.d service command
// returns command code or error if any
func runInitdCommand(action string) (int, error) {
	confPath := "/etc/init.d/" + serviceName
	code, _, err := util.RunCommand("sh", "-c", confPath+" "+action)
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
// https://github.com/AdguardTeam/AdGuardHome/issues/1386
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
command_args="-P ${pidfile} -r -f {{.WorkingDirectory}}/{{.Name}}"
run_rc_command "$1"
`
