package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
	agentjobeventstore "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/agentjobeventstore"
	agentjobstore "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/agentjobstore"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/auditjsonl"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/inboxstore"
	knowledgecheckpointstore "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/knowledgecheckpointstore"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/localmedia"
	mediaassetstore "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/mediaassetstore"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/onebotdelivery"
	outboxeventstore "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/outboxeventstore"
	outboxstore "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/outboxstore"
	proactivestate "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/proactivestate"
	sendledgerstore "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/sendledgerstore"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/telegramdelivery"
	httptrigger "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/trigger/http"
)

func main() {
	addr, addrSource := envOrFirstDefaultWithSource(
		[]string{"AKASHIC_RUNTIME_ADDR", "AKASHIC_GATEWAY_ADDR"},
		":8780",
	)
	botIDs := csvEnvOrDefault("AKASHIC_BOT_IDS", []string{"1049511700", "2365524513"})

	store := memory.NewStore()
	var auditLog outport.AuditLog = store
	var shadowReader outport.ShadowAuditReader = store
	if auditPath := strings.TrimSpace(os.Getenv("AKASHIC_SHADOW_AUDIT_PATH")); auditPath != "" {
		auditStore, err := auditjsonl.NewStore(auditPath)
		if err != nil {
			log.Fatalf("init shadow audit jsonl store: %v", err)
		}
		auditLog = auditStore
		shadowReader = auditStore
		log.Printf("shadow audit jsonl enabled: %s", auditPath)
	}
	classifier := domainservice.NewProvenanceClassifier(botIDs)
	loopGuard := domainservice.NewLoopGuard(botIDs, 15*time.Second, 6)
	agentJobRepository, err := newAgentJobRepository()
	if err != nil {
		log.Fatalf("init agent job repository: %v", err)
	}
	agentJobEventStore, err := newAgentJobEventStore()
	if err != nil {
		log.Fatalf("init agent job event store: %v", err)
	}
	mediaAssetRepository, err := newMediaAssetRepository()
	if err != nil {
		log.Fatalf("init media asset repository: %v", err)
	}
	sendLedgerRepository, err := newSendLedgerRepository()
	if err != nil {
		log.Fatalf("init send ledger repository: %v", err)
	}
	outboxRepository, outboxQueue, err := newOutboxStore()
	if err != nil {
		log.Fatalf("init outbox repository: %v", err)
	}
	outboxEventStore, err := newOutboxEventStore()
	if err != nil {
		log.Fatalf("init outbox event store: %v", err)
	}
	deliveryAdapters, err := newDeliveryAdapters()
	if err != nil {
		log.Fatalf("init delivery adapters: %v", err)
	}
	inboxEventRepository, err := newInboxEventRepository()
	if err != nil {
		log.Fatalf("init inbox event repository: %v", err)
	}
	knowledgeCheckpointRepository, err := newKnowledgeCheckpointRepository()
	if err != nil {
		log.Fatalf("init knowledge checkpoint repository: %v", err)
	}
	proactiveStateRepository, err := newProactiveStateRepository()
	if err != nil {
		log.Fatalf("init proactive state repository: %v", err)
	}
	queueBackendView, err := queueBackendViewFromEnv()
	if err != nil {
		log.Fatalf("init queue backend config: %v", err)
	}
	workQueue, closeWorkQueue, err := newWorkQueuePublisher(queueBackendView)
	if err != nil {
		log.Fatalf("init work queue publisher: %v", err)
	}
	if closeWorkQueue != nil {
		defer closeWorkQueue()
	}
	if workQueue != nil {
		queueBackendView.ExternalQueueActive = true
		queueBackendView.MigrationPhase = queueBackendActivePhase(queueBackendView.Mode)
		queueBackendView.Notes = append(queueBackendView.Notes, "NATS JetStream work notification publisher is active")
	}
	var workQueueDiagnostics outport.WorkQueuePublishDiagnosticReader
	if diagnostics, ok := workQueue.(outport.WorkQueuePublishDiagnosticReader); ok {
		workQueueDiagnostics = diagnostics
	}
	queueCompare := appservice.NewWorkQueueCompareService(outboxRepository, agentJobRepository)
	compareConsumer, closeCompareConsumer, err := newWorkQueueCompareConsumer(queueBackendView)
	if err != nil {
		log.Fatalf("init work queue compare consumer: %v", err)
	}
	if closeCompareConsumer != nil {
		defer closeCompareConsumer()
	}
	if compareConsumer != nil {
		queueBackendView.ExternalQueueActive = true
		queueBackendView.MigrationPhase = "dual_read_compare"
		queueBackendView.Notes = append(queueBackendView.Notes, "NATS JetStream dual_read_compare consumer is active")
		compareCtx, cancelCompare := context.WithCancel(context.Background())
		defer cancelCompare()
		go func() {
			if err := compareConsumer.Run(compareCtx, queueCompare); err != nil && err != context.Canceled {
				log.Printf("work queue dual_read_compare consumer stopped: %v", err)
			}
		}()
	}

	ingestor := appservice.NewMessageIngestServiceWithRuntimeStores(
		store,
		auditLog,
		sendLedgerRepository,
		store,
		classifier,
		loopGuard,
		mediaAssetRepository,
		inboxEventRepository,
	)
	sender := appservice.NewMessageSendServiceWithOutboxEventsAndWorkQueue(store, sendLedgerRepository, outboxRepository, outboxQueue, outboxEventStore, workQueue)
	imageJobs := appservice.NewImageJobServiceWithAgentJobs(store, store, store)
	outbox := appservice.NewOutboxServiceWithEvents(outboxRepository, outboxQueue, outboxEventStore)
	outboxEvents := appservice.NewOutboxDeliveryEventService(outboxEventStore)
	deliveryDispatch := appservice.NewDeliveryDispatchServiceWithAdapters(outboxRepository, deliveryAdapters...)
	agentJobs := appservice.NewAgentJobServiceWithEventsAndWorkQueue(
		agentJobRepository,
		agentJobEventStore,
		workQueue,
		appservice.WithStrictAgentJobLeaseToken(boolEnv("AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN")),
	)
	stopAgentJobRecovery, err := startAgentJobLeaseRecovery(agentJobs)
	if err != nil {
		log.Fatalf("init agent job lease recovery: %v", err)
	}
	if stopAgentJobRecovery != nil {
		defer stopAgentJobRecovery()
	}
	externalLeaseConsumer, closeExternalLeaseConsumer, err := newWorkQueueExternalLeaseConsumer(queueBackendView)
	if err != nil {
		log.Fatalf("init work queue external lease consumer: %v", err)
	}
	if closeExternalLeaseConsumer != nil {
		defer closeExternalLeaseConsumer()
	}
	if externalLeaseConsumer != nil {
		queueBackendView.ExternalQueueActive = true
		queueBackendView.MigrationPhase = "external_lease"
		queueBackendView.LeaseOwner = "nats_jetstream"
		queueBackendView.OutboxQueueSource = "nats_jetstream"
		queueBackendView.Notes = append(queueBackendView.Notes, "NATS JetStream external lease outbox consumer is active")
		externalLeaseExecutor := appservice.NewWorkQueueExternalLeaseService(
			outbox,
			deliveryDispatch,
			appservice.WithExternalLeaseChannelByAccount(keyValueCSVEnv("AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT")),
			appservice.WithExternalLeaseWorker(
				strings.TrimSpace(os.Getenv("AKASHIC_QUEUE_EXTERNAL_LEASE_WORKER_ID")),
				positiveIntEnvOrDefault("AKASHIC_QUEUE_EXTERNAL_LEASE_TTL_SECONDS", 300, 86400),
			),
			appservice.WithExternalLeaseAgentJobs(agentJobs),
		)
		externalLeaseCtx, cancelExternalLease := context.WithCancel(context.Background())
		defer cancelExternalLease()
		go func() {
			if err := externalLeaseConsumer.Run(externalLeaseCtx, externalLeaseExecutor); err != nil && err != context.Canceled {
				log.Printf("work queue external lease consumer stopped: %v", err)
			}
		}()
	}
	stopOutboxDeliveryWorker, err := startOutboxDeliveryWorker(outbox, deliveryDispatch, queueBackendView)
	if err != nil {
		log.Fatalf("init outbox delivery worker: %v", err)
	}
	if stopOutboxDeliveryWorker != nil {
		defer stopOutboxDeliveryWorker()
	}
	mediaContentReader, err := newMediaAssetContentReader()
	if err != nil {
		log.Fatalf("init media content reader: %v", err)
	}
	mediaAssets := appservice.NewMediaAssetServiceWithContent(mediaAssetRepository, mediaContentReader)
	agentJobEvents := appservice.NewAgentJobEventService(agentJobEventStore)
	sendLedger := appservice.NewSendLedgerService(sendLedgerRepository)
	inboxEvents := appservice.NewInboxEventService(inboxEventRepository)
	knowledgeCheckpoints := appservice.NewKnowledgeCheckpointService(knowledgeCheckpointRepository)
	knowledgeDiagnostics := appservice.NewKnowledgeWorkerDiagnosticsService(agentJobRepository, knowledgeCheckpointRepository)
	deliveryAdapterDiagnostics := appservice.NewDeliveryAdapterDiagnosticsService(deliveryAdapterDiagnosticsFromEnv())
	proactiveState := appservice.NewProactiveStateService(proactiveStateRepository)
	shadowQueries := appservice.NewShadowQueryService(shadowReader)
	inboxMetrics := appservice.NewInboxMetricsService(inboxEventRepository)
	agentJobMetrics := appservice.NewAgentJobMetricsService(agentJobRepository, agentJobEventStore)
	outboxMetrics := appservice.NewOutboxMetricsService(outboxRepository, outboxEventStore)
	queueBackend := appservice.NewQueueBackendServiceWithDiagnostics(queueBackendView, appservice.QueueBackendDiagnosticsDeps{
		Diagnostics:    workQueueDiagnostics,
		Compare:        queueCompare,
		OutboxRepo:     outboxRepository,
		OutboxEvents:   outboxEventStore,
		AgentJobRepo:   agentJobRepository,
		AgentJobEvents: agentJobEventStore,
	})
	runtimeOverview := appservice.NewRuntimeOverviewService(appservice.RuntimeOverviewDeps{
		QueueBackend:         queueBackend,
		DeliveryAdapters:     deliveryAdapterDiagnostics,
		SendLedger:           sendLedger,
		InboxMetrics:         inboxMetrics,
		AgentJobMetrics:      agentJobMetrics,
		OutboxMetrics:        outboxMetrics,
		KnowledgeDiagnostics: knowledgeDiagnostics,
	})

	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, sendLedger, inboxEvents)
	httptrigger.RegisterKnowledgeCheckpointRoutes(mux, knowledgeCheckpoints)
	httptrigger.RegisterKnowledgeDiagnosticsRoutes(mux, knowledgeDiagnostics)
	httptrigger.RegisterAgentJobEventRoutes(mux, agentJobEvents)
	httptrigger.RegisterInboxMetricsRoutes(mux, inboxMetrics)
	httptrigger.RegisterAgentJobMetricsRoutes(mux, agentJobMetrics)
	httptrigger.RegisterOutboxEventRoutes(mux, outboxEvents)
	httptrigger.RegisterOutboxMetricsRoutes(mux, outboxMetrics)
	httptrigger.RegisterQueueBackendRoutes(mux, queueBackend)
	httptrigger.RegisterDeliveryDispatchRoutes(mux, deliveryDispatch)
	httptrigger.RegisterDeliveryAdapterDiagnosticsRoutes(mux, deliveryAdapterDiagnostics)
	httptrigger.RegisterRuntimeOverviewRoutes(mux, runtimeOverview)
	httptrigger.RegisterProactiveStateRoutes(mux, proactiveState)

	log.Printf(
		"queue backend provider=%s mode=%s phase=%s external_active=%t",
		queueBackendView.Provider,
		queueBackendView.Mode,
		queueBackendView.MigrationPhase,
		queueBackendView.ExternalQueueActive,
	)
	log.Printf("agent job strict lease token=%t", boolEnv("AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN"))
	log.Printf("akashic agent runtime listening on %s (configured by %s); bot_ids=%s", addr, addrSource, strings.Join(botIDs, ","))
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func queueBackendActivePhase(mode string) string {
	if mode == "dual_read_compare" {
		return "dual_read_compare"
	}
	if mode == "external_lease" {
		return "external_lease_gate"
	}
	return "shadow_publish"
}

