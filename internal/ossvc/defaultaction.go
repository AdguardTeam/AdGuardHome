package ossvc

// TODO(e.burkov):  Declare actions for each OS.

// ActionInstall is the implementation of the [Action] interface.
type ActionInstall struct {
	// ServiceName is the name of the service to install.  It must not be empty.
	ServiceName ServiceName

	// DisplayName is the human-readable name of the service.  It must not be
	// empty.
	DisplayName string

	// Description is the description of the service.
	Description string

	// WorkingDirectory is the working directory of the service.  It must not be
	// empty.
	WorkingDirectory string

	// ExecutablePath is the path to the executable of the service.  If it's
	// empty, the path to the current executable is used.
	ExecutablePath string

	// Version is the version of the service to put into the service info.
	Version string

	// Arguments are the command-line arguments to pass to the executable of the
	// service.
	Arguments []string
}

// Name implements the [Action] interface for *ActionInstall.
func (a *ActionInstall) Name() (name ActionName) { return ActionNameInstall }

// isAction implements the [Action] interface for *ActionInstall.
func (a *ActionInstall) isAction() {}

// ActionRestart is the implementation of the [Action] interface.
type ActionRestart struct {
	ServiceName ServiceName
}

// Name implements the [Action] interface for *ActionRestart.
func (a *ActionRestart) Name() (name ActionName) { return ActionNameRestart }

// isAction implements the [Action] interface for *ActionRestart.
func (a *ActionRestart) isAction() {}

// ActionStart is the implementation of the [Action] interface.
type ActionStart struct {
	ServiceName ServiceName
}

// Name implements the [Action] interface for *ActionStart.
func (a *ActionStart) Name() (name ActionName) { return ActionNameStart }

// isAction implements the [Action] interface for *ActionStart.
func (a *ActionStart) isAction() {}

// ActionStop is the implementation of the [Action] interface.
type ActionStop struct {
	ServiceName ServiceName
}

// Name implements the [Action] interface for *ActionStop.
func (a *ActionStop) Name() (name ActionName) { return ActionNameStop }

// isAction implements the [Action] interface for *ActionStop.
func (a *ActionStop) isAction() {}

// ActionUninstall is the implementation of the [Action] interface.
type ActionUninstall struct {
	ServiceName ServiceName
}

// Name implements the [Action] interface for *ActionUninstall.
func (a *ActionUninstall) Name() (name ActionName) { return ActionNameUninstall }

// isAction implements the [Action] interface for *ActionUninstall.
func (a *ActionUninstall) isAction() {}
