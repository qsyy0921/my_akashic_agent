package command

type PlanDeliveryDispatchCommand struct {
	EventID          string
	ChannelByAccount map[string]string
}

type DispatchDeliveryCommand struct {
	EventID          string
	ChannelByAccount map[string]string
}
