package home

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/kardianos/service"
)

// TODO(a.garipov): Consider moving the shell templates into actual files and
// using go:embed instead of using large string constants.

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
	// TODO(s.chzhen):  Remove this.
	ctx           context.Context
	clientBuildFS fs.FS
	signals       chan os.Signal
	done          chan struct{}
	opts          options
	baseLogger    *slog.Logger
	logger        *slog.Logger
	sigHdlr       *signalHandler
}

// type check
var _ service.Interface = (*program)(nil)

// Start implements service.Interface interface for *program.
func (p *program) Start(_ service.Service) (err error) {
	// Start should not block.  Do the actual work async.
	args := p.opts
	args.runningAsService = true

	go run(p.ctx, p.baseLogger, args, p.clientBuildFS, p.done, p.sigHdlr)

	return nil
}

// Stop implements service.Interface interface for *program.
func (p *program) Stop(_ service.Service) (err error) {
	p.logger.InfoContext(p.ctx, "stopping: waiting for cleanup")

	aghos.SendShutdownSignal(p.signals)

	// Wait for other goroutines to complete their job.
	<-p.done

	return nil
}

// svcStatus returns the service's status.
//
// On OpenWrt, the service utility may not exist.  We use our service script
// directly in this case.
func svcStatus(ctx context.Context, s service.Service) (status service.Status, err error) {
	status, err = s.Status()
	if err != nil && service.Platform() == "unix-systemv" {
		var code int
		code, err = runInitdCommand(ctx, "status")
		if err != nil || code != 0 {
			return service.StatusStopped, nil
		}

		return service.StatusRunning, nil
	}

	return status, err
}

// svcAction performs the action on the service.  l must not be nil.
//
// On OpenWrt, the service utility may not exist.  We use our service script
// directly in this case.
func svcAction(ctx context.Context, l *slog.Logger, s service.Service, action string) (err error) {
	if action == "start" {
		if err = aghos.PreCheckActionStart(); err != nil {
			l.ErrorContext(ctx, "starting service", slogutil.KeyError, err)
		}
	}

	err = service.Control(s, action)
	if err != nil && service.Platform() == "unix-systemv" &&
		(action == "start" || action == "stop" || action == "restart") {
		_, err = runInitdCommand(ctx, action)
	}

	return err
}

// Send SIGHUP to a process with PID taken from our .pid file.  If it doesn't
// exist, find our PID using 'ps' command.  baseLogger and l must not be nil.
func sendSigReload(ctx context.Context, baseLogger, l *slog.Logger) {
	if runtime.GOOS == "windows" {
		l.ErrorContext(ctx, "not implemented on windows")

		return
	}

	pidFile := fmt.Sprintf("/var/run/%s.pid", serviceName)
	var pid int
	data, err := os.ReadFile(pidFile)
	if errors.Is(err, os.ErrNotExist) {
		aghosLogger := baseLogger.With(slogutil.KeyPrefix, "aghos")
		if pid, err = aghos.PIDByCommand(ctx, aghosLogger, serviceName, os.Getpid()); err != nil {
			l.ErrorContext(ctx, "finding adguardhome process", slogutil.KeyError, err)

			return
		}
	} else if err != nil {
		l.ErrorContext(ctx, "reading", "pid_file", pidFile, slogutil.KeyError, err)

		return
	} else {
		parts := strings.SplitN(string(data), "\n", 2)
		if len(parts) == 0 {
			l.ErrorContext(ctx, "splitting", "pid_file", pidFile, slogutil.KeyError, "bad value")

			return
		}

		if pid, err = strconv.Atoi(strings.TrimSpace(parts[0])); err != nil {
			l.ErrorContext(ctx, "parsing", "pid_file", pidFile, slogutil.KeyError, err)

			return
		}
	}

	var proc *os.Process
	if proc, err = os.FindProcess(pid); err != nil {
		l.ErrorContext(ctx, "finding process for", "pid", pid, slogutil.KeyError, err)

		return
	}

	if err = proc.Signal(syscall.SIGHUP); err != nil {
		l.ErrorContext(ctx, "sending sighup to", "pid", pid, slogutil.KeyError, err)

		return
	}

	l.DebugContext(ctx, "sent sighup to", "pid", pid)
}

