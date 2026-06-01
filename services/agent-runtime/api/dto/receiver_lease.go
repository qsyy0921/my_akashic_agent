package dto

type AcquireReceiverLeaseRequest struct {
	ReceiverID  string            `json:"receiver_id"`
	Kind        string            `json:"kind"`
	ChannelName string            `json:"channel_name"`
	AccountID   string            `json:"account_id"`
	HolderID    string            `json:"holder_id"`
	TTLSeconds  int               `json:"ttl_seconds"`
	Metadata    map[string]string `json:"metadata"`
	Timestamp   string            `json:"timestamp"`
}

type RenewReceiverLeaseRequest struct {
	ReceiverID string `json:"receiver_id"`
	HolderID   string `json:"holder_id"`
	LeaseToken string `json:"lease_token"`
	TTLSeconds int    `json:"ttl_seconds"`
	Timestamp  string `json:"timestamp"`
}

type ReleaseReceiverLeaseRequest struct {
	ReceiverID string `json:"receiver_id"`
	HolderID   string `json:"holder_id"`
	LeaseToken string `json:"lease_token"`
	Timestamp  string `json:"timestamp"`
}

type CleanupExpiredReceiverLeasesRequest struct {
	Timestamp string `json:"timestamp"`
}
