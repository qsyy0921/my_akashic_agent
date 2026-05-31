package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

const (
	defaultRuntimeOverviewLimit             = 200
	defaultRuntimeOverviewEventLimit        = 50
	defaultRuntimeOverviewStaleAfterSeconds = 15 * 60
	maxRuntimeOverviewLimit                 = 200
)

type RuntimeOverviewDeps struct {
	QueueBackend         runtimeQueueBackendGetter
	RuntimeConfig        runtimeConfigGetter
	DeliveryAdapters     runtimeDeliveryAdapterLister
	SendLedger           runtimeSendLedgerMetricsGetter
	InboxMetrics         runtimeInboxMetricsGetter
	InboundDedupe        runtimeInboundDedupeMetricsGetter
	AgentJobMetrics      runtimeAgentJobMetricsGetter
	OutboxMetrics        runtimeOutboxMetricsGetter
	KnowledgeDiagnostics runtimeKnowledgeDiagnosticsGetter
	RuntimeWorkers       runtimeWorkerDiagnosticsGetter
	ObserveTargets       runtimeObserveTargetsGetter
	ObserveCapture       runtimeObserveCaptureGetter
	ReceiverStatuses     runtimeReceiverStatusesGetter
	ReceiverLeases       runtimeReceiverLeasesGetter
}

type runtimeQueueBackendGetter interface {
	Get(ctx context.Context) (query.QueueBackendView, error)
}

type runtimeConfigGetter interface {
	GetRuntimeConfig(ctx context.Context) (query.RuntimeConfigView, error)
}

type runtimeDeliveryAdapterLister interface {
	ListDeliveryAdapters(ctx context.Context) ([]query.DeliveryAdapterDiagnosticsView, error)
}

type runtimeSendLedgerMetricsGetter interface {
	Metrics(ctx context.Context, filter query.SendLedgerMetricsFilter) (query.SendLedgerMetricsView, error)
}

type runtimeInboxMetricsGetter interface {
	Get(ctx context.Context, filter query.InboxMetricsFilter) (query.InboxMetricsView, error)
}

type runtimeInboundDedupeMetricsGetter interface {
	Metrics(ctx context.Context, filter query.InboundDedupeMetricsFilter) (query.InboundDedupeMetricsView, error)
}

type runtimeAgentJobMetricsGetter interface {
	Get(ctx context.Context, filter query.AgentJobMetricsFilter) (query.AgentJobMetricsView, error)
}

type runtimeOutboxMetricsGetter interface {
	Get(ctx context.Context, filter query.OutboxMetricsFilter) (query.OutboxMetricsView, error)
}

type runtimeKnowledgeDiagnosticsGetter interface {
	Get(ctx context.Context, filter query.KnowledgeWorkerDiagnosticsFilter) (query.KnowledgeWorkerDiagnosticsView, error)
}

type runtimeWorkerDiagnosticsGetter interface {
	GetRuntimeWorkers(ctx context.Context) (query.RuntimeWorkerDiagnosticsView, error)
}

type runtimeObserveTargetsGetter interface {
	ListObserveTargets(ctx context.Context) (query.ObserveTargetsView, error)
}

type runtimeObserveCaptureGetter interface {
	GetObserveCaptureDiagnostics(ctx context.Context, filter query.ObserveCaptureDiagnosticsFilter) (query.ObserveCaptureDiagnosticsView, error)
}

type runtimeReceiverStatusesGetter interface {
	ListReceiverStatuses(ctx context.Context) (query.ReceiverStatusesView, error)
}

type runtimeReceiverLeasesGetter interface {
	ListReceiverLeases(ctx context.Context) (query.ReceiverLeasesView, error)
}

type RuntimeOverviewService struct {
	deps RuntimeOverviewDeps
}

func NewRuntimeOverviewService(deps RuntimeOverviewDeps) *RuntimeOverviewService {
	return &RuntimeOverviewService{deps: deps}
}

