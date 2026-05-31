package model

import (
	"errors"
	"strings"
	"time"
)

type SchedulerJob struct {
	ID              string
	Trigger         string
	Tier            string
	FireAt          time.Time
	Channel         string
	ChatID          string
	IntervalSeconds *int
	CronExpr        string
	Message         string
	Prompt          string
	Name            string
	Timezone        string
	CreatedAt       time.Time
	RunCount        int
	Enabled         bool
}

func NewSchedulerJob(
	id string,
	trigger string,
	tier string,
	fireAt time.Time,
	channel string,
	chatID string,
	createdAt time.Time,
) (SchedulerJob, error) {
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	job := SchedulerJob{
		ID:        strings.TrimSpace(id),
		Trigger:   strings.TrimSpace(trigger),
		Tier:      strings.TrimSpace(tier),
		FireAt:    fireAt.UTC(),
		Channel:   strings.TrimSpace(channel),
		ChatID:    strings.TrimSpace(chatID),
		Timezone:  "UTC",
		CreatedAt: createdAt.UTC(),
		Enabled:   true,
	}
	if err := job.Validate(); err != nil {
		return SchedulerJob{}, err
	}
	return job, nil
}

func (j SchedulerJob) Validate() error {
	if strings.TrimSpace(j.ID) == "" {
		return errors.New("scheduler job requires id")
	}
	switch strings.TrimSpace(j.Trigger) {
	case "at", "after", "every":
	default:
		return errors.New("scheduler job has invalid trigger")
	}
	switch strings.TrimSpace(j.Tier) {
	case "instant", "soft":
	default:
		return errors.New("scheduler job has invalid tier")
	}
	if j.FireAt.IsZero() {
		return errors.New("scheduler job requires fire_at")
	}
	if strings.TrimSpace(j.Channel) == "" {
		return errors.New("scheduler job requires channel")
	}
	if strings.TrimSpace(j.ChatID) == "" {
		return errors.New("scheduler job requires chat_id")
	}
	if j.CreatedAt.IsZero() {
		return errors.New("scheduler job requires created_at")
	}
	if j.RunCount < 0 {
		return errors.New("scheduler job requires non-negative run_count")
	}
	if j.IntervalSeconds != nil && *j.IntervalSeconds <= 0 {
		return errors.New("scheduler job interval_seconds must be positive")
	}
	if strings.TrimSpace(j.Timezone) == "" {
		return errors.New("scheduler job requires timezone")
	}
	if j.Tier == "instant" && strings.TrimSpace(j.Message) == "" {
		return errors.New("instant scheduler job requires message")
	}
	if j.Tier == "soft" && strings.TrimSpace(j.Prompt) == "" {
		return errors.New("soft scheduler job requires prompt")
	}
	return nil
}
