package query

type AgentJobPrioritySummaryView struct {
	JobTypes             int `json:"job_types"`
	HighPriorityJobTypes int `json:"high_priority_job_types"`
	BlockedJobTypes      int `json:"blocked_job_types"`
	WarningJobTypes      int `json:"warning_job_types"`
	MaxPriorityScore     int `json:"max_priority_score"`
}

type AgentJobPriorityPlanItemView struct {
	Rank                    int                        `json:"rank"`
	JobType                 string                     `json:"job_type"`
	PriorityClass           string                     `json:"priority_class"`
	PriorityScore           int                        `json:"priority_score"`
	Action                  string                     `json:"action"`
	Recommendation          string                     `json:"recommendation"`
	Pending                 int                        `json:"pending"`
	Leased                  int                        `json:"leased"`
	Running                 int                        `json:"running"`
	Active                  int                        `json:"active"`
	OldestPendingAgeSeconds int                        `json:"oldest_pending_age_seconds"`
	HighPressure            bool                       `json:"high_pressure"`
	PressureReason          string                     `json:"pressure_reason,omitempty"`
	Coverage                AgentJobWorkerCoverageView `json:"coverage"`
}

type AgentJobPriorityPlanStep struct {
	StepIndex int    `json:"step_index"`
	Phase     string `json:"phase"`
	Action    string `json:"action"`
	Method    string `json:"method,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type AgentJobPriorityPlanView struct {
	Ready             bool                           `json:"ready"`
	Reason            string                         `json:"reason"`
	Summary           AgentJobPrioritySummaryView    `json:"summary"`
	Items             []AgentJobPriorityPlanItemView `json:"items,omitempty"`
	VerificationSteps []AgentJobPriorityPlanStep     `json:"verification_steps,omitempty"`
	Blockers          []string                       `json:"blockers,omitempty"`
	Attributes        map[string]string              `json:"attributes,omitempty"`
	Notes             []string                       `json:"notes,omitempty"`
	SideEffect        string                         `json:"side_effect"`
}
