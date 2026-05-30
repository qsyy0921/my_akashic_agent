package dto

type SyncObserveTargetsRequest struct {
	Source  string                 `json:"source,omitempty"`
	Targets []ObserveTargetRequest `json:"targets"`
}

type ObserveTargetRequest struct {
	TargetID     string            `json:"target_id,omitempty"`
	Channel      ChannelDTO        `json:"channel"`
	ObserveOnly  bool              `json:"observe_only"`
	ReplyAllowed bool              `json:"reply_allowed"`
	RequireAt    bool              `json:"require_at"`
	AllowFrom    []string          `json:"allow_from,omitempty"`
	Enabled      bool              `json:"enabled"`
	Source       string            `json:"source,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}