func (s *RuntimeOverviewService) Get(ctx context.Context, filter query.RuntimeOverviewFilter) (query.RuntimeOverviewView, error) {
	if err := ctx.Err(); err != nil {
		return query.RuntimeOverviewView{}, err
	}
	limit := boundedRuntimeOverviewLimit(filter.Limit, defaultRuntimeOverviewLimit)
	eventLimit := boundedRuntimeOverviewLimit(filter.EventLimit, defaultRuntimeOverviewEventLimit)
	staleAfterSeconds := filter.StaleAfterSeconds
	if staleAfterSeconds <= 0 {
		staleAfterSeconds = defaultRuntimeOverviewStaleAfterSeconds
	}
	if staleAfterSeconds > 24*60*60 {
		staleAfterSeconds = 24 * 60 * 60
	}

	var (
		errors           []query.RuntimeOverviewErrorView
		deliveryAdapters []query.DeliveryAdapterDiagnosticsView
		queueBackend     query.QueueBackendView
		runtimeConfig    query.RuntimeConfigView
		sendLedger       query.SendLedgerMetricsView
		inboxMetrics     query.InboxMetricsView
		inboundDedupe    query.InboundDedupeMetricsView
		agentJobMetrics  query.AgentJobMetricsView
		outboxMetrics    query.OutboxMetricsView
		diagnostics      query.KnowledgeWorkerDiagnosticsView
		runtimeWorkers   query.RuntimeWorkerDiagnosticsView
		observeTargets   query.ObserveTargetsView
		observeCapture   query.ObserveCaptureDiagnosticsView
		receiverStatuses query.ReceiverStatusesView
		receiverLeases   query.ReceiverLeasesView
	)

	if s == nil {
		errors = append(errors, runtimeOverviewError("runtime-overview", fmt.Errorf("runtime overview service is nil")))
	} else {
		deps := s.deps
		if deps.QueueBackend == nil {
			errors = append(errors, runtimeOverviewError("queue-backend", fmt.Errorf("queue backend viewer disabled")))
		} else if item, err := deps.QueueBackend.Get(ctx); err != nil {
			errors = append(errors, runtimeOverviewError("queue-backend", err))
		} else {
			queueBackend = item
		}

		if deps.RuntimeConfig != nil {
			if item, err := deps.RuntimeConfig.GetRuntimeConfig(ctx); err != nil {
				errors = append(errors, runtimeOverviewError("runtime-config", err))
			} else {
				runtimeConfig = item
			}
		}

		if deps.DeliveryAdapters == nil {
			errors = append(errors, runtimeOverviewError("delivery-adapters", fmt.Errorf("delivery adapter diagnostics disabled")))
		} else if items, err := deps.DeliveryAdapters.ListDeliveryAdapters(ctx); err != nil {
			errors = append(errors, runtimeOverviewError("delivery-adapters", err))
		} else {
			deliveryAdapters = items
		}

		if deps.SendLedger == nil {
			errors = append(errors, runtimeOverviewError("send-ledger-metrics", fmt.Errorf("send ledger metrics disabled")))
		} else if item, err := deps.SendLedger.Metrics(ctx, query.SendLedgerMetricsFilter{Limit: limit}); err != nil {
			errors = append(errors, runtimeOverviewError("send-ledger-metrics", err))
		} else {
			sendLedger = item
		}

		if deps.InboxMetrics == nil {
			errors = append(errors, runtimeOverviewError("inbox-metrics", fmt.Errorf("inbox metrics disabled")))
		} else if item, err := deps.InboxMetrics.Get(ctx, query.InboxMetricsFilter{Limit: limit}); err != nil {
			errors = append(errors, runtimeOverviewError("inbox-metrics", err))
		} else {
			inboxMetrics = item
		}

		if deps.InboundDedupe == nil {
			errors = append(errors, runtimeOverviewError("inbound-dedupe-metrics", fmt.Errorf("inbound dedupe metrics disabled")))
		} else if item, err := deps.InboundDedupe.Metrics(ctx, query.InboundDedupeMetricsFilter{Limit: limit}); err != nil {
			errors = append(errors, runtimeOverviewError("inbound-dedupe-metrics", err))
		} else {
			inboundDedupe = item
		}

		if deps.AgentJobMetrics == nil {
			errors = append(errors, runtimeOverviewError("job-metrics", fmt.Errorf("agent job metrics disabled")))
		} else if item, err := deps.AgentJobMetrics.Get(ctx, query.AgentJobMetricsFilter{JobLimit: limit, EventLimit: eventLimit}); err != nil {
			errors = append(errors, runtimeOverviewError("job-metrics", err))
		} else {
			agentJobMetrics = item
		}

		if deps.OutboxMetrics == nil {
			errors = append(errors, runtimeOverviewError("outbox-metrics", fmt.Errorf("outbox metrics disabled")))
		} else if item, err := deps.OutboxMetrics.Get(ctx, query.OutboxMetricsFilter{DeliveryLimit: limit, EventLimit: eventLimit}); err != nil {
			errors = append(errors, runtimeOverviewError("outbox-metrics", err))
		} else {
			outboxMetrics = item
		}

		if deps.KnowledgeDiagnostics == nil {
			errors = append(errors, runtimeOverviewError("knowledge-worker-diagnostics", fmt.Errorf("knowledge diagnostics disabled")))
		} else if item, err := deps.KnowledgeDiagnostics.Get(ctx, query.KnowledgeWorkerDiagnosticsFilter{
			Limit:             limit,
			StaleAfterSeconds: staleAfterSeconds,
		}); err != nil {
			errors = append(errors, runtimeOverviewError("knowledge-worker-diagnostics", err))
		} else {
			diagnostics = item
		}

		if deps.RuntimeWorkers == nil {
			errors = append(errors, runtimeOverviewError("runtime-workers", fmt.Errorf("runtime worker diagnostics disabled")))
		} else if item, err := deps.RuntimeWorkers.GetRuntimeWorkers(ctx); err != nil {
			errors = append(errors, runtimeOverviewError("runtime-workers", err))
		} else {
			runtimeWorkers = item
		}

		if deps.ObserveTargets == nil {
			errors = append(errors, runtimeOverviewError("observe-targets", fmt.Errorf("observe target diagnostics disabled")))
		} else if item, err := deps.ObserveTargets.ListObserveTargets(ctx); err != nil {
			errors = append(errors, runtimeOverviewError("observe-targets", err))
		} else {
			observeTargets = item
		}

		if deps.ObserveCapture == nil {
			errors = append(errors, runtimeOverviewError("observe-capture-diagnostics", fmt.Errorf("observe capture diagnostics disabled")))
		} else if item, err := deps.ObserveCapture.GetObserveCaptureDiagnostics(ctx, query.ObserveCaptureDiagnosticsFilter{Limit: limit}); err != nil {
			errors = append(errors, runtimeOverviewError("observe-capture-diagnostics", err))
		} else {
			observeCapture = item
		}

		if deps.ReceiverStatuses == nil {
			errors = append(errors, runtimeOverviewError("receiver-statuses", fmt.Errorf("receiver status diagnostics disabled")))
		} else if item, err := deps.ReceiverStatuses.ListReceiverStatuses(ctx); err != nil {
			errors = append(errors, runtimeOverviewError("receiver-statuses", err))
		} else {
			receiverStatuses = item
		}

		if deps.ReceiverLeases == nil {
			errors = append(errors, runtimeOverviewError("receiver-leases", fmt.Errorf("receiver lease diagnostics disabled")))
		} else if item, err := deps.ReceiverLeases.ListReceiverLeases(ctx); err != nil {
			errors = append(errors, runtimeOverviewError("receiver-leases", err))
		} else {
			receiverLeases = item
		}
	}

	summary := runtimeOverviewSummary(
		deliveryAdapters,
		queueBackend,
		runtimeConfig,
		sendLedger,
		inboxMetrics,
		inboundDedupe,
		agentJobMetrics,
		outboxMetrics,
		diagnostics,
		runtimeWorkers,
		observeTargets,
		observeCapture,
		receiverStatuses,
		receiverLeases,
	)
	cards := runtimeOverviewCards(
		summary,
		deliveryAdapters,
		queueBackend,
		runtimeConfig,
		sendLedger,
		inboxMetrics,
		inboundDedupe,
		agentJobMetrics,
		outboxMetrics,
		diagnostics,
		runtimeWorkers,
		observeTargets,
		observeCapture,
		receiverStatuses,
		receiverLeases,
		errors,
	)

	return query.RuntimeOverviewView{
		Summary:           summary,
		Cards:             cards,
		DeliveryAdapters:  deliveryAdapters,
		QueueBackend:      queueBackend,
		RuntimeConfig:     runtimeConfig,
		RuntimeWorkers:    runtimeWorkers,
		ObserveTargets:    observeTargets,
		ObserveCapture:    observeCapture,
		ReceiverStatuses:  receiverStatuses,
		ReceiverLeases:    receiverLeases,
		SendLedgerMetrics: sendLedger,
		InboxMetrics:      inboxMetrics,
		InboundDedupe:     inboundDedupe,
		AgentJobMetrics:   agentJobMetrics,
		OutboxMetrics:     outboxMetrics,
		Diagnostics:       diagnostics,
		Status: query.RuntimeOverviewStatusView{
			RuntimeAvailable: true,
			HealthAvailable:  true,
			Partial:          len(errors) > 0,
			Errors:           errors,
		},
	}, nil
}

