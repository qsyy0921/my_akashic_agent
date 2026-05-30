package dto

type ReceiverStatusRequest struct {
	ReceiverID  string            `json:"receiver_id"`
	Kind        string            `json:"kind"`
	ChannelName string            `json:"channel_name"`
	AccountID   string            `json:"account_id"`
	Endpoint    string            `json:"endpoint"`
	Status      string            `json:"status"`
	Reason      string            `json:"reason"`
	LastError   string            `json:"last_error"`
	Source      string            `json:"source"`
	Metadata    map[string]string `json:"metadata"`
	Timestamp   string            `json:"timestamp"`
}
