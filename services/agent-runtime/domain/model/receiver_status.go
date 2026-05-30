package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReceiverLifecycleStatus string

const (
	ReceiverStatusStarting  ReceiverLifecycleStatus = "starting"
	ReceiverStatusConnected ReceiverLifecycleStatus = "connected"
	ReceiverStatusSuspended ReceiverLifecycleStatus = "suspended"
	ReceiverStatusFailed    ReceiverLifecycleStatus = "failed"
	ReceiverStatusStopped   ReceiverLifecycleStatus = "stopped"
)

type ReceiverStatus struct {
	ReceiverID  string
	Kind        ChannelKind
	ChannelName string
	AccountID   string
	Endpoint    string
	Status      ReceiverLifecycleStatus
	Reason      string
	LastError   string
	Source      string
	Metadata    map[string]string
	UpdatedAt   time.Time
}

type ReceiverStatusSpec struct {
	ReceiverID  string
	Kind        ChannelKind
	ChannelName string
	AccountID   string
	Endpoint    string
	Status      string
	Reason      string
	LastError   string
	Source      string
	Metadata    map[string]string
}

func NewReceiverStatus(spec ReceiverStatusSpec, updatedAt time.Time) (ReceiverStatus, error) {
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	status := ReceiverStatus{
		ReceiverID:  strings.TrimSpace(spec.ReceiverID),
		Kind:        ChannelKind(strings.TrimSpace(string(spec.Kind))),
		ChannelName: strings.TrimSpace(spec.ChannelName),
		AccountID:   strings.TrimSpace(spec.AccountID),
		Endpoint:    strings.TrimSpace(spec.Endpoint),
		Status:      normalizeReceiverStatus(spec.Status),
		Reason:      strings.TrimSpace(spec.Reason),
		LastError:   strings.TrimSpace(spec.LastError),
		Source:      strings.TrimSpace(spec.Source),
		Metadata:    cloneStringMap(spec.Metadata),
		UpdatedAt:   updatedAt.UTC(),
	}
	if status.Source == "" {
		status.Source = "unknown"
	}
	if status.ReceiverID == "" {
		status.ReceiverID = status.stableID()
	}
	return status, status.Validate()
}

func (s ReceiverStatus) Validate() error {
	if strings.TrimSpace(s.ReceiverID) == "" {
		return errors.New("receiver status requires receiver_id")
	}
	if strings.TrimSpace(string(s.Kind)) == "" {
		return errors.New("receiver status requires kind")
	}
	if strings.TrimSpace(s.ChannelName) == "" {
		return errors.New("receiver status requires channel_name")
	}
	if strings.TrimSpace(string(s.Status)) == "" {
		return errors.New("receiver status requires status")
	}
	if !isKnownReceiverStatus(s.Status) {
		return fmt.Errorf("unknown receiver status: %s", s.Status)
	}
	if s.UpdatedAt.IsZero() {
		return errors.New("receiver status requires updated_at")
	}
	return nil
}

func (s ReceiverStatus) stableID() string {
	accountID := s.AccountID
	if accountID == "" {
		accountID = s.ChannelName
	}
	return fmt.Sprintf(
		"%s:%s:%s",
		strings.TrimSpace(string(s.Kind)),
		strings.TrimSpace(accountID),
		strings.TrimSpace(s.ChannelName),
	)
}

func normalizeReceiverStatus(status string) ReceiverLifecycleStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "starting":
		return ReceiverStatusStarting
	case "connected", "running", "ok":
		return ReceiverStatusConnected
	case "suspended", "paused":
		return ReceiverStatusSuspended
	case "failed", "error", "unhealthy":
		return ReceiverStatusFailed
	case "stopped", "disabled":
		return ReceiverStatusStopped
	default:
		return ReceiverLifecycleStatus(strings.TrimSpace(status))
	}
}

func isKnownReceiverStatus(status ReceiverLifecycleStatus) bool {
	switch status {
	case ReceiverStatusStarting, ReceiverStatusConnected, ReceiverStatusSuspended, ReceiverStatusFailed, ReceiverStatusStopped:
		return true
	default:
		return false
	}
}

func SortedReceiverStatuses(items []ReceiverStatus) []ReceiverStatus {
	sorted := append([]ReceiverStatus(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		if sorted[i].ChannelName != sorted[j].ChannelName {
			return sorted[i].ChannelName < sorted[j].ChannelName
		}
		if sorted[i].AccountID != sorted[j].AccountID {
			return sorted[i].AccountID < sorted[j].AccountID
		}
		return sorted[i].ReceiverID < sorted[j].ReceiverID
	})
	return sorted
}
