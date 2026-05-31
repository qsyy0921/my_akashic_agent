package command

type PlanAgentJobCapacityCommand struct {
	JobLimit          int
	EventLimit        int
	StaleAfterSeconds int
}