func runtimeOverviewSummary(
	deliveryAdapters []query.DeliveryAdapterDiagnosticsView,
	queueBackend query.QueueBackendView,
	runtimeConfig query.RuntimeConfigView,
	sendLedger query.SendLedgerMetricsView,
	inboxMetrics query.InboxMetricsView,
	inboundDedupe query.InboundDedupeMetricsView,
	agentJobMetrics query.AgentJobMetricsView,
	outboxMetrics query.OutboxMetricsView,
	diagnostics query.KnowledgeWorkerDiagnosticsView,
	runtimeWorkers query.RuntimeWorkerDiagnosticsView,
	observeTargets query.ObserveTargetsView,
	observeCapture query.ObserveCaptureDiagnosticsView,
	receiverStatuses query.ReceiverStatusesView,
	receiverLeases query.ReceiverLeasesView,
) map[string]any {
	enabledAdapters := 0
	for _, item := range deliveryAdapters {
		if item.Enabled {
			enabledAdapters++
		}
	}
	agentJobLeases := intFromMap(agentJobMetrics.JobsByStatus, "leased") + intFromMap(agentJobMetrics.JobsByStatus, "running")
	outboxLeases := intFromMap(outboxMetrics.DeliveriesByStatus, "leased") + intFromMap(outboxMetrics.DeliveriesByStatus, "dispatching")
	checkpointLagMax := maxCheckpointLag(diagnostics)
	externalLeaseReady := false
	if queueBackend.ExternalLease != nil {
		externalLeaseReady = queueBackend.ExternalLease.AllowExecution
	}

	return map[string]any{
		"jobs_total":                                agentJobMetrics.SampledJobs,
		"outbox_total":                              outboxMetrics.SampledDeliveries,
		"checkpoints_total":                         intFromMap(diagnostics.Totals, "checkpoints"),
		"worker_leases":                             agentJobLeases + outboxLeases,
		"agent_job_leases":                          agentJobLeases,
		"outbox_leases":                             outboxLeases,
		"stale_jobs":                                intFromMap(diagnostics.Totals, "stale_leases"),
		"dead_letters":                              agentJobMetrics.DeadLetters.CurrentTotal + outboxMetrics.DeadLetters.CurrentTotal,
		"checkpoint_lag_max":                        checkpointLagMax,
		"job_events":                                agentJobMetrics.SampledEvents,
		"outbox_events":                             outboxMetrics.SampledEvents,
		"rag_eval_failures":                         intFromMap(agentJobMetrics.DeadLetters.ByType, "rag_eval"),
		"delivery_adapters":                         len(deliveryAdapters),
		"delivery_adapters_enabled":                 enabledAdapters,
		"delivery_adapters_disabled":                len(deliveryAdapters) - enabledAdapters,
		"queue_backend_provider":                    queueBackend.Provider,
		"queue_backend_mode":                        queueBackend.Mode,
		"queue_consumer_concurrency":                queueBackend.ConsumerConcurrency,
		"queue_max_in_flight":                       queueBackend.MaxInFlight,
		"queue_external_lease_ready":                externalLeaseReady,
		"runtime_config_blockers":                   len(runtimeConfig.Readiness.Blockers),
		"runtime_config_onebot_missing":             len(runtimeConfig.Delivery.OneBotMissingChannels),
		"runtime_workers":                           intFromMap(runtimeWorkers.Totals, "workers"),
		"runtime_workers_enabled":                   intFromMap(runtimeWorkers.Totals, "enabled"),
		"runtime_workers_running":                   intFromMap(runtimeWorkers.Totals, "running"),
		"observe_targets":                           intFromMap(observeTargets.Totals, "targets"),
		"observe_targets_enabled":                   intFromMap(observeTargets.Totals, "enabled"),
		"observe_targets_observe_only":              intFromMap(observeTargets.Totals, "observe_only"),
		"observe_targets_reply_allowed":             intFromMap(observeTargets.Totals, "reply_allowed"),
		"observe_target_groups":                     intFromMap(observeTargets.Totals, "groups"),
		"observe_capture_targets":                   intFromMap(observeCapture.Totals, "targets"),
		"observe_capture_ready":                     intFromMap(observeCapture.Totals, "ready"),
		"observe_capture_warning":                   intFromMap(observeCapture.Totals, "warning"),
		"observe_capture_blocked":                   intFromMap(observeCapture.Totals, "blocked"),
		"observe_capture_text":                      intFromMap(observeCapture.Totals, "text_covered"),
		"observe_capture_image":                     intFromMap(observeCapture.Totals, "image_covered"),
		"observe_capture_file":                      intFromMap(observeCapture.Totals, "file_covered"),
		"observe_capture_content_ready":             intFromMap(observeCapture.Totals, "content_ready_assets"),
		"observe_capture_receiver_connected":        intFromMap(observeCapture.Totals, "receiver_connected"),
		"observe_capture_receiver_status_connected": intFromMap(observeCapture.Totals, "receiver_status_connected"),
		"observe_capture_receiver_activity_recent":  intFromMap(observeCapture.Totals, "receiver_activity_recent"),
		"receiver_statuses":                         intFromMap(receiverStatuses.Totals, "receivers"),
		"receiver_status_connected":                 intFromMap(receiverStatuses.Totals, "connected"),
		"receiver_status_suspended":                 intFromMap(receiverStatuses.Totals, "suspended"),
		"receiver_status_failed":                    intFromMap(receiverStatuses.Totals, "failed"),
		"receiver_status_qq":                        intFromMap(receiverStatuses.Totals, "qq"),
		"receiver_status_telegram":                  intFromMap(receiverStatuses.Totals, "telegram"),
		"receiver_leases":                           intFromMap(receiverLeases.Totals, "leases"),
		"receiver_leases_active":                    intFromMap(receiverLeases.Totals, "active"),
		"receiver_leases_expired":                   intFromMap(receiverLeases.Totals, "expired"),
		"send_ledger_records":                       sendLedger.SampledRecords,
		"send_ledger_repeated_hashes":               sendLedger.RepeatedContentHashes,
		"inbox_metric_events":                       inboxMetrics.SampledEvents,
		"inbox_metric_observe_only":                 inboxMetrics.ObserveOnlyTotal,
		"inbox_metric_with_attachments":             inboxMetrics.WithAttachments,
		"inbound_dedupe_records":                    inboundDedupe.SampledRecords,
		"inbound_dedupe_active_records":             inboundDedupe.ActiveRecords,
		"inbound_dedupe_duplicate_records":          inboundDedupe.DuplicateRecords,
		"inbound_dedupe_seen_total":                 inboundDedupe.SeenTotal,
		"inbound_dedupe_duplicate_seen_total":       inboundDedupe.DuplicateSeenTotal,
		"inbound_dedupe_scopes":                     len(inboundDedupe.Scopes),
		"agent_job_metric_events":                   agentJobMetrics.SampledEvents,
		"agent_job_metric_dead_letters":             agentJobMetrics.DeadLetters.CurrentTotal,
		"outbox_metric_events":                      outboxMetrics.SampledEvents,
		"outbox_metric_dead_letters":                outboxMetrics.DeadLetters.CurrentTotal,
	}
}

