package command

import "time"

type PlanKnowledgeJobsCommand struct {
	PlannerID       string
	AgentID         string
	IntervalSeconds int
	MaxAttempts     int
	RagMaxMessages  int
	RagParse        bool
	Timestamp       time.Time
}

type CheckKnowledgeJobPlannerReadinessCommand struct {
	Plan              PlanKnowledgeJobsCommand
	StaleAfterSeconds int
}
