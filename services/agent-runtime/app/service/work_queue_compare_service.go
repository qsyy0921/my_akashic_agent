package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const queueCompareRecentLimit = 50

type WorkQueueCompareService struct {
	outboxRepo   outport.OutboxRepository
	agentJobRepo outport.AgentJobRepository

	mu      sync.Mutex
	records []query.QueueCandidateComparisonView
	reasons map[string]int
	kinds   map[string]*queueCompareKindStats
}

type queueCompareKindStats struct {
	workKind   string
	compared   int
	matched    int
	mismatched int
}

func NewWorkQueueCompareService(
	outboxRepo outport.OutboxRepository,
	agentJobRepo outport.AgentJobRepository,
) *WorkQueueCompareService {
	return &WorkQueueCompareService{
		outboxRepo:   outboxRepo,
		agentJobRepo: agentJobRepo,
		records:      make([]query.QueueCandidateComparisonView, 0, queueCompareRecentLimit),
		reasons:      make(map[string]int),
		kinds:        make(map[string]*queueCompareKindStats),
	}
}

func (s *WorkQueueCompareService) CompareWorkQueueCandidate(
	ctx context.Context,
	cmd command.CompareWorkQueueCandidateCommand,
) (query.QueueCandidateComparisonView, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueCandidateComparisonView{}, err
	}
	if s == nil {
		return query.QueueCandidateComparisonView{}, errors.New("work queue compare service is nil")
	}
	comparedAt := time.Now().UTC()
	if cmd.ObservedAt.IsZero() {
		cmd.ObservedAt = comparedAt
	}
	workKind := normalizeWorkKind(cmd.WorkKind)
	workID := strings.TrimSpace(cmd.WorkID)
	if workID == "" {
		workID = strings.TrimSpace(cmd.AggregateID)
	}

	view := query.QueueCandidateComparisonView{
		WorkKind:    workKind,
		WorkID:      workID,
		AggregateID: strings.TrimSpace(cmd.AggregateID),
		Subject:     strings.TrimSpace(cmd.Subject),
		ObservedAt:  formatQueueCompareTime(cmd.ObservedAt),
		ComparedAt:  formatQueueCompareTime(comparedAt),
	}
	view.Matched, view.Reason, view.StateStatus = s.compare(ctx, workKind, workID, comparedAt)
	s.record(view)
	return view, nil
}

func (s *WorkQueueCompareService) SnapshotDualReadDiagnostics(ctx context.Context) (query.QueueDualReadDiagnostics, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueDualReadDiagnostics{}, err
	}
	if s == nil {
		return query.QueueDualReadDiagnostics{}, errors.New("work queue compare service is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	diagnostics := query.QueueDualReadDiagnostics{
		Enabled:     true,
		SampleLimit: queueCompareRecentLimit,
		Notes: []string{
			"dual_read_compare validates queue candidates against Go authoritative state only",
			"no platform send, Python worker execution, or external lease is performed in this mode",
		},
	}
	diagnostics.RecentComparisons = append(diagnostics.RecentComparisons, s.records...)
	for _, record := range s.records {
		diagnostics.LastComparedAt = laterQueueTime(diagnostics.LastComparedAt, record.ComparedAt)
		if !record.Matched {
			diagnostics.LastMismatchAt = laterQueueTime(diagnostics.LastMismatchAt, record.ComparedAt)
		}
	}
	for reason, count := range s.reasons {
		diagnostics.Reasons = append(diagnostics.Reasons, query.QueueCompareReasonStats{
			Reason: reason,
			Count:  count,
		})
	}
	for _, kind := range s.kinds {
		diagnostics.WorkKinds = append(diagnostics.WorkKinds, query.QueueCompareWorkKindStats{
			WorkKind:      kind.workKind,
			ComparedCount: kind.compared,
			MatchedCount:  kind.matched,
			MismatchCount: kind.mismatched,
		})
		diagnostics.ComparedTotal += kind.compared
		diagnostics.MatchedTotal += kind.matched
		diagnostics.MismatchedTotal += kind.mismatched
	}
	sort.Slice(diagnostics.Reasons, func(i, j int) bool {
		return diagnostics.Reasons[i].Reason < diagnostics.Reasons[j].Reason
	})
	sort.Slice(diagnostics.WorkKinds, func(i, j int) bool {
		return diagnostics.WorkKinds[i].WorkKind < diagnostics.WorkKinds[j].WorkKind
	})
	return diagnostics, nil
}

func (s *WorkQueueCompareService) compare(ctx context.Context, workKind string, workID string, now time.Time) (bool, string, string) {
	if workID == "" {
		return false, "missing_work_id", ""
	}
	switch workKind {
	case "outbox_delivery":
		if s.outboxRepo == nil {
			return false, "repository_unavailable", ""
		}
		delivery, ok, err := s.outboxRepo.FindOutboxDelivery(ctx, workID)
		if err != nil {
			return false, "state_lookup_error", ""
		}
		if !ok {
			return false, "missing_state", ""
		}
		status := string(delivery.Status)
		if !delivery.CanLease(now) {
			return false, "not_leaseable", status
		}
		return true, "leaseable_state", status
	case "agent_job":
		if s.agentJobRepo == nil {
			return false, "repository_unavailable", ""
		}
		job, ok, err := s.agentJobRepo.FindAgentJob(ctx, workID)
		if err != nil {
			return false, "state_lookup_error", ""
		}
		if !ok {
			return false, "missing_state", ""
		}
		status := string(job.Status)
		if !job.CanLease(now) {
			return false, "not_leaseable", status
		}
		return true, "leaseable_state", status
	default:
		return false, "unsupported_work_kind", ""
	}
}

func (s *WorkQueueCompareService) record(view query.QueueCandidateComparisonView) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append([]query.QueueCandidateComparisonView{view}, s.records...)
	if len(s.records) > queueCompareRecentLimit {
		s.records = s.records[:queueCompareRecentLimit]
	}
	s.reasons[view.Reason]++
	kind := s.kinds[view.WorkKind]
	if kind == nil {
		kind = &queueCompareKindStats{workKind: view.WorkKind}
		s.kinds[view.WorkKind] = kind
	}
	kind.compared++
	if view.Matched {
		kind.matched++
	} else {
		kind.mismatched++
	}
}

func normalizeWorkKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "outbox", "outbox_delivery", "delivery":
		return "outbox_delivery"
	case "job", "agent_job", "agentjob":
		return "agent_job"
	default:
		if value == "" {
			return "unknown"
		}
		return value
	}
}

func formatQueueCompareTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func laterQueueTime(left string, right string) string {
	if right == "" {
		return left
	}
	if left == "" || right > left {
		return right
	}
	return left
}
