package dto

type CheckInboundDedupeRequest struct {
	Scope      string            `json:"scope"`
	MessageKey string            `json:"message_key"`
	TTLSeconds int               `json:"ttl_seconds,omitempty"`
	Timestamp  string            `json:"timestamp,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
