package query

type OutboxAttachmentView struct {
	ID        string `json:"id,omitempty"`
	Kind      string `json:"kind"`
	URL       string `json:"url,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Name      string `json:"name,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type OutboxChannelView struct {
	Kind             string `json:"kind"`
	AccountID        string `json:"account_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
}

type OutboxDeliveryView struct {
	EventID        string                 `json:"event_id"`
	Channel        OutboxChannelView      `json:"channel"`
	Content        string                 `json:"content"`
	Attachments    []OutboxAttachmentView `json:"attachments,omitempty"`
	Status         string                 `json:"status"`
	Attempts       int                    `json:"attempts"`
	MaxAttempts    int                    `json:"max_attempts"`
	LeaseOwner     string                 `json:"lease_owner,omitempty"`
	LeaseExpiresAt string                 `json:"lease_expires_at,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
}
