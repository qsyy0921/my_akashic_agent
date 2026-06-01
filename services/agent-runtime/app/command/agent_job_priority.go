package command

type PlanAgentJobPriorityCommand struct {
	JobLimit          int
	EventLimit        int
	StaleAfterSeconds int
}
