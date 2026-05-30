package query

type OutboxMetricsFilter struct {
	DeliveryLimit int
	EventLimit    int
}

type OutboxChannelKindMetricsView struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
}

type OutboxThroughputMetricsView struct {
	EventsByType   map[string]int `json:"events_by_type"`
	Queued         int            `json:"queued"`
	Leased         int            `json:"leased"`
	Dispatching    int            `json:"dispatching"`
	Succeeded      int            `json:"succeeded"`
	Failed         int            `json:"failed"`
	Retry          int            `json:"retry"`
	DeadLettered   int            `json:"dead_lettered"`
	TerminalEvents int            `json:"terminal_events"`
}

type OutboxDeadLetterSampleView struct {
	DeliveryID   string `json:"delivery_id"`
	ChannelKind  string `json:"channel_kind"`
	EventType    string `json:"event_type"`
	Status       string `json:"status"`
	ErrorKind    string `json:"error_kind,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Attempt      int    `json:"attempt"`
	MaxAttempts  int    `json:"max_attempts"`
	OccurredAt   string `json:"occurred_at"`
}

type OutboxDeadLetterMetricsView struct {
	CurrentTotal  int                          `json:"current_total"`
	ByChannelKind map[string]int               `json:"by_channel_kind"`
	Recent        []OutboxDeadLetterSampleView `json:"recent"`
}

type OutboxMetricsView struct {
	SampledDeliveries       int                                     `json:"sampled_deliveries"`
	SampledEvents           int                                     `json:"sampled_events"`
	DeliveriesByStatus      map[string]int                          `json:"deliveries_by_status"`
	DeliveriesByChannelKind map[string]OutboxChannelKindMetricsView `json:"deliveries_by_channel_kind"`
	Throughput              OutboxThroughputMetricsView             `json:"throughput"`
	DeadLetters             OutboxDeadLetterMetricsView             `json:"dead_letters"`
	Notes                   []string                                `json:"notes,omitempty"`
}
