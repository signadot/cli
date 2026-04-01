package config

type PlanExecution struct {
	*Plan
}

type PlanExecGet struct {
	*PlanExecution
}

type PlanExecCancel struct {
	*PlanExecution
}

type PlanExecOutputs struct {
	*PlanExecution
}

type PlanExecGetOutput struct {
	*PlanExecution

	// Flags
	All      bool
	Dir      string
	Metadata bool
}
