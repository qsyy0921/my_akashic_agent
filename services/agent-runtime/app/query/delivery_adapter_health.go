package query

type DeliveryAdapterHealthFilter struct {
	TimeoutSeconds int
}

type DeliveryAdapterHealthSummaryView struct {
	Items  []DeliveryAdapterHealthView `json:"items"`
	Totals map[string]int              `json:"totals"`
	Notes  []string                    `json:"notes,omitempty"`
}

type DeliveryAdapterHealthView struct {
	Provider           string            `json:"provider"`
	Channel            string            `json:"channel"`
	Transport          string            `json:"transport"`
	Endpoint           string            `json:"endpoint,omitempty"`
	Healthy            bool              `json:"healthy"`
	Reachable          bool              `json:"reachable"`
	Authenticated      bool              `json:"authenticated"`
	AccountID          string            `json:"account_id,omitempty"`
	AccountName        string            `json:"account_name,omitempty"`
	ErrorKind          string            `json:"error_kind,omitempty"`
	ErrorMessage       string            `json:"error_message,omitempty"`
	CheckedAt          string            `json:"checked_at"`
	LatencyMs          int               `json:"latency_ms,omitempty"`
	SideEffect         string            `json:"side_effect"`
	Attributes         map[string]string `json:"attributes,omitempty"`
	AccessTokenPresent bool              `json:"access_token_present"`
	Notes              []string          `json:"notes,omitempty"`
}
