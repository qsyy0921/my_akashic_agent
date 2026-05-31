package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/natsqueue"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/queuediagnostics"
)

func queueBackendViewFromEnv() (query.QueueBackendView, error) {
	provider, err := normalizeQueueProvider(os.Getenv("AKASHIC_QUEUE_BACKEND"))
	if err != nil {
		return query.QueueBackendView{}, err
	}
	mode, err := normalizeQueueMode(os.Getenv("AKASHIC_QUEUE_MODE"), provider)
	if err != nil {
		return query.QueueBackendView{}, err
	}
	consumerConcurrency, err := positiveIntEnv("AKASHIC_QUEUE_CONSUMER_CONCURRENCY", 4, 128)
	if err != nil {
		return query.QueueBackendView{}, err
	}
	maxInFlight, err := positiveIntEnv("AKASHIC_QUEUE_MAX_IN_FLIGHT", consumerConcurrency*4, 4096)
	if err != nil {
		return query.QueueBackendView{}, err
	}
	if maxInFlight < consumerConcurrency {
		return query.QueueBackendView{}, fmt.Errorf("AKASHIC_QUEUE_MAX_IN_FLIGHT must be >= AKASHIC_QUEUE_CONSUMER_CONCURRENCY")
	}
	dsn := strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_DSN"))
	dsnConfigured := dsn != ""
	externalConfigured := provider != "local" && dsnConfigured
	notes := queueBackendNotes(provider, mode, dsnConfigured)
	stream := strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_STREAM"))
	if stream == "" {
		stream = "AKASHIC_WORK"
	}
	subjectPrefix := strings.Trim(strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_SUBJECT_PREFIX")), ".")
	if subjectPrefix == "" {
		subjectPrefix = "akashic.work"
	}
	externalLease := queueExternalLeaseGate(provider, mode, dsnConfigured)
	agentJobQueueSource := "agent_job_state_store"
	if externalLeaseAllowsAgentJobs(externalLease) {
		agentJobQueueSource = "agent_job_state_store_with_nats_result_ack"
	}
	outboxOwner := queueOutboxExecutionOwner(externalLease)
	agentJobOwner := queueAgentJobExecutionOwner(externalLease)
	providerCapabilities := queueProviderCapabilities(provider)

	return query.QueueBackendView{
		Provider:                   provider,
		Mode:                       mode,
		MigrationPhase:             queueMigrationPhase(provider, mode),
		Stream:                     stream,
		SubjectPrefix:              subjectPrefix,
		ExternalQueueConfigured:    externalConfigured,
		ExternalQueueActive:        false,
		StateStoreAuthoritative:    true,
		LeaseOwner:                 "go_state_store",
		ConsumerModel:              "goroutine_worker_pool",
		ConsumerConcurrency:        consumerConcurrency,
		MaxInFlight:                maxInFlight,
		OutboxQueueSource:          "outbox_state_store",
		AgentJobQueueSource:        agentJobQueueSource,
		OutboxExecutionOwner:       outboxOwner,
		AgentJobExecutionOwner:     agentJobOwner,
		DSNConfigured:              dsnConfigured,
		DSNRedacted:                redactQueueDSN(dsn),
		RecommendedFirstBackend:    "nats_jetstream",
		SupportedProviders:         []string{"local", "nats_jetstream", "redis_streams", "rabbitmq"},
		ProviderCapabilities:       providerCapabilities,
		SelectedProviderCapability: selectedQueueProviderCapability(provider, providerCapabilities),
		Notes:                      notes,
		ExternalLease:              externalLease,
	}, nil
}

func queueOutboxExecutionOwner(externalLease *query.QueueExternalLeaseGate) string {
	if externalLease != nil && externalLease.AllowExecution {
		return "nats_external_lease"
	}
	if boolEnv("AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED") {
		return "go_local_outbox_worker"
	}
	return "go_state_store_api"
}

func queueAgentJobExecutionOwner(externalLease *query.QueueExternalLeaseGate) string {
	if externalLeaseAllowsAgentJobs(externalLease) {
		return "python_ai_worker_with_nats_result_ack"
	}
	return "python_ai_worker_state_store_lease"
}

func queueProviderCapabilities(selectedProvider string) []query.QueueProviderCapabilityView {
	items := []query.QueueProviderCapabilityView{
		{
			Provider:                "local",
			Label:                   "Local Go state store",
			Status:                  "available",
			Implemented:             true,
			SupportsStateStoreLease: true,
			ConsumerModel:           "state_store_lease",
			AdapterBoundary:         "in_process_state_store",
			Notes:                   []string{"default safe mode; no external MQ dependency", "state store remains authoritative during every migration phase"},
		},
		{
			Provider:                    "nats_jetstream",
			Label:                       "NATS JetStream",
			Status:                      "available",
			Recommended:                 true,
			RecommendedPhase:            "first_external_mq",
			Implemented:                 true,
			SupportsExternalQueue:       true,
			SupportsShadowPublish:       true,
			SupportsDualReadCompare:     true,
			SupportsExternalLease:       true,
			SupportsAgentJobResultAck:   true,
			SupportsConcurrentConsumers: true,
			SupportsDelayedNack:         true,
			ConsumerModel:               "goroutine_worker_pool",
			AdapterBoundary:             "infrastructure/natsqueue",
			Notes:                       []string{"current recommended external MQ for staged cutover", "external lease remains gate-protected by runtime readiness checks"},
		},
		{
			Provider:                    "redis_streams",
			Label:                       "Redis Streams",
			Status:                      "planned",
			RecommendedPhase:            "future_adapter",
			SupportsExternalQueue:       true,
			SupportsConcurrentConsumers: true,
			ConsumerModel:               "consumer_group",
			AdapterBoundary:             "future infrastructure adapter behind WorkQueue ports",
			Notes:                       []string{"kept as a replaceable provider boundary", "not implemented for external lease in this runtime yet"},
			Blockers:                    []string{"adapter_not_implemented", "external_lease_smoke_missing"},
		},
		{
			Provider:                    "rabbitmq",
			Label:                       "RabbitMQ",
			Status:                      "planned",
			RecommendedPhase:            "future_adapter",
			SupportsExternalQueue:       true,
			SupportsConcurrentConsumers: true,
			SupportsDelayedNack:         true,
			ConsumerModel:               "competing_consumers",
			AdapterBoundary:             "future infrastructure adapter behind WorkQueue ports",
			Notes:                       []string{"kept as a replaceable provider boundary", "not implemented for external lease in this runtime yet"},
			Blockers:                    []string{"adapter_not_implemented", "external_lease_smoke_missing"},
		},
	}
	for index := range items {
		if items[index].Provider == selectedProvider {
			items[index].Status = "selected"
		}
	}
	return items
}

func selectedQueueProviderCapability(provider string, items []query.QueueProviderCapabilityView) *query.QueueProviderCapabilityView {
	for _, item := range items {
		if item.Provider == provider {
			selected := item
			return &selected
		}
	}
	return nil
}

func normalizeQueueProvider(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = "local"
	}
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "local", "memory", "json", "jsonl", "state_store":
		return "local", nil
	case "nats", "nats_js", "nats_jetstream", "jetstream":
		return "nats_jetstream", nil
	case "redis", "redis_stream", "redis_streams":
		return "redis_streams", nil
	case "rabbit", "rabbitmq", "amqp":
		return "rabbitmq", nil
	default:
		return "", fmt.Errorf("unsupported AKASHIC_QUEUE_BACKEND %q", raw)
	}
}

