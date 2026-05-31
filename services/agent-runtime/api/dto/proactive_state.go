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
