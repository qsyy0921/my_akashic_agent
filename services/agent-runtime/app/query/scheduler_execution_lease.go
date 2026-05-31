package query

type SchedulerExecutionLeaseView struct {
	JobID             string            `json:"job_id"`
	HolderID          string            `json:"holder_id,omitempty"`
	LeaseToken        string            `json:"lease_token,omitempty"`
	LeaseTokenPresent bool              `json:"lease_token_present"`
	Active            bool              `json:"active"`
	Acquired          *bool             `json:"acquired,omitempty"`
	DeniedReason      string            `json:"denied_reason,omitempty"`
	ExpiresAt         string            `json:"expires_at,omitempty"`
	AcquiredAt        string            `json:"acquired_at,omitempty"`
	UpdatedAt         string            `json:"updated_at,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	SideEffect        string            `json:"side_effect,omitempty"`
}

type SchedulerExecutionLeasesView struct {
	Leases     []SchedulerExecutionLeaseView `json:"leases"`
	Totals     map[string]int                `json:"totals"`
	Notes      []string                      `json:"notes"`
	SideEffect string                        `json:"side_effect"`
}