func envOrFirstDefaultWithSource(keys []string, fallback string) (string, string) {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value, key
		}
	}
	return fallback, "default"
}

func newDeliveryAdapters() ([]outport.DeliveryAdapter, error) {
	adapters := make([]outport.DeliveryAdapter, 0, 2)
	if token := telegramBotTokenFromEnv(); token != "" {
		channels := csvEnvOrDefault("AKASHIC_TELEGRAM_CHANNELS", []string{"telegram"})
		adapter, err := telegramdelivery.NewAdapter(telegramdelivery.Config{
			Token:    token,
			BaseURL:  strings.TrimSpace(os.Getenv("AKASHIC_TELEGRAM_API_BASE_URL")),
			Channels: channels,
		})
		if err != nil {
			return nil, err
		}
		adapters = append(adapters, adapter)
		log.Printf("telegram delivery adapter enabled for channels=%s", strings.Join(channels, ","))
	}
	onebotEndpoints := onebotEndpointsFromEnv()
	if len(onebotEndpoints) > 0 {
		adapter, err := onebotdelivery.NewAdapter(onebotdelivery.Config{Endpoints: onebotEndpoints})
		if err != nil {
			return nil, err
		}
		adapters = append(adapters, adapter)
		log.Printf("onebot delivery adapter enabled for channels=%s", strings.Join(sortedMapKeys(onebotEndpoints), ","))
	}
	return adapters, nil
}

