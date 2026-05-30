package command

type PlanDeliveryDispatchCommand struct {
	EventID          string
	ChannelByAccount map[string]string
}
