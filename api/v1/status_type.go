package v1

const (
	StatusPhaseCreating = "Creating"
	StatusPhaseRunning  = "Running"
	StatusPhaseSuccess  = "Success"
	StatusPhaseFailed   = "Failed"
	StatusPhaseDeleting = "Deleting"
)

const (
	StatusReasonDependsUnavailable = "DependsUnavailable"
	StatusReasonDependsAvailable   = "DependsAvailable"
)
