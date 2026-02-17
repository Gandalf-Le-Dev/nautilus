package enums

// ExecutionMode defines how a job is executed
type ExecutionMode int

const (
	// ModeOneShot executes the job once, runs Teardown, and exits
	ModeOneShot ExecutionMode = iota

	// ModePeriodic executes the job repeatedly on a schedule (interval or cron)
	ModePeriodic

	// ModeContinuous executes the job once and blocks indefinitely
	// Used for queue consumers and long-running processes
	// On context cancel, Execute returns and Teardown runs
	ModeContinuous
)

// String returns the string representation of the execution mode
func (m ExecutionMode) String() string {
	switch m {
	case ModeOneShot:
		return "oneshot"
	case ModePeriodic:
		return "periodic"
	case ModeContinuous:
		return "continuous"
	default:
		return "unknown"
	}
}
