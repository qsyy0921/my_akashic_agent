package command

import "time"

type CheckInboundDedupeCommand struct {
	Scope      string
	MessageKey string
	TTLSeconds int
	Timestamp  time.Time
	Metadata   map[string]string
}
