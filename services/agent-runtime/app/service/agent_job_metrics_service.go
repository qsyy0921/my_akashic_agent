package service

import (
	"context"
	"errors"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	defaultAgentJobMetricLimit = 200
	maxAgentJobMetricLimit     = 200
	maxDeadLetterSamples       = 10
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

	return summarizeAgentJobMetrics(jobs, events, notes), nil
}

func summarizeAgentJobMetrics(
	jobs []model.AgentJob,
	events []model.AgentJobEvent,
	notes []string,
) query.AgentJobMetricsView {
	jobsByStatus := make(map[string]int)
	jobsByType := make(map[string]query.AgentJobTypeMetricsView)
	deadByType := make(map[string]int)
	throughput := query.AgentJobThroughputMetricsView{
		EventsByType: make(map[string]int),
	}
	recentDeadLetters := make([]query.AgentJobDeadLetterSampleView, 0, maxDeadLetterSamples)

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
		Notes: notes,
	}
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
