package model

type DeliveryDispatchStepKind string

const (
	DeliveryDispatchStepText  DeliveryDispatchStepKind = "text"
	DeliveryDispatchStepImage DeliveryDispatchStepKind = "image"
	DeliveryDispatchStepFile  DeliveryDispatchStepKind = "file"
)

type DeliveryDispatchStep struct {
	StepIndex int
	Kind      DeliveryDispatchStepKind
	Channel   string
	ChatID    string
	Message   string
	Image     string
	File      string
}

type DeliveryDispatchPlan struct {
	EventID    string
	Channel    string
	ChatID     string
	StepCount  int
	Steps      []DeliveryDispatchStep
	Attributes map[string]string
}

type DeliveryDispatchResultStatus string

const (
	DeliveryDispatchSent DeliveryDispatchResultStatus = "sent"
)

type DeliveryDispatchResult struct {
	StepIndex         int
	Kind              DeliveryDispatchStepKind
	Channel           string
	ChatID            string
	Status            DeliveryDispatchResultStatus
	Provider          string
	ProviderMessageID string
	Attributes        map[string]string
}

type DeliveryDispatchExecution struct {
	EventID    string
	StepCount  int
	Results    []DeliveryDispatchResult
	Attributes map[string]string
}
