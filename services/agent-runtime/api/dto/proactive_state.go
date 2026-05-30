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
