package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OutboundCutoverPlanDeps struct {
	Readiness    outboundCutoverReadinessChecker
	QueueBackend outboundCutoverQueueBackendGetter
}

type outboundCutoverReadinessChecker interface {
	CheckOutboundCutoverReadiness(ctx context.Context, cmd command.CheckOutboundCutoverReadinessCommand) (query.OutboundCutoverReadinessView, error)
}

type OutboundCutoverPlanService struct {
	deps OutboundCutoverPlanDeps
}

func NewOutboundCutoverPlanService(deps OutboundCutoverPlanDeps) *OutboundCutoverPlanService {
	return &OutboundCutoverPlanService{deps: deps}
}

func (s *OutboundCutoverPlanService) PlanOutboundCutover(
	ctx context.Context,
	cmd command.PlanOutboundCutoverCommand,
) (query.OutboundCutoverPlanView, error) {
	if err := ctx.Err(); err != nil {
		return query.OutboundCutoverPlanView{}, err
	}
	if s == nil {
		return query.OutboundCutoverPlanView{}, errors.New("outbound cutover plan service is nil")
	}

	var blockers []string
	var readiness query.OutboundCutoverReadinessView
	if s.deps.Readiness == nil {
		blockers = append(blockers, "outbound_cutover_readiness_unavailable")
	} else {
		item, err := s.deps.Readiness.CheckOutboundCutoverReadiness(ctx, cmd.Readiness)
		if err != nil {
			return query.OutboundCutoverPlanView{}, err
		}
		readiness = item
		blockers = append(blockers, item.Blockers...)
	}

	var queueBackend query.QueueBackendView
	if s.deps.QueueBackend == nil {
		blockers = append(blockers, "queue_backend_unavailable")
	} else if item, err := s.deps.QueueBackend.Get(ctx); err != nil {
		blockers = append(blockers, "queue_backend_unavailable")
	} else {
		queueBackend = item
	}

	desired, desiredOK := normalizeOutboundCutoverExecutionOwner(cmd.DesiredExecutionOwner)
	recommended := recommendOutboundCutoverExecutionOwner(readiness, queueBackend)
	if desired == "auto" {
		desired = recommended
	}
	if !desiredOK {
		blockers = append(blockers, "invalid_desired_execution_owner")
		desired = recommended
	}

	current := strings.TrimSpace(readiness.ExecutionOwner)
	if current == "" {
		current = strings.TrimSpace(queueBackend.OutboxExecutionOwner)
	}

	switch desired {
	case "go_local_outbox_worker":
		if !readiness.LocalOutboxWorkerReady {
			blockers = append(blockers, "local_outbox_worker_not_ready")
		}
	case "nats_external_lease":
		if !readiness.ExternalLeaseOutboxReady {
			blockers = append(blockers, "external_lease_outbox_not_ready")
			if queueBackend.ExternalLease != nil {
				blockers = append(blockers, queueBackend.ExternalLease.Blockers...)
			}
		}
	default:
		blockers = append(blockers, "desired_execution_owner_not_supported")
	}

	blockers = sortedUniqueSmokeStrings(blockers)
	ready := readiness.Ready && desired != "" && desired == current && len(blockers) == 0
	decision := "blocked"
	if ready {
		decision = "ready_to_cutover"
	} else if readiness.Ready && desired != current && len(blockers) == 0 {
		decision = "ready_on_different_execution_owner"
	}

	return query.OutboundCutoverPlanView{
		Ready:                     ready,
		Decision:                  decision,
		DesiredExecutionOwner:     desired,
		RecommendedExecutionOwner: recommended,
		CurrentExecutionOwner:     current,
		Readiness:                 readiness,
		RequiredChecks:            outboundCutoverRequiredChecks(desired),
		EnableSteps:               outboundCutoverEnableSteps(desired, readiness, queueBackend),
		VerificationSteps:         outboundCutoverVerificationSteps(desired),
		RollbackSteps:             outboundCutoverRollbackSteps(desired),
		Blockers:                  blockers,
		Attributes: map[string]string{
			"planned_by":  "agent_runtime_outbound_cutover_plan",
			"side_effect": "none",
		},
		Notes: []string{
			"read-only cutover plan; no environment variables are changed and no platform messages are sent",
			"Python remains responsible for AI reasoning and content creation; Go only owns deterministic delivery infrastructure after explicit operator cutover",
		},
		SideEffect: "none",
	}, nil
}

