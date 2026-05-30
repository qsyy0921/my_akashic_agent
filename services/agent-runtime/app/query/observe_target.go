package query

type ObserveTargetsView struct {
	Targets    []ObserveTargetView `json:"targets"`
	Totals     map[string]int      `json:"totals"`
	Notes      []string            `json:"notes,omitempty"`
	SideEffect string              `json:"side_effect"`
}

type ObserveTargetView struct {
	TargetID     string                   `json:"target_id"`
	Channel      ObserveTargetChannelView `json:"channel"`
	ObserveOnly  bool                     `json:"observe_only"`
	ReplyAllowed bool                     `json:"reply_allowed"`
	RequireAt    bool                     `json:"require_at"`
	AllowFrom    []string                 `json:"allow_from,omitempty"`
	Enabled      bool                     `json:"enabled"`
	Source       string                   `json:"source"`
	Metadata     map[string]string        `json:"metadata,omitempty"`
	UpdatedAt    string                   `json:"updated_at"`
}

type ObserveTargetChannelView struct {
	Kind             string `json:"kind"`
	AccountID        string `json:"account_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
}
