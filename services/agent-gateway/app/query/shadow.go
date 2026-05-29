package query

type ShadowObservedEventView struct {
	EventID          string                 `json:"event_id"`
	Platform         string                 `json:"platform"`
	AccountID        string                 `json:"account_id"`
	ConversationID   string                 `json:"conversation_id"`
	ConversationType string                 `json:"conversation_type"`
	SenderID         string                 `json:"sender_id"`
	Content          string                 `json:"content"`
	Timestamp        string                 `json:"timestamp"`
	AttachmentCount  int                    `json:"attachment_count"`
	Attachments      []ShadowAttachmentView `json:"attachments,omitempty"`
	DecisionAction   string                 `json:"decision_action"`
	DecisionReason   string                 `json:"decision_reason"`
	Metadata         map[string]string      `json:"metadata,omitempty"`
}

type ShadowAttachmentView struct {
	ID        string `json:"id,omitempty"`
	Kind      string `json:"kind"`
	URL       string `json:"url,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Name      string `json:"name,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}