func onebotEndpointsFromEnv() map[string]onebotdelivery.EndpointConfig {
	endpoints := make(map[string]onebotdelivery.EndpointConfig)
	accessTokens := keyValueCSVEnv("AKASHIC_ONEBOT_ACCESS_TOKENS")
	defaultToken := strings.TrimSpace(os.Getenv("AKASHIC_ONEBOT_ACCESS_TOKEN"))
	for channel, baseURL := range keyValueCSVEnv("AKASHIC_ONEBOT_HTTP_BASE_URLS") {
		if baseURL == "" {
			continue
		}
		endpoints[channel] = onebotdelivery.EndpointConfig{
			BaseURL:     baseURL,
			AccessToken: onebotTokenForChannel(channel, accessTokens, defaultToken),
		}
	}
	if baseURL := strings.TrimSpace(os.Getenv("AKASHIC_ONEBOT_HTTP_BASE_URL")); baseURL != "" {
		channels := csvEnvOrDefault("AKASHIC_ONEBOT_CHANNELS", []string{"qq"})
		for _, channel := range channels {
			channel = strings.TrimSpace(channel)
			if channel == "" {
				continue
			}
			endpoints[channel] = onebotdelivery.EndpointConfig{
				BaseURL:     baseURL,
				AccessToken: onebotTokenForChannel(channel, accessTokens, defaultToken),
			}
		}
	}
	for channel, webSocketURL := range mergedKeyValueCSVEnv("AKASHIC_ONEBOT_WS_URLS", "AKASHIC_ONEBOT_WEBSOCKET_URLS") {
		if webSocketURL == "" {
			continue
		}
		endpoint := endpoints[channel]
		endpoint.WebSocketURL = webSocketURL
		endpoint.AccessToken = onebotTokenForChannel(channel, accessTokens, defaultToken)
		endpoints[channel] = endpoint
	}
	if webSocketURL := firstEnvValue("AKASHIC_ONEBOT_WS_URL", "AKASHIC_ONEBOT_WEBSOCKET_URL"); webSocketURL != "" {
		channels := csvEnvOrDefault("AKASHIC_ONEBOT_CHANNELS", []string{"qq"})
		for _, channel := range channels {
			channel = strings.TrimSpace(channel)
			if channel == "" {
				continue
			}
			endpoint := endpoints[channel]
			endpoint.WebSocketURL = webSocketURL
			endpoint.AccessToken = onebotTokenForChannel(channel, accessTokens, defaultToken)
			endpoints[channel] = endpoint
		}
	}
	return endpoints
}

