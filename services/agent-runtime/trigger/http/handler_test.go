package httptrigger_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/localmedia"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
	httptrigger "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/trigger/http"
)

func TestJSONHandlersDeclareUTF8(t *testing.T) {
	response := httptest.NewRecorder()

	httptrigger.HealthHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("expected utf-8 json content type, got %q", got)
	}
}

func TestShadowIngestEndpointAuditsWithoutAgentInbound(t *testing.T) {
	store := memory.NewStore()
	ingestor := appservice.NewMessageIngestServiceWithMediaAssets(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
		store,
	)
	sender := appservice.NewMessageSendService(store, store, store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	outbox := appservice.NewOutboxService(store, store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))

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
		"content": "/ask hello",
		"attachments": []map[string]any{{
			"id":         "asset:qq:image:1049511700:msg-1:1",
			"kind":       "image",
			"url":        "E:/agent/akashic/.akashic-workspace/uploads/qq-image.png",
			"mime_type":  "image/png",
			"name":       "qq-image.png",
			"size_bytes": 123,
		}},
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
	if len(store.MediaAssets()) != 1 {
		t.Fatalf("expected 1 media asset, got %d", len(store.MediaAssets()))
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
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))

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

func TestInboxEndpointListsRawObservedGroupEvents(t *testing.T) {
	store := memory.NewStore()
	ingestor := appservice.NewMessageIngestServiceWithRuntimeStores(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
		store,
		store,
	)
	sender := appservice.NewMessageSendService(store, store, store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	outbox := appservice.NewOutboxService(store, store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	inboxEvents := appservice.NewInboxEventService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), inboxEvents)

	body := map[string]any{
		"event_id": "qq:1049511700:group:27234224:msg-inbox-1",
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "1049511700",
			"conversation_id":   "27234224",
			"conversation_type": "group",
		},
		"sender": map[string]any{
			"id":   "2948770636",
			"kind": "human",
		},
		"content":   "raw inbox hardware message",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"metadata":  map[string]string{"observe_only": "true", "seq": "1"},
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/inbox?channel_kind=qq&conversation_id=27234224&conversation_type=group&observe_only=true&limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected inbox list 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("raw inbox hardware message")) {
		t.Fatalf("inbox response missing raw event content: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"observe_only":true`)) {
		t.Fatalf("inbox response missing observe-only flag: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/inbox?channel_kind=qq&conversation_id=27234224&conversation_type=group&observe_only=true&after_seq=0&order=asc&limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected cursor inbox list 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("raw inbox hardware message")) {
		t.Fatalf("cursor inbox response missing raw event content: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/inbox/qq:1049511700:group:27234224:msg-inbox-1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected inbox get 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"event_id":"qq:1049511700:group:27234224:msg-inbox-1"`)) {
		t.Fatalf("inbox get response missing event id: %s", response.Body.String())
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
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))

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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/outbox-http-1/failed", bytes.NewReader([]byte(`{"error_kind":"platform_timeout","error_message":"platform timeout"}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected failed 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"failed"`)) {
		t.Fatalf("failed response missing status: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"error_kind":"platform_timeout"`)) {
		t.Fatalf("failed response missing error kind: %s", response.Body.String())
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

func TestOutboxEndpointDeadLettersNonRetryableFailureKind(t *testing.T) {
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
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))

	body := map[string]any{
		"event_id": "outbox-http-nonretryable",
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "1049511700",
			"conversation_id":   "2365524513",
			"conversation_type": "private",
		},
		"content":   "bad route",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"metadata":  map[string]string{"max_attempts": "3"},
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/outbox-http-nonretryable/dispatching", bytes.NewReader([]byte(`{}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected dispatching 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/outbox-http-nonretryable/failed", bytes.NewReader([]byte(`{"error_kind":"validation_error","error_message":"invalid recipient"}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected failed 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"dead_lettered"`)) {
		t.Fatalf("failed response missing dead-letter status: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"error_kind":"validation_error"`)) {
		t.Fatalf("failed response missing validation kind: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/outbox-http-nonretryable/retry", bytes.NewReader([]byte(`{}`))))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected retry 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestOutboxLeaseNextEndpointLeasesQueuedDelivery(t *testing.T) {
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
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))

	body := map[string]any{
		"event_id": "outbox-http-lease-1",
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "1049511700",
			"conversation_id":   "2365524513",
			"conversation_type": "private",
		},
		"content":   "generated image is ready",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/lease-next", bytes.NewReader([]byte(`{"worker_id":"qq-dispatcher","ttl_seconds":60}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected lease-next 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"event_id":"outbox-http-lease-1"`)) {
		t.Fatalf("lease response missing event id: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"dispatching"`)) {
		t.Fatalf("lease response missing dispatching status: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"lease_owner":"qq-dispatcher"`)) {
		t.Fatalf("lease response missing owner: %s", response.Body.String())
	}
}

func TestDeliveryDispatchPlanEndpointBuildsSendSteps(t *testing.T) {
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
	deliveryDispatch := appservice.NewDeliveryDispatchService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))
	httptrigger.RegisterDeliveryDispatchRoutes(mux, deliveryDispatch)

	body := map[string]any{
		"event_id": "outbox-dispatch-plan-1",
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "2365524513",
			"conversation_id":   "1049511700",
			"conversation_type": "private",
		},
		"content":   "hello",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"attachments": []map[string]any{
			{"kind": "image", "url": "file:///E:/agent/akashic/.tmp/a.png"},
			{"kind": "file", "url": "file:///E:/agent/akashic/.tmp/a.pdf"},
		},
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

	request := []byte(`{"event_id":"outbox-dispatch-plan-1","channel_by_account":{"2365524513":"qq_2365524513"}}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/delivery-dispatch/plan", bytes.NewReader(request)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected plan 200, got %d: %s", response.Code, response.Body.String())
	}
	bodyText := response.Body.String()
	for _, expected := range []string{
		`"event_id":"outbox-dispatch-plan-1"`,
		`"channel":"qq_2365524513"`,
		`"step_count":2`,
		`"kind":"image"`,
		`"image":"E:/agent/akashic/.tmp/a.png"`,
		`"kind":"file"`,
		`"file":"E:/agent/akashic/.tmp/a.pdf"`,
	} {
		if !strings.Contains(bodyText, expected) {
			t.Fatalf("plan response missing %s: %s", expected, bodyText)
		}
	}
}

func TestMediaAssetEndpointRegistersListsAndServesContentRoute(t *testing.T) {
	assetRoot := t.TempDir()
	assetPath := filepath.Join(assetRoot, "qq-image.txt")
	if err := os.WriteFile(assetPath, []byte("qq image bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	contentReader, err := localmedia.NewReader([]string{assetRoot})
	if err != nil {
		t.Fatal(err)
	}
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
	mediaAssets := appservice.NewMediaAssetServiceWithContent(store, contentReader)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))

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
		"url":               assetPath,
		"mime_type":         "text/plain",
		"name":              "qq-image.txt",
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
	if !bytes.Contains(response.Body.Bytes(), []byte("qq-image.txt")) {
		t.Fatalf("list response missing file name: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/v1/media-assets?channel_kind=qq&conversation_id=27234224&conversation_type=group&source_message_id_suffix=498&limit=10", nil),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected filtered list 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("qq-image.txt")) {
		t.Fatalf("filtered list response missing file name: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/media-assets/"+assetID+"/content", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected content 200, got %d: %s", response.Code, response.Body.String())
	}
	if response.Body.String() != "qq image bytes" {
		t.Fatalf("unexpected content body: %s", response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/plain" {
		t.Fatalf("unexpected content type: %s", got)
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
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))

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

func TestAgentJobEventsEndpointListsLifecycleStream(t *testing.T) {
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
	agentJobs := appservice.NewAgentJobServiceWithEvents(store, store)
	jobEvents := appservice.NewAgentJobEventService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))
	httptrigger.RegisterAgentJobEventRoutes(mux, jobEvents)

	body := []byte(`{
		"job_id":"job-events-http-1",
		"job_type":"rag_ingest",
		"agent_id":"main",
		"route":{
			"platform":"qq",
			"account_id":"1049511700",
			"conversation_id":"27234224",
			"conversation_type":"group"
		},
		"payload":{"source":"group"},
		"max_attempts":2
	}`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected job accepted, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/lease-next", bytes.NewReader([]byte(`{"worker_id":"worker-events","job_type":"rag_ingest","ttl_seconds":60}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected lease 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/job-events?job_id=job-events-http-1&limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected events 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"event_type":"leased"`)) {
		t.Fatalf("events response missing leased event: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"event_type":"created"`)) {
		t.Fatalf("events response missing created event: %s", response.Body.String())
	}
}

