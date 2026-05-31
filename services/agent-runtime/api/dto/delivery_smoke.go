package dto

type DeliverySmokeReadinessRequest struct {
	Cases                 []DeliverySmokeCaseRequest `json:"cases,omitempty"`
	GroupIDs              []string                   `json:"group_ids,omitempty"`
	ChannelByAccount      map[string]string          `json:"channel_by_account,omitempty"`
	IncludeSyntheticMedia bool                       `json:"include_synthetic_media,omitempty"`
}

type OutboundCutoverPlanRequest struct {
	DeliverySmokeReadinessRequest
	DesiredExecutionOwner string `json:"desired_execution_owner,omitempty"`
}

type DeliverySmokeCaseRequest struct {
	Name             string                           `json:"name,omitempty"`
	ChannelKind      string                           `json:"channel_kind,omitempty"`
	AccountID        string                           `json:"account_id,omitempty"`
	ConversationID   string                           `json:"conversation_id,omitempty"`
	ConversationType string                           `json:"conversation_type,omitempty"`
	Content          string                           `json:"content,omitempty"`
	Attachments      []DeliverySmokeAttachmentRequest `json:"attachments,omitempty"`
	Metadata         map[string]string                `json:"metadata,omitempty"`
}

type DeliverySmokeAttachmentRequest struct {
	Kind     string `json:"kind,omitempty"`
	URL      string `json:"url,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}