func normalizeQueueMode(raw string, provider string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		if provider == "local" {
			return "local_state_store", nil
		}
		return "shadow_publish", nil
	}
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "local", "local_state_store", "state_store":
		return "local_state_store", nil
	case "shadow", "shadow_publish", "shadow_only":
		return "shadow_publish", nil
	case "dual", "dual_read", "dual_read_compare":
		return "dual_read_compare", nil
	case "external", "external_lease", "external_lease_owner":
		return "external_lease", nil
	default:
		return "", fmt.Errorf("unsupported AKASHIC_QUEUE_MODE %q", raw)
	}
}

func positiveIntEnv(key string, fallback int, maxValue int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	if maxValue > 0 && parsed > maxValue {
		return 0, fmt.Errorf("%s must be <= %d", key, maxValue)
	}
	return parsed, nil
}

func nonNegativeIntEnv(key string, fallback int, maxValue int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	if maxValue > 0 && parsed > maxValue {
		return 0, fmt.Errorf("%s must be <= %d", key, maxValue)
	}
	return parsed, nil
}

func queueMigrationPhase(provider string, mode string) string {
	if provider == "local" {
		return "local_only"
	}
	switch mode {
	case "shadow_publish":
		return "shadow_ready"
	case "dual_read_compare":
		return "dual_read_compare"
	case "external_lease":
		return "external_lease_gate"
	default:
		return "local_only"
	}
}

