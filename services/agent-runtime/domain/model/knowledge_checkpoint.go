package model

import (
	"errors"
	"strings"
	"time"
)

type KnowledgeCheckpoint struct {
	CheckpointID string
	Cursor       int
	UpdatedAt    time.Time
	Metadata     map[string]string
}

func NewKnowledgeCheckpoint(checkpointID string, cursor int, metadata map[string]string, now time.Time) (KnowledgeCheckpoint, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	checkpoint := KnowledgeCheckpoint{
		CheckpointID: strings.TrimSpace(checkpointID),
		Cursor:       cursor,
		UpdatedAt:    now,
		Metadata:     copyStringMap(metadata),
	}
	if err := checkpoint.Validate(); err != nil {
		return KnowledgeCheckpoint{}, err
	}
	return checkpoint, nil
}

func (c KnowledgeCheckpoint) Validate() error {
	if strings.TrimSpace(c.CheckpointID) == "" {
		return errors.New("knowledge checkpoint requires checkpoint id")
	}
	if c.Cursor < -1 {
		return errors.New("knowledge checkpoint cursor must be >= -1")
	}
	if c.UpdatedAt.IsZero() {
		return errors.New("knowledge checkpoint requires updated_at")
	}
	return nil
}

func (c *KnowledgeCheckpoint) Advance(cursor int, metadata map[string]string, now time.Time) error {
	if c == nil {
		return errors.New("knowledge checkpoint is nil")
	}
	if cursor < c.Cursor {
		return errors.New("knowledge checkpoint cursor cannot move backwards")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	c.Cursor = cursor
	c.UpdatedAt = now
	if metadata != nil {
		c.Metadata = copyStringMap(metadata)
	}
	return c.Validate()
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
