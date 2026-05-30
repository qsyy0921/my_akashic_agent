package query

type InboxEventFilter struct {
	Limit            int
	AfterSeq         int
	AfterSeqSet      bool
	Order            string
	ChannelKind      string
	AccountID        string
	ConversationID   string
	ConversationType string
	SenderID         string
	DecisionAction   string
	ObserveOnly      string
}

type InboxChannelView struct {
	Kind             string `json:"kind"`
	AccountID        string `json:"account_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
}

type InboxSenderView struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
	Kind        string `json:"kind,omitempty"`
}

type InboxProvenanceView struct {
	Type           string `json:"type,omitempty"`
	FromBotID      string `json:"from_bot_id,omitempty"`
	ContentHash    string `json:"content_hash,omitempty"`
	Nonce          string `json:"nonce,omitempty"`
	Hop            int    `json:"hop,omitempty"`
	HasProtocolTag bool   `json:"has_protocol_tag,omitempty"`
}

type InboxEventView struct {
	EventID         string                 `json:"event_id"`
	Channel         InboxChannelView       `json:"channel"`
	Sender          InboxSenderView        `json:"sender"`
	Content         string                 `json:"content"`
	Timestamp       string                 `json:"timestamp"`
	ReceivedAt      string                 `json:"received_at"`
	AttachmentCount int                    `json:"attachment_count"`
	Attachments     []ShadowAttachmentView `json:"attachments,omitempty"`
	DecisionAction  string                 `json:"decision_action"`
	DecisionReason  string                 `json:"decision_reason"`
	ObserveOnly     bool                   `json:"observe_only"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
	Provenance      InboxProvenanceView    `json:"provenance,omitempty"`
}
