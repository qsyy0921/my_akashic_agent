package service

import (
	"context"
	"errors"
	"sort"

	inport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/in"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const queueDiagnosticsSampleLimit = 200

type QueueBackendService struct {
	view           query.QueueBackendView
	diagnostics    outport.WorkQueuePublishDiagnosticReader
	compare        inport.WorkQueueCompareDiagnosticReader
	outboxRepo     outport.OutboxRepository
	outboxEvents   outport.OutboxDeliveryEventStore
	agentJobRepo   outport.AgentJobRepository
	agentJobEvents outport.AgentJobEventStore
}

func NewQueueBackendService(view query.QueueBackendView) *QueueBackendService {
	return &QueueBackendService{view: view}
}

type QueueBackendDiagnosticsDeps struct {
	Diagnostics    outport.WorkQueuePublishDiagnosticReader
	Compare        inport.WorkQueueCompareDiagnosticReader
	OutboxRepo     outport.OutboxRepository
	OutboxEvents   outport.OutboxDeliveryEventStore
	AgentJobRepo   outport.AgentJobRepository
	AgentJobEvents outport.AgentJobEventStore
}

func NewQueueBackendServiceWithDiagnostics(view query.QueueBackendView, deps QueueBackendDiagnosticsDeps) *QueueBackendService {
	return &QueueBackendService{
		view:           view,
		diagnostics:    deps.Diagnostics,
		compare:        deps.Compare,
		outboxRepo:     deps.OutboxRepo,
		outboxEvents:   deps.OutboxEvents,
		agentJobRepo:   deps.AgentJobRepo,
		agentJobEvents: deps.AgentJobEvents,
	}
}

func (s *QueueBackendService) Get(ctx context.Context) (query.QueueBackendView, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueBackendView{}, err
	}
	if s == nil {
		return query.QueueBackendView{}, errors.New("queue backend service is nil")
	}
	view := s.view
	if view.Mode == "shadow_publish" || view.Mode == "dual_read_compare" {
		diagnostics, err := s.shadowPublishDiagnostics(ctx)
		if err != nil {
			return query.QueueBackendView{}, err
		}
		view.ShadowPublish = &diagnostics
	}
	if view.Mode == "dual_read_compare" {
		diagnostics, err := s.dualReadDiagnostics(ctx)
		if err != nil {
			return query.QueueBackendView{}, err
		}
		view.DualReadCompare = &diagnostics
	}
	return view, nil
}

func (s *QueueBackendService) dualReadDiagnostics(ctx context.Context) (query.QueueDualReadDiagnostics, error) {
	diagnostics := query.QueueDualReadDiagnostics{
		Enabled:     s != nil && s.compare != nil,
		SampleLimit: queueCompareRecentLimit,
		Notes: []string{
			"dual_read_compare is compare-only and must not execute queue side effects",
		},
	}
	if s == nil {
		return diagnostics, nil
	}
	if s.compare == nil {
		diagnostics.Notes = append(diagnostics.Notes, "no active queue candidate compare recorder")
		return diagnostics, nil
	}
	snapshot, err := s.compare.SnapshotDualReadDiagnostics(ctx)
	if err != nil {
		return query.QueueDualReadDiagnostics{}, err
	}
	snapshot.Notes = append(diagnostics.Notes, snapshot.Notes...)
	return snapshot, nil
}

