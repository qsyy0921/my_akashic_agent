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
	AssetID                   string                `json:"asset_id"`
	Channel                   MediaAssetChannelView `json:"channel"`
	SourceMessageID           string                `json:"source_message_id"`
	SenderID                  string                `json:"sender_id"`
	Kind                      string                `json:"kind"`
	MimeType                  string                `json:"mime_type,omitempty"`
	Name                      string                `json:"name,omitempty"`
	SizeBytes                 int64                 `json:"size_bytes,omitempty"`
	ContentStatus             string                `json:"content_status"`
	ContentReason             string                `json:"content_reason"`
	ContentEndpoint           string                `json:"content_endpoint"`
	ContentAccessPlanEndpoint string                `json:"content_access_plan_endpoint"`
	ContentMimeType           string                `json:"content_mime_type,omitempty"`
	ContentSizeBytes          int64                 `json:"content_size_bytes,omitempty"`
	UpdatedAt                 string                `json:"updated_at"`
}

type MediaAssetContentDiagnosticsView struct {
	Items      []MediaAssetContentDiagnosticItemView `json:"items"`
	Totals     map[string]int                        `json:"totals"`
	SideEffect string                                `json:"side_effect"`
	Notes      []string                              `json:"notes,omitempty"`
}

type MediaAssetContentAccessStep struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Endpoint    string            `json:"endpoint,omitempty"`
	Method      string            `json:"method,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type MediaAssetContentAccessPlanView struct {
	Ready            bool                          `json:"ready"`
	Reason           string                        `json:"reason"`
	Blockers         []string                      `json:"blockers,omitempty"`
	AssetID          string                        `json:"asset_id,omitempty"`
	Asset            *MediaAssetView               `json:"asset,omitempty"`
	ContentEndpoint  string                        `json:"content_endpoint,omitempty"`
	ContentMimeType  string                        `json:"content_mime_type,omitempty"`
	ContentSizeBytes int64                         `json:"content_size_bytes,omitempty"`
	RequiredSteps    []MediaAssetContentAccessStep `json:"required_steps"`
	VerifySteps      []MediaAssetContentAccessStep `json:"verify_steps"`
	FallbackSteps    []MediaAssetContentAccessStep `json:"fallback_steps"`
	SideEffect       string                        `json:"side_effect"`
	Notes            []string                      `json:"notes,omitempty"`
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

type MediaAssetRetentionCleanupPreflightFilter struct {
	RetentionFilter MediaAssetRetentionDiagnosticsFilter
	TargetID        string
	OperatorID      string
	ApprovalID      string
}

type MediaAssetRetentionCleanupPreflightView struct {
	Ready            bool                               `json:"ready"`
	Reason           string                             `json:"reason"`
	Blockers         []string                           `json:"blockers,omitempty"`
	TargetKind       string                             `json:"target_kind"`
	TargetID         string                             `json:"target_id"`
	Action           string                             `json:"action"`
	OperatorID       string                             `json:"operator_id"`
	ApprovalID       string                             `json:"approval_id"`
	CandidateCount   int                                `json:"candidate_count"`
	AssetCount       int                                `json:"asset_count"`
	Plan             MediaAssetRetentionPlanView        `json:"plan"`
	ControlPreflight ControlMutationPreflightView       `json:"control_preflight"`
	SuggestedAudit   *ControlMutationSuggestedAuditView `json:"suggested_audit,omitempty"`
	Notes            []string                           `json:"notes,omitempty"`
	SideEffect       string                             `json:"side_effect"`
}

type MediaAssetRetentionCleanupView struct {
	Ready           bool                                    `json:"ready"`
	Applied         bool                                    `json:"applied"`
	DryRun          bool                                    `json:"dry_run"`
	Reason          string                                  `json:"reason"`
	Blockers        []string                                `json:"blockers,omitempty"`
	TargetKind      string                                  `json:"target_kind"`
	TargetID        string                                  `json:"target_id"`
	Action          string                                  `json:"action"`
	OperatorID      string                                  `json:"operator_id"`
	ApprovalID      string                                  `json:"approval_id"`
	MutationID      string                                  `json:"mutation_id,omitempty"`
	CandidateCount  int                                     `json:"candidate_count"`
	DeletedCount    int                                     `json:"deleted_count"`
	DeletedAssetIDs []string                                `json:"deleted_asset_ids,omitempty"`
	Candidates      []MediaAssetRetentionDiagnosticItemView `json:"candidates,omitempty"`
	Preflight       MediaAssetRetentionCleanupPreflightView `json:"preflight"`
	AppliedAudit    *ControlMutationAuditView               `json:"applied_audit,omitempty"`
	FailedAudit     *ControlMutationAuditView               `json:"failed_audit,omitempty"`
	Notes           []string                                `json:"notes,omitempty"`
	SideEffect      string                                  `json:"side_effect"`
}
