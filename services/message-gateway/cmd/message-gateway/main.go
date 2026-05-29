package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	appservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/service"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/service"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/infrastructure/memory"
	httptrigger "github.com/kachofugetsu09/akashic-agent/services/message-gateway/trigger/http"
)

func main() {
	addr := envOrDefault("AKASHIC_GATEWAY_ADDR", ":8780")
	botIDs := csvEnvOrDefault("AKASHIC_BOT_IDS", []string{"1049511700", "2365524513"})

	store := memory.NewStore()
	classifier := domainservice.NewProvenanceClassifier(botIDs)
	loopGuard := domainservice.NewLoopGuard(botIDs, 15*time.Second, 6)

	ingestor := appservice.NewMessageIngestService(
		store,
		store,
		store,
		store,
		classifier,
		loopGuard,
	)
	sender := appservice.NewMessageSendService(store, store)
	imageJobs := appservice.NewImageJobService(store, store)

	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, sender, imageJobs)

	log.Printf("akashic message gateway listening on %s; bot_ids=%s", addr, strings.Join(botIDs, ","))
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
