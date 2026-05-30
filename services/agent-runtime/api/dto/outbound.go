package dto

type SendMessageRequest struct {
	EventID         string            `json:"event_id"`
	Channel         ChannelDTO        `json:"channel"`
	Content         string            `json:"content"`
	Attachments     []AttachmentDTO   `json:"attachments,omitempty"`
	Timestamp       string            `json:"timestamp,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	WithBotProtocol bool              `json:"with_bot_protocol,omitempty"`
	ProtocolFromBot string            `json:"protocol_from_bot,omitempty"`
	ProtocolNonce   string            `json:"protocol_nonce,omitempty"`
	ProtocolNextHop int               `json:"protocol_next_hop,omitempty"`
}

type OutboxStateRequest struct {
	Timestamp    string `json:"timestamp,omitempty"`
	ErrorKind    string `json:"error_kind,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type OutboxLeaseRequest struct {
	WorkerID   string `json:"worker_id,omitempty"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}
