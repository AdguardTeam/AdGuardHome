package main

import (
	"os"

	"github.com/hmage/golibs/log"
	"github.com/kardianos/service"
)

// Represents the program that will be launched by a service or daemon
type program struct {
}

// Start should quickly start the program
func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	args := options{}
	go run(args)
	return nil
}

// Stop stops the program
func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	cleanup()
	return nil
}

// handleServiceControlAction one of the possible control actions:
// install -- installs a service/daemon
// uninstall -- uninstalls it
// status -- prints the service status
// start -- starts the previously installed service
// stop -- stops the previously installed service
// restart - restarts the previously installed service
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
	}
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
	} else {
		err = service.Control(s, action)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Action %s has been done successfully", action)
	}
}
