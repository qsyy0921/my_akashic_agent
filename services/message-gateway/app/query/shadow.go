package query

type ShadowObservedEventView struct {
	EventID          string            `json:"event_id"`
	Platform         string            `json:"platform"`
	AccountID        string            `json:"account_id"`
	ConversationID   string            `json:"conversation_id"`
	ConversationType string            `json:"conversation_type"`
	SenderID         string            `json:"sender_id"`
	Content          string            `json:"content"`
	Timestamp        string            `json:"timestamp"`
	AttachmentCount  int               `json:"attachment_count"`
	DecisionAction   string            `json:"decision_action"`
	DecisionReason   string            `json:"decision_reason"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}