func normalizeOutboundCutoverExecutionOwner(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "auto", true
	}
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "auto":
		return "auto", true
	case "local", "go_local", "local_outbox_worker", "go_local_outbox_worker":
		return "go_local_outbox_worker", true
	case "nats", "nats_external", "external_lease", "nats_external_lease":
		return "nats_external_lease", true
	default:
		return value, false
	}
}

func recommendOutboundCutoverExecutionOwner(readiness query.OutboundCutoverReadinessView, queueBackend query.QueueBackendView) string {
	if readiness.ExternalLeaseOutboxReady {
		return "nats_external_lease"
	}
	if readiness.LocalOutboxWorkerReady {
		return "go_local_outbox_worker"
	}
	if queueBackend.Provider == "nats_jetstream" || queueBackend.Mode == "external_lease" || queueBackend.DSNConfigured {
		return "nats_external_lease"
	}
	return "go_local_outbox_worker"
}

func outboundCutoverRequiredChecks(desired string) []query.OutboundCutoverPlanStep {
	steps := []query.OutboundCutoverPlanStep{
		{
			Phase:    "precheck",
			Action:   "check_outbound_cutover_readiness",
			Method:   "POST",
			Endpoint: "/v1/outbound-cutover/readiness",
			Detail:   "must be ready before enabling Go-owned platform delivery",
		},
		{
			Phase:    "precheck",
			Action:   "check_delivery_adapter_health",
			Method:   "GET",
			Endpoint: "/v1/delivery-adapters/health",
			Detail:   "OneBot/NapCat channels must be reachable before live send smoke",
		},
		{
			Phase:  "precheck",
			Action: "run_manual_live_send_smoke",
			Detail: "operator should verify QQ private text, group text, image and file sends on the intended accounts before broad cutover",
		},
	}
	if desired == "nats_external_lease" {
		steps = append(steps, query.OutboundCutoverPlanStep{
			Phase:    "precheck",
			Action:   "check_queue_external_lease_gate",
			Method:   "GET",
			Endpoint: "/v1/queue-backend",
			Detail:   "external_lease.allow_execution must be true and include outbox_delivery",
		})
	}
	return indexedOutboundCutoverSteps(steps)
}

func outboundCutoverEnableSteps(
	desired string,
	readiness query.OutboundCutoverReadinessView,
	queueBackend query.QueueBackendView,
) []query.OutboundCutoverPlanStep {
	switch desired {
	case "nats_external_lease":
		return indexedOutboundCutoverSteps([]query.OutboundCutoverPlanStep{
			{
				Phase:  "enable",
				Action: "configure_nats_external_lease",
				Env: map[string]string{
					"AKASHIC_QUEUE_BACKEND":                           "nats_jetstream",
					"AKASHIC_QUEUE_MODE":                              "external_lease",
					"AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER":            "true",
					"AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED":            "true",
					"AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED":      "true",
					"AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED":          "false",
					"AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT":             channelByAccountHint(readiness),
					"AKASHIC_QUEUE_CONSUMER_CONCURRENCY":              concurrencyHint(queueBackend),
					"AKASHIC_QUEUE_MAX_IN_FLIGHT":                     maxInFlightHint(queueBackend),
					"AKASHIC_QUEUE_EXTERNAL_LEASE_NACK_DELAY_SECONDS": "30",
				},
				Detail: "restart agent-runtime after setting these flags; AKASHIC_QUEUE_DSN/stream/subject_prefix must already point at the live NATS deployment",
			},
			{
				Phase:    "enable",
				Action:   "confirm_nats_external_lease_worker_running",
				Method:   "GET",
				Endpoint: "/v1/runtime-workers",
				Detail:   "nats_external_lease should be enabled and running; local outbox worker must remain disabled",
			},
		})
	case "go_local_outbox_worker":
		return indexedOutboundCutoverSteps([]query.OutboundCutoverPlanStep{
			{
				Phase:  "enable",
				Action: "enable_go_local_outbox_worker",
				Env: map[string]string{
					"AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED":    "true",
					"AKASHIC_OUTBOX_DELIVERY_WORKER_BATCH_SIZE": "1",
					"AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT":       channelByAccountHint(readiness),
				},
				Detail: "restart agent-runtime after enabling; keep NATS external lease cutover disabled to avoid two execution owners",
			},
			{
				Phase:    "enable",
				Action:   "confirm_local_outbox_worker_running",
				Method:   "GET",
				Endpoint: "/v1/runtime-workers",
				Detail:   "outbox_delivery_worker should be enabled and running",
			},
		})
	default:
		return nil
	}
}

