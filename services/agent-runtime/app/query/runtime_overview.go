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

type RuntimeOverviewView struct {
	Summary           map[string]any                   `json:"summary"`
	Cards             []RuntimeOverviewCardView        `json:"cards"`
	DeliveryAdapters  []DeliveryAdapterDiagnosticsView `json:"delivery_adapters,omitempty"`
	QueueBackend      QueueBackendView                 `json:"queue_backend"`
	RuntimeWorkers    RuntimeWorkerDiagnosticsView     `json:"runtime_workers"`
	SendLedgerMetrics SendLedgerMetricsView            `json:"send_ledger_metrics"`
	InboxMetrics      InboxMetricsView                 `json:"inbox_metrics"`
	AgentJobMetrics   AgentJobMetricsView              `json:"agent_job_metrics"`
	OutboxMetrics     OutboxMetricsView                `json:"outbox_metrics"`
	Diagnostics       KnowledgeWorkerDiagnosticsView   `json:"diagnostics"`
	Status            RuntimeOverviewStatusView        `json:"status"`
}
