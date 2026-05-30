package query

type RuntimeWorkerDiagnosticsView struct {
	Workers []RuntimeWorkerView `json:"workers"`
	Totals  map[string]int      `json:"totals"`
	Notes   []string            `json:"notes,omitempty"`
}

type RuntimeWorkerView struct {
	Name                string            `json:"name"`
	Kind                string            `json:"kind"`
	Enabled             bool              `json:"enabled"`
	Running             bool              `json:"running"`
	Mode                string            `json:"mode,omitempty"`
	ExecutionScope      string            `json:"execution_scope,omitempty"`
	WorkerID            string            `json:"worker_id,omitempty"`
	IntervalSeconds     int               `json:"interval_seconds,omitempty"`
	LeaseTTLSeconds     int               `json:"lease_ttl_seconds,omitempty"`
	BatchSize           int               `json:"batch_size,omitempty"`
	RunOnStart          bool              `json:"run_on_start,omitempty"`
	ConsumerConcurrency int               `json:"consumer_concurrency,omitempty"`
	MaxInFlight         int               `json:"max_in_flight,omitempty"`
	Attributes          map[string]string `json:"attributes,omitempty"`
	Notes               []string          `json:"notes,omitempty"`
}
