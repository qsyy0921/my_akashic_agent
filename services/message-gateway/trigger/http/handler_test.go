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
	sender := appservice.NewMessageSendService(store, store, store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	outbox := appservice.NewOutboxService(store, store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs)

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
	sender := appservice.NewMessageSendService(store, store, store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	outbox := appservice.NewOutboxService(store, store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs)

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
		"content": "shadow group message",
		"attachments": []map[string]any{
			{
				"id":        "asset:qq:image:1",
				"kind":      "image",
				"url":       "file:///E:/agent/akashic/.tmp/image.png",
				"mime_type": "image/png",
				"name":      "image.png",
			},
		},
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
	if !bytes.Contains(response.Body.Bytes(), []byte("image.png")) {
		t.Fatalf("observed response missing attachment metadata: %s", response.Body.String())
	}
}

func TestOutboxEndpointTracksDeliveryFailureAndRetry(t *testing.T) {
	store := memory.NewStore()
	ingestor := appservice.NewMessageIngestService(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
	)
	sender := appservice.NewMessageSendService(store, store, store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	outbox := appservice.NewOutboxService(store, store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs)

	body := map[string]any{
		"event_id": "outbox-http-1",
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "1049511700",
			"conversation_id":   "2365524513",
			"conversation_type": "private",
		},
		"content":   "generated image is ready",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"metadata":  map[string]string{"max_attempts": "2"},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbound", bytes.NewReader(raw)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected outbound accepted, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/outbox-http-1/dispatching", bytes.NewReader([]byte(`{}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected dispatching 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"dispatching"`)) {
		t.Fatalf("dispatching response missing status: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/outbox-http-1/failed", bytes.NewReader([]byte(`{"error_message":"platform timeout"}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected failed 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"failed"`)) {
		t.Fatalf("failed response missing status: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/outbox-http-1/retry", bytes.NewReader([]byte(`{}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected retry 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"queued"`)) {
		t.Fatalf("retry response missing queued status: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/outbox?limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("outbox-http-1")) {
		t.Fatalf("list response missing delivery: %s", response.Body.String())
	}
}

func TestMediaAssetEndpointRegistersListsAndRejectsContentRoute(t *testing.T) {
	store := memory.NewStore()
	ingestor := appservice.NewMessageIngestService(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
	)
	sender := appservice.NewMessageSendService(store, store, store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	outbox := appservice.NewOutboxService(store, store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs)

	body := map[string]any{
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "1049511700",
			"conversation_id":   "27234224",
			"conversation_type": "group",
		},
		"source_message_id": "qq:gqq:27234224:498",
		"sender_id":         "2948770636",
		"kind":              "image",
		"url":               "https://example.invalid/qq-image.png",
		"mime_type":         "image/png",
		"name":              "qq-image.png",
		"index":             1,
		"timestamp":         time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/media-assets", bytes.NewReader(raw)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected media asset accepted, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("asset:qq:1049511700:group:27234224:qq:gqq:27234224:498:1")) {
		t.Fatalf("register response missing generated asset id: %s", response.Body.String())
	}

	assetID := "asset:qq:1049511700:group:27234224:qq:gqq:27234224:498:1"
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/media-assets/"+assetID, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected get 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"account_id":"1049511700"`)) {
		t.Fatalf("get response missing account id: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/media-assets?limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("qq-image.png")) {
		t.Fatalf("list response missing file name: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/media-assets/"+assetID+"/content", nil))
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("content route must not be enabled yet, got %d: %s", response.Code, response.Body.String())
	}
}

func TestAgentJobEndpointCreatesLeasesAndCompletesJob(t *testing.T) {
	store := memory.NewStore()
	ingestor := appservice.NewMessageIngestService(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
	)
	sender := appservice.NewMessageSendService(store, store, store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	outbox := appservice.NewOutboxService(store, store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs)

	body := map[string]any{
		"job_id":   "job-http-1",
		"job_type": "rag_ingest",
		"agent_id": "main",
		"route": map[string]any{
			"platform":          "qq",
			"account_id":        "1049511700",
			"conversation_id":   "27234224",
			"conversation_type": "group",
		},
		"source_event_ids": []string{"qq:gqq:27234224:1"},
		"source_asset_ids": []string{"asset:1"},
		"payload":          map[string]string{"source": "group"},
		"max_attempts":     2,
		"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs", bytes.NewReader(raw)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected job accepted, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"pending"`)) {
		t.Fatalf("create response missing pending status: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/lease-next", bytes.NewReader([]byte(`{"worker_id":"worker-http","job_type":"rag_ingest","ttl_seconds":60}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected lease-next 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"leased"`)) {
		t.Fatalf("lease response missing leased status: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/job-http-1/running", bytes.NewReader([]byte(`{}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected running 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/job-http-1/succeeded", bytes.NewReader([]byte(`{"result":{"indexed":"true"}}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected succeeded 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"succeeded"`)) {
		t.Fatalf("succeeded response missing status: %s", response.Body.String())
	}
}
