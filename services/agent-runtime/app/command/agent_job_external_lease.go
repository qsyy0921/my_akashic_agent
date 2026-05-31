package command

type CheckAgentJobExternalLeaseReadinessCommand struct {
	JobLimit          int
	EventLimit        int
	StaleAfterSeconds int
}

type PlanAgentJobExternalLeaseCommand struct {
	Readiness             CheckAgentJobExternalLeaseReadinessCommand
	DesiredExecutionOwner string
}
