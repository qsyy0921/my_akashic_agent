package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestDefaultMediaAssetRootsDiscoverRepoFromRepoRoot(t *testing.T) {
	repoRoot := fakeAkashicRepo(t)

	roots := defaultMediaAssetRootsFrom([]string{repoRoot})

	assertContainsPath(t, roots, filepath.Join(repoRoot, ".akashic-workspace", "uploads"))
	assertContainsPath(t, roots, filepath.Join(repoRoot, ".akashic-workspace", "generated_images"))
	assertContainsPath(t, roots, filepath.Join(repoRoot, "generated_images"))
	assertNotContainsPath(t, roots, filepath.Join(filepath.Dir(repoRoot), ".akashic-workspace", "uploads"))
}

func TestDefaultMediaAssetRootsDiscoverRepoFromServiceOrBinDir(t *testing.T) {
	repoRoot := fakeAkashicRepo(t)
	serviceDir := filepath.Join(repoRoot, "services", "agent-runtime")
	binDir := filepath.Join(repoRoot, ".tmp", "bin")
	if err := os.MkdirAll(serviceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}

	roots := defaultMediaAssetRootsFrom([]string{serviceDir, binDir})

	assertContainsPath(t, roots, filepath.Join(repoRoot, ".akashic-workspace", "uploads"))
	if got := countPath(roots, filepath.Join(repoRoot, ".akashic-workspace", "uploads")); got != 1 {
		t.Fatalf("expected upload root once, got %d in %#v", got, roots)
	}
}

func TestDefaultRuntimeStateDirDiscoversRepoFromServiceDir(t *testing.T) {
	repoRoot := fakeAkashicRepo(t)
	serviceDir := filepath.Join(repoRoot, "services", "agent-runtime")

	dir, ok := defaultRuntimeStateDirFrom([]string{serviceDir})

	if !ok {
		t.Fatalf("expected runtime state dir")
	}
	if dir != filepath.Join(repoRoot, ".akashic-workspace", "agent-runtime") {
		t.Fatalf("unexpected runtime state dir: %s", dir)
	}
}

func TestDefaultRuntimeStateDirCanBeDisabled(t *testing.T) {
	t.Setenv("AKASHIC_RUNTIME_STATE_DIR", "memory")

	dir, ok := defaultRuntimeStateDirFrom([]string{fakeAkashicRepo(t)})

	if ok || dir != "" {
		t.Fatalf("expected memory runtime state, got ok=%v dir=%q", ok, dir)
	}
}

func TestDefaultRuntimeStatePathUsesExplicitDir(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "runtime-state")
	t.Setenv("AKASHIC_RUNTIME_STATE_DIR", stateDir)

	path, ok := defaultRuntimeStatePath("inbox.json")

	if !ok {
		t.Fatalf("expected runtime state path")
	}
	if path != filepath.Join(stateDir, "inbox.json") {
		t.Fatalf("unexpected runtime state path: %s", path)
	}
}

func TestNewInboxEventRepositoryDefaultsToFileBackedRuntimeState(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "runtime-state")
	t.Setenv("AKASHIC_RUNTIME_STATE_DIR", stateDir)
	t.Setenv("AKASHIC_INBOX_DSN", "")
	t.Setenv("AKASHIC_INBOX_PATH", "")

	repo, err := newInboxEventRepository()
	if err != nil {
		t.Fatal(err)
	}
	event := newRuntimeStateInboxEvent(t)
	if err := repo.SaveInboxEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	reopened, err := newInboxEventRepository()
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.ListInboxEvents(context.Background(), query.InboxEventFilter{
		ConversationID: "27234224",
		Limit:          10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected default file-backed inbox persistence, got %d", len(items))
	}
	if _, err := os.Stat(filepath.Join(stateDir, "inbox.json")); err != nil {
		t.Fatalf("expected inbox state file: %v", err)
	}
}

func TestNewInboxEventRepositoryRespectsMemoryOverride(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "runtime-state")
	t.Setenv("AKASHIC_RUNTIME_STATE_DIR", stateDir)
	t.Setenv("AKASHIC_INBOX_DSN", "memory")
	t.Setenv("AKASHIC_INBOX_PATH", "")

	repo, err := newInboxEventRepository()
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveInboxEvent(context.Background(), newRuntimeStateInboxEvent(t)); err != nil {
		t.Fatal(err)
	}

	reopened, err := newInboxEventRepository()
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.ListInboxEvents(context.Background(), query.InboxEventFilter{
		ConversationID: "27234224",
		Limit:          10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected memory override to avoid persistence, got %d", len(items))
	}
}

