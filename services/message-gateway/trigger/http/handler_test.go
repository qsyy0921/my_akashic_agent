package httptrigger_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/app/service"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/service"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/infrastructure/memory"
	httptrigger "github.com/kachofugetsu09/akashic-agent/services/message-gateway/trigger/http"
)

func TestShadowIngestEndpointAuditsWithoutAgentInbound(t *testing.T) {
	store := memory.NewStore()
	ingestor := appservice.NewMessageIngestService(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
	)
	sender := appservice.NewMessageSendService(store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs)

	body := map[string]any{
		"event_id": "qq:2365524513:private:1049511700:msg-1",
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "2365524513",
			"conversation_id":   "1049511700",
			"conversation_type": "private",
		},
		"sender": map[string]any{
			"id":   "1049511700",
			"kind": "human",
		},
		"content":   "/ask hello",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"metadata":  map[string]string{"shadow_mode": "true"},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/shadow/inbound", bytes.NewReader(raw))
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", response.Code, response.Body.String())
	}
	if len(store.Observed()) != 1 {
		t.Fatalf("expected 1 observed event, got %d", len(store.Observed()))
	}
	if len(store.AgentInbound()) != 0 {
		t.Fatalf("shadow endpoint must not publish agent inbound, got %d", len(store.AgentInbound()))
	}
	if len(store.Audits()) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(store.Audits()))
	}
}

func TestShadowObservedEndpointReturnsRecentEvents(t *testing.T) {
	store := memory.NewStore()
	ingestor := appservice.NewMessageIngestService(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
	)
	sender := appservice.NewMessageSendService(store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs)

	body := map[string]any{
		"event_id": "qq:2365524513:group:27234224:msg-1",
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "2365524513",
			"conversation_id":   "27234224",
			"conversation_type": "group",
		},
		"sender": map[string]any{
			"id":   "2948770636",
			"kind": "human",
		},
		"content":   "shadow group message",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"metadata":  map[string]string{"observe_only": "true"},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	mux.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/v1/shadow/inbound", bytes.NewReader(raw)),
	)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/shadow/observed?limit=10", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("shadow group message")) {
		t.Fatalf("observed response missing event content: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("decision_action")) {
		t.Fatalf("observed response missing decision fields: %s", response.Body.String())
	}
}
