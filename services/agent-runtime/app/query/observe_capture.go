package query

type ObserveCaptureDiagnosticsFilter struct {
	Limit int
}

type ObserveCaptureTargetDiagnosticsView struct {
	TargetID                 string                   `json:"target_id"`
	Channel                  ObserveTargetChannelView `json:"channel"`
	Enabled                  bool                     `json:"enabled"`
	ObserveOnly              bool                     `json:"observe_only"`
	ReceiverConnected        bool                     `json:"receiver_connected"`
	ReceiverStatusConnected  bool                     `json:"receiver_status_connected"`
	ReceiverActivityRecent   bool                     `json:"receiver_activity_recent"`
	ReceiverConnectionSource string                   `json:"receiver_connection_source,omitempty"`
	ReceiverID               string                   `json:"receiver_id,omitempty"`
	ReceiverStatus           string                   `json:"receiver_status,omitempty"`
	Status                   string                   `json:"status"`
	Blockers                 []string                 `json:"blockers,omitempty"`
	InboxEvents              int                      `json:"inbox_events"`
	TextEvents               int                      `json:"text_events"`
	AttachmentEvents         int                      `json:"attachment_events"`
	AttachmentCount          int                      `json:"attachment_count"`
	MediaAssets              int                      `json:"media_assets"`
	ImageAssets              int                      `json:"image_assets"`
	FileAssets               int                      `json:"file_assets"`
	ContentReadyAssets       int                      `json:"content_ready_assets"`
	ContentUnavailableAssets int                      `json:"content_unavailable_assets"`
	ContentForbiddenAssets   int                      `json:"content_forbidden_assets"`
	ContentDisabledAssets    int                      `json:"content_disabled_assets"`
	LatestReceivedAt         string                   `json:"latest_received_at,omitempty"`
	LatestAssetAt            string                   `json:"latest_asset_at,omitempty"`
	Coverage                 ObserveCaptureCoverage   `json:"coverage"`
}

type ObserveCaptureCoverage struct {
	TextSeen          bool `json:"text_seen"`
	AttachmentSeen    bool `json:"attachment_seen"`
	ImageSeen         bool `json:"image_seen"`
	FileSeen          bool `json:"file_seen"`
	MediaContentReady bool `json:"media_content_ready"`
}

type ObserveCaptureDiagnosticsView struct {
	Targets    []ObserveCaptureTargetDiagnosticsView `json:"targets"`
	Totals     map[string]int                        `json:"totals"`
	Notes      []string                              `json:"notes"`
	SideEffect string                                `json:"side_effect"`
}