func TestNewObserveTargetServiceDefaultsToFileBackedRuntimeState(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "runtime-state")
	t.Setenv("AKASHIC_RUNTIME_STATE_DIR", stateDir)
	t.Setenv("AKASHIC_OBSERVE_TARGETS_DSN", "")
	t.Setenv("AKASHIC_OBSERVE_TARGETS_PATH", "")

	service, err := newObserveTargetService()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SyncObserveTargets(context.Background(), command.SyncObserveTargetsCommand{
		Source:    "python_config",
		Timestamp: time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC),
		Targets: []command.ObserveTargetCommand{{
			Channel: command.ChannelCommand{
				Kind:             "qq",
				AccountID:        "1049511700",
				ConversationID:   "27234224",
				ConversationType: "group",
			},
			ObserveOnly: true,
			Enabled:     true,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	reopened, err := newObserveTargetService()
	if err != nil {
		t.Fatal(err)
	}
	view, err := reopened.ListObserveTargets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if view.Totals["targets"] != 1 || view.Targets[0].Channel.ConversationID != "27234224" {
		t.Fatalf("expected default file-backed observe target persistence: %#v", view)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "observe-targets.json")); err != nil {
		t.Fatalf("expected observe targets state file: %v", err)
	}
}

func TestNewReceiverStatusServiceDefaultsToFileBackedRuntimeState(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "runtime-state")
	t.Setenv("AKASHIC_RUNTIME_STATE_DIR", stateDir)
	t.Setenv("AKASHIC_RECEIVER_STATUSES_DSN", "")
	t.Setenv("AKASHIC_RECEIVER_STATUSES_PATH", "")

	service, err := newReceiverStatusService()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReportReceiverStatus(context.Background(), command.ReportReceiverStatusCommand{
		Kind:        "qq",
		ChannelName: "qq",
		AccountID:   "1049511700",
		Status:      "connected",
		Reason:      "heartbeat",
		Source:      "python_channel",
		Timestamp:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	reopened, err := newReceiverStatusService()
	if err != nil {
		t.Fatal(err)
	}
	view, err := reopened.ListReceiverStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if view.Totals["receivers"] != 1 || view.Totals["connected"] != 1 {
		t.Fatalf("expected default file-backed receiver status persistence: %#v", view)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "receiver-statuses.json")); err != nil {
		t.Fatalf("expected receiver statuses state file: %v", err)
	}
}

func TestReceiverStatusStaleAfterFromEnvClampsMinimum(t *testing.T) {
	t.Setenv("AKASHIC_RECEIVER_STATUS_STALE_SECONDS", "10")

	got, err := receiverStatusStaleAfterFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got != 30*time.Second {
		t.Fatalf("expected minimum stale threshold, got %s", got)
	}
}

func TestOneBotEndpointsFromEnvReadsMultiEndpointConfig(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_HTTP_BASE_URLS", "qq_1049511700=http://127.0.0.1:3001, qq_2365524513=http://127.0.0.1:3002")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKENS", "qq_1049511700=token-a,qq_2365524513=token-b")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKEN", "default-token")

	endpoints := onebotEndpointsFromEnv()

	if len(endpoints) != 2 {
		t.Fatalf("expected two endpoints, got %#v", endpoints)
	}
	if got := endpoints["qq_1049511700"].BaseURL; got != "http://127.0.0.1:3001" {
		t.Fatalf("unexpected endpoint A base url: %q", got)
	}
	if got := endpoints["qq_1049511700"].AccessToken; got != "token-a" {
		t.Fatalf("unexpected endpoint A token: %q", got)
	}
	if got := endpoints["qq_2365524513"].BaseURL; got != "http://127.0.0.1:3002" {
		t.Fatalf("unexpected endpoint B base url: %q", got)
	}
	if got := endpoints["qq_2365524513"].AccessToken; got != "token-b" {
		t.Fatalf("unexpected endpoint B token: %q", got)
	}
}

func TestOneBotEndpointsFromEnvReadsSingleEndpointConfig(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_HTTP_BASE_URL", "http://127.0.0.1:3001")
	t.Setenv("AKASHIC_ONEBOT_CHANNELS", "qq,qq_2365524513")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKEN", "shared-token")

	endpoints := onebotEndpointsFromEnv()

	if len(endpoints) != 2 {
		t.Fatalf("expected two channel aliases, got %#v", endpoints)
	}
	for _, channel := range []string{"qq", "qq_2365524513"} {
		endpoint, ok := endpoints[channel]
		if !ok {
			t.Fatalf("missing endpoint for %s: %#v", channel, endpoints)
		}
		if endpoint.BaseURL != "http://127.0.0.1:3001" || endpoint.AccessToken != "shared-token" {
			t.Fatalf("unexpected endpoint for %s: %#v", channel, endpoint)
		}
	}
}

func TestOneBotEndpointsFromEnvReadsWebSocketEndpointConfig(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_WS_URLS", "qq_1049511700=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKEN", "shared-token")

	endpoints := onebotEndpointsFromEnv()

	if len(endpoints) != 2 {
		t.Fatalf("expected two websocket endpoints, got %#v", endpoints)
	}
	if got := endpoints["qq_1049511700"].WebSocketURL; got != "ws://127.0.0.1:3001" {
		t.Fatalf("unexpected endpoint A websocket url: %q", got)
	}
	if got := endpoints["qq_2365524513"].WebSocketURL; got != "ws://127.0.0.1:3002" {
		t.Fatalf("unexpected endpoint B websocket url: %q", got)
	}
	if got := endpoints["qq_2365524513"].AccessToken; got != "shared-token" {
		t.Fatalf("unexpected endpoint B token: %q", got)
	}
}

func TestOneBotEndpointsFromEnvCombinesHTTPAndWebSocketEndpointConfig(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_HTTP_BASE_URLS", "qq=http://127.0.0.1:3003")
	t.Setenv("AKASHIC_ONEBOT_WEBSOCKET_URLS", "qq=ws://127.0.0.1:3001")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKEN", "token")

	endpoints := onebotEndpointsFromEnv()
	endpoint := endpoints["qq"]

	if endpoint.BaseURL != "http://127.0.0.1:3003" || endpoint.WebSocketURL != "ws://127.0.0.1:3001" {
		t.Fatalf("expected combined endpoint, got %#v", endpoint)
	}
	if endpoint.AccessToken != "token" {
		t.Fatalf("unexpected token: %q", endpoint.AccessToken)
	}
}

func TestDeliveryAdapterDiagnosticsFromEnvReportsConfiguredAliases(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_WS_URLS", "qq=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKENS", "qq=NcatBot,qq_2365524513=NcatBot")
	t.Setenv("AKASHIC_TELEGRAM_BOT_TOKEN", "telegram-token")
	t.Setenv("AKASHIC_TELEGRAM_CHANNELS", "telegram,dongri0909bot")

	items := deliveryAdapterDiagnosticsFromEnv()

	if len(items) != 4 {
		t.Fatalf("expected four adapter diagnostics, got %#v", items)
	}
	qq := findDeliveryAdapterDiagnostic(t, items, "onebot", "qq")
	if qq.Transport != "websocket" || !qq.Enabled || !qq.EndpointConfigured || !qq.AccessTokenConfigured {
		t.Fatalf("unexpected qq diagnostic: %#v", qq)
	}
	if qq.Endpoint != "ws://127.0.0.1:3001" {
		t.Fatalf("unexpected redacted qq endpoint: %q", qq.Endpoint)
	}
	bot := findDeliveryAdapterDiagnostic(t, items, "onebot", "qq_2365524513")
	if bot.Transport != "websocket" || bot.Endpoint != "ws://127.0.0.1:3002" {
		t.Fatalf("unexpected bot diagnostic: %#v", bot)
	}
	telegram := findDeliveryAdapterDiagnostic(t, items, "telegram", "dongri0909bot")
	if telegram.Transport != "http" || !telegram.AccessTokenConfigured || telegram.Endpoint == "" {
		t.Fatalf("unexpected telegram diagnostic: %#v", telegram)
	}
}

func TestRuntimeConfigFromEnvReportsSanitizedOneBotReadiness(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_WS_URLS", "qq=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKENS", "qq=NcatBot,qq_2365524513=NcatBot")
	t.Setenv("AKASHIC_TELEGRAM_BOT_TOKEN", "telegram-token")
	t.Setenv("TELEGRAM_BOT_TOKEN", "telegram-token-fallback")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED", "true")
	t.Setenv("AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN", "true")

	view := runtimeConfigFromEnv(":8780", "AKASHIC_RUNTIME_ADDR", []string{"1049511700", "2365524513"})

	if view.SideEffect != "none" {
		t.Fatalf("runtime config must be read-only: %#v", view)
	}
	if view.Runtime.Address != ":8780" || view.Runtime.AddressSource != "AKASHIC_RUNTIME_ADDR" {
		t.Fatalf("unexpected runtime address: %#v", view.Runtime)
	}
	if !view.Delivery.OneBotReadyForHealthProbe || !view.Delivery.OneBotReadyForDualAccountSmoke {
		t.Fatalf("expected onebot readiness: %#v", view.Delivery)
	}
	if len(view.Delivery.OneBotMissingChannels) != 0 || len(view.Readiness.Blockers) != 0 {
		t.Fatalf("unexpected blockers: %#v %#v", view.Delivery.OneBotMissingChannels, view.Readiness.Blockers)
	}
	if !view.Workers.OutboxDeliveryWorkerEnabled || !view.Workers.AgentJobStrictLeaseToken {
		t.Fatalf("unexpected worker config flags: %#v", view.Workers)
	}
	tokenEnv := findRuntimeEnvVar(t, view.Environment, "AKASHIC_ONEBOT_ACCESS_TOKENS")
	if !tokenEnv.Present || !tokenEnv.Secret {
		t.Fatalf("token env should expose presence only: %#v", tokenEnv)
	}
	if tokenEnv.ValueRedacted == "qq=NcatBot,qq_2365524513=NcatBot" {
		t.Fatalf("token value was not redacted: %#v", tokenEnv)
	}
	if tokenEnv.ValueRedacted != "qq=redacted,qq_2365524513=redacted" {
		t.Fatalf("token value should be fully redacted: %#v", tokenEnv)
	}
	strictTokenFlag := findRuntimeEnvVar(t, view.Environment, "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN")
	if strictTokenFlag.Secret || strictTokenFlag.ValueRedacted != "true" {
		t.Fatalf("strict lease token flag is a boolean config, not a secret: %#v", strictTokenFlag)
	}
	telegramToken := findRuntimeEnvVar(t, view.Environment, "TELEGRAM_BOT_TOKEN")
	if telegramToken.ValueRedacted != "redacted" {
		t.Fatalf("telegram token should be fully redacted: %#v", telegramToken)
	}
}

func TestRuntimeConfigFromEnvReportsMissingExpectedOneBotAliases(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_WS_URLS", "qq=ws://127.0.0.1:3001")

	view := runtimeConfigFromEnv(":8780", "default", []string{"1049511700", "2365524513"})

	if view.Delivery.OneBotReadyForDualAccountSmoke {
		t.Fatalf("dual account smoke must not be ready with missing alias: %#v", view.Delivery)
	}
	assertContainsString(t, view.Delivery.OneBotMissingChannels, "qq_2365524513")
	assertContainsString(t, view.Readiness.Blockers, "onebot_expected_channels_missing")
}

func TestDeliveryAdapterHealthProbesSelectProbeCapableAdapters(t *testing.T) {
	probes := deliveryAdapterHealthProbes([]outport.DeliveryAdapter{
		fakeDeliveryAdapter{},
		fakeDeliveryAdapterHealthProbe{},
	})

	if len(probes) != 1 {
		t.Fatalf("expected one health probe, got %#v", probes)
	}
}

func TestDeliverySmokeReadinessConfigFromEnvBuildsDualAccountMediaMatrix(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_WS_URLS", "qq=ws://127.0.0.1:3001,qq_1049511700=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002")
	t.Setenv("AKASHIC_DELIVERY_SMOKE_GROUP_IDS", "27234224")

	endpoints := onebotEndpointsFromEnv()
	cases, channelByAccount := deliverySmokeReadinessConfigFromEnv([]string{"1049511700", "2365524513"}, endpoints)

	if channelByAccount["1049511700"] != "qq_1049511700" {
		t.Fatalf("expected exact 104 alias, got %#v", channelByAccount)
	}
	if channelByAccount["2365524513"] != "qq_2365524513" {
		t.Fatalf("expected exact 236 alias, got %#v", channelByAccount)
	}
	if len(cases) != 12 {
		t.Fatalf("expected private and group text/image/file matrix, got %d cases: %#v", len(cases), cases)
	}
	assertSmokeCase(t, cases, "qq_private_text_1049511700_to_2365524513")
	assertSmokeCase(t, cases, "qq_private_image_2365524513_to_1049511700")
	assertSmokeCase(t, cases, "qq_group_file_1049511700_to_27234224")
}

func TestQueueBackendViewFromEnvDefaultsLocal(t *testing.T) {
	view, err := queueBackendViewFromEnv()
	if err != nil {
		t.Fatalf("queue backend view: %v", err)
	}

	if view.Provider != "local" || view.Mode != "local_state_store" || view.MigrationPhase != "local_only" {
		t.Fatalf("unexpected default queue backend: %#v", view)
	}
	if view.Stream != "AKASHIC_WORK" || view.SubjectPrefix != "akashic.work" {
		t.Fatalf("unexpected default queue names: %#v", view)
	}
	if view.ExternalQueueActive {
		t.Fatalf("external queue must not be active by default")
	}
	if !view.StateStoreAuthoritative {
		t.Fatalf("state store must remain authoritative")
	}
}

func TestQueueBackendViewFromEnvNormalizesNATSJetStream(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_BACKEND", "nats")
	t.Setenv("AKASHIC_QUEUE_DSN", "nats://token@127.0.0.1:4222")
	t.Setenv("AKASHIC_QUEUE_STREAM", "AKASHIC_TEST")
	t.Setenv("AKASHIC_QUEUE_SUBJECT_PREFIX", "akashic.test")
	t.Setenv("AKASHIC_QUEUE_CONSUMER_CONCURRENCY", "16")
	t.Setenv("AKASHIC_QUEUE_MAX_IN_FLIGHT", "128")

	view, err := queueBackendViewFromEnv()
	if err != nil {
		t.Fatalf("queue backend view: %v", err)
	}

	if view.Provider != "nats_jetstream" || view.Mode != "shadow_publish" || view.MigrationPhase != "shadow_ready" {
		t.Fatalf("unexpected nats queue backend: %#v", view)
	}
	if !view.ExternalQueueConfigured || !view.DSNConfigured {
		t.Fatalf("expected configured external queue diagnostics: %#v", view)
	}
	if view.ConsumerModel != "goroutine_worker_pool" || view.ConsumerConcurrency != 16 || view.MaxInFlight != 128 {
		t.Fatalf("unexpected consumer concurrency diagnostics: %#v", view)
	}
	if view.Stream != "AKASHIC_TEST" || view.SubjectPrefix != "akashic.test" {
		t.Fatalf("unexpected queue names: %#v", view)
	}
	if view.RecommendedFirstBackend != "nats_jetstream" {
		t.Fatalf("unexpected recommended backend: %q", view.RecommendedFirstBackend)
	}
	if view.ExternalQueueActive {
		t.Fatalf("external queue adapter should not be active yet: %#v", view)
	}
	if view.DSNRedacted == "" || view.DSNRedacted == "nats://token@127.0.0.1:4222" {
		t.Fatalf("expected redacted dsn, got %q", view.DSNRedacted)
	}
}

func TestQueueBackendViewFromEnvSupportsDualReadCompareMode(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_BACKEND", "nats")
	t.Setenv("AKASHIC_QUEUE_MODE", "dual_read_compare")
	t.Setenv("AKASHIC_QUEUE_DSN", "nats://127.0.0.1:4222")
	t.Setenv("AKASHIC_QUEUE_CONSUMER_CONCURRENCY", "6")
	t.Setenv("AKASHIC_QUEUE_MAX_IN_FLIGHT", "24")

	view, err := queueBackendViewFromEnv()
	if err != nil {
		t.Fatalf("queue backend view: %v", err)
	}

	if view.Provider != "nats_jetstream" || view.Mode != "dual_read_compare" || view.MigrationPhase != "dual_read_compare" {
		t.Fatalf("unexpected dual-read queue backend: %#v", view)
	}
	if view.ConsumerConcurrency != 6 || view.MaxInFlight != 24 {
		t.Fatalf("unexpected consumer sizing: %#v", view)
	}
}

func TestQueueBackendViewFromEnvReportsExternalLeaseGate(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_BACKEND", "nats")
	t.Setenv("AKASHIC_QUEUE_MODE", "external_lease")
	t.Setenv("AKASHIC_QUEUE_DSN", "nats://127.0.0.1:4222")

	view, err := queueBackendViewFromEnv()
	if err != nil {
		t.Fatalf("queue backend view: %v", err)
	}

	if view.Mode != "external_lease" || view.MigrationPhase != "external_lease_gate" {
		t.Fatalf("unexpected external lease mode: %#v", view)
	}
	if view.ExternalLease == nil || !view.ExternalLease.Enabled {
		t.Fatalf("expected external lease gate: %#v", view.ExternalLease)
	}
	if view.ExternalLease.AllowExecution || view.ExternalLease.GateState != "blocked" {
		t.Fatalf("external lease must be blocked by default: %#v", view.ExternalLease)
	}
	assertContainsString(t, view.ExternalLease.Blockers, "explicit_cutover")
	assertContainsString(t, view.ExternalLease.Blockers, "dual_read_smoke_passed")
	assertContainsString(t, view.ExternalLease.Blockers, "state_lease_workers_disabled")
	if view.ExternalLease.AckPolicy != "ack_after_go_state_terminal" {
		t.Fatalf("unexpected ack policy: %#v", view.ExternalLease)
	}
	if view.ExternalLease.ExecutionScope != "none" {
		t.Fatalf("blocked gate should have no execution scope: %#v", view.ExternalLease)
	}
	assertBlockedWorkKind(t, view.ExternalLease.BlockedWorkKinds, "agent_job")
}

func TestQueueBackendViewFromEnvAllowsExternalLeaseAfterExplicitGates(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_BACKEND", "nats")
	t.Setenv("AKASHIC_QUEUE_MODE", "external_lease")
	t.Setenv("AKASHIC_QUEUE_DSN", "nats://127.0.0.1:4222")
	t.Setenv("AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER", "true")
	t.Setenv("AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED", "true")
	t.Setenv("AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED", "true")

	view, err := queueBackendViewFromEnv()
	if err != nil {
		t.Fatalf("queue backend view: %v", err)
	}

	if view.ExternalLease == nil || !view.ExternalLease.CutoverRequested {
		t.Fatalf("expected cutover requested gate: %#v", view.ExternalLease)
	}
	if !view.ExternalLease.AllowExecution || view.ExternalLease.GateState != "ready" {
		t.Fatalf("external lease should be ready after explicit gates: %#v", view.ExternalLease)
	}
	if view.ExternalLease.ExecutionScope != "outbox_delivery_only" {
		t.Fatalf("external lease must stay scoped to outbox delivery: %#v", view.ExternalLease)
	}
	assertContainsString(t, view.ExternalLease.AllowedWorkKinds, "outbox_delivery")
	assertBlockedWorkKind(t, view.ExternalLease.BlockedWorkKinds, "agent_job")
	if view.AgentJobQueueSource != "agent_job_state_store" {
		t.Fatalf("agent jobs must remain on state-store lease: %#v", view)
	}
	if len(view.ExternalLease.Blockers) != 0 {
		t.Fatalf("unexpected blockers: %#v", view.ExternalLease.Blockers)
	}
}

func TestQueueBackendViewFromEnvAllowsAgentJobResultAckAfterExplicitGates(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_BACKEND", "nats")
	t.Setenv("AKASHIC_QUEUE_MODE", "external_lease")
	t.Setenv("AKASHIC_QUEUE_DSN", "nats://127.0.0.1:4222")
	t.Setenv("AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER", "true")
	t.Setenv("AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED", "true")
	t.Setenv("AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED", "true")
	t.Setenv("AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED", "true")
	t.Setenv("AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED", "true")
	t.Setenv("AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED", "true")
	t.Setenv("AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN", "true")

	view, err := queueBackendViewFromEnv()
	if err != nil {
		t.Fatalf("queue backend view: %v", err)
	}

	if view.ExternalLease == nil || !view.ExternalLease.AllowExecution {
		t.Fatalf("expected external lease ready: %#v", view.ExternalLease)
	}
	if view.ExternalLease.ExecutionScope != "outbox_delivery_and_agent_job_result_ack" {
		t.Fatalf("expected expanded execution scope: %#v", view.ExternalLease)
	}
	assertContainsString(t, view.ExternalLease.AllowedWorkKinds, "outbox_delivery")
	assertContainsString(t, view.ExternalLease.AllowedWorkKinds, "agent_job")
	if len(view.ExternalLease.BlockedWorkKinds) != 0 {
		t.Fatalf("agent_job should not remain blocked: %#v", view.ExternalLease.BlockedWorkKinds)
	}
	if view.AgentJobQueueSource != "agent_job_state_store_with_nats_result_ack" {
		t.Fatalf("unexpected agent job queue source: %#v", view)
	}
}

func TestQueueBackendViewFromEnvKeepsAgentJobBlockedWithoutFlowSmoke(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_BACKEND", "nats")
	t.Setenv("AKASHIC_QUEUE_MODE", "external_lease")
	t.Setenv("AKASHIC_QUEUE_DSN", "nats://127.0.0.1:4222")
	t.Setenv("AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER", "true")
	t.Setenv("AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED", "true")
	t.Setenv("AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED", "true")
	t.Setenv("AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED", "true")
	t.Setenv("AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED", "true")
	t.Setenv("AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN", "true")

	view, err := queueBackendViewFromEnv()
	if err != nil {
		t.Fatalf("queue backend view: %v", err)
	}

	if view.ExternalLease == nil || !view.ExternalLease.AllowExecution {
		t.Fatalf("base external lease gate should still be ready: %#v", view.ExternalLease)
	}
	if view.ExternalLease.ExecutionScope != "outbox_delivery_only" {
		t.Fatalf("agent job result-ack must stay blocked without flow smoke: %#v", view.ExternalLease)
	}
	assertBlockedWorkKind(t, view.ExternalLease.BlockedWorkKinds, "agent_job")
	if view.AgentJobQueueSource != "agent_job_state_store" {
		t.Fatalf("agent jobs must remain on state-store lease without flow smoke: %#v", view)
	}
}

func TestQueueBackendViewFromEnvRejectsUnknownProvider(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_BACKEND", "kafka")

	if _, err := queueBackendViewFromEnv(); err == nil {
		t.Fatalf("expected unsupported provider error")
	}
}

func TestQueueBackendViewFromEnvRejectsInvalidConcurrency(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_CONSUMER_CONCURRENCY", "0")

	if _, err := queueBackendViewFromEnv(); err == nil {
		t.Fatalf("expected invalid concurrency error")
	}
}

func TestQueueBackendViewFromEnvRejectsMaxInFlightBelowConcurrency(t *testing.T) {
	t.Setenv("AKASHIC_QUEUE_CONSUMER_CONCURRENCY", "8")
	t.Setenv("AKASHIC_QUEUE_MAX_IN_FLIGHT", "4")

	if _, err := queueBackendViewFromEnv(); err == nil {
		t.Fatalf("expected max in-flight validation error")
	}
}

func TestAgentJobLeaseRecoveryConfigDisabledByDefault(t *testing.T) {
	_, enabled, err := agentJobLeaseRecoveryConfigFromEnv()
	if err != nil {
		t.Fatalf("recovery config: %v", err)
	}
	if enabled {
		t.Fatal("expected recovery runner disabled by default")
	}
}

func TestAgentJobLeaseRecoveryConfigFromEnv(t *testing.T) {
	t.Setenv("AKASHIC_AGENT_JOB_RECOVERY_ENABLED", "true")
	t.Setenv("AKASHIC_AGENT_JOB_RECOVERY_INTERVAL_SECONDS", "120")
	t.Setenv("AKASHIC_AGENT_JOB_RECOVERY_LIMIT", "25")
	t.Setenv("AKASHIC_AGENT_JOB_RECOVERY_RUN_ON_START", "false")

	config, enabled, err := agentJobLeaseRecoveryConfigFromEnv()
	if err != nil {
		t.Fatalf("recovery config: %v", err)
	}
	if !enabled {
		t.Fatal("expected recovery runner enabled")
	}
	if config.Interval != 120*time.Second || config.Limit != 25 || config.RunOnStart {
		t.Fatalf("unexpected recovery config: %+v", config)
	}
}

func TestOutboxDeliveryWorkerConfigDisabledByDefault(t *testing.T) {
	_, enabled, err := outboxDeliveryWorkerConfigFromEnv()
	if err != nil {
		t.Fatalf("outbox worker config: %v", err)
	}
	if enabled {
		t.Fatal("expected outbox delivery worker disabled by default")
	}
}

func TestOutboxDeliveryWorkerConfigFromEnv(t *testing.T) {
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED", "true")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_INTERVAL_SECONDS", "7")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_BATCH_SIZE", "3")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_ID", "runtime-outbox-a")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_LEASE_TTL_SECONDS", "120")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_RUN_ON_START", "false")
	t.Setenv("AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT", "1049511700=qq_1049511700,2365524513=qq_2365524513")

	config, enabled, err := outboxDeliveryWorkerConfigFromEnv()
	if err != nil {
		t.Fatalf("outbox worker config: %v", err)
	}
	if !enabled {
		t.Fatal("expected outbox delivery worker enabled")
	}
	if config.Interval != 7*time.Second ||
		config.BatchSize != 3 ||
		config.WorkerID != "runtime-outbox-a" ||
		config.LeaseTTLSeconds != 120 ||
		config.RunOnStart {
		t.Fatalf("unexpected outbox worker config: %+v", config)
	}
	if config.ChannelByAccount["1049511700"] != "qq_1049511700" ||
		config.ChannelByAccount["2365524513"] != "qq_2365524513" {
		t.Fatalf("unexpected channel map: %#v", config.ChannelByAccount)
	}
}