// restartService restarts the service.  It returns error if the service is not
// running.  l must not be nil.
func restartService(ctx context.Context, l *slog.Logger) (err error) {
	// Call chooseSystem explicitly to introduce OpenBSD support for service
	// package.  It's a noop for other GOOS values.
	chooseSystem()

	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	svcConfig := &service.Config{
		Name:             serviceName,
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		WorkingDirectory: pwd,
	}
	configureService(svcConfig)

	var s service.Service
	if s, err = service.New(&program{}, svcConfig); err != nil {
		return fmt.Errorf("initializing service: %w", err)
	}

	if err = svcAction(ctx, l, s, "restart"); err != nil {
		return fmt.Errorf("restarting service: %w", err)
	}

	return nil
}

// handleServiceControlAction one of the possible control actions:
//
//   - install:  Installs a service/daemon.
//   - uninstall:  Uninstalls it.
//   - status:  Prints the service status.
//   - start:  Starts the previously installed service.
//   - stop:  Stops the previously installed service.
//   - restart:  Restarts the previously installed service.
//   - run:  This is a special command that is not supposed to be used directly
//     it is specified when we register a service, and it indicates to the app
//     that it is being run as a service/daemon.
func handleServiceControlAction(
	ctx context.Context,
	baseLogger *slog.Logger,
	l *slog.Logger,
	opts options,
	clientBuildFS fs.FS,
	signals chan os.Signal,
	done chan struct{},
	sigHdlr *signalHandler,
) {
	// Call chooseSystem explicitly to introduce OpenBSD support for service
	// package.  It's a noop for other GOOS values.
	chooseSystem()

	action := opts.serviceControlAction
	l.InfoContext(ctx, version.Full())
	l.InfoContext(ctx, "control", "action", action)

	if action == "reload" {
		sendSigReload(ctx, baseLogger, l)

		return
	}

	pwd, err := os.Getwd()
	if err != nil {
		l.ErrorContext(ctx, "getting current directory", slogutil.KeyError, err)
		os.Exit(osutil.ExitCodeFailure)
	}

	runOpts := opts
	runOpts.serviceControlAction = "run"

	args := optsToArgs(runOpts)
	l.DebugContext(ctx, "using", "args", args)

	svcConfig := &service.Config{
		Name:             serviceName,
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		WorkingDirectory: pwd,
		Arguments:        args,
	}
	configureService(svcConfig)

	s, err := service.New(&program{
		ctx:           ctx,
		clientBuildFS: clientBuildFS,
		signals:       signals,
		done:          done,
		opts:          runOpts,
		baseLogger:    l,
		logger:        l.With(slogutil.KeyPrefix, "service"),
		sigHdlr:       sigHdlr,
	}, svcConfig)
	if err != nil {
		l.ErrorContext(ctx, "initializing service", slogutil.KeyError, err)
		os.Exit(osutil.ExitCodeFailure)
	}

	err = handleServiceCommand(ctx, l, s, action, opts)
	if err != nil {
		l.ErrorContext(ctx, "handling command", slogutil.KeyError, err)
		os.Exit(osutil.ExitCodeFailure)
	}

	l.InfoContext(
		ctx,
		"action has been done successfully",
		"action", action,
		"system", service.ChosenSystem(),
	)
}

// handleServiceCommand handles service command.
func handleServiceCommand(
	ctx context.Context,
	l *slog.Logger,
	s service.Service,
	action string,
	opts options,
) (err error) {
	switch action {
	case "status":
		handleServiceStatusCommand(ctx, l, s)
	case "run":
		if err = s.Run(); err != nil {
			return fmt.Errorf("failed to run service: %w", err)
		}
	case "install":
		if err = initWorkingDir(opts); err != nil {
			return fmt.Errorf("failed to init working dir: %w", err)
		}

		initConfigFilename(opts)

		handleServiceInstallCommand(ctx, l, s)
	case "uninstall":
		handleServiceUninstallCommand(ctx, l, s)
	default:
		if err = svcAction(ctx, l, s, action); err != nil {
			return fmt.Errorf("executing action %q: %w", action, err)
		}
	}

	return nil
}

