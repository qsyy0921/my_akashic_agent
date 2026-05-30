package command

import "time"

type UpsertKnowledgeCheckpointCommand struct {
	CheckpointID string
	Cursor       int
	Metadata     map[string]string
	Timestamp    time.Time
}
