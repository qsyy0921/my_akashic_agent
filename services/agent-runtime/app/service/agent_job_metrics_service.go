package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	defaultAgentJobMetricLimit  = 200
	maxAgentJobMetricLimit      = 200
	maxDeadLetterSamples        = 10
	agentJobPressurePendingWarn = 10
	agentJobPressureActiveWarn  = 5
	agentJobPressureAgeWarn     = 15 * time.Minute
)

type AgentJobMetricsService struct {
	jobs   outport.AgentJobRepository
	events outport.AgentJobEventStore
}

func NewAgentJobMetricsService(
	jobs outport.AgentJobRepository,
	events outport.AgentJobEventStore,
) *AgentJobMetricsService {
	return &AgentJobMetricsService{jobs: jobs, events: events}
}

func (s *AgentJobMetricsService) Get(ctx context.Context, filter query.AgentJobMetricsFilter) (query.AgentJobMetricsView, error) {
	if s == nil || s.jobs == nil {
		return query.AgentJobMetricsView{}, errors.New("agent job metrics service requires job repository")
	}
	jobLimit := boundedAgentJobMetricLimit(filter.JobLimit)
	eventLimit := boundedAgentJobMetricLimit(filter.EventLimit)

	jobs, err := s.jobs.ListAgentJobs(ctx, query.AgentJobFilter{Limit: jobLimit})
	if err != nil {
		return query.AgentJobMetricsView{}, err
	}
	events := make([]model.AgentJobEvent, 0)
	notes := make([]string, 0)
	if s.events == nil {
		notes = append(notes, "agent job event stream unavailable")
	} else {
		events, err = s.events.ListAgentJobEvents(ctx, query.AgentJobEventFilter{Limit: eventLimit})
		if err != nil {
			return query.AgentJobMetricsView{}, err
		}
	}

	now := filter.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	return summarizeAgentJobMetrics(jobs, events, notes, now), nil
}

func summarizeAgentJobMetrics(
	jobs []model.AgentJob,
	events []model.AgentJobEvent,
	notes []string,
	now time.Time,
) query.AgentJobMetricsView {
	jobsByStatus := make(map[string]int)
	jobsByType := make(map[string]query.AgentJobTypeMetricsView)
	deadByType := make(map[string]int)
	pressureByType := make(map[string]*query.AgentJobTypePressureView)
	throughput := query.AgentJobThroughputMetricsView{
		EventsByType: make(map[string]int),
	}
	recentDeadLetters := make([]query.AgentJobDeadLetterSampleView, 0, maxDeadLetterSamples)
	if now.IsZero() {
		now = time.Now().UTC()
	}

	for _, job := range jobs {
		status := string(job.Status)
		jobType := string(job.JobType)
		jobsByStatus[status]++
		typeMetrics := jobsByType[jobType]
		if typeMetrics.ByStatus == nil {
			typeMetrics.ByStatus = make(map[string]int)
		}
		typeMetrics.Total++
		typeMetrics.ByStatus[status]++
		jobsByType[jobType] = typeMetrics
		if job.Status == model.AgentJobDeadLettered {
			deadByType[jobType]++
		}
		pressure := agentJobPressureForType(pressureByType, jobType)
		switch job.Status {
		case model.AgentJobPending:
			pressure.Pending++
			age := ageSeconds(now, job.CreatedAt)
			if age > pressure.OldestPendingAgeSeconds {
				pressure.OldestPendingAgeSeconds = age
			}
		case model.AgentJobLeased:
			pressure.Leased++
		case model.AgentJobRunning:
			pressure.Running++
		}
	}

	for _, event := range events {
		eventType := string(event.EventType)
		status := string(event.Status)
		throughput.EventsByType[eventType]++
		switch event.EventType {
		case model.AgentJobEventCreated:
			throughput.Created++
		case model.AgentJobEventLeased:
			throughput.Leased++
		case model.AgentJobEventRenewed:
			throughput.Renewed++
		case model.AgentJobEventRunning:
			throughput.Running++
		case model.AgentJobEventSucceeded:
			throughput.Succeeded++
		case model.AgentJobEventFailed:
			throughput.Failed++
		case model.AgentJobEventRetry:
			throughput.Retry++
		case model.AgentJobEventLeaseExpired:
			throughput.LeaseExpired++
		case model.AgentJobEventCancelled:
			throughput.Cancelled++
		}
		if isTerminalAgentJobStatus(event.Status) {
			throughput.TerminalEvents++
		}
		if status == string(model.AgentJobDeadLettered) && len(recentDeadLetters) < maxDeadLetterSamples {
			recentDeadLetters = append(recentDeadLetters, query.AgentJobDeadLetterSampleView{
				JobID:       event.JobID,
				JobType:     string(event.JobType),
				EventType:   eventType,
				Status:      status,
				Attempt:     event.Attempt,
				MaxAttempts: event.MaxAttempts,
				OccurredAt:  formatAgentJobMetricTime(event.OccurredAt),
			})
		}
	}

	pressure := summarizeAgentJobPressure(pressureByType)

	return query.AgentJobMetricsView{
		SampledJobs:   len(jobs),
		SampledEvents: len(events),
		JobsByStatus:  jobsByStatus,
		JobsByType:    jobsByType,
		Throughput:    throughput,
		DeadLetters: query.AgentJobDeadLetterMetricsView{
			CurrentTotal: jobsByStatus[string(model.AgentJobDeadLettered)],
			ByType:       deadByType,
			Recent:       recentDeadLetters,
		},
		Pressure: pressure,
		Notes:    notes,
	}
}