func onebotTokenForChannel(channel string, accessTokens map[string]string, defaultToken string) string {
	if token := strings.TrimSpace(accessTokens[channel]); token != "" {
		return token
	}
	return strings.TrimSpace(defaultToken)
}

func mergedKeyValueCSVEnv(keys ...string) map[string]string {
	result := make(map[string]string)
	for _, key := range keys {
		for itemKey, itemValue := range keyValueCSVEnv(key) {
			result[itemKey] = itemValue
		}
	}
	return result
}

func firstEnvValue(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func keyValueCSVEnv(key string) map[string]string {
	return parseKeyValueCSV(os.Getenv(key))
}

func parseKeyValueCSV(raw string) map[string]string {
	result := make(map[string]string)
	for _, item := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(item), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			result[key] = value
		}
	}
	return result
}

func sortedMapKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func newAgentJobRepository() (outport.AgentJobRepository, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_AGENT_JOBS_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			return memory.NewStore(), nil
		}
		return agentjobstore.NewStore(dsn)
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_AGENT_JOBS_PATH")); path != "" {
		return agentjobstore.NewStore(path)
	}
	return memory.NewStore(), nil
}

func newAgentJobEventStore() (outport.AgentJobEventStore, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_AGENT_JOB_EVENTS_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			return memory.NewStore(), nil
		}
		return agentjobeventstore.NewStore(dsn)
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_AGENT_JOB_EVENTS_PATH")); path != "" {
		return agentjobeventstore.NewStore(path)
	}
	return memory.NewStore(), nil
}

func newMediaAssetRepository() (outport.MediaAssetRepository, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_MEDIA_ASSETS_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			return memory.NewStore(), nil
		}
		return mediaassetstore.NewStore(dsn)
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_MEDIA_ASSETS_PATH")); path != "" {
		return mediaassetstore.NewStore(path)
	}
	return memory.NewStore(), nil
}