func outboundCutoverVerificationSteps(desired string) []query.OutboundCutoverPlanStep {
	steps := []query.OutboundCutoverPlanStep{
		{Phase: "verify", Action: "read_runtime_config", Method: "GET", Endpoint: "/v1/runtime-config"},
		{Phase: "verify", Action: "read_queue_backend", Method: "GET", Endpoint: "/v1/queue-backend"},
		{Phase: "verify", Action: "read_runtime_workers", Method: "GET", Endpoint: "/v1/runtime-workers"},
		{Phase: "verify", Action: "recheck_cutover_readiness", Method: "POST", Endpoint: "/v1/outbound-cutover/readiness"},
		{Phase: "verify", Action: "watch_outbox_metrics", Method: "GET", Endpoint: "/v1/outbox-metrics"},
		{Phase: "verify", Action: "watch_send_ledger", Method: "GET", Endpoint: "/v1/send-ledger/recent"},
	}
	if desired == "nats_external_lease" {
		steps = append(steps, query.OutboundCutoverPlanStep{
			Phase:    "verify",
			Action:   "watch_external_lease_diagnostics",
			Method:   "GET",
			Endpoint: "/v1/queue-backend",
			Detail:   "external_lease.diagnostics should show ack/nack/term counts for outbox_delivery work",
		})
	}
	return indexedOutboundCutoverSteps(steps)
}

func outboundCutoverRollbackSteps(desired string) []query.OutboundCutoverPlanStep {
	switch desired {
	case "nats_external_lease":
		return indexedOutboundCutoverSteps([]query.OutboundCutoverPlanStep{
			{
				Phase:  "rollback",
				Action: "disable_external_lease_cutover",
				Env: map[string]string{
					"AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER": "false",
					"AKASHIC_QUEUE_MODE":                   "dual_read_compare",
				},
				Detail: "restart agent-runtime; state store remains authoritative and queued outbox records stay recoverable",
			},
			{
				Phase:    "rollback",
				Action:   "confirm_execution_owner_reverted",
				Method:   "GET",
				Endpoint: "/v1/queue-backend",
				Detail:   "outbox_execution_owner should no longer be nats_external_lease",
			},
		})
	case "go_local_outbox_worker":
		return indexedOutboundCutoverSteps([]query.OutboundCutoverPlanStep{
			{
				Phase:  "rollback",
				Action: "disable_local_outbox_worker",
				Env: map[string]string{
					"AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED": "false",
				},
				Detail: "restart agent-runtime; Python compatibility sending can continue while Go outbox remains a state store",
			},
			{
				Phase:    "rollback",
				Action:   "confirm_worker_disabled",
				Method:   "GET",
				Endpoint: "/v1/runtime-workers",
				Detail:   "outbox_delivery_worker should be disabled and not running",
			},
		})
	default:
		return nil
	}
}

func indexedOutboundCutoverSteps(steps []query.OutboundCutoverPlanStep) []query.OutboundCutoverPlanStep {
	for index := range steps {
		steps[index].StepIndex = index + 1
	}
	return steps
}

func channelByAccountHint(readiness query.OutboundCutoverReadinessView) string {
	if len(readiness.ExpectedOneBotChannels) == 0 {
		return "1049511700=qq_1049511700,2365524513=qq_2365524513"
	}
	pairs := make([]string, 0, len(readiness.ExpectedOneBotChannels))
	for _, channel := range readiness.ExpectedOneBotChannels {
		accountID, ok := strings.CutPrefix(strings.TrimSpace(channel), "qq_")
		if !ok || accountID == "" {
			continue
		}
		pairs = append(pairs, accountID+"="+channel)
	}
	if len(pairs) == 0 {
		return "1049511700=qq_1049511700,2365524513=qq_2365524513"
	}
	return strings.Join(pairs, ",")
}

func concurrencyHint(queueBackend query.QueueBackendView) string {
	if queueBackend.ConsumerConcurrency > 0 {
		return intToString(queueBackend.ConsumerConcurrency)
	}
	return "4"
}

func maxInFlightHint(queueBackend query.QueueBackendView) string {
	if queueBackend.MaxInFlight > 0 {
		return intToString(queueBackend.MaxInFlight)
	}
	return "16"
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
