package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OutboundCutoverReadinessDeps struct {
	RuntimeConfig  outboundCutoverRuntimeConfigGetter
	DeliverySmoke  outboundCutoverDeliverySmokeChecker
	QueueBackend   outboundCutoverQueueBackendGetter
	RuntimeWorkers outboundCutoverRuntimeWorkerDiagnosticsGetter
}

type outboundCutoverRuntimeConfigGetter interface {
	GetRuntimeConfig(ctx context.Context) (query.RuntimeConfigView, error)
}

type outboundCutoverDeliverySmokeChecker interface {
	CheckDeliverySmokeReadiness(ctx context.Context, cmd command.CheckDeliverySmokeReadinessCommand) (query.DeliverySmokeReadinessView, error)
}

type outboundCutoverQueueBackendGetter interface {
	Get(ctx context.Context) (query.QueueBackendView, error)
}

type outboundCutoverRuntimeWorkerDiagnosticsGetter interface {
	GetRuntimeWorkers(ctx context.Context) (query.RuntimeWorkerDiagnosticsView, error)
}

type OutboundCutoverReadinessService struct {
	deps OutboundCutoverReadinessDeps
}

func NewOutboundCutoverReadinessService(deps OutboundCutoverReadinessDeps) *OutboundCutoverReadinessService {
	return &OutboundCutoverReadinessService{deps: deps}
}

func (s *OutboundCutoverReadinessService) CheckOutboundCutoverReadiness(
	ctx context.Context,
	cmd command.CheckOutboundCutoverReadinessCommand,
) (query.OutboundCutoverReadinessView, error) {
	if err := ctx.Err(); err != nil {
		return query.OutboundCutoverReadinessView{}, err
	}
	if s == nil {
		return query.OutboundCutoverReadinessView{}, errors.New("outbound cutover readiness service is nil")
	}

	var (
		runtimeConfig query.RuntimeConfigView
		queueBackend  query.QueueBackendView
		runtimeWorker query.RuntimeWorkerDiagnosticsView
		smoke         query.DeliverySmokeReadinessView
		blockers      []string
	)

	if s.deps.RuntimeConfig == nil {
		blockers = append(blockers, "runtime_config_unavailable")
	} else if item, err := s.deps.RuntimeConfig.GetRuntimeConfig(ctx); err != nil {
		blockers = append(blockers, "runtime_config_unavailable")
	} else {
		runtimeConfig = item
	}

	if s.deps.QueueBackend == nil {
		blockers = append(blockers, "queue_backend_unavailable")
	} else if item, err := s.deps.QueueBackend.Get(ctx); err != nil {
		blockers = append(blockers, "queue_backend_unavailable")
	} else {
		queueBackend = item
	}

	if s.deps.RuntimeWorkers == nil {
		blockers = append(blockers, "runtime_workers_unavailable")
	} else if item, err := s.deps.RuntimeWorkers.GetRuntimeWorkers(ctx); err != nil {
		blockers = append(blockers, "runtime_workers_unavailable")
	} else {
		runtimeWorker = item
	}

	if s.deps.DeliverySmoke == nil {
		blockers = append(blockers, "delivery_smoke_unavailable")
	} else {
		item, err := s.deps.DeliverySmoke.CheckDeliverySmokeReadiness(ctx, cmd.Smoke)
		if err != nil {
			return query.OutboundCutoverReadinessView{}, err
		}
		smoke = item
	}

	onebotReady := runtimeConfig.Readiness.OneBotConfigured && runtimeConfig.Readiness.OneBotExpectedChannelsOK
	if !onebotReady {
		if !runtimeConfig.Readiness.OneBotConfigured {
			blockers = append(blockers, "onebot_not_configured")
		}
		if !runtimeConfig.Readiness.OneBotExpectedChannelsOK {
			blockers = append(blockers, "onebot_expected_channels_missing")
		}
	}

	smokeReady := smoke.Ready && smoke.SideEffect != ""
	if !smokeReady {
		blockers = append(blockers, "delivery_smoke_not_ready")
		blockers = append(blockers, smoke.Blockers...)
	}

	localOutboxReady := outboundCutoverLocalOutboxReady(queueBackend, runtimeWorker)
	externalLeaseReady := outboundCutoverExternalLeaseOutboxReady(queueBackend)
	executionReady := localOutboxReady || externalLeaseReady
	if !executionReady {
		blockers = append(blockers, "outbox_execution_path_not_ready")
	}

	blockers = sortedUniqueSmokeStrings(blockers)
	ready := onebotReady && smokeReady && executionReady && len(blockers) == 0
	reason := "outbound_cutover_ready"
	if !ready {
		reason = "outbound_cutover_not_ready"
	}

	return query.OutboundCutoverReadinessView{
		Ready:                    ready,
		Reason:                   reason,
		OneBotReady:              onebotReady,
		SmokeReady:               smokeReady,
		ExecutionReady:           executionReady,
		ExecutionOwner:           strings.TrimSpace(queueBackend.OutboxExecutionOwner),
		LocalOutboxWorkerReady:   localOutboxReady,
		ExternalLeaseOutboxReady: externalLeaseReady,
		QueueProvider:            queueBackend.Provider,
		QueueMode:                queueBackend.Mode,
		ExternalLeaseScope:       outboundCutoverExternalLeaseScope(queueBackend),
		ExpectedOneBotChannels:   append([]string(nil), runtimeConfig.Delivery.OneBotExpectedChannels...),
		MissingOneBotChannels:    append([]string(nil), runtimeConfig.Delivery.OneBotMissingChannels...),
		RuntimeConfigReadiness:   runtimeConfig.Readiness,
		DeliverySmokeReadiness:   smoke,
		Blockers:                 blockers,
		Attributes: map[string]string{
			"checked_by":  "agent_runtime_outbound_cutover_readiness",
			"side_effect": "none",
		},
		Notes: []string{
			"read-only outbound cutover preflight; no platform messages are sent",
			"ready requires OneBot config, delivery smoke adapter support, and a running Go outbox execution path",
		},
		SideEffect: "none",
	}, nil
}

func outboundCutoverLocalOutboxReady(queueBackend query.QueueBackendView, runtimeWorkers query.RuntimeWorkerDiagnosticsView) bool {
	if strings.TrimSpace(queueBackend.OutboxExecutionOwner) != "go_local_outbox_worker" {
		return false
	}
	for _, worker := range runtimeWorkers.Workers {
		if worker.Name == "outbox_delivery_worker" {
			return worker.Enabled && worker.Running
		}
	}
	return false
}

func outboundCutoverExternalLeaseOutboxReady(queueBackend query.QueueBackendView) bool {
	if strings.TrimSpace(queueBackend.OutboxExecutionOwner) != "nats_external_lease" {
		return false
	}
	if queueBackend.ExternalLease == nil || !queueBackend.ExternalLease.AllowExecution || !queueBackend.ExternalQueueActive || queueBackend.LeaseOwner != "nats_jetstream" {
		return false
	}
	for _, kind := range queueBackend.ExternalLease.AllowedWorkKinds {
		if strings.TrimSpace(kind) == "outbox_delivery" {
			return true
		}
	}
	return false
}

func outboundCutoverExternalLeaseScope(queueBackend query.QueueBackendView) string {
	if queueBackend.ExternalLease == nil {
		return ""
	}
	return queueBackend.ExternalLease.ExecutionScope
}
