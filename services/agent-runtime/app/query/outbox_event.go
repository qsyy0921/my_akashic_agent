package query

type OutboxDeliveryEventFilter struct {
	DeliveryID string
	Status     string
	EventType  string
	Limit      int
}

type OutboxDeliveryEventView struct {
	EventID        string            `json:"event_id"`
	DeliveryID     string            `json:"delivery_id"`
	Channel        OutboxChannelView `json:"channel"`
	EventType      string            `json:"event_type"`
	Status         string            `json:"status"`
	Attempt        int               `json:"attempt"`
	MaxAttempts    int               `json:"max_attempts"`
	LeaseOwner     string            `json:"lease_owner,omitempty"`
	LeaseExpiresAt string            `json:"lease_expires_at,omitempty"`
	ErrorKind      string            `json:"error_kind,omitempty"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	OccurredAt     string            `json:"occurred_at"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}