func queueBackendNotes(provider string, mode string, dsnConfigured bool) []string {
	notes := []string{
		"outbox and generic agent-job leases still use Go state stores as the source of truth",
		"external queue adapters activate only when provider, mode, and DSN match a concrete implementation",
	}
	if provider == "local" {
		return append(notes, "local provider keeps all work scheduling on memory or JSON stores")
	}
	if !dsnConfigured {
		notes = append(notes, "external provider selected without AKASHIC_QUEUE_DSN; runtime remains local-only")
	} else {
		notes = append(notes, "AKASHIC_QUEUE_DSN is configured for staged queue migration")
	}
	if provider == "nats_jetstream" {
		notes = append(notes, "NATS JetStream is the recommended first external backend for event-subject routing and Go-native runtime infrastructure")
	}
	if mode == "external_lease" {
		notes = append(notes, "external lease ownership is guarded by explicit cutover checks and remains blocked until every gate passes")
	}
	return notes
}

func queueExternalLeaseGate(provider string, mode string, dsnConfigured bool) *query.QueueExternalLeaseGate {
	if mode != "external_lease" {
		return nil
	}
	cutoverRequested := boolEnv("AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER")
	dualReadSmokePassed := boolEnv("AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED")
	stateLeaseWorkersDisabled := boolEnv("AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED")
	localOutboxWorkerDisabled := !boolEnv("AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED")
	agentJobRequested := boolEnv("AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED")
	agentJobDuplicateSmokePassed := boolEnv("AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED")
	agentJobFlowSmokePassed := boolEnv("AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED")
	strictAgentJobLeaseToken := boolEnv("AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN")
	executorImplemented := true
	gate := &query.QueueExternalLeaseGate{
		Enabled:          true,
		CutoverRequested: cutoverRequested,
		GateState:        "blocked",
		ExecutionScope:   "none",
		AckPolicy:        envOrDefault("AKASHIC_QUEUE_EXTERNAL_LEASE_ACK_POLICY", "ack_after_go_state_terminal"),
		NackPolicy:       envOrDefault("AKASHIC_QUEUE_EXTERNAL_LEASE_NACK_POLICY", "nack_when_go_lease_rejected"),
		RetryPolicy:      envOrDefault("AKASHIC_QUEUE_EXTERNAL_LEASE_RETRY_POLICY", "go_domain_retry_then_dead_letter"),
		DeadLetterPolicy: envOrDefault("AKASHIC_QUEUE_EXTERNAL_LEASE_DEAD_LETTER_POLICY", "go_domain_dead_letter_is_final"),
		RollbackPolicy:   envOrDefault("AKASHIC_QUEUE_EXTERNAL_LEASE_ROLLBACK_POLICY", "state_store_recovery_and_queue_replay"),
		Notes: []string{
			"external lease is a guarded cutover state, not enabled by AKASHIC_QUEUE_MODE alone",
			"state store remains authoritative and validates each queue candidate before side effects",
		},
	}
	addExternalLeaseCheck(gate, "provider_supported", provider == "nats_jetstream", "first external lease target is NATS JetStream")
	addExternalLeaseCheck(gate, "dsn_configured", dsnConfigured, "AKASHIC_QUEUE_DSN must point at the external queue")
	addExternalLeaseCheck(gate, "explicit_cutover", cutoverRequested, "set AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER=true after smoke tests")
	addExternalLeaseCheck(gate, "dual_read_smoke_passed", dualReadSmokePassed, "set AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED=true after local NATS smoke")
	addExternalLeaseCheck(gate, "state_lease_workers_disabled", stateLeaseWorkersDisabled, "set AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED=true after stopping legacy state-store lease workers")
	addExternalLeaseCheck(gate, "local_outbox_worker_disabled", localOutboxWorkerDisabled, "unset AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED or set it false before external lease cutover")
	addExternalLeaseCheck(gate, "executor_implemented", executorImplemented, "outbox delivery external lease executor is implemented in agent-runtime")
	if len(gate.Blockers) == 0 {
		gate.AllowExecution = true
		gate.GateState = "ready"
		gate.ExecutionScope = "outbox_delivery_only"
		gate.AllowedWorkKinds = []string{"outbox_delivery"}
	}
	if gate.AllowExecution && agentJobRequested && agentJobDuplicateSmokePassed && agentJobFlowSmokePassed && strictAgentJobLeaseToken {
		gate.ExecutionScope = "outbox_delivery_and_agent_job_result_ack"
		gate.AllowedWorkKinds = append(gate.AllowedWorkKinds, "agent_job")
		gate.Notes = append(gate.Notes, "agent_job NATS subjects are enabled only for result acknowledgement; Python still executes model/RAG/memory work")
	} else {
		required := "set AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED=true, AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED=true, AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED=true, and AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN=true before moving agent_job subjects to external lease"
		if !gate.AllowExecution {
			required = "pass the base external-lease gates before enabling agent_job subjects"
		}
		gate.BlockedWorkKinds = append(gate.BlockedWorkKinds, query.QueueExternalLeaseBlock{
			WorkKind:       "agent_job",
			Reason:         "agent jobs are executed by Python workers, so NATS ack must be tied to Python result writeback rather than Go dispatch completion",
			RequiredChange: required,
		})
	}
	return gate
}

