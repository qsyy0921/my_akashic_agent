package query

type DeliveryDispatchStepView struct {
	StepIndex        int    `json:"step_index"`
	Kind             string `json:"kind"`
	Channel          string `json:"channel"`
	ChatID           string `json:"chat_id"`
	ConversationType string `json:"conversation_type,omitempty"`
	Message          string `json:"message,omitempty"`
	Image            string `json:"image,omitempty"`
	File             string `json:"file,omitempty"`
}

type DeliveryDispatchPlanView struct {
	EventID    string                     `json:"event_id"`
	Channel    string                     `json:"channel"`
	ChatID     string                     `json:"chat_id"`
	StepCount  int                        `json:"step_count"`
	Steps      []DeliveryDispatchStepView `json:"steps"`
	Attributes map[string]string          `json:"attributes,omitempty"`
}

type DeliveryDispatchResultStepView struct {
	StepIndex         int               `json:"step_index"`
	Kind              string            `json:"kind"`
	Channel           string            `json:"channel"`
	ChatID            string            `json:"chat_id"`
	Status            string            `json:"status"`
	Provider          string            `json:"provider,omitempty"`
	ProviderMessageID string            `json:"provider_message_id,omitempty"`
	Attributes        map[string]string `json:"attributes,omitempty"`
}

type DeliveryDispatchResultView struct {
	EventID    string                           `json:"event_id"`
	StepCount  int                              `json:"step_count"`
	Results    []DeliveryDispatchResultStepView `json:"results"`
	Attributes map[string]string                `json:"attributes,omitempty"`
}
