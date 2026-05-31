package query

type RuntimeOverviewFilter struct {
	Limit             int
	EventLimit        int
	StaleAfterSeconds int
}

type RuntimeOverviewErrorView struct {
	Endpoint string `json:"endpoint"`
	Error    string `json:"error"`
}

type RuntimeOverviewStatusView struct {
	RuntimeAvailable bool                       `json:"runtime_available"`
	HealthAvailable  bool                       `json:"health_available"`
	Partial          bool                       `json:"partial"`
	Errors           []RuntimeOverviewErrorView `json:"errors"`
}

type RuntimeOverviewCardView struct {
	ID     string         `json:"id"`
	Label  string         `json:"label"`
	Value  any            `json:"value"`
	Status string         `json:"status"`
	Detail map[string]any `json:"detail,omitempty"`
}

type AgentJobWorkerCoverageView struct {
	JobType             string   `json:"job_type"`
	ExpectedWorkerTypes []string `json:"expected_worker_types,omitempty"`
	HighPressure        bool     `json:"high_pressure"`
	PressureReason      string   `json:"pressure_reason,omitempty"`
	WorkerCount         int      `json:"worker_count"`
	ActiveWorkers       int      `json:"active_workers"`
	RunningWorkers      int      `json:"running_workers"`
	FailedWorkers       int      `json:"failed_workers"`
	StaleWorkers        int      `json:"stale_workers"`
	CoverageStatus      string   `json:"coverage_status"`
	CoverageReason      string   `json:"coverage_reason,omitempty"`
}

type RuntimeOverviewView struct {
	Summary                map[string]any                   `json:"summary"`
	Cards                  []RuntimeOverviewCardView        `json:"cards"`
	DeliveryAdapters       []DeliveryAdapterDiagnosticsView `json:"delivery_adapters,omitempty"`
	QueueBackend           QueueBackendView                 `json:"queue_backend"`
	RuntimeConfig          RuntimeConfigView                `json:"runtime_config,omitempty"`
	RuntimeWorkers         RuntimeWorkerDiagnosticsView     `json:"runtime_workers"`
	AgentWorkers           AgentWorkerStatusesView          `json:"agent_workers"`
	ObserveTargets         ObserveTargetsView               `json:"observe_targets"`
	ObserveCapture         ObserveCaptureDiagnosticsView    `json:"observe_capture"`
	ReceiverStatuses       ReceiverStatusesView             `json:"receiver_statuses"`
	ReceiverLeases         ReceiverLeasesView               `json:"receiver_leases"`
	SchedulerJobs          SchedulerJobDiagnosticsView      `json:"scheduler_jobs"`
	SendLedgerMetrics      SendLedgerMetricsView            `json:"send_ledger_metrics"`
	InboxMetrics           InboxMetricsView                 `json:"inbox_metrics"`
	InboundDedupe          InboundDedupeMetricsView         `json:"inbound_dedupe_metrics"`
	AgentJobMetrics        AgentJobMetricsView              `json:"agent_job_metrics"`
	AgentJobWorkerCoverage []AgentJobWorkerCoverageView     `json:"agent_job_worker_coverage,omitempty"`
	OutboxMetrics          OutboxMetricsView                `json:"outbox_metrics"`
	Diagnostics            KnowledgeWorkerDiagnosticsView   `json:"diagnostics"`
	Status                 RuntimeOverviewStatusView        `json:"status"`
}
