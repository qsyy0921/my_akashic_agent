package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
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

	return query.QueueBackendView{
		Provider:                provider,
		Mode:                    mode,
		MigrationPhase:          queueMigrationPhase(provider, mode),
		ExternalQueueConfigured: externalConfigured,
		ExternalQueueActive:     false,
		StateStoreAuthoritative: true,
		LeaseOwner:              "go_state_store",
		ConsumerModel:           "goroutine_worker_pool",
		ConsumerConcurrency:     consumerConcurrency,
		MaxInFlight:             maxInFlight,
		OutboxQueueSource:       "outbox_state_store",
		AgentJobQueueSource:     "agent_job_state_store",
		DSNConfigured:           dsnConfigured,
		DSNRedacted:             redactQueueDSN(dsn),
		RecommendedFirstBackend: "nats_jetstream",
		SupportedProviders:      []string{"local", "nats_jetstream", "redis_streams", "rabbitmq"},
		Notes:                   notes,
	}, nil
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

func queueMigrationPhase(provider string, mode string) string {
	if provider == "local" {
		return "local_only"
	}
	switch mode {
	case "shadow_publish":
		return "shadow_ready"
	case "dual_read_compare":
		return "dual_read_design"
	case "external_lease":
		return "external_lease_design"
	default:
		return "local_only"
	}
}

func queueBackendNotes(provider string, mode string, dsnConfigured bool) []string {
	notes := []string{
		"outbox and generic agent-job leases still use Go state stores as the source of truth",
		"external queue adapters are not active in this runtime slice",
	}
	if provider == "local" {
		return append(notes, "local provider keeps all work scheduling on memory or JSON stores")
	}
	if !dsnConfigured {
		notes = append(notes, "external provider selected without AKASHIC_QUEUE_DSN; runtime remains local-only")
	} else {
		notes = append(notes, "AKASHIC_QUEUE_DSN is configured for migration diagnostics only")
	}
	if provider == "nats_jetstream" {
		notes = append(notes, "NATS JetStream is the recommended first external backend for event-subject routing and Go-native runtime infrastructure")
	}
	if mode == "external_lease" {
		notes = append(notes, "external lease ownership requires a future adapter before it can become active")
	}
	return notes
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
