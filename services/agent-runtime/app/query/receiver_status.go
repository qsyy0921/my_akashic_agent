package query

type ReceiverStatusView struct {
	ReceiverID  string            `json:"receiver_id"`
	Kind        string            `json:"kind"`
	ChannelName string            `json:"channel_name"`
	AccountID   string            `json:"account_id,omitempty"`
	Endpoint    string            `json:"endpoint,omitempty"`
	Status      string            `json:"status"`
	Reason      string            `json:"reason,omitempty"`
	LastError   string            `json:"last_error,omitempty"`
	Source      string            `json:"source"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	UpdatedAt   string            `json:"updated_at"`
}

type ReceiverStatusesView struct {
	Receivers  []ReceiverStatusView `json:"receivers"`
	Totals     map[string]int       `json:"totals"`
	Notes      []string             `json:"notes"`
	SideEffect string               `json:"side_effect"`
}

type ReceiverStatusCleanupView struct {
	Deleted    []ReceiverStatusView `json:"deleted"`
	Remaining  []ReceiverStatusView `json:"remaining"`
	Totals     map[string]int       `json:"totals"`
	Notes      []string             `json:"notes"`
	SideEffect string               `json:"side_effect"`
}
