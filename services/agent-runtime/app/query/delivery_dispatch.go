package query

type DeliveryDispatchStepView struct {
	StepIndex int    `json:"step_index"`
	Kind      string `json:"kind"`
	Channel   string `json:"channel"`
	ChatID    string `json:"chat_id"`
	Message   string `json:"message,omitempty"`
	Image     string `json:"image,omitempty"`
	File      string `json:"file,omitempty"`
}

type DeliveryDispatchPlanView struct {
	EventID    string                     `json:"event_id"`
	Channel    string                     `json:"channel"`
	ChatID     string                     `json:"chat_id"`
	StepCount  int                        `json:"step_count"`
	Steps      []DeliveryDispatchStepView `json:"steps"`
	Attributes map[string]string          `json:"attributes,omitempty"`
}