// statusRestartOnFail is a custom status value used to indicate the service's
// state of restarting after failed start.
const statusRestartOnFail = service.StatusStopped + 1

// handleServiceStatusCommand handles service "status" command.
func handleServiceStatusCommand(
	ctx context.Context,
	l *slog.Logger,
	s service.Service,
) {
	status, errSt := svcStatus(ctx, s)
	if errSt != nil {
		l.ErrorContext(ctx, "failed to get service status", slogutil.KeyError, errSt)
		os.Exit(osutil.ExitCodeFailure)
	}

	switch status {
	case service.StatusUnknown:
		l.InfoContext(ctx, "status is unknown")
	case service.StatusStopped:
		l.InfoContext(ctx, "stopped")
	case service.StatusRunning:
		l.InfoContext(ctx, "running")
	case statusRestartOnFail:
		l.InfoContext(ctx, "restarting after failed start")
	}
}

// handleServiceInstallCommand handles service "install" command.
func handleServiceInstallCommand(ctx context.Context, l *slog.Logger, s service.Service) {
	err := svcAction(ctx, l, s, "install")
	if err != nil {
		l.ErrorContext(ctx, "executing install", slogutil.KeyError, err)
		os.Exit(osutil.ExitCodeFailure)
	}

	if aghos.IsOpenWrt() {
		// On OpenWrt it is important to run enable after the service
		// installation.  Otherwise, the service won't start on the system
		// startup.
		_, err = runInitdCommand(ctx, "enable")
		if err != nil {
			l.ErrorContext(ctx, "running init enable", slogutil.KeyError, err)
			os.Exit(osutil.ExitCodeFailure)
		}
	}

	// Start automatically after install.
	err = svcAction(ctx, l, s, "start")
	if err != nil {
		l.ErrorContext(ctx, "starting", slogutil.KeyError, err)
		os.Exit(osutil.ExitCodeFailure)
	}
	l.InfoContext(ctx, "started")

	if detectFirstRun() {
		slogutil.PrintLines(ctx, l, slog.LevelInfo, "", "Almost ready!\n"+
			"AdGuard Home is successfully installed and will automatically start on boot.\n"+
			"There are a few more things that must be configured before you can use it.\n"+
			"Click on the link below and follow the Installation Wizard steps to finish setup.\n"+
			"AdGuard Home is now available at the following addresses:")
		printHTTPAddresses(urlutil.SchemeHTTP, nil)
	}
}

// handleServiceUninstallCommand handles service "uninstall" command.
func handleServiceUninstallCommand(ctx context.Context, l *slog.Logger, s service.Service) {
	if aghos.IsOpenWrt() {
		// On OpenWrt it is important to run disable command first
		// as it will remove the symlink
		_, err := runInitdCommand(ctx, "disable")
		if err != nil {
			l.ErrorContext(ctx, "running init disable", slogutil.KeyError, err)
			os.Exit(osutil.ExitCodeFailure)
		}
	}

	if err := svcAction(ctx, l, s, "stop"); err != nil {
		l.DebugContext(ctx, "executing action stop", slogutil.KeyError, err)
	}

	if err := svcAction(ctx, l, s, "uninstall"); err != nil {
		l.ErrorContext(ctx, "executing action uninstall", slogutil.KeyError, err)
		os.Exit(osutil.ExitCodeFailure)
	}

	if runtime.GOOS == "darwin" {
		// Remove log files on cleanup and log errors.
		err := os.Remove(launchdStdoutPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			l.WarnContext(ctx, "removing stdout file", slogutil.KeyError, err)
		}

		err = os.Remove(launchdStderrPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			l.WarnContext(ctx, "removing stderr file", slogutil.KeyError, err)
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
	c.Option["SvcInfo"] = fmt.Sprintf("%s %s", version.Full(), time.Now())
}

// runInitdCommand runs init.d service command
// returns command code or error if any
func runInitdCommand(ctx context.Context, action string) (int, error) {
	confPath := "/etc/init.d/" + serviceName
	// Pass the script and action as a single string argument.
	cmdCons := executil.SystemCommandConstructor{}
	code, _, err := aghos.RunCommand(ctx, cmdCons, "sh", "-c", confPath+" "+action)

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
