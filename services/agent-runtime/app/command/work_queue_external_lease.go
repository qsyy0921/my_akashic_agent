package command

import "time"

type ExecuteWorkQueueLeaseCommand struct {
	WorkKind                                  string
	WorkID                                    string
	AggregateID                               string
	Subject                                   string
	WorkerID                                  string
	LeaseTTLSeconds                           int
	ObservedAt                                time.Time
	Timestamp                                 time.Time
	ChannelByAccount                          map[string]string
	AllowedStepKinds                          []string
	AllowedStepKindsByAccount                 map[string][]string
	AllowedStepKindsByAccountConversationType map[string]map[string][]string
	AllowedStepKindsByAccountConversationID   map[string]map[string]map[string][]string
	Metadata                                  map[string]string
}
