package model

import (
	"errors"
	"sort"
	"strings"
	"time"
)

type SchedulerExecutionLease struct {
	JobID      string
	HolderID   string
	LeaseToken string
	ExpiresAt  time.Time
	AcquiredAt time.Time
	UpdatedAt  time.Time
	Metadata   map[string]string
}

type SchedulerExecutionLeaseSpec struct {
	JobID      string
	HolderID   string
	LeaseToken string
	ExpiresAt  time.Time
	AcquiredAt time.Time
	UpdatedAt  time.Time
	Metadata   map[string]string
}

func NewSchedulerExecutionLease(spec SchedulerExecutionLeaseSpec) (SchedulerExecutionLease, error) {
	now := spec.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	acquiredAt := spec.AcquiredAt
	if acquiredAt.IsZero() {
		acquiredAt = now
	}
	lease := SchedulerExecutionLease{
		JobID:      strings.TrimSpace(spec.JobID),
		HolderID:   strings.TrimSpace(spec.HolderID),
		LeaseToken: strings.TrimSpace(spec.LeaseToken),
		ExpiresAt:  spec.ExpiresAt.UTC(),
		AcquiredAt: acquiredAt.UTC(),
		UpdatedAt:  now.UTC(),
		Metadata:   cloneStringMap(spec.Metadata),
	}
	return lease, lease.Validate()
}

func (l SchedulerExecutionLease) Validate() error {
	if strings.TrimSpace(l.JobID) == "" {
		return errors.New("scheduler execution lease requires job_id")
	}
	if strings.TrimSpace(l.HolderID) == "" {
		return errors.New("scheduler execution lease requires holder_id")
	}
	if strings.TrimSpace(l.LeaseToken) == "" {
		return errors.New("scheduler execution lease requires lease_token")
	}
	if l.ExpiresAt.IsZero() {
		return errors.New("scheduler execution lease requires expires_at")
	}
	if l.UpdatedAt.IsZero() {
		return errors.New("scheduler execution lease requires updated_at")
	}
	return nil
}

func (l SchedulerExecutionLease) ActiveAt(now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return l.ExpiresAt.After(now.UTC())
}

func (l SchedulerExecutionLease) Matches(holderID string, leaseToken string) bool {
	return strings.TrimSpace(l.HolderID) == strings.TrimSpace(holderID) &&
		strings.TrimSpace(l.LeaseToken) == strings.TrimSpace(leaseToken)
}

func (l SchedulerExecutionLease) Renew(expiresAt time.Time, updatedAt time.Time) (SchedulerExecutionLease, error) {
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	next := l
	next.ExpiresAt = expiresAt.UTC()
	next.UpdatedAt = updatedAt.UTC()
	return next, next.Validate()
}

func SortedSchedulerExecutionLeases(items []SchedulerExecutionLease) []SchedulerExecutionLease {
	sorted := append([]SchedulerExecutionLease(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].JobID != sorted[j].JobID {
			return sorted[i].JobID < sorted[j].JobID
		}
		return sorted[i].HolderID < sorted[j].HolderID
	})
	return sorted
}
