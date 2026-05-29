package query

type MediaAssetFilter struct {
	Limit                 int
	ChannelKind           string
	AccountID             string
	ConversationID        string
	ConversationType      string
	SourceMessageID       string
	SourceMessageIDSuffix string
	Kind                  string
}

type MediaAssetChannelView struct {
	Kind             string `json:"kind"`
	AccountID        string `json:"account_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
}

type MediaAssetView struct {
	AssetID         string                `json:"asset_id"`
	Channel         MediaAssetChannelView `json:"channel"`
	SourceMessageID string                `json:"source_message_id"`
	SenderID        string                `json:"sender_id"`
	Kind            string                `json:"kind"`
	URL             string                `json:"url,omitempty"`
	MimeType        string                `json:"mime_type,omitempty"`
	Name            string                `json:"name,omitempty"`
	SizeBytes       int64                 `json:"size_bytes,omitempty"`
	ContentHash     string                `json:"content_hash,omitempty"`
	Retention       string                `json:"retention"`
	CreatedAt       string                `json:"created_at"`
	UpdatedAt       string                `json:"updated_at"`
	Metadata        map[string]string     `json:"metadata,omitempty"`
}
