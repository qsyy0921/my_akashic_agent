package query

type AgentJobMetricsFilter struct {
	JobLimit   int
	EventLimit int
}

type AgentJobTypeMetricsView struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
}

type AgentJobThroughputMetricsView struct {
	EventsByType   map[string]int `json:"events_by_type"`
	Created        int            `json:"created"`
	Leased         int            `json:"leased"`
	Renewed        int            `json:"renewed"`
	Running        int            `json:"running"`
	Succeeded      int            `json:"succeeded"`
	Failed         int            `json:"failed"`
	Retry          int            `json:"retry"`
	LeaseExpired   int            `json:"lease_expired"`
	Cancelled      int            `json:"cancelled"`
	TerminalEvents int            `json:"terminal_events"`
}

type AgentJobDeadLetterSampleView struct {
	JobID       string `json:"job_id"`
	JobType     string `json:"job_type"`
	EventType   string `json:"event_type"`
	Status      string `json:"status"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
	OccurredAt  string `json:"occurred_at"`
}

type AgentJobDeadLetterMetricsView struct {
	CurrentTotal int                            `json:"current_total"`
	ByType       map[string]int                 `json:"by_type"`
	Recent       []AgentJobDeadLetterSampleView `json:"recent"`
}

type AgentJobMetricsView struct {
	SampledJobs   int                                `json:"sampled_jobs"`
	SampledEvents int                                `json:"sampled_events"`
	JobsByStatus  map[string]int                     `json:"jobs_by_status"`
	JobsByType    map[string]AgentJobTypeMetricsView `json:"jobs_by_type"`
	Throughput    AgentJobThroughputMetricsView      `json:"throughput"`
	DeadLetters   AgentJobDeadLetterMetricsView      `json:"dead_letters"`
	Notes         []string                           `json:"notes,omitempty"`
}