func TestRuntimeWorkerDiagnosticsFromEnvIncludesConfiguredWorkers(t *testing.T) {
	t.Setenv("AKASHIC_AGENT_JOB_RECOVERY_ENABLED", "true")
	t.Setenv("AKASHIC_AGENT_JOB_RECOVERY_INTERVAL_SECONDS", "120")
	t.Setenv("AKASHIC_AGENT_JOB_RECOVERY_LIMIT", "25")
	t.Setenv("AKASHIC_AGENT_JOB_RECOVERY_RUN_ON_START", "false")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED", "true")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_INTERVAL_SECONDS", "7")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_BATCH_SIZE", "3")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_ID", "runtime-outbox-a")
	t.Setenv("AKASHIC_OUTBOX_DELIVERY_WORKER_LEASE_TTL_SECONDS", "120")
	t.Setenv("AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT", "1049511700=qq_1049511700")

	view, err := runtimeWorkerDiagnosticsFromEnv(query.QueueBackendView{
		Provider:                "nats_jetstream",
		Mode:                    "dual_read_compare",
		MigrationPhase:          "dual_read_compare",
		Stream:                  "AKASHIC_WORK",
		SubjectPrefix:           "akashic.work",
		ExternalQueueConfigured: true,
		ExternalQueueActive:     true,
		ConsumerConcurrency:     4,
		MaxInFlight:             16,
	})
	if err != nil {
		t.Fatalf("runtime worker diagnostics: %v", err)
	}

	recovery := findRuntimeWorker(t, view.Workers, "agent_job_recovery")
	if !recovery.Enabled || !recovery.Running || recovery.IntervalSeconds != 120 || recovery.BatchSize != 25 || recovery.RunOnStart {
		t.Fatalf("unexpected recovery worker: %#v", recovery)
	}
	outbox := findRuntimeWorker(t, view.Workers, "outbox_delivery_worker")
	if !outbox.Enabled || !outbox.Running || outbox.WorkerID != "runtime-outbox-a" || outbox.LeaseTTLSeconds != 120 {
		t.Fatalf("unexpected outbox worker: %#v", outbox)
	}
	if outbox.Attributes["1049511700"] != "qq_1049511700" {
		t.Fatalf("unexpected outbox channel attributes: %#v", outbox.Attributes)
	}
	compare := findRuntimeWorker(t, view.Workers, "nats_dual_read_compare")
	if !compare.Enabled || !compare.Running || compare.ConsumerConcurrency != 4 || compare.MaxInFlight != 16 {
		t.Fatalf("unexpected dual-read worker: %#v", compare)
	}
	externalLease := findRuntimeWorker(t, view.Workers, "nats_external_lease")
	if externalLease.Enabled || externalLease.Running {
		t.Fatalf("external lease should remain disabled without cutover gate: %#v", externalLease)
	}
}

