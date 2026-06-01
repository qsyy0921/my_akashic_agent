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

type MediaAssetRetentionDiagnosticsFilter struct {
	AssetID               string
	Limit                 int
	ChannelKind           string
	AccountID             string
	ConversationID        string
	ConversationType      string
	SourceMessageID       string
	SourceMessageIDSuffix string
	Kind                  string
	Timestamp             string
	DefaultTTLHours       int
	EphemeralTTLHours     int
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

type MediaAssetRetentionDiagnosticItemView struct {
	AssetID         string                `json:"asset_id"`
	Channel         MediaAssetChannelView `json:"channel"`
	SourceMessageID string                `json:"source_message_id"`
	SenderID        string                `json:"sender_id"`
	Kind            string                `json:"kind"`
	MimeType        string                `json:"mime_type,omitempty"`
	Name            string                `json:"name,omitempty"`
	SizeBytes       int64                 `json:"size_bytes,omitempty"`
	Retention       string                `json:"retention"`
	RetentionClass  string                `json:"retention_class"`
	CleanupDue      bool                  `json:"cleanup_due"`
	AgeSeconds      int                   `json:"age_seconds"`
	TTLSeconds      int                   `json:"ttl_seconds,omitempty"`
	CleanupAfter    string                `json:"cleanup_after,omitempty"`
	CleanupReason   string                `json:"cleanup_reason"`
	CreatedAt       string                `json:"created_at"`
	UpdatedAt       string                `json:"updated_at"`
}

type MediaAssetRetentionDiagnosticsView struct {
	Items      []MediaAssetRetentionDiagnosticItemView `json:"items"`
	Totals     map[string]int                          `json:"totals"`
	SideEffect string                                  `json:"side_effect"`
	Notes      []string                                `json:"notes,omitempty"`
}

type MediaAssetRetentionPlanStep struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Endpoint    string            `json:"endpoint,omitempty"`
	Method      string            `json:"method,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type MediaAssetRetentionPlanView struct {
	Ready          bool                                    `json:"ready"`
	Reason         string                                  `json:"reason"`
	Blockers       []string                                `json:"blockers,omitempty"`
	AssetCount     int                                     `json:"asset_count"`
	CandidateCount int                                     `json:"candidate_count"`
	Candidates     []MediaAssetRetentionDiagnosticItemView `json:"candidates,omitempty"`
	RequiredSteps  []MediaAssetRetentionPlanStep           `json:"required_steps"`
	VerifySteps    []MediaAssetRetentionPlanStep           `json:"verify_steps"`
	RollbackSteps  []MediaAssetRetentionPlanStep           `json:"rollback_steps"`
	Diagnostics    MediaAssetRetentionDiagnosticsView      `json:"diagnostics"`
	SideEffect     string                                  `json:"side_effect"`
	Notes          []string                                `json:"notes,omitempty"`
}
