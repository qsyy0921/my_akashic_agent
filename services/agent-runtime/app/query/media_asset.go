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

type MediaAssetContentDiagnosticsFilter struct {
	AssetID               string
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

type MediaAssetContentDiagnosticItemView struct {
	AssetID          string                `json:"asset_id"`
	Channel          MediaAssetChannelView `json:"channel"`
	SourceMessageID  string                `json:"source_message_id"`
	SenderID         string                `json:"sender_id"`
	Kind             string                `json:"kind"`
	MimeType         string                `json:"mime_type,omitempty"`
	Name             string                `json:"name,omitempty"`
	SizeBytes        int64                 `json:"size_bytes,omitempty"`
	ContentStatus    string                `json:"content_status"`
	ContentReason    string                `json:"content_reason"`
	ContentEndpoint  string                `json:"content_endpoint"`
	ContentMimeType  string                `json:"content_mime_type,omitempty"`
	ContentSizeBytes int64                 `json:"content_size_bytes,omitempty"`
	UpdatedAt        string                `json:"updated_at"`
}

type MediaAssetContentDiagnosticsView struct {
	Items      []MediaAssetContentDiagnosticItemView `json:"items"`
	Totals     map[string]int                        `json:"totals"`
	SideEffect string                                `json:"side_effect"`
	Notes      []string                              `json:"notes,omitempty"`
}