func TestParseKeyValueCSVSkipsMalformedEntries(t *testing.T) {
	got := parseKeyValueCSV("a=1, malformed, b = 2, =missing, c= ")

	if len(got) != 2 || got["a"] != "1" || got["b"] != "2" {
		t.Fatalf("unexpected parse result: %#v", got)
	}
}

func fakeAkashicRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, ".akashic-workspace", "uploads"),
		filepath.Join(root, "services", "agent-runtime"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\nname = \"akashic-test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "services", "agent-runtime", "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func newRuntimeStateInboxEvent(t *testing.T) model.InboxEvent {
	t.Helper()
	event, err := model.NewInboxEvent(model.MessageEnvelope{
		EventID: "qq:1049511700:group:27234224:runtime-state-test",
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: model.ConversationTypeGroup,
		},
		Sender: model.SenderRef{
			ID:   "2948770636",
			Kind: model.SenderKindHuman,
		},
		Content:   "runtime state survives restart",
		Timestamp: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		Metadata:  map[string]string{"observe_only": "true"},
	}, model.LoopDecision{Action: model.LoopActionAllow, Reason: "accepted"}, time.Date(2026, 5, 31, 0, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func assertContainsPath(t *testing.T, paths []string, want string) {
	t.Helper()
	if countPath(paths, want) > 0 {
		return
	}
	t.Fatalf("expected %s in %#v", filepath.Clean(want), paths)
}

func assertNotContainsPath(t *testing.T, paths []string, want string) {
	t.Helper()
	if countPath(paths, want) == 0 {
		return
	}
	t.Fatalf("did not expect %s in %#v", filepath.Clean(want), paths)
}

func countPath(paths []string, want string) int {
	want = filepath.Clean(want)
	count := 0
	for _, path := range paths {
		if filepath.Clean(path) == want {
			count++
		}
	}
	return count
}

func assertContainsString(t *testing.T, items []string, want string) {
	t.Helper()
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("expected %q in %#v", want, items)
}

func assertSmokeCase(t *testing.T, items []command.DeliverySmokeCaseCommand, want string) {
	t.Helper()
	for _, item := range items {
		if item.Name == want {
			return
		}
	}
	t.Fatalf("expected smoke case %q in %#v", want, items)
}

func findDeliveryAdapterDiagnostic(t *testing.T, items []query.DeliveryAdapterDiagnosticsView, provider string, channel string) query.DeliveryAdapterDiagnosticsView {
	t.Helper()
	for _, item := range items {
		if item.Provider == provider && item.Channel == channel {
			return item
		}
	}
	t.Fatalf("missing delivery adapter diagnostic provider=%s channel=%s in %#v", provider, channel, items)
	return query.DeliveryAdapterDiagnosticsView{}
}

func findRuntimeWorker(t *testing.T, items []query.RuntimeWorkerView, name string) query.RuntimeWorkerView {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("missing runtime worker %q in %#v", name, items)
	return query.RuntimeWorkerView{}
}

func findRuntimeEnvVar(t *testing.T, items []query.RuntimeEnvVarView, key string) query.RuntimeEnvVarView {
	t.Helper()
	for _, item := range items {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("missing env var %q in %#v", key, items)
	return query.RuntimeEnvVarView{}
}

type fakeDeliveryAdapter struct{}

func (fakeDeliveryAdapter) SupportsDeliveryChannel(string) bool {
	return false
}

func (fakeDeliveryAdapter) DispatchDeliveryStep(context.Context, model.DeliveryDispatchStep) (model.DeliveryDispatchResult, error) {
	return model.DeliveryDispatchResult{}, nil
}

type fakeDeliveryAdapterHealthProbe struct {
	fakeDeliveryAdapter
}

func (fakeDeliveryAdapterHealthProbe) CheckDeliveryAdapterHealth(context.Context, query.DeliveryAdapterHealthFilter) ([]query.DeliveryAdapterHealthView, error) {
	return nil, nil
}

var _ outport.DeliveryAdapter = fakeDeliveryAdapter{}
var _ outport.DeliveryAdapter = fakeDeliveryAdapterHealthProbe{}
var _ outport.DeliveryAdapterHealthProbe = fakeDeliveryAdapterHealthProbe{}

func assertBlockedWorkKind(t *testing.T, items []query.QueueExternalLeaseBlock, want string) {
	t.Helper()
	for _, item := range items {
		if item.WorkKind == want {
			if item.Reason == "" {
				t.Fatalf("blocked work kind %q should include a reason: %#v", want, item)
			}
			return
		}
	}
	t.Fatalf("expected blocked work kind %q in %#v", want, items)
}
