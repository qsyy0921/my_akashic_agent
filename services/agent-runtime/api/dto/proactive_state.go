package dto

type RecordProactiveDeliveryRequest struct {
	SessionKey  string `json:"session_key"`
	DeliveryKey string `json:"delivery_key"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type RecordProactiveSessionRequest struct {
	SessionKey string `json:"session_key"`
	Timestamp  string `json:"timestamp,omitempty"`
}

type RecordProactiveDriftFinishRequest struct {
	SkillUsed     string `json:"skill_used"`
	OneLine       string `json:"one_line"`
	Next          string `json:"next"`
	MessageResult string `json:"message_result"`
	Note          string `json:"note,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
}

type RecordProactiveTimestampRequest struct {
	Timestamp string `json:"timestamp,omitempty"`
}

type ProactiveAnyActionQuotaRequest struct {
	QuotaKey  string `json:"quota_key"`
	ResetHour int    `json:"reset_hour"`
	Timezone  string `json:"timezone"`
	Timestamp string `json:"timestamp,omitempty"`
}

type ProactiveSourceItemEntry struct {
	SourceKey string `json:"source_key"`
	ItemID    string `json:"item_id"`
}

type MarkProactiveItemsRequest struct {
	Entries   []ProactiveSourceItemEntry `json:"entries"`
	Timestamp string                     `json:"timestamp,omitempty"`
}

type MarkProactiveRejectionCooldownRequest struct {
	Entries   []ProactiveSourceItemEntry `json:"entries"`
	Hours     int                        `json:"hours"`
	Timestamp string                     `json:"timestamp,omitempty"`
}

type CleanupProactiveStateRequest struct {
	SeenTTLHours              int    `json:"seen_ttl_hours"`
	DeliveryTTLHours          int    `json:"delivery_ttl_hours"`
	ContextOnlyTTLHours       int    `json:"context_only_ttl_hours,omitempty"`
	RejectionCooldownTTLHours int    `json:"rejection_cooldown_ttl_hours,omitempty"`
	Timestamp                 string `json:"timestamp,omitempty"`
}
