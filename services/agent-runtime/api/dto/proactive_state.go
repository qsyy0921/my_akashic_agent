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