func runtimeOverviewCards(
	summary map[string]any,
	deliveryAdapters []query.DeliveryAdapterDiagnosticsView,
	queueBackend query.QueueBackendView,
	runtimeConfig query.RuntimeConfigView,
	sendLedger query.SendLedgerMetricsView,
	inboxMetrics query.InboxMetricsView,
	inboundDedupe query.InboundDedupeMetricsView,
	agentJobMetrics query.AgentJobMetricsView,
	outboxMetrics query.OutboxMetricsView,
	diagnostics query.KnowledgeWorkerDiagnosticsView,
	runtimeWorkers query.RuntimeWorkerDiagnosticsView,
	observeTargets query.ObserveTargetsView,
	observeCapture query.ObserveCaptureDiagnosticsView,
	receiverStatuses query.ReceiverStatusesView,
	receiverLeases query.ReceiverLeasesView,
	errors []query.RuntimeOverviewErrorView,
) []query.RuntimeOverviewCardView {
	queueValue := fmt.Sprintf("%s/%s", emptyAsUnknown(queueBackend.Provider), emptyAsUnknown(queueBackend.Mode))
	cards := []query.RuntimeOverviewCardView{
		runtimeOverviewCard("runtime_health", "Runtime Health", "ok", runtimeHealthStatus(errors), map[string]any{"errors": errors}),
		runtimeOverviewCard("worker_leases", "Worker Leases", intSummary(summary, "worker_leases"), statusIfPositive(intSummary(summary, "stale_jobs"), "warn", "ok"), map[string]any{"diagnostics": diagnostics}),
		runtimeOverviewCard("stale_jobs", "Stale Jobs", intSummary(summary, "stale_jobs"), statusIfPositive(intSummary(summary, "stale_jobs"), "danger", "ok"), map[string]any{"diagnostics": diagnostics}),
		runtimeOverviewCard("dead_letters", "Dead Letters", intSummary(summary, "dead_letters"), statusIfPositive(intSummary(summary, "dead_letters"), "danger", "ok"), map[string]any{"agent_job_metrics": agentJobMetrics, "outbox_metrics": outboxMetrics}),
		runtimeOverviewCard("checkpoint_lag", "Checkpoint Lag", intSummary(summary, "checkpoint_lag_max"), statusIfPositive(intSummary(summary, "checkpoint_lag_max"), "warn", "ok"), map[string]any{"diagnostics": diagnostics}),
		runtimeOverviewCard("job_events", "Job Events", intSummary(summary, "job_events"), statusIfPositive(intSummary(summary, "job_events"), "ok", "muted"), map[string]any{"agent_job_metrics": agentJobMetrics}),
		runtimeOverviewCard("agent_job_metrics", "Agent Job Metrics", intSummary(summary, "agent_job_metric_events"), statusIfPositive(intSummary(summary, "agent_job_metric_dead_letters"), "danger", "ok"), map[string]any{"agent_job_metrics": agentJobMetrics}),
		runtimeOverviewCard("outbox_metrics", "Outbox Metrics", intSummary(summary, "outbox_metric_events"), statusIfPositive(intSummary(summary, "outbox_metric_dead_letters"), "danger", "ok"), map[string]any{"outbox_metrics": outboxMetrics}),
		runtimeOverviewCard("outbox_events", "Outbox Events", intSummary(summary, "outbox_events"), statusIfPositive(intSummary(summary, "outbox_events"), "ok", "muted"), map[string]any{"outbox_metrics": outboxMetrics}),
		runtimeOverviewCard("rag_eval_failures", "RAG Eval Failures", intSummary(summary, "rag_eval_failures"), statusIfPositive(intSummary(summary, "rag_eval_failures"), "danger", "ok"), map[string]any{"agent_job_metrics": agentJobMetrics}),
		runtimeOverviewCard("delivery_adapters", "Delivery Adapters", intSummary(summary, "delivery_adapters_enabled"), deliveryAdapterStatus(deliveryAdapters), map[string]any{"items": deliveryAdapters}),
		runtimeOverviewCard("queue_backend", "Queue Backend", queueValue, queueBackendStatus(queueBackend), map[string]any{"queue_backend": queueBackend}),
		runtimeOverviewCard("runtime_workers", "Runtime Workers", intSummary(summary, "runtime_workers_running"), runtimeWorkerStatus(runtimeWorkers), map[string]any{"runtime_workers": runtimeWorkers}),
		runtimeOverviewCard("observe_targets", "Observe Targets", intSummary(summary, "observe_targets_enabled"), observeTargetStatus(observeTargets), map[string]any{"observe_targets": observeTargets}),
		runtimeOverviewCard("observe_capture", "Observe Capture", observeCaptureValue(observeCapture), observeCaptureStatus(observeCapture), map[string]any{"observe_capture": observeCapture}),
		runtimeOverviewCard("receiver_statuses", "Receiver Statuses", intSummary(summary, "receiver_status_connected"), receiverStatusStatus(receiverStatuses), map[string]any{"receiver_statuses": receiverStatuses}),
		runtimeOverviewCard("receiver_leases", "Receiver Leases", intSummary(summary, "receiver_leases_active"), receiverLeaseStatus(receiverLeases), map[string]any{"receiver_leases": receiverLeases}),
		runtimeOverviewCard("send_ledger_metrics", "Send Ledger Metrics", intSummary(summary, "send_ledger_records"), statusIfPositive(intSummary(summary, "send_ledger_repeated_hashes"), "warn", statusIfPositive(intSummary(summary, "send_ledger_records"), "ok", "muted")), map[string]any{"send_ledger_metrics": sendLedger}),
		runtimeOverviewCard("inbox_metrics", "Inbox Metrics", intSummary(summary, "inbox_metric_events"), statusIfPositive(intSummary(summary, "inbox_metric_events"), "ok", "muted"), map[string]any{"inbox_metrics": inboxMetrics}),
		runtimeOverviewCard("inbound_dedupe_metrics", "Inbound Dedupe", intSummary(summary, "inbound_dedupe_duplicate_seen_total"), statusIfPositive(intSummary(summary, "inbound_dedupe_duplicate_seen_total"), "warn", statusIfPositive(intSummary(summary, "inbound_dedupe_records"), "ok", "muted")), map[string]any{"inbound_dedupe_metrics": inboundDedupe}),
	}
	if runtimeConfig.SideEffect != "" {
		cards = append(cards, runtimeOverviewCard("runtime_config", "Runtime Config", runtimeConfigCardValue(runtimeConfig), runtimeConfigStatus(runtimeConfig), map[string]any{"runtime_config": runtimeConfig}))
	}
	return cards
}

