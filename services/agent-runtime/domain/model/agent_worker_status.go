package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type AgentWorkerLifecycleStatus string

const (
	AgentWorkerStatusStarting AgentWorkerLifecycleStatus = "starting"
	AgentWorkerStatusIdle     AgentWorkerLifecycleStatus = "idle"
	AgentWorkerStatusRunning  AgentWorkerLifecycleStatus = "running"
	AgentWorkerStatusFailed   AgentWorkerLifecycleStatus = "failed"
	AgentWorkerStatusStopped  AgentWorkerLifecycleStatus = "stopped"
)

type AgentWorkerStatus struct {
	WorkerID       string
	WorkerType     string
	Status         AgentWorkerLifecycleStatus
	CurrentJobID   string
	LastJobID      string
	LastError      string
	ProcessedTotal int
	FailedTotal    int
	Source         string
	Metadata       map[string]string
	UpdatedAt      time.Time
}

type AgentWorkerStatusSpec struct {
	WorkerID       string
	WorkerType     string
	Status         string
	CurrentJobID   string
	LastJobID      string
	LastError      string
	ProcessedTotal int
	FailedTotal    int
	Source         string
	Metadata       map[string]string
}

func NewAgentWorkerStatus(spec AgentWorkerStatusSpec, updatedAt time.Time) (AgentWorkerStatus, error) {
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	status := AgentWorkerStatus{
		WorkerID:       strings.TrimSpace(spec.WorkerID),
		WorkerType:     strings.TrimSpace(spec.WorkerType),
		Status:         normalizeAgentWorkerStatus(spec.Status),
		CurrentJobID:   strings.TrimSpace(spec.CurrentJobID),
		LastJobID:      strings.TrimSpace(spec.LastJobID),
		LastError:      strings.TrimSpace(spec.LastError),
		ProcessedTotal: spec.ProcessedTotal,
		FailedTotal:    spec.FailedTotal,
		Source:         strings.TrimSpace(spec.Source),
		Metadata:       cloneStringMap(spec.Metadata),
		UpdatedAt:      updatedAt.UTC(),
	}
	if status.Source == "" {
		status.Source = "unknown"
	}
	return status, status.Validate()
}

func (s AgentWorkerStatus) Validate() error {
	if strings.TrimSpace(s.WorkerID) == "" {
		return errors.New("agent worker status requires worker_id")
	}
	if strings.TrimSpace(s.WorkerType) == "" {
		return errors.New("agent worker status requires worker_type")
	}
	if strings.TrimSpace(string(s.Status)) == "" {
		return errors.New("agent worker status requires status")
	}
	if !isKnownAgentWorkerStatus(s.Status) {
		return fmt.Errorf("unknown agent worker status: %s", s.Status)
	}
	if s.ProcessedTotal < 0 {
		return errors.New("agent worker status requires non-negative processed_total")
	}
	if s.FailedTotal < 0 {
		return errors.New("agent worker status requires non-negative failed_total")
	}
	if s.UpdatedAt.IsZero() {
		return errors.New("agent worker status requires updated_at")
	}
	return nil
}

func (s AgentWorkerStatus) HeartbeatActive() bool {
	switch s.Status {
	case AgentWorkerStatusStarting, AgentWorkerStatusIdle, AgentWorkerStatusRunning:
		return true
	default:
		return false
	}
}

func (s AgentWorkerStatus) WithStaleHeartbeat(now time.Time, staleAfter time.Duration) AgentWorkerStatus {
	if staleAfter <= 0 || !s.HeartbeatActive() {
		return s
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if s.UpdatedAt.IsZero() || now.UTC().Sub(s.UpdatedAt.UTC()) <= staleAfter {
		return s
	}
	next := s
	next.Status = AgentWorkerStatusStopped
	if next.LastError == "" {
		next.LastError = "last agent worker heartbeat exceeded stale threshold"
	}
	next.Metadata = cloneStringMap(next.Metadata)
	if next.Metadata == nil {
		next.Metadata = map[string]string{}
	}
	next.Metadata["last_status"] = string(s.Status)
	next.Metadata["stale_after_seconds"] = fmt.Sprintf("%d", int(staleAfter.Seconds()))
	return next
}

func normalizeAgentWorkerStatus(status string) AgentWorkerLifecycleStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "starting", "start":
		return AgentWorkerStatusStarting
	case "idle", "no_job", "waiting":
		return AgentWorkerStatusIdle
	case "running", "processing", "busy":
		return AgentWorkerStatusRunning
	case "failed", "error", "unhealthy":
		return AgentWorkerStatusFailed
	case "stopped", "disabled", "stop":
		return AgentWorkerStatusStopped
	default:
		return AgentWorkerLifecycleStatus(strings.TrimSpace(status))
	}
}

func isKnownAgentWorkerStatus(status AgentWorkerLifecycleStatus) bool {
	switch status {
	case AgentWorkerStatusStarting, AgentWorkerStatusIdle, AgentWorkerStatusRunning, AgentWorkerStatusFailed, AgentWorkerStatusStopped:
		return true
	default:
		return false
	}
}

func SortedAgentWorkerStatuses(items []AgentWorkerStatus) []AgentWorkerStatus {
	sorted := append([]AgentWorkerStatus(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].WorkerType != sorted[j].WorkerType {
			return sorted[i].WorkerType < sorted[j].WorkerType
		}
		return sorted[i].WorkerID < sorted[j].WorkerID
	})
	return sorted
}
