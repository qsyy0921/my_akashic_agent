package dto

type IngestMessageRequest struct {
	EventID     string            `json:"event_id"`
	Channel     ChannelDTO        `json:"channel"`
	Sender      SenderDTO         `json:"sender"`
	Content     string            `json:"content"`
	Attachments []AttachmentDTO   `json:"attachments,omitempty"`
	Timestamp   string            `json:"timestamp,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ShadowIngestResponse struct {
	Decision LoopDecisionDTO `json:"decision"`
}

type LoopDecisionDTO struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type ChannelDTO struct {
	Kind             string `json:"kind,omitempty"`
	Platform         string `json:"platform,omitempty"`
	AccountID        string `json:"account_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
}

func (c ChannelDTO) RoutePlatform() string {
	if c.Platform != "" {
		return c.Platform
	}
	return c.Kind
}

type SenderDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
	Kind        string `json:"kind,omitempty"`
}

type AttachmentDTO struct {
	ID        string `json:"id,omitempty"`
	Kind      string `json:"kind"`
	URL       string `json:"url,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Name      string `json:"name,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

