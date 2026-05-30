package dto

type PlanDeliveryDispatchRequest struct {
	EventID          string            `json:"event_id"`
	ChannelByAccount map[string]string `json:"channel_by_account,omitempty"`
}

type DispatchDeliveryRequest struct {
	EventID          string            `json:"event_id"`
	ChannelByAccount map[string]string `json:"channel_by_account,omitempty"`
}
