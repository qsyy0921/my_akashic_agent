package command

import "time"

type SyncObserveTargetsCommand struct {
	Source    string
	Timestamp time.Time
	Targets   []ObserveTargetCommand
}

type ObserveTargetCommand struct {
	TargetID     string
	Channel      ChannelCommand
	ObserveOnly  bool
	ReplyAllowed bool
	RequireAt    bool
	AllowFrom    []string
	Enabled      bool
	Source       string
	Metadata     map[string]string
}
