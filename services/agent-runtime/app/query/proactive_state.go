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

type ProactiveTimestampView struct {
	SessionKey string `json:"session_key"`
	Key        string `json:"key"`
	Timestamp  string `json:"timestamp"`
	Found      bool   `json:"found"`
}