func newSendLedgerRepository() (outport.SendLedger, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_SEND_LEDGER_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			return memory.NewStore(), nil
		}
		return sendledgerstore.NewStore(dsn)
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_SEND_LEDGER_PATH")); path != "" {
		return sendledgerstore.NewStore(path)
	}
	return memory.NewStore(), nil
}

func newOutboxStore() (outport.OutboxRepository, outport.OutboxQueue, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_OUTBOX_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			store := memory.NewStore()
			return store, store, nil
		}
		store, err := outboxstore.NewStore(dsn)
		if err != nil {
			return nil, nil, err
		}
		return store, store, nil
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_OUTBOX_PATH")); path != "" {
		store, err := outboxstore.NewStore(path)
		if err != nil {
			return nil, nil, err
		}
		return store, store, nil
	}
	store := memory.NewStore()
	return store, store, nil
}

func newOutboxEventStore() (outport.OutboxDeliveryEventStore, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_OUTBOX_EVENTS_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			return memory.NewStore(), nil
		}
		return outboxeventstore.NewStore(dsn)
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_OUTBOX_EVENTS_PATH")); path != "" {
		return outboxeventstore.NewStore(path)
	}
	return memory.NewStore(), nil
}

func newInboxEventRepository() (outport.InboxEventRepository, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_INBOX_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			return memory.NewStore(), nil
		}
		return inboxstore.NewStore(dsn)
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_INBOX_PATH")); path != "" {
		return inboxstore.NewStore(path)
	}
	return memory.NewStore(), nil
}

func newKnowledgeCheckpointRepository() (outport.KnowledgeCheckpointRepository, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_KNOWLEDGE_CHECKPOINTS_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			return memory.NewStore(), nil
		}
		return knowledgecheckpointstore.NewStore(dsn)
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_KNOWLEDGE_CHECKPOINTS_PATH")); path != "" {
		return knowledgecheckpointstore.NewStore(path)
	}
	return memory.NewStore(), nil
}

func newProactiveStateRepository() (outport.ProactiveStateRepository, error) {
	if dsn := strings.TrimSpace(os.Getenv("AKASHIC_PROACTIVE_STATE_DSN")); dsn != "" {
		if strings.EqualFold(dsn, "memory") {
			return memory.NewStore(), nil
		}
		return proactivestate.NewStore(dsn)
	}

	if path := strings.TrimSpace(os.Getenv("AKASHIC_PROACTIVE_STATE_PATH")); path != "" {
		return proactivestate.NewStore(path)
	}
	return memory.NewStore(), nil
}

func newMediaAssetContentReader() (outport.MediaAssetContentReader, error) {
	roots := csvEnvOrDefault("AKASHIC_MEDIA_ASSET_ROOTS", nil)
	explicitRoots := len(roots) > 0
	if len(roots) == 0 {
		roots = defaultMediaAssetRoots()
	}
	return localmedia.NewReaderWithDiscoveredRoots(roots, !explicitRoots)
}

func defaultMediaAssetRoots() []string {
	starts := make([]string, 0, 2)
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	if executable, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(executable))
	}
	return defaultMediaAssetRootsFrom(starts)
}

func defaultMediaAssetRootsFrom(starts []string) []string {
	roots := make([]string, 0, 6)
	seen := make(map[string]struct{})
	for _, start := range starts {
		for _, repoRoot := range discoverAkashicRoots(start) {
			for _, candidate := range []string{
				filepath.Join(repoRoot, ".akashic-workspace", "uploads"),
				filepath.Join(repoRoot, ".akashic-workspace", "generated_images"),
				filepath.Join(repoRoot, "generated_images"),
			} {
				cleaned := filepath.Clean(candidate)
				key := strings.ToLower(cleaned)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				roots = append(roots, cleaned)
			}
		}
	}
	return roots
}

func discoverAkashicRoots(start string) []string {
	if strings.TrimSpace(start) == "" {
		return nil
	}
	current, err := filepath.Abs(start)
	if err != nil {
		return nil
	}
	current = filepath.Clean(current)
	roots := make([]string, 0, 1)
	for {
		if isAkashicRoot(current) {
			roots = append(roots, current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return roots
}

func isAkashicRoot(path string) bool {
	if stat, err := os.Stat(filepath.Join(path, ".akashic-workspace")); err == nil && stat.IsDir() {
		return true
	}
	if _, err := os.Stat(filepath.Join(path, "pyproject.toml")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "services", "agent-runtime", "go.mod")); err != nil {
		return false
	}
	return true
}

func csvEnvOrDefault(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
