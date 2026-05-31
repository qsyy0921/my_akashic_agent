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
