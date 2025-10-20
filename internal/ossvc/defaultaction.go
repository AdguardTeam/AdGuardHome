package ossvc

import "github.com/kardianos/service"

// TODO(e.burkov):  Declare actions for each OS.

// ActionInstall is the implementation of the [Action] interface.
type ActionInstall struct {
	// ServiceConf is the configuration for the service to control.
	//
	// TODO(e.burkov):  Get rid of github.com/kardianos/service dependency and
	// replace with the actual configuration.
	ServiceConf *service.Config
}

// Name implements the [Action] interface for *ActionInstall.
func (a *ActionInstall) Name() (name ActionName) { return ActionNameInstall }

// isAction implements the [Action] interface for *ActionInstall.
func (a *ActionInstall) isAction() {}

// ActionReload is the implementation of the [Action] interface.
type ActionReload struct {
	// ServiceConf is the configuration for the service to control.
	//
	// TODO(e.burkov):  Get rid of github.com/kardianos/service dependency and
	// replace with the actual configuration.
	ServiceConf *service.Config
}

// Name implements the [Action] interface for *ActionReload.
func (a *ActionReload) Name() (name ActionName) { return ActionNameReload }

// isAction implements the [Action] interface for *ActionReload.
func (a *ActionReload) isAction() {}

// ActionStart is the implementation of the [Action] interface.
type ActionStart struct {
	// ServiceConf is the configuration for the service to control.
	//
	// TODO(e.burkov):  Get rid of github.com/kardianos/service dependency and
	// replace with the actual configuration.
	ServiceConf *service.Config
}

// Name implements the [Action] interface for *ActionStart.
func (a *ActionStart) Name() (name ActionName) { return ActionNameStart }

// isAction implements the [Action] interface for *ActionStart.
func (a *ActionStart) isAction() {}

// ActionStop is the implementation of the [Action] interface.
type ActionStop struct {
	// ServiceConf is the configuration for the service to control.
	//
	// TODO(e.burkov):  Get rid of github.com/kardianos/service dependency and
	// replace with the actual configuration.
	ServiceConf *service.Config
}

// Name implements the [Action] interface for *ActionStop.
func (a *ActionStop) Name() (name ActionName) { return ActionNameStop }

// isAction implements the [Action] interface for *ActionStop.
func (a *ActionStop) isAction() {}

// ActionUninstall is the implementation of the [Action] interface.
type ActionUninstall struct {
	// ServiceConf is the configuration for the service to control.
	//
	// TODO(e.burkov):  Get rid of github.com/kardianos/service dependency and
	// replace with the actual configuration.
	ServiceConf *service.Config
}

// Name implements the [Action] interface for *ActionUninstall.
func (a *ActionUninstall) Name() (name ActionName) { return ActionNameUninstall }

// isAction implements the [Action] interface for *ActionUninstall.
func (a *ActionUninstall) isAction() {}
