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
