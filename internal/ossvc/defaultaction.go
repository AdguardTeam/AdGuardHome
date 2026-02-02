package ossvc

// TODO(e.burkov):  Declare actions for each OS.

// ActionInstall is the implementation of the [Action] interface.
type ActionInstall struct {
	ServiceName      ServiceName
	DisplayName      string
	Description      string
	WorkingDirectory string
	Version          string
	Arguments        []string
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