func TestKnowledgeCheckpointEndpointUpsertsAndGetsCheckpoint(t *testing.T) {
	store := memory.NewStore()
	mux := http.NewServeMux()
	httptrigger.RegisterKnowledgeCheckpointRoutes(
		mux,
		appservice.NewKnowledgeCheckpointService(store),
	)

	body := []byte(`{"cursor":42,"metadata":{"group_id":"27234224","dataset_id":"ds1"}}`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPut, "/v1/knowledge-checkpoints/ragflow%3Aqq%3A27234224%3Ads1", bytes.NewReader(body)),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected checkpoint upsert 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"cursor":42`)) {
		t.Fatalf("checkpoint response missing cursor: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/v1/knowledge-checkpoints/ragflow%3Aqq%3A27234224%3Ads1", nil),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected checkpoint get 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"checkpoint_id":"ragflow:qq:27234224:ds1"`)) {
		t.Fatalf("checkpoint get missing id: %s", response.Body.String())
	}

	body = []byte(`{"cursor":43,"metadata":{"group_id":"3219982","dataset_id":"ds1"}}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPut, "/v1/knowledge-checkpoints/ragflow%3Aqq%3A3219982%3Ads1", bytes.NewReader(body)),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected second checkpoint upsert 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/v1/knowledge-checkpoints?prefix=ragflow%3Aqq%3A&limit=10", nil),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected checkpoint list 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"checkpoint_id":"ragflow:qq:3219982:ds1"`)) {
		t.Fatalf("checkpoint list missing newest item: %s", response.Body.String())
	}
}

