package enums

// State represents the current lifecycle state of Nautilus
type State int

const (
	// StateCreated is the initial state after New() returns
	StateCreated State = iota

	// StateInitializing is set when plugins and job Setup is running
	StateInitializing

	// StateReady is set after successful initialization, before first Execute
	StateReady

	// StateExecuting is set when job Execute is running
	StateExecuting

	// StateShuttingDown is set when Shutdown() has been called
	StateShuttingDown

	// StateStopped is set after all cleanup is complete
	StateStopped
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateExecuting:
		return "executing"
	case StateShuttingDown:
		return "shutting_down"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}