func externalLeaseAllowsAgentJobs(gate *query.QueueExternalLeaseGate) bool {
	if gate == nil || !gate.AllowExecution {
		return false
	}
	for _, kind := range gate.AllowedWorkKinds {
		if kind == "agent_job" {
			return true
		}
	}
	return false
}

func addExternalLeaseCheck(gate *query.QueueExternalLeaseGate, name string, passed bool, detail string) {
	status := "passed"
	if !passed {
		status = "blocked"
		gate.Blockers = append(gate.Blockers, name)
	}
	gate.RequiredChecks = append(gate.RequiredChecks, query.QueueExternalLeaseCheck{
		Name:   name,
		Status: status,
		Detail: detail,
	})
}

func boolEnv(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	default:
		return false
	}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func redactQueueDSN(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Scheme != "" {
		if parsed.User != nil {
			username := parsed.User.Username()
			if username == "" {
				parsed.User = url.UserPassword("redacted", "redacted")
			} else {
				parsed.User = url.UserPassword(username, "redacted")
			}
		}
		if parsed.RawQuery != "" {
			parsed.RawQuery = "redacted=true"
		}
		return parsed.String()
	}
	if len(raw) <= 12 {
		return "redacted"
	}
	return raw[:6] + "..." + raw[len(raw)-4:]
}

func newWorkQueuePublisher(view query.QueueBackendView) (outport.WorkQueuePublisher, func(), error) {
	if view.Provider != "nats_jetstream" || !queueModePublishesWork(view.Mode) || !view.DSNConfigured {
		return nil, nil, nil
	}
	timeoutSeconds, err := positiveIntEnv("AKASHIC_QUEUE_TIMEOUT_SECONDS", 3, 60)
	if err != nil {
		return nil, nil, err
	}
	publisher, err := natsqueue.NewPublisher(natsqueue.Config{
		URL:           strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_DSN")),
		Stream:        view.Stream,
		SubjectPrefix: view.SubjectPrefix,
		Timeout:       time.Duration(timeoutSeconds) * time.Second,
	})
	if err != nil {
		return nil, nil, err
	}
	recorder, err := queuediagnostics.NewRecorder(publisher, queuediagnostics.SubjectResolver{
		OutboxDelivery: func(delivery model.OutboxDelivery) string {
			return natsqueue.OutboxSubject(view.SubjectPrefix, delivery)
		},
		AgentJob: func(job model.AgentJob) string {
			return natsqueue.AgentJobSubject(view.SubjectPrefix, job)
		},
	})
	if err != nil {
		publisher.Close()
		return nil, nil, err
	}
	return recorder, publisher.Close, nil
}

