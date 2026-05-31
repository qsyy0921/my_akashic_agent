package query

type AgentJobCapacitySummaryView struct {
	JobTypes                int `json:"job_types"`
	MappedJobTypes          int `json:"mapped_job_types"`
	UnmappedJobTypes        int `json:"unmapped_job_types"`
	HighPressureJobTypes    int `json:"high_pressure_job_types"`
	CapacityBlockedJobTypes int `json:"capacity_blocked_job_types"`
	WorkerWarningJobTypes   int `json:"worker_warning_job_types"`
	ActiveWorkerJobTypes    int `json:"active_worker_job_types"`
	StaleWorkerJobTypes     int `json:"stale_worker_job_types"`
	FailedWorkerJobTypes    int `json:"failed_worker_job_types"`
	MaxPending              int `json:"max_pending"`
	MaxActive               int `json:"max_active"`
	OldestPendingAgeSeconds int `json:"oldest_pending_age_seconds"`
}

type AgentJobCapacityPlanItemView struct {
	JobType                 string                     `json:"job_type"`
	Severity                string                     `json:"severity"`
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

type AgentJobCapacityPlanStep struct {
	StepIndex int    `json:"step_index"`
	Phase     string `json:"phase"`
	Action    string `json:"action"`
	Method    string `json:"method,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type AgentJobCapacityPlanView struct {
	Ready             bool                           `json:"ready"`
	Reason            string                         `json:"reason"`
	Summary           AgentJobCapacitySummaryView    `json:"summary"`
	Items             []AgentJobCapacityPlanItemView `json:"items,omitempty"`
	VerificationSteps []AgentJobCapacityPlanStep     `json:"verification_steps,omitempty"`
	Blockers          []string                       `json:"blockers,omitempty"`
	Attributes        map[string]string              `json:"attributes,omitempty"`
	Notes             []string                       `json:"notes,omitempty"`
	SideEffect        string                         `json:"side_effect"`
}