func (s *QueueBackendService) shadowPublishDiagnostics(ctx context.Context) (query.QueueShadowPublishDiagnostics, error) {
	diagnostics := query.QueueShadowPublishDiagnostics{
		Enabled:     s != nil && s.diagnostics != nil,
		SampleLimit: queueDiagnosticsSampleLimit,
		Notes: []string{
			"state store remains authoritative; queue publish diagnostics are advisory",
			"reconciliation counts are capped by sample_limit",
		},
	}
	if s == nil {
		return diagnostics, nil
	}
	if s.diagnostics != nil {
		snapshot, err := s.diagnostics.SnapshotWorkQueuePublishDiagnostics(ctx)
		if err != nil {
			return query.QueueShadowPublishDiagnostics{}, err
		}
		diagnostics = mergeQueuePublishSnapshot(diagnostics, snapshot)
	} else {
		diagnostics.Notes = append(diagnostics.Notes, "no active queue publisher diagnostics recorder")
	}

	if s.outboxRepo != nil {
		items, err := s.outboxRepo.ListOutboxDeliveries(ctx, queueDiagnosticsSampleLimit)
		if err != nil {
			return query.QueueShadowPublishDiagnostics{}, err
		}
		diagnostics.StateStoreTotal += len(items)
		mergeWorkKindReconciliation(&diagnostics, "outbox_delivery", len(items), 0)
	} else {
		diagnostics.Notes = append(diagnostics.Notes, "outbox repository unavailable for reconciliation")
	}
	if s.outboxEvents != nil {
		events, err := s.outboxEvents.ListOutboxDeliveryEvents(ctx, query.OutboxDeliveryEventFilter{
			EventType: "queued",
			Limit:     queueDiagnosticsSampleLimit,
		})
		if err != nil {
			return query.QueueShadowPublishDiagnostics{}, err
		}
		diagnostics.EventStreamTotal += len(events)
		mergeWorkKindReconciliation(&diagnostics, "outbox_delivery", 0, len(events))
	} else {
		diagnostics.Notes = append(diagnostics.Notes, "outbox event stream unavailable for reconciliation")
	}
	if s.agentJobRepo != nil {
		items, err := s.agentJobRepo.ListAgentJobs(ctx, query.AgentJobFilter{Limit: queueDiagnosticsSampleLimit})
		if err != nil {
			return query.QueueShadowPublishDiagnostics{}, err
		}
		diagnostics.StateStoreTotal += len(items)
		mergeWorkKindReconciliation(&diagnostics, "agent_job", len(items), 0)
	} else {
		diagnostics.Notes = append(diagnostics.Notes, "agent job repository unavailable for reconciliation")
	}
	if s.agentJobEvents != nil {
		events, err := s.agentJobEvents.ListAgentJobEvents(ctx, query.AgentJobEventFilter{
			EventType: "created",
			Limit:     queueDiagnosticsSampleLimit,
		})
		if err != nil {
			return query.QueueShadowPublishDiagnostics{}, err
		}
		diagnostics.EventStreamTotal += len(events)
		mergeWorkKindReconciliation(&diagnostics, "agent_job", 0, len(events))
	} else {
		diagnostics.Notes = append(diagnostics.Notes, "agent job event stream unavailable for reconciliation")
	}

	for i := range diagnostics.WorkKinds {
		item := &diagnostics.WorkKinds[i]
		item.StateMinusPublished = item.StateStoreCount - item.SucceededCount
		item.EventMinusPublished = item.EventStreamCount - item.SucceededCount
		diagnostics.StateMinusPublishedTotal += item.StateMinusPublished
		diagnostics.EventMinusPublishedTotal += item.EventMinusPublished
	}
	sort.Slice(diagnostics.WorkKinds, func(i, j int) bool {
		return diagnostics.WorkKinds[i].WorkKind < diagnostics.WorkKinds[j].WorkKind
	})
	return diagnostics, nil
}

func mergeQueuePublishSnapshot(
	base query.QueueShadowPublishDiagnostics,
	snapshot query.QueueShadowPublishDiagnostics,
) query.QueueShadowPublishDiagnostics {
	base.Enabled = snapshot.Enabled
	base.AttemptTotal = snapshot.AttemptTotal
	base.SucceededTotal = snapshot.SucceededTotal
	base.FailedTotal = snapshot.FailedTotal
	base.LastPublishedAt = snapshot.LastPublishedAt
	base.LastFailedAt = snapshot.LastFailedAt
	base.LastError = snapshot.LastError
	base.Subjects = append(base.Subjects, snapshot.Subjects...)
	for _, item := range snapshot.WorkKinds {
		mergeWorkKindPublishStats(&base, item)
	}
	if len(snapshot.Notes) > 0 {
		base.Notes = append(base.Notes, snapshot.Notes...)
	}
	return base
}

func mergeWorkKindPublishStats(diagnostics *query.QueueShadowPublishDiagnostics, incoming query.QueuePublishWorkKindStats) {
	for i := range diagnostics.WorkKinds {
		item := &diagnostics.WorkKinds[i]
		if item.WorkKind != incoming.WorkKind {
			continue
		}
		item.AttemptCount += incoming.AttemptCount
		item.SucceededCount += incoming.SucceededCount
		item.FailedCount += incoming.FailedCount
		if incoming.LastPublishedAt > item.LastPublishedAt {
			item.LastPublishedAt = incoming.LastPublishedAt
		}
		if incoming.LastFailedAt > item.LastFailedAt {
			item.LastFailedAt = incoming.LastFailedAt
		}
		if incoming.LastError != "" {
			item.LastError = incoming.LastError
		}
		return
	}
	diagnostics.WorkKinds = append(diagnostics.WorkKinds, incoming)
}

func mergeWorkKindReconciliation(diagnostics *query.QueueShadowPublishDiagnostics, workKind string, stateCount int, eventCount int) {
	for i := range diagnostics.WorkKinds {
		item := &diagnostics.WorkKinds[i]
		if item.WorkKind != workKind {
			continue
		}
		item.StateStoreCount += stateCount
		item.EventStreamCount += eventCount
		return
	}
	diagnostics.WorkKinds = append(diagnostics.WorkKinds, query.QueuePublishWorkKindStats{
		WorkKind:         workKind,
		StateStoreCount:  stateCount,
		EventStreamCount: eventCount,
	})
}