func runtimeOverviewCard(id string, label string, value any, status string, detail map[string]any) query.RuntimeOverviewCardView {
	return query.RuntimeOverviewCardView{ID: id, Label: label, Value: value, Status: status, Detail: detail}
}

func runtimeOverviewError(endpoint string, err error) query.RuntimeOverviewErrorView {
	return query.RuntimeOverviewErrorView{Endpoint: endpoint, Error: err.Error()}
}

func boundedRuntimeOverviewLimit(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value > maxRuntimeOverviewLimit {
		return maxRuntimeOverviewLimit
	}
	return value
}

func intFromMap(items map[string]int, key string) int {
	if items == nil {
		return 0
	}
	return items[key]
}

func intSummary(summary map[string]any, key string) int {
	value, ok := summary[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func statusIfPositive(value int, positive string, zero string) string {
	if value > 0 {
		return positive
	}
	return zero
}

func runtimeHealthStatus(errors []query.RuntimeOverviewErrorView) string {
	if len(errors) > 0 {
		return "warn"
	}
	return "ok"
}

func deliveryAdapterStatus(items []query.DeliveryAdapterDiagnosticsView) string {
	if len(items) == 0 {
		return "muted"
	}
	for _, item := range items {
		if !item.Enabled {
			return "warn"
		}
	}
	return "ok"
}

func queueBackendStatus(view query.QueueBackendView) string {
	if view.Provider == "" || view.Provider == "unknown" {
		return "muted"
	}
	if view.Provider == "local" || view.Mode == "local_state_store" {
		return "ok"
	}
	if view.ExternalQueueActive {
		return "ok"
	}
	if view.Mode == "external_lease" && (view.ExternalLease == nil || !view.ExternalLease.AllowExecution) {
		return "warn"
	}
	if view.ExternalQueueConfigured {
		return "ok"
	}
	return "warn"
}

func runtimeWorkerStatus(view query.RuntimeWorkerDiagnosticsView) string {
	enabled := intFromMap(view.Totals, "enabled")
	running := intFromMap(view.Totals, "running")
	if enabled == 0 {
		return "muted"
	}
	if running < enabled {
		return "warn"
	}
	return "ok"
}

func observeTargetStatus(view query.ObserveTargetsView) string {
	if intFromMap(view.Totals, "targets") == 0 {
		return "muted"
	}
	if intFromMap(view.Totals, "enabled") == 0 {
		return "warn"
	}
	return "ok"
}

func observeCaptureStatus(view query.ObserveCaptureDiagnosticsView) string {
	if intFromMap(view.Totals, "targets") == 0 {
		return "muted"
	}
	if intFromMap(view.Totals, "blocked") > 0 {
		return "danger"
	}
	if intFromMap(view.Totals, "ready") < intFromMap(view.Totals, "enabled") {
		return "warn"
	}
	return "ok"
}

func observeCaptureValue(view query.ObserveCaptureDiagnosticsView) string {
	return fmt.Sprintf("%d/%d", intFromMap(view.Totals, "ready"), intFromMap(view.Totals, "enabled"))
}

func receiverStatusStatus(view query.ReceiverStatusesView) string {
	if intFromMap(view.Totals, "receivers") == 0 {
		return "muted"
	}
	if intFromMap(view.Totals, "failed") > 0 {
		return "danger"
	}
	if intFromMap(view.Totals, "suspended") > 0 {
		return "warn"
	}
	if intFromMap(view.Totals, "connected") == 0 {
		return "warn"
	}
	return "ok"
}

func receiverLeaseStatus(view query.ReceiverLeasesView) string {
	if intFromMap(view.Totals, "leases") == 0 {
		return "muted"
	}
	if intFromMap(view.Totals, "expired") > 0 {
		return "warn"
	}
	if intFromMap(view.Totals, "active") == 0 {
		return "warn"
	}
	return "ok"
}

func runtimeConfigStatus(view query.RuntimeConfigView) string {
	if view.SideEffect == "" {
		return "muted"
	}
	if len(view.Readiness.Blockers) > 0 {
		return "warn"
	}
	return "ok"
}

func runtimeConfigCardValue(view query.RuntimeConfigView) string {
	if view.Runtime.Address == "" {
		return "unknown"
	}
	if len(view.Readiness.Blockers) == 0 {
		return view.Runtime.Address
	}
	return fmt.Sprintf("%s (%d blockers)", view.Runtime.Address, len(view.Readiness.Blockers))
}

func emptyAsUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func maxCheckpointLag(diagnostics query.KnowledgeWorkerDiagnosticsView) int {
	maxLag := 0
	for _, worker := range diagnostics.Workers {
		for _, checkpoint := range worker.Checkpoints {
			latest, ok := intMetadata(checkpoint.Metadata, "latest_source_seq")
			if !ok {
				continue
			}
			lag := latest - checkpoint.Cursor
			if lag > maxLag {
				maxLag = lag
			}
		}
	}
	return maxLag
}

func intMetadata(items map[string]string, key string) (int, bool) {
	if items == nil {
		return 0, false
	}
	value, ok := items[key]
	if !ok {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