func newWorkQueueCompareConsumer(view query.QueueBackendView) (*natsqueue.CompareConsumer, func(), error) {
	if view.Provider != "nats_jetstream" || view.Mode != "dual_read_compare" || !view.DSNConfigured {
		return nil, nil, nil
	}
	timeoutSeconds, err := positiveIntEnv("AKASHIC_QUEUE_TIMEOUT_SECONDS", 3, 60)
	if err != nil {
		return nil, nil, err
	}
	consumer, err := natsqueue.NewCompareConsumer(natsqueue.CompareConsumerConfig{
		URL:                 strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_DSN")),
		Stream:              view.Stream,
		SubjectPrefix:       view.SubjectPrefix,
		Durable:             strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_DUAL_READ_DURABLE")),
		Timeout:             time.Duration(timeoutSeconds) * time.Second,
		ConsumerConcurrency: view.ConsumerConcurrency,
		MaxInFlight:         view.MaxInFlight,
	})
	if err != nil {
		return nil, nil, err
	}
	return consumer, consumer.Close, nil
}

func newWorkQueueExternalLeaseConsumer(view query.QueueBackendView) (*natsqueue.ExternalLeaseConsumer, func(), error) {
	if view.Provider != "nats_jetstream" || view.Mode != "external_lease" || !view.DSNConfigured || view.ExternalLease == nil || !view.ExternalLease.AllowExecution {
		return nil, nil, nil
	}
	timeoutSeconds, err := positiveIntEnv("AKASHIC_QUEUE_TIMEOUT_SECONDS", 3, 60)
	if err != nil {
		return nil, nil, err
	}
	ttlSeconds, err := positiveIntEnv("AKASHIC_QUEUE_EXTERNAL_LEASE_TTL_SECONDS", 300, 86400)
	if err != nil {
		return nil, nil, err
	}
	nackDelaySeconds, err := positiveIntEnv("AKASHIC_QUEUE_EXTERNAL_LEASE_NACK_DELAY_SECONDS", 30, 86400)
	if err != nil {
		return nil, nil, err
	}
	consumer, err := natsqueue.NewExternalLeaseConsumer(natsqueue.ExternalLeaseConsumerConfig{
		URL:                 strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_DSN")),
		Stream:              view.Stream,
		SubjectPrefix:       view.SubjectPrefix,
		Durable:             strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_EXTERNAL_LEASE_DURABLE")),
		WorkerID:            strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_EXTERNAL_LEASE_WORKER_ID")),
		LeaseTTLSeconds:     ttlSeconds,
		NackDelay:           time.Duration(nackDelaySeconds) * time.Second,
		Timeout:             time.Duration(timeoutSeconds) * time.Second,
		ConsumerConcurrency: view.ConsumerConcurrency,
		MaxInFlight:         view.MaxInFlight,
		ChannelByAccount:    keyValueCSVEnv("AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT"),
		IncludeAgentJobs:    externalLeaseAllowsAgentJobs(view.ExternalLease),
	})
	if err != nil {
		return nil, nil, err
	}
	return consumer, consumer.Close, nil
}

func queueModePublishesWork(mode string) bool {
	return mode == "shadow_publish" || mode == "dual_read_compare" || mode == "external_lease"
}

func positiveIntEnvOrDefault(key string, fallback int, maxValue int) int {
	value, err := positiveIntEnv(key, fallback, maxValue)
	if err != nil {
		return fallback
	}
	return value
}
