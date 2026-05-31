package command

type CheckAgentJobExternalLeaseReadinessCommand struct {
	JobLimit          int
	EventLimit        int
	StaleAfterSeconds int
}
