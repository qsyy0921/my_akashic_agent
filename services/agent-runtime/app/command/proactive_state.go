package command

import "time"

type RecordProactiveDeliveryCommand struct {
	SessionKey  string
	DeliveryKey string
	Timestamp   time.Time
}

type CheckProactiveDeliveryDuplicateCommand struct {
	SessionKey  string
	DeliveryKey string
	WindowHours int
	Timestamp   time.Time
}

type CountProactiveDeliveriesCommand struct {
	SessionKey  string
	WindowHours int
	Timestamp   time.Time
}

type ProactiveSourceItemEntry struct {
	SourceKey string
	ItemID    string
}

type CheckProactiveItemSeenCommand struct {
	SourceKey string
	ItemID    string
	TTLHours  int
	Timestamp time.Time
}

type MarkProactiveItemsSeenCommand struct {
	Entries   []ProactiveSourceItemEntry
	Timestamp time.Time
}

type CheckProactiveRejectionCooldownCommand struct {
	SourceKey string
	ItemID    string
	TTLHours  int
	Timestamp time.Time
}

type MarkProactiveRejectionCooldownCommand struct {
	Entries   []ProactiveSourceItemEntry
	Hours     int
	Timestamp time.Time
}

type RecordProactiveContextOnlyCommand struct {
	SessionKey string
	Timestamp  time.Time
}

type CountProactiveContextOnlyCommand struct {
	SessionKey  string
	WindowHours int
	Timestamp   time.Time
}

type RecordProactiveDriftRunCommand struct {
	SessionKey string
	Timestamp  time.Time
}

type RecordProactiveDriftFinishCommand struct {
	SkillUsed     string
	OneLine       string
	Next          string
	MessageResult string
	Note          string
	Timestamp     time.Time
}

type RecordProactiveTickLogStartCommand struct {
	TickID     string
	SessionKey string
	StartedAt  time.Time
	GateExit   string
}

type RecordProactiveTickLogFinishCommand struct {
	TickID         string
	SessionKey     string
	StartedAt      time.Time
	FinishedAt     time.Time
	GateExit       string
	TerminalAction string
	SkipReason     string
	StepsTaken     int
	AlertCount     int
	ContentCount   int
	ContextCount   int
	InterestingIDs []string
	DiscardedIDs   []string
	CitedIDs       []string
	DriftEntered   bool
	FinalMessage   string
}

type RecordProactiveTickStepLogCommand struct {
	TickID              string
	StepIndex           int
	Phase               string
	ToolName            string
	ToolCallID          string
	ToolArgs            map[string]any
	ToolResultText      string
	TerminalActionAfter string
	SkipReasonAfter     string
	InterestingIDsAfter []string
	DiscardedIDsAfter   []string
	CitedIDsAfter       []string
	FinalMessageAfter   string
}

type RecordProactiveBGContextMainCommand struct {
	Timestamp time.Time
}

type SnapshotProactiveAnyActionQuotaCommand struct {
	QuotaKey  string
	ResetHour int
	Timezone  string
	Timestamp time.Time
}

type RecordProactiveAnyActionCommand struct {
	QuotaKey  string
	ResetHour int
	Timezone  string
	Timestamp time.Time
}

type CleanupProactiveStateCommand struct {
	SeenTTLHours              int
	DeliveryTTLHours          int
	ContextOnlyTTLHours       int
	RejectionCooldownTTLHours int
	Timestamp                 time.Time
}
