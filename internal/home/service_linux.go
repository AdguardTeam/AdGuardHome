//go:build linux

package home

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/kardianos/service"
)

// chooseSystem checks the current system detected and substitutes it with local
// implementation if needed.
func chooseSystem() {
	sys := service.ChosenSystem()
	// By default, package service uses the SysV system if it cannot detect
	// anything other, but the update-rc.d fix should not be applied on OpenWrt,
	// so exclude it explicitly.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/4480 and
	// https://github.com/AdguardTeam/AdGuardHome/issues/4677.
	if sys.String() == "unix-systemv" && !aghos.IsOpenWrt() {
		service.ChooseSystem(sysvSystem{System: sys})
	}
}

// sysvSystem is a wrapper for service.System that wraps the service.Service
// while creating a new one.
//
// TODO(e.burkov):  File a PR to github.com/kardianos/service.
type sysvSystem struct {
	// System is expected to have an unexported type
	// *service.linuxSystemService.
	service.System
}

// New returns a wrapped service.Service.
func (sys sysvSystem) New(i service.Interface, c *service.Config) (s service.Service, err error) {
	s, err = sys.System.New(i, c)
	if err != nil {
		return s, err
	}

	return sysvService{
		Service: s,
		name:    c.Name,
	}, nil
}

// sysvService is a wrapper for a service.Service that also calls update-rc.d in
// a proper way on installing and uninstalling.
type sysvService struct {
	// Service is expected to have an unexported type *service.sysv.
	service.Service
	// name stores the name of the service to call updating script with it.
	name string
}

// Install wraps service.Service.Install call with calling the updating script.
func (svc sysvService) Install() (err error) {
	err = svc.Service.Install()
	if err != nil {
		// Don't wrap an error since it's informative enough as is.
		return err
	}

	_, _, err = aghos.RunCommand("update-rc.d", svc.name, "defaults")

	// Don't wrap an error since it's informative enough as is.
	return err
}

// Uninstall wraps service.Service.Uninstall call with calling the updating
// script.
func (svc sysvService) Uninstall() (err error) {
	err = svc.Service.Uninstall()
	if err != nil {
		// Don't wrap an error since it's informative enough as is.
		return err
	}

	_, _, err = aghos.RunCommand("update-rc.d", svc.name, "remove")

	// Don't wrap an error since it's informative enough as is.
	return err
}
