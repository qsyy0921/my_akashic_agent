package main

import (
	"os"
	"sort"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/onebotdelivery"
)

func runtimeConfigFromEnv(addr string, addrSource string, botIDs []string) query.RuntimeConfigView {
	onebotEndpoints := onebotEndpointsFromEnv()
	expectedOneBotChannels := expectedOneBotChannelsFromEnv(botIDs)
	missingOneBotChannels := missingChannels(expectedOneBotChannels, onebotEndpoints)
	telegramChannels := csvEnvOrDefault("AKASHIC_TELEGRAM_CHANNELS", []string{"telegram"})
	telegramEndpoint := strings.TrimSpace(os.Getenv("AKASHIC_TELEGRAM_API_BASE_URL"))
	if telegramEndpoint == "" {
		telegramEndpoint = "https://api.telegram.org"
	}
	blockers := runtimeConfigBlockers(onebotEndpoints, expectedOneBotChannels, missingOneBotChannels)

	return query.RuntimeConfigView{
		Runtime: query.RuntimeProcessConfigView{
			Address:       addr,
			AddressSource: addrSource,
			BotIDs:        append([]string(nil), botIDs...),
		},
		Delivery: query.RuntimeDeliveryConfigView{
			TelegramChannels:               telegramChannels,
			TelegramTokenConfigured:        telegramBotTokenFromEnv() != "",
			TelegramEndpoint:               redactQueueDSN(telegramEndpoint),
			OneBotEndpoints:                runtimeOneBotEndpointViews(onebotEndpoints),
			OneBotExpectedChannels:         expectedOneBotChannels,
			OneBotMissingChannels:          missingOneBotChannels,
			OneBotDefaultTokenConfigured:   strings.TrimSpace(os.Getenv("AKASHIC_ONEBOT_ACCESS_TOKEN")) != "",
			OneBotPerChannelTokenChannels:  sortedMapKeys(keyValueCSVEnv("AKASHIC_ONEBOT_ACCESS_TOKENS")),
			OneBotReadyForHealthProbe:      len(onebotEndpoints) > 0,
			OneBotReadyForDualAccountSmoke: len(onebotEndpoints) > 0 && len(missingOneBotChannels) == 0,
		},
		Workers: query.RuntimeWorkerConfigView{
			AgentJobRecoveryEnabled:          boolEnv("AKASHIC_AGENT_JOB_RECOVERY_ENABLED"),
			OutboxDeliveryWorkerEnabled:      boolEnv("AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED"),
			AgentJobStrictLeaseToken:         boolEnv("AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN"),
			QueueExternalLeaseAgentJobEnable: boolEnv("AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"),
		},
		Environment: runtimeConfigEnvVars(),
		Readiness: query.RuntimeConfigReadinessView{
			DeliveryAdaptersConfigured:  len(onebotEndpoints) > 0 || telegramBotTokenFromEnv() != "",
			OneBotConfigured:            len(onebotEndpoints) > 0,
			OneBotExpectedChannelsOK:    len(missingOneBotChannels) == 0,
			OneBotHealthProbeReady:      len(onebotEndpoints) > 0,
			OneBotDualAccountSmokeReady: len(onebotEndpoints) > 0 && len(missingOneBotChannels) == 0,
			Blockers:                    blockers,
		},
		Notes: []string{
			"read-only sanitized runtime configuration snapshot; secret values are never returned",
			"generic qq channel is treated as the local primary OneBot alias; set AKASHIC_ONEBOT_EXPECTED_CHANNELS to override expected aliases",
		},
		SideEffect: "none",
	}
}

func expectedOneBotChannelsFromEnv(botIDs []string) []string {
	if channels := csvEnvOrDefault("AKASHIC_ONEBOT_EXPECTED_CHANNELS", nil); len(channels) > 0 {
		return sortedUniqueStrings(channels)
	}
	expected := []string{"qq"}
	for _, botID := range botIDs[1:] {
		botID = strings.TrimSpace(botID)
		if botID == "" {
			continue
		}
		expected = append(expected, "qq_"+botID)
	}
	return sortedUniqueStrings(expected)
}

func missingChannels(expected []string, endpoints map[string]onebotdelivery.EndpointConfig) []string {
	missing := make([]string, 0)
	for _, channel := range expected {
		if _, ok := endpoints[channel]; !ok {
			missing = append(missing, channel)
		}
	}
	return missing
}

func runtimeConfigBlockers(
	onebotEndpoints map[string]onebotdelivery.EndpointConfig,
	expectedChannels []string,
	missingChannels []string,
) []string {
	blockers := make([]string, 0)
	if len(onebotEndpoints) == 0 {
		blockers = append(blockers, "onebot_endpoints_missing")
	}
	if len(expectedChannels) > 0 && len(missingChannels) > 0 {
		blockers = append(blockers, "onebot_expected_channels_missing")
	}
	return blockers
}

