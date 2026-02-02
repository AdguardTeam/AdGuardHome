package ossvc

// ActionName is the type for actions' names.  It has the following valid
// values:
//   - [ActionNameInstall]
//   - [ActionNameRestart]
//   - [ActionNameStart]
//   - [ActionNameStop]
//   - [ActionNameUninstall]
type ActionName string

const (
	ActionNameInstall   ActionName = "install"
	ActionNameRestart   ActionName = "restart"
	ActionNameStart     ActionName = "start"
	ActionNameStop      ActionName = "stop"
	ActionNameUninstall ActionName = "uninstall"
)

// Action is the interface for actions that can be performed by [Manager].
type Action interface {
	// Name returns the name of the action.
	Name() (name ActionName)

	// isAction is a marker method to prevent types from other packages from
	// implementing this interface.
	isAction()
}
