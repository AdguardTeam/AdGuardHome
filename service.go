package main

import (
	"os"
	"runtime"

	"github.com/hmage/golibs/log"
	"github.com/kardianos/service"
)

const (
	launchdStdoutPath = "/var/log/AdGuardHome.stdout.log"
	launchdStderrPath = "/var/log/AdGuardHome.stderr.log"
)

// Represents the program that will be launched by a service or daemon
type program struct {
}

// Start should quickly start the program
func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	args := options{runningAsService: true}
	go run(args)
	return nil
}

// Stop stops the program
func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	cleanup()
	os.Exit(0)
	return nil
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
func handleServiceControlAction(action string) {
	log.Printf("Service control action: %s", action)

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Unable to find the path to the current directory")
	}
	svcConfig := &service.Config{
		Name:             "AdGuardHome",
		DisplayName:      "AdGuard Home service",
		Description:      "AdGuard Home: Network-level blocker",
		WorkingDirectory: pwd,
		Arguments:        []string{"-s", "run"},
	}
	configureService(svcConfig)
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if action == "status" {
		status, errSt := s.Status()
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
	} else if action == "run" {
		err = s.Run()
		if err != nil {
			log.Fatalf("Failed to run service: %s", err)
		}
	} else {
		if action == "uninstall" {
			// In case of Windows and Linux when a running service is being uninstalled,
			// it is just marked for deletion but not stopped
			// So we explicitly stop it here
			_ = s.Stop()
		}

		err = service.Control(s, action)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Action %s has been done successfully on %s", action, service.ChosenSystem().String())

		if action == "install" {
			// Start automatically after install
			err = service.Control(s, "start")
			if err != nil {
				log.Fatalf("Failed to start the service: %s", err)
			}
			log.Printf("Service has been started")
		} else if action == "uninstall" {
			cleanupService()
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

	// Windows
	if runtime.GOOS == "windows" {
		c.UserName = "NT AUTHORITY\\NetworkService"
	}
}

// cleanupService called on the service uninstall, cleans up additional files if needed
func cleanupService() {
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