func agentJobPressureForType(items map[string]*query.AgentJobTypePressureView, jobType string) *query.AgentJobTypePressureView {
	item := items[jobType]
	if item == nil {
		item = &query.AgentJobTypePressureView{JobType: jobType}
		items[jobType] = item
	}
	return item
}

func summarizeAgentJobPressure(items map[string]*query.AgentJobTypePressureView) query.AgentJobPressureMetricsView {
	if len(items) == 0 {
		return query.AgentJobPressureMetricsView{}
	}
	view := query.AgentJobPressureMetricsView{
		JobTypes: len(items),
		ByType:   make([]query.AgentJobTypePressureView, 0, len(items)),
	}
	for _, item := range items {
		current := *item
		current.Active = current.Leased + current.Running
		current.HighPressure, current.PressureReason = agentJobPressureStatus(current)
		if current.HighPressure {
			view.HighPressureJobTypes++
		}
		if current.Pending > view.MaxPending {
			view.MaxPending = current.Pending
		}
		if current.Active > view.MaxActive {
			view.MaxActive = current.Active
		}
		if current.OldestPendingAgeSeconds > view.OldestPendingAgeSeconds {
			view.OldestPendingAgeSeconds = current.OldestPendingAgeSeconds
		}
		view.ByType = append(view.ByType, current)
	}
	sort.Slice(view.ByType, func(i, j int) bool {
		left := view.ByType[i]
		right := view.ByType[j]
		if left.HighPressure != right.HighPressure {
			return left.HighPressure
		}
		if left.Pending != right.Pending {
			return left.Pending > right.Pending
		}
		if left.Active != right.Active {
			return left.Active > right.Active
		}
		return left.JobType < right.JobType
	})
	return view
}

func agentJobPressureStatus(item query.AgentJobTypePressureView) (bool, string) {
	if item.Pending >= agentJobPressurePendingWarn {
		return true, fmt.Sprintf("pending>=%d", agentJobPressurePendingWarn)
	}
	if item.Active >= agentJobPressureActiveWarn {
		return true, fmt.Sprintf("active>=%d", agentJobPressureActiveWarn)
	}
	if item.OldestPendingAgeSeconds >= int(agentJobPressureAgeWarn/time.Second) {
		return true, fmt.Sprintf("oldest_pending_age>=%ds", int(agentJobPressureAgeWarn/time.Second))
	}
	return false, ""
}

func ageSeconds(now time.Time, createdAt time.Time) int {
	if now.IsZero() || createdAt.IsZero() || now.Before(createdAt) {
		return 0
	}
	return int(now.Sub(createdAt) / time.Second)
}

func isTerminalAgentJobStatus(status model.AgentJobStatus) bool {
	switch status {
	case model.AgentJobSucceeded, model.AgentJobDeadLettered, model.AgentJobCancelled:
		return true
	default:
		return false
	}
}

func boundedAgentJobMetricLimit(value int) int {
	if value <= 0 {
		return defaultAgentJobMetricLimit
	}
	if value > maxAgentJobMetricLimit {
		return maxAgentJobMetricLimit
	}
	return value
}

func formatAgentJobMetricTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
