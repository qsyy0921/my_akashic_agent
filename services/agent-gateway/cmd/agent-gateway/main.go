package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/port/out"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/service"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/service"
	agentjobstore "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/infrastructure/agentjobstore"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/infrastructure/auditjsonl"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/infrastructure/memory"
	httptrigger "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/trigger/http"
)

func main() {
	addr := envOrDefault("AKASHIC_GATEWAY_ADDR", ":8780")
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

	ingestor := appservice.NewMessageIngestService(
		store,
		auditLog,
		store,
		store,
		classifier,
		loopGuard,
	)
	sender := appservice.NewMessageSendService(store, store, store, store)
	imageJobs := appservice.NewImageJobServiceWithAgentJobs(store, store, store)
	outbox := appservice.NewOutboxService(store, store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(agentJobRepository)
	shadowQueries := appservice.NewShadowQueryService(shadowReader)

	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs)

	log.Printf("akashic agent gateway listening on %s; bot_ids=%s", addr, strings.Join(botIDs, ","))
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
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