func TestKnowledgeWorkerDiagnosticsEndpointSummarizesJobsAndCheckpoints(t *testing.T) {
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
	sendLedger := appservice.NewSendLedgerService(store)
	inboxEvents := appservice.NewInboxEventService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	checkpoints := appservice.NewKnowledgeCheckpointService(store)
	diagnostics := appservice.NewKnowledgeWorkerDiagnosticsService(store, store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, sendLedger, inboxEvents)
	httptrigger.RegisterKnowledgeCheckpointRoutes(mux, checkpoints)
	httptrigger.RegisterKnowledgeDiagnosticsRoutes(mux, diagnostics)

	jobBody := []byte(`{
		"job_id":"rag_ingest:qq:3219982:ds1:1",
		"job_type":"rag_ingest",
		"agent_id":"knowledge-worker",
		"route":{
			"platform":"qq",
			"account_id":"1049511700",
			"conversation_id":"3219982",
			"conversation_type":"group"
		},
		"payload":{"group_id":"3219982","dataset_id":"ds1"},
		"max_attempts":2
	}`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs", bytes.NewReader(jobBody)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected job accepted, got %d: %s", response.Code, response.Body.String())
	}

	checkpointBody := []byte(`{"cursor":99,"metadata":{"group_id":"3219982","dataset_id":"ds1"}}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPut, "/v1/knowledge-checkpoints/ragflow%3Aqq%3A3219982%3Ads1", bytes.NewReader(checkpointBody)),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected checkpoint upsert 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/v1/knowledge-worker-diagnostics?limit=10&stale_after_seconds=60", nil),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected diagnostics 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"job_type":"rag_ingest"`)) {
		t.Fatalf("diagnostics missing rag worker: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"checkpoint_id":"ragflow:qq:3219982:ds1"`)) {
		t.Fatalf("diagnostics missing checkpoint: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status_counts":{"pending":1}`)) {
		t.Fatalf("diagnostics missing pending count: %s", response.Body.String())
	}
}

func TestSendLedgerEndpointRecordsListsAndChecksRecentEcho(t *testing.T) {
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
	sendLedger := appservice.NewSendLedgerService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, sendLedger, appservice.NewInboxEventService(store))

	body := map[string]any{
		"from_bot_id":     "1049511700",
		"conversation_id": "2365524513",
		"content":         "image generated",
		"timestamp":       time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/send-ledger/records", bytes.NewReader(raw)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected send ledger accepted, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"from_bot_id":"1049511700"`)) {
		t.Fatalf("record response missing bot id: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/send-ledger/recent?from_bot_id=1049511700&conversation_id=2365524513&content=image+generated&window_seconds=60", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected recent 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"recent":true`)) {
		t.Fatalf("recent response did not detect echo: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/send-ledger/records?from_bot_id=1049511700&conversation_id=2365524513&limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"conversation_id":"2365524513"`)) {
		t.Fatalf("list response missing conversation id: %s", response.Body.String())
	}
}

func TestProactiveStateEndpointsRecordAndQuerySchedulingState(t *testing.T) {
	store := memory.NewStore()
	proactiveState := appservice.NewProactiveStateService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterProactiveStateRoutes(mux, proactiveState)

	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	body := []byte(`{"session_key":"telegram:1","delivery_key":"delivery-a","timestamp":"` + now + `"}`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/proactive/deliveries", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected delivery record 202, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/proactive/deliveries/duplicate?session_key=telegram:1&delivery_key=delivery-a&window_hours=24&timestamp=2026-05-30T11:00:00Z", nil)
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected duplicate 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"duplicate":true`)) {
		t.Fatalf("duplicate response missing true flag: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/v1/proactive/deliveries/count?session_key=telegram:1&window_hours=24&timestamp=2026-05-30T11:00:00Z", nil)
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected count 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"count":1`)) {
		t.Fatalf("count response missing one delivery: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/proactive/context-only", bytes.NewReader([]byte(`{"session_key":"telegram:1","timestamp":"2026-05-30T11:30:00Z"}`))))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected context-only 202, got %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/proactive/context-only/last?session_key=telegram:1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected context last 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"found":true`)) {
		t.Fatalf("context last response missing found flag: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/proactive/drift-runs", bytes.NewReader([]byte(`{"session_key":"telegram:1","timestamp":"2026-05-30T12:00:00Z"}`))))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected drift 202, got %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/proactive/drift-runs/last?session_key=telegram:1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected drift last 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"key":"drift_last_at"`)) {
		t.Fatalf("drift response missing marker key: %s", response.Body.String())
	}
}