func runtimeOneBotEndpointViews(endpoints map[string]onebotdelivery.EndpointConfig) []query.RuntimeOneBotEndpointConfigView {
	items := make([]query.RuntimeOneBotEndpointConfigView, 0, len(endpoints))
	for _, channel := range sortedMapKeys(endpoints) {
		endpoint := endpoints[channel]
		items = append(items, query.RuntimeOneBotEndpointConfigView{
			Channel:               channel,
			Transport:             onebotTransport(endpoint),
			HTTPConfigured:        strings.TrimSpace(endpoint.BaseURL) != "",
			WebSocketConfigured:   strings.TrimSpace(endpoint.WebSocketURL) != "",
			AccessTokenConfigured: strings.TrimSpace(endpoint.AccessToken) != "",
			Endpoint:              onebotDiagnosticEndpoint(endpoint),
			Notes:                 onebotDiagnosticNotes(endpoint),
		})
	}
	return items
}

func runtimeConfigEnvVars() []query.RuntimeEnvVarView {
	keys := []string{
		"AKASHIC_RUNTIME_ADDR",
		"AKASHIC_GATEWAY_ADDR",
		"AKASHIC_RUNTIME_STATE_DIR",
		"AKASHIC_OBSERVE_TARGETS_DSN",
		"AKASHIC_OBSERVE_TARGETS_PATH",
		"AKASHIC_RECEIVER_STATUSES_DSN",
		"AKASHIC_RECEIVER_STATUSES_PATH",
		"AKASHIC_RECEIVER_LEASES_DSN",
		"AKASHIC_RECEIVER_LEASES_PATH",
		"AKASHIC_RECEIVER_STATUS_STALE_SECONDS",
		"AKASHIC_BOT_IDS",
		"AKASHIC_ONEBOT_EXPECTED_CHANNELS",
		"AKASHIC_ONEBOT_WS_URLS",
		"AKASHIC_ONEBOT_WEBSOCKET_URLS",
		"AKASHIC_ONEBOT_WS_URL",
		"AKASHIC_ONEBOT_WEBSOCKET_URL",
		"AKASHIC_ONEBOT_HTTP_BASE_URLS",
		"AKASHIC_ONEBOT_HTTP_BASE_URL",
		"AKASHIC_ONEBOT_CHANNELS",
		"AKASHIC_ONEBOT_ACCESS_TOKENS",
		"AKASHIC_ONEBOT_ACCESS_TOKEN",
		"AKASHIC_TELEGRAM_CHANNELS",
		"AKASHIC_TELEGRAM_BOT_TOKEN",
		"TELEGRAM_BOT_TOKEN",
		"AKASHIC_TELEGRAM_API_BASE_URL",
		"AKASHIC_QUEUE_BACKEND",
		"AKASHIC_QUEUE_DSN",
		"AKASHIC_QUEUE_MODE",
		"AKASHIC_QUEUE_CONSUMER_CONCURRENCY",
		"AKASHIC_QUEUE_MAX_IN_FLIGHT",
		"AKASHIC_AGENT_JOB_RECOVERY_ENABLED",
		"AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED",
		"AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN",
		"AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED",
	}
	items := make([]query.RuntimeEnvVarView, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		secret := runtimeConfigSecretEnv(key)
		redacted := ""
		if value != "" {
			redacted = runtimeConfigRedactedEnvValue(key, value)
		}
		items = append(items, query.RuntimeEnvVarView{
			Key:           key,
			Present:       value != "",
			Secret:        secret,
			ValueRedacted: redacted,
		})
	}
	return items
}

func runtimeConfigSecretEnv(key string) bool {
	if key == "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN" {
		return false
	}
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "DSN")
}

func runtimeConfigRedactedEnvValue(key string, value string) string {
	if key == "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN" {
		return value
	}
	upper := strings.ToUpper(key)
	if strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") {
		return redactSecretKeyValueCSV(value)
	}
	if strings.Contains(upper, "DSN") || strings.Contains(upper, "URL") {
		return redactKeyValueCSVOrDSN(value)
	}
	return value
}

func redactSecretKeyValueCSV(value string) string {
	if strings.Contains(value, "=") && strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		for index, item := range parts {
			key, _, ok := strings.Cut(strings.TrimSpace(item), "=")
			if !ok {
				parts[index] = "redacted"
				continue
			}
			parts[index] = strings.TrimSpace(key) + "=redacted"
		}
		return strings.Join(parts, ",")
	}
	if key, _, ok := strings.Cut(strings.TrimSpace(value), "="); ok {
		return strings.TrimSpace(key) + "=redacted"
	}
	return "redacted"
}

func redactKeyValueCSVOrDSN(value string) string {
	if strings.Contains(value, "=") && strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		for index, item := range parts {
			key, itemValue, ok := strings.Cut(strings.TrimSpace(item), "=")
			if !ok {
				continue
			}
			parts[index] = strings.TrimSpace(key) + "=" + redactQueueDSN(strings.TrimSpace(itemValue))
		}
		return strings.Join(parts, ",")
	}
	if key, itemValue, ok := strings.Cut(strings.TrimSpace(value), "="); ok {
		return strings.TrimSpace(key) + "=" + redactQueueDSN(strings.TrimSpace(itemValue))
	}
	return redactQueueDSN(value)
}

func sortedUniqueStrings(items []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}
