package query

type ReceiverLeaseView struct {
	ReceiverID        string            `json:"receiver_id"`
	Kind              string            `json:"kind"`
	ChannelName       string            `json:"channel_name"`
	AccountID         string            `json:"account_id,omitempty"`
	HolderID          string            `json:"holder_id,omitempty"`
	LeaseToken        string            `json:"lease_token,omitempty"`
	LeaseTokenPresent bool              `json:"lease_token_present"`
	Active            bool              `json:"active"`
	Acquired          *bool             `json:"acquired,omitempty"`
	DeniedReason      string            `json:"denied_reason,omitempty"`
	ExpiresAt         string            `json:"expires_at,omitempty"`
	AcquiredAt        string            `json:"acquired_at,omitempty"`
	UpdatedAt         string            `json:"updated_at,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

type ReceiverLeasesView struct {
	Leases     []ReceiverLeaseView `json:"leases"`
	Totals     map[string]int      `json:"totals"`
	Notes      []string            `json:"notes"`
	SideEffect string              `json:"side_effect"`
}
