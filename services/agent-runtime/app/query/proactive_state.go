package query

type ProactiveDeliveryFilter struct {
	Limit       int
	SessionKey  string
	DeliveryKey string
}

type ProactiveDeliveryView struct {
	SessionKey  string `json:"session_key"`
	DeliveryKey string `json:"delivery_key"`
	SentAt      string `json:"sent_at"`
}

type ProactiveDuplicateView struct {
	Duplicate   bool   `json:"duplicate"`
	SessionKey  string `json:"session_key"`
	DeliveryKey string `json:"delivery_key"`
	WindowHours int    `json:"window_hours"`
}

type ProactiveCountView struct {
	Count       int    `json:"count"`
	SessionKey  string `json:"session_key"`
	WindowHours int    `json:"window_hours"`
}

type ProactiveSeenView struct {
	Seen       bool   `json:"seen"`
	SourceKey  string `json:"source_key"`
	ItemID     string `json:"item_id"`
	TTLHours   int    `json:"ttl_hours"`
	SeenAt     string `json:"seen_at,omitempty"`
	SideEffect string `json:"side_effect"`
}

type ProactiveRejectionCooldownView struct {
	Cooled     bool   `json:"cooled"`
	SourceKey  string `json:"source_key"`
	ItemID     string `json:"item_id"`
	TTLHours   int    `json:"ttl_hours"`
	RejectedAt string `json:"rejected_at,omitempty"`
	SideEffect string `json:"side_effect"`
}

type ProactiveMarkItemsView struct {
	Count      int    `json:"count"`
	Timestamp  string `json:"timestamp"`
	SideEffect string `json:"side_effect"`
}

type ProactiveCleanupView struct {
	RemovedDeliveries         int    `json:"removed_deliveries"`
	RemovedSeenItems          int    `json:"removed_seen_items"`
	RemovedContextOnly        int    `json:"removed_context_only"`
	RemovedRejectionCooldowns int    `json:"removed_rejection_cooldowns"`
	Timestamp                 string `json:"timestamp"`
	SideEffect                string `json:"side_effect"`
}

type ProactiveTimestampView struct {
	SessionKey string `json:"session_key"`
	Key        string `json:"key"`
	Timestamp  string `json:"timestamp"`
	Found      bool   `json:"found"`
}

type ProactiveDriftSkillStateView struct {
	SkillName  string `json:"skill_name"`
	LastRunAt  string `json:"last_run_at,omitempty"`
	RunCount   int    `json:"run_count"`
	Status     string `json:"status"`
	Next       string `json:"next"`
	Found      bool   `json:"found"`
	SideEffect string `json:"side_effect"`
}

type ProactiveDriftRecentRunView struct {
	SkillName     string `json:"skill"`
	RunAt         string `json:"run_at"`
	OneLine       string `json:"one_line"`
	MessageResult string `json:"message_result"`
}

type ProactiveDriftSummaryView struct {
	Version    int                           `json:"version"`
	RecentRuns []ProactiveDriftRecentRunView `json:"recent_runs"`
	Note       string                        `json:"note"`
	SideEffect string                        `json:"side_effect"`
}

type ProactiveDriftFinishView struct {
	SkillState ProactiveDriftSkillStateView `json:"skill_state"`
	RecentRun  ProactiveDriftRecentRunView  `json:"recent_run"`
	Note       string                       `json:"note"`
	SideEffect string                       `json:"side_effect"`
}

type ProactiveAnyActionQuotaView struct {
	QuotaKey     string `json:"quota_key"`
	WindowKey    string `json:"window_key"`
	NextResetAt  string `json:"next_reset_at"`
	Used         int    `json:"used"`
	LastActionAt string `json:"last_action_at"`
	Found        bool   `json:"found"`
	SideEffect   string `json:"side_effect"`
}
