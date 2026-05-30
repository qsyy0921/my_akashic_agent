package main

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func runtimeWorkerDiagnosticsFromEnv(queueBackend query.QueueBackendView) (query.RuntimeWorkerDiagnosticsView, error) {
	agentJobRecovery, agentJobRecoveryEnabled, err := agentJobLeaseRecoveryConfigFromEnv()
	if err != nil {
		return query.RuntimeWorkerDiagnosticsView{}, err
	}
	outboxDelivery, outboxDeliveryEnabled, err := outboxDeliveryWorkerConfigFromEnv()
	if err != nil {
		return query.RuntimeWorkerDiagnosticsView{}, err
	}

	workers := []query.RuntimeWorkerView{
		{
			Name:            "agent_job_recovery",
			Kind:            "agent_job_recovery",
			Enabled:         agentJobRecoveryEnabled,
			Running:         agentJobRecoveryEnabled,
			WorkerID:        "agent-runtime",
			IntervalSeconds: durationSeconds(agentJobRecovery.Interval),
			BatchSize:       agentJobRecovery.Limit,
			RunOnStart:      agentJobRecovery.RunOnStart,
			Notes:           []string{"recovers expired leased/running agent jobs; does not execute Python agent work"},
		},
		{
			Name:            "outbox_delivery_worker",
			Kind:            "outbox_delivery",
			Enabled:         outboxDeliveryEnabled,
			Running:         outboxDeliveryEnabled,
			WorkerID:        outboxDelivery.WorkerID,
			IntervalSeconds: durationSeconds(outboxDelivery.Interval),
			LeaseTTLSeconds: outboxDelivery.LeaseTTLSeconds,
			BatchSize:       outboxDelivery.BatchSize,
			RunOnStart:      outboxDelivery.RunOnStart,
			Attributes:      outboxDelivery.ChannelByAccount,
			Notes:           []string{"leases the Go outbox state store and dispatches through Go delivery adapters"},
		},
	}
	workers = append(workers, queueRuntimeWorkerViews(queueBackend)...)

	return query.RuntimeWorkerDiagnosticsView{
		Workers: workers,
		Notes:   []string{"read-only runtime diagnostics; use endpoint status plus logs for live failure details"},
	}, nil
}

func queueRuntimeWorkerViews(queueBackend query.QueueBackendView) []query.RuntimeWorkerView {
	shadowEnabled := queueBackend.ExternalQueueConfigured &&
		(queueBackend.Mode == "shadow_publish" || queueBackend.Mode == "dual_read_compare" || queueBackend.Mode == "external_lease")
	dualReadEnabled := queueBackend.ExternalQueueConfigured && queueBackend.Mode == "dual_read_compare"
	externalLeaseEnabled := queueBackend.ExternalLease != nil && queueBackend.ExternalLease.AllowExecution

	workers := []query.RuntimeWorkerView{
		{
			Name:       "nats_shadow_publisher",
			Kind:       "work_queue_publish",
			Enabled:    shadowEnabled,
			Running:    shadowEnabled && queueBackend.ExternalQueueActive,
			Mode:       queueBackend.Mode,
			WorkerID:   "agent-runtime",
			Attributes: queueWorkerAttributes(queueBackend),
			Notes:      []string{"publishes state-store work notifications to NATS JetStream while Go state remains authoritative"},
		},
		{
			Name:                "nats_dual_read_compare",
			Kind:                "work_queue_compare",
			Enabled:             dualReadEnabled,
			Running:             dualReadEnabled && queueBackend.ExternalQueueActive,
			Mode:                queueBackend.Mode,
			WorkerID:            "agent-runtime-dual-read-compare",
			ConsumerConcurrency: queueBackend.ConsumerConcurrency,
			MaxInFlight:         queueBackend.MaxInFlight,
			Attributes:          queueWorkerAttributes(queueBackend),
			Notes:               []string{"compares NATS notifications against authoritative Go state before queue cutover"},
		},
		{
			Name:                "nats_external_lease",
			Kind:                "work_queue_external_lease",
			Enabled:             externalLeaseEnabled,
			Running:             externalLeaseEnabled && queueBackend.ExternalQueueActive && queueBackend.LeaseOwner == "nats_jetstream",
			Mode:                queueBackend.Mode,
			ExecutionScope:      queueExecutionScope(queueBackend),
			WorkerID:            "agent-runtime-external-lease",
			ConsumerConcurrency: queueBackend.ConsumerConcurrency,
			MaxInFlight:         queueBackend.MaxInFlight,
			Attributes:          queueWorkerAttributes(queueBackend),
			Notes:               []string{"leases eligible work from NATS only after explicit cutover gates pass"},
		},
	}

	return workers
}

func queueExecutionScope(queueBackend query.QueueBackendView) string {
	if queueBackend.ExternalLease == nil {
		return ""
	}
	return queueBackend.ExternalLease.ExecutionScope
}

func queueWorkerAttributes(queueBackend query.QueueBackendView) map[string]string {
	attributes := map[string]string{
		"provider":        queueBackend.Provider,
		"stream":          queueBackend.Stream,
		"subject_prefix":  queueBackend.SubjectPrefix,
		"migration_phase": queueBackend.MigrationPhase,
	}
	if queueBackend.LeaseOwner != "" {
		attributes["lease_owner"] = queueBackend.LeaseOwner
	}
	return attributes
}

func durationSeconds(value time.Duration) int {
	if value <= 0 {
		return 0
	}
	return int(value / time.Second)
}
