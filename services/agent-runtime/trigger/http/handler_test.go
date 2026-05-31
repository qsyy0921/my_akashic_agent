package httptrigger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
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

func TestQueueBackendEndpointReturnsReadOnlyDiagnostics(t *testing.T) {
	viewer := appservice.NewQueueBackendService(queryQueueBackendViewForTest())
	mux := http.NewServeMux()
	httptrigger.RegisterQueueBackendRoutes(mux, viewer)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/queue-backend", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"provider":"nats_jetstream"`)) {
		t.Fatalf("response missing provider: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"external_queue_active":false`)) {
		t.Fatalf("response must show adapter inactive: %s", response.Body.String())
	}
}

func TestRuntimeOverviewEndpointReturnsGoOwnedAggregate(t *testing.T) {
	mux := http.NewServeMux()
	httptrigger.RegisterRuntimeOverviewRoutes(mux, staticRuntimeOverviewViewer{
		view: query.RuntimeOverviewView{
			Summary: map[string]any{
				"queue_backend_provider": "nats_jetstream",
				"inbox_metric_events":    9,
			},
			Cards: []query.RuntimeOverviewCardView{{
				ID:     "inbox_metrics",
				Label:  "Inbox Metrics",
				Value:  9,
				Status: "ok",
			}},
			Status: query.RuntimeOverviewStatusView{
				RuntimeAvailable: true,
				HealthAvailable:  true,
			},
		},
	})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/runtime-overview?limit=50&event_limit=10&stale_after_seconds=60", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"queue_backend_provider":"nats_jetstream"`)) {
		t.Fatalf("response missing summary: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"id":"inbox_metrics"`)) {
		t.Fatalf("response missing card: %s", response.Body.String())
	}
}

func TestRuntimeWorkerDiagnosticsEndpointReturnsReadOnlyWorkers(t *testing.T) {
	viewer := appservice.NewRuntimeWorkerDiagnosticsService(query.RuntimeWorkerDiagnosticsView{
		Workers: []query.RuntimeWorkerView{{
			Name:            "outbox_delivery_worker",
			Kind:            "outbox_delivery",
			Enabled:         true,
			Running:         true,
			WorkerID:        "agent-runtime-outbox-worker",
			IntervalSeconds: 2,
		}},
	})
	mux := http.NewServeMux()
	httptrigger.RegisterRuntimeWorkerDiagnosticsRoutes(mux, viewer)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/runtime-workers", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		`"name":"outbox_delivery_worker"`,
		`"worker_id":"agent-runtime-outbox-worker"`,
		`"running":1`,
	} {
		if !bytes.Contains(response.Body.Bytes(), []byte(expected)) {
			t.Fatalf("response missing %s: %s", expected, response.Body.String())
		}
	}
}

func TestAgentWorkerStatusEndpointReportsAndListsWorkers(t *testing.T) {
	manager := appservice.NewAgentWorkerStatusService()
	mux := http.NewServeMux()
	httptrigger.RegisterAgentWorkerStatusRoutes(mux, manager)

	body := strings.NewReader(`{
		"worker_id":"worker-a",
		"worker_type":"knowledge",
		"status":"running",
		"current_job_id":"group-memory-1",
		"processed_total":2,
		"failed_total":1,
		"source":"python",
		"timestamp":"2026-05-31T10:00:00Z"
	}`)
	report := httptest.NewRecorder()
	mux.ServeHTTP(report, httptest.NewRequest(http.MethodPost, "/v1/agent-worker-statuses/report", body))

	if report.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", report.Code, report.Body.String())
	}
	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/v1/agent-worker-statuses", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", list.Code, list.Body.String())
	}
	for _, expected := range []string{
		`"worker_id":"worker-a"`,
		`"worker_type":"knowledge"`,
		`"running":1`,
		`"side_effect":"none"`,
	} {
		if !bytes.Contains(list.Body.Bytes(), []byte(expected)) {
			t.Fatalf("response missing %s: %s", expected, list.Body.String())
		}
	}
}

func TestRuntimeConfigEndpointReturnsSanitizedReadOnlyConfig(t *testing.T) {
	viewer := appservice.NewRuntimeConfigService(query.RuntimeConfigView{
		Runtime: query.RuntimeProcessConfigView{
			Address:       ":8780",
			AddressSource: "AKASHIC_RUNTIME_ADDR",
			BotIDs:        []string{"1049511700", "2365524513"},
		},
		Delivery: query.RuntimeDeliveryConfigView{
			OneBotExpectedChannels: []string{"qq", "qq_2365524513"},
			OneBotEndpoints: []query.RuntimeOneBotEndpointConfigView{{
				Channel:               "qq_2365524513",
				Transport:             "websocket",
				WebSocketConfigured:   true,
				AccessTokenConfigured: true,
				Endpoint:              "ws://127.0.0.1:3002",
			}},
			OneBotReadyForHealthProbe: true,
		},
		Environment: []query.RuntimeEnvVarView{{
			Key:           "AKASHIC_ONEBOT_ACCESS_TOKENS",
			Present:       true,
			Secret:        true,
			ValueRedacted: "qq_2365524513=redacted",
		}},
		Readiness:  query.RuntimeConfigReadinessView{DeliveryAdaptersConfigured: true, OneBotConfigured: true},
		SideEffect: "none",
	})
	mux := http.NewServeMux()
	httptrigger.RegisterRuntimeConfigRoutes(mux, viewer)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/runtime-config", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{
		`"address":":8780"`,
		`"side_effect":"none"`,
		`"key":"AKASHIC_ONEBOT_ACCESS_TOKENS"`,
		`"value_redacted":"qq_2365524513=redacted"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("runtime config response missing %s: %s", expected, body)
		}
	}
	if strings.Contains(body, "NcatBot") {
		t.Fatalf("runtime config leaked token: %s", body)
	}
}

func TestProactiveAnyActionQuotaEndpointSnapshotsAndRecords(t *testing.T) {
	manager := appservice.NewProactiveStateService(memory.NewStore())
	mux := http.NewServeMux()
	httptrigger.RegisterProactiveStateRoutes(mux, manager)

	snapshot := httptest.NewRecorder()
	mux.ServeHTTP(snapshot, httptest.NewRequest(http.MethodGet, "/v1/proactive/anyaction/quota?reset_hour=12&timezone=Asia%2FShanghai&timestamp=2026-05-30T03:00:00Z", nil))
	if snapshot.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", snapshot.Code, snapshot.Body.String())
	}
	if !bytes.Contains(snapshot.Body.Bytes(), []byte(`"window_key":"2026-05-29@12@Asia/Shanghai"`)) {
		t.Fatalf("response missing window key: %s", snapshot.Body.String())
	}

	record := httptest.NewRecorder()
	body := strings.NewReader(`{"reset_hour":12,"timezone":"Asia/Shanghai","timestamp":"2026-05-30T03:01:00Z"}`)
	mux.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/v1/proactive/anyaction/actions", body))
	if record.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", record.Code, record.Body.String())
	}
	if !bytes.Contains(record.Body.Bytes(), []byte(`"used":1`)) {
		t.Fatalf("response missing used count: %s", record.Body.String())
	}
}

func TestDeliveryAdaptersEndpointReturnsReadOnlyDiagnostics(t *testing.T) {
	viewer := appservice.NewDeliveryAdapterDiagnosticsService([]query.DeliveryAdapterDiagnosticsView{
		{
			Provider:              "onebot",
			Channel:               "qq_2365524513",
			Transport:             "websocket",
			Enabled:               true,
			EndpointConfigured:    true,
			AccessTokenConfigured: true,
			Endpoint:              "ws://127.0.0.1:3002",
		},
	})
	mux := http.NewServeMux()
	httptrigger.RegisterDeliveryAdapterDiagnosticsRoutes(mux, viewer)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/delivery-adapters", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"provider":"onebot"`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"channel":"qq_2365524513"`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"transport":"websocket"`)) {
		t.Fatalf("response missing adapter diagnostics: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"access_token_configured":true`)) {
		t.Fatalf("response should expose token presence only: %s", response.Body.String())
	}
}

func TestObserveTargetsEndpointSyncsAndListsConfiguredTargets(t *testing.T) {
	manager := appservice.NewObserveTargetService()
	mux := http.NewServeMux()
	httptrigger.RegisterObserveTargetRoutes(mux, manager)

	body := []byte(`{
		"source": "python_config",
		"targets": [{
			"channel": {
				"kind": "qq",
				"account_id": "1049511700",
				"conversation_id": "27234224",
				"conversation_type": "group"
			},
			"observe_only": true,
			"reply_allowed": false,
			"enabled": true,
			"metadata": {"channel_name": "qq"}
		}]
	}`)
	syncResponse := httptest.NewRecorder()
	mux.ServeHTTP(syncResponse, httptest.NewRequest(http.MethodPut, "/v1/observe-targets/sync", bytes.NewReader(body)))
	if syncResponse.Code != http.StatusOK {
		t.Fatalf("expected sync 200, got %d: %s", syncResponse.Code, syncResponse.Body.String())
	}
	if !bytes.Contains(syncResponse.Body.Bytes(), []byte(`"side_effect":"none"`)) ||
		!bytes.Contains(syncResponse.Body.Bytes(), []byte(`"target_id":"qq:1049511700:group:27234224"`)) {
		t.Fatalf("sync response missing target diagnostics: %s", syncResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/v1/observe-targets", nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}
	for _, expected := range []string{
		`"observe_only":1`,
		`"reply_allowed":0`,
		`"groups":1`,
		`"channel_name":"qq"`,
	} {
		if !bytes.Contains(listResponse.Body.Bytes(), []byte(expected)) {
			t.Fatalf("list response missing %s: %s", expected, listResponse.Body.String())
		}
	}
}

func TestObserveCaptureDiagnosticsEndpointReturnsReadOnlyCoverage(t *testing.T) {
	mux := http.NewServeMux()
	httptrigger.RegisterObserveCaptureDiagnosticsRoutes(mux, staticObserveCaptureDiagnostics{
		view: query.ObserveCaptureDiagnosticsView{
			Targets: []query.ObserveCaptureTargetDiagnosticsView{{
				TargetID: "qq:1049511700:group:27234224",
				Channel: query.ObserveTargetChannelView{
					Kind:             "qq",
					AccountID:        "1049511700",
					ConversationID:   "27234224",
					ConversationType: "group",
				},
				Enabled:           true,
				ObserveOnly:       true,
				ReceiverConnected: true,
				Status:            "ok",
				TextEvents:        1,
				ImageAssets:       1,
				FileAssets:        1,
			}},
			Totals:     map[string]int{"targets": 1, "ready": 1, "text_covered": 1, "image_covered": 1, "file_covered": 1},
			SideEffect: "none",
		},
	})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/observe-capture-diagnostics?limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected diagnostics 200, got %d: %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		`"side_effect":"none"`,
		`"target_id":"qq:1049511700:group:27234224"`,
		`"text_covered":1`,
		`"image_covered":1`,
		`"file_covered":1`,
	} {
		if !bytes.Contains(response.Body.Bytes(), []byte(expected)) {
			t.Fatalf("observe capture response missing %s: %s", expected, response.Body.String())
		}
	}
}

func TestReceiverStatusEndpointReportsAndListsReceivers(t *testing.T) {
	manager := appservice.NewReceiverStatusService()
	mux := http.NewServeMux()
	httptrigger.RegisterReceiverStatusRoutes(mux, manager)

	body := []byte(`{
		"kind": "telegram",
		"channel_name": "telegram",
		"account_id": "7689386159",
		"status": "suspended",
		"reason": "getupdates_conflict",
		"source": "python_channel",
		"metadata": {"polling": "stopped"}
	}`)
	reportResponse := httptest.NewRecorder()
	mux.ServeHTTP(reportResponse, httptest.NewRequest(http.MethodPost, "/v1/receiver-statuses/report", bytes.NewReader(body)))
	if reportResponse.Code != http.StatusOK {
		t.Fatalf("expected report 200, got %d: %s", reportResponse.Code, reportResponse.Body.String())
	}
	if !bytes.Contains(reportResponse.Body.Bytes(), []byte(`"receiver_id":"telegram:7689386159:telegram"`)) ||
		!bytes.Contains(reportResponse.Body.Bytes(), []byte(`"suspended":1`)) {
		t.Fatalf("report response missing receiver status: %s", reportResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/v1/receiver-statuses", nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}
	for _, expected := range []string{
		`"side_effect":"none"`,
		`"telegram":1`,
		`"reason":"getupdates_conflict"`,
		`"polling":"stopped"`,
	} {
		if !bytes.Contains(listResponse.Body.Bytes(), []byte(expected)) {
			t.Fatalf("list response missing %s: %s", expected, listResponse.Body.String())
		}
	}
}

func TestReceiverLeaseEndpointAcquireDenyRenewRelease(t *testing.T) {
	manager := appservice.NewReceiverStatusService()
	mux := http.NewServeMux()
	httptrigger.RegisterReceiverStatusRoutes(mux, manager)

	acquireBody := []byte(`{
		"kind": "telegram",
		"channel_name": "telegram",
		"account_id": "7689386159",
		"holder_id": "python:1",
		"ttl_seconds": 60
	}`)
	acquireResponse := httptest.NewRecorder()
	mux.ServeHTTP(acquireResponse, httptest.NewRequest(http.MethodPost, "/v1/receiver-leases/acquire", bytes.NewReader(acquireBody)))
	if acquireResponse.Code != http.StatusOK {
		t.Fatalf("expected acquire 200, got %d: %s", acquireResponse.Code, acquireResponse.Body.String())
	}
	var acquirePayload struct {
		Data query.ReceiverLeaseView `json:"data"`
	}
	if err := json.Unmarshal(acquireResponse.Body.Bytes(), &acquirePayload); err != nil {
		t.Fatalf("decode acquire: %v", err)
	}
	if acquirePayload.Data.Acquired == nil || !*acquirePayload.Data.Acquired || acquirePayload.Data.LeaseToken == "" {
		t.Fatalf("unexpected acquire payload: %s", acquireResponse.Body.String())
	}

	denyBody := []byte(`{
		"kind": "telegram",
		"channel_name": "telegram",
		"account_id": "7689386159",
		"holder_id": "python:2",
		"ttl_seconds": 60
	}`)
	denyResponse := httptest.NewRecorder()
	mux.ServeHTTP(denyResponse, httptest.NewRequest(http.MethodPost, "/v1/receiver-leases/acquire", bytes.NewReader(denyBody)))
	if denyResponse.Code != http.StatusOK {
		t.Fatalf("expected deny 200, got %d: %s", denyResponse.Code, denyResponse.Body.String())
	}
	if !bytes.Contains(denyResponse.Body.Bytes(), []byte(`"acquired":false`)) ||
		!bytes.Contains(denyResponse.Body.Bytes(), []byte(`"denied_reason":"active_lease_held"`)) {
		t.Fatalf("deny response missing active lease state: %s", denyResponse.Body.String())
	}

	renewBody, _ := json.Marshal(map[string]any{
		"receiver_id": acquirePayload.Data.ReceiverID,
		"holder_id":   "python:1",
		"lease_token": acquirePayload.Data.LeaseToken,
		"ttl_seconds": 120,
	})
	renewResponse := httptest.NewRecorder()
	mux.ServeHTTP(renewResponse, httptest.NewRequest(http.MethodPost, "/v1/receiver-leases/renew", bytes.NewReader(renewBody)))
	if renewResponse.Code != http.StatusOK {
		t.Fatalf("expected renew 200, got %d: %s", renewResponse.Code, renewResponse.Body.String())
	}

	releaseBody, _ := json.Marshal(map[string]any{
		"receiver_id": acquirePayload.Data.ReceiverID,
		"holder_id":   "python:1",
		"lease_token": acquirePayload.Data.LeaseToken,
	})
	releaseResponse := httptest.NewRecorder()
	mux.ServeHTTP(releaseResponse, httptest.NewRequest(http.MethodPost, "/v1/receiver-leases/release", bytes.NewReader(releaseBody)))
	if releaseResponse.Code != http.StatusOK {
		t.Fatalf("expected release 200, got %d: %s", releaseResponse.Code, releaseResponse.Body.String())
	}
	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/v1/receiver-leases", nil))
	if !bytes.Contains(listResponse.Body.Bytes(), []byte(`"leases":0`)) {
		t.Fatalf("list response should show no leases: %s", listResponse.Body.String())
	}
}

func TestDeliveryAdapterHealthEndpointReturnsReadOnlyProbeResults(t *testing.T) {
	viewer := appservice.NewDeliveryAdapterHealthService(staticHTTPDeliveryHealthProbe{
		items: []query.DeliveryAdapterHealthView{{
			Provider:      "onebot",
			Channel:       "qq_2365524513",
			Transport:     "websocket",
			Healthy:       true,
			Reachable:     true,
			Authenticated: true,
			AccountID:     "2365524513",
			SideEffect:    "none",
		}},
	})
	mux := http.NewServeMux()
	httptrigger.RegisterDeliveryAdapterHealthRoutes(mux, viewer)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/delivery-adapters/health?timeout_seconds=1", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		`"provider":"onebot"`,
		`"account_id":"2365524513"`,
		`"side_effect":"none"`,
		`"healthy":true`,
	} {
		if !bytes.Contains(response.Body.Bytes(), []byte(expected)) {
			t.Fatalf("response missing %s: %s", expected, response.Body.String())
		}
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

func TestInboxMetricsEndpointReturnsObserveOnlyCollectionSummary(t *testing.T) {
	store := memory.NewStore()
	metrics := appservice.NewInboxMetricsService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterInboxMetricsRoutes(mux, metrics)

	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	events := []model.InboxEvent{
		mustHTTPInboxEvent(t, "qq:1049511700:group:27234224:metrics-1", "27234224", "2948770636", nil, map[string]string{"observe_only": "true", "seq": "10"}, now),
		mustHTTPInboxEvent(t, "qq:1049511700:group:27234224:metrics-2", "27234224", "99887766", []model.Attachment{{
			ID:       "asset:qq:image:metrics-2:1",
			Kind:     model.AttachmentKindImage,
			MimeType: "image/png",
			Name:     "qq-image.png",
		}}, map[string]string{"observe_only": "true", "seq": "11"}, now.Add(time.Second)),
	}
	for _, event := range events {
		if err := store.SaveInboxEvent(context.Background(), event); err != nil {
			t.Fatalf("save inbox event: %v", err)
		}
	}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/inbox-metrics?channel_kind=qq&conversation_id=27234224&conversation_type=group&observe_only=true&limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{
		`"sampled_events":2`,
		`"observe_only_total":2`,
		`"reply_eligible_total":0`,
		`"with_attachments":1`,
		`"attachment_count":1`,
		`"unique_senders":2`,
		`"latest_seq":11`,
		`"event_id":"qq:1049511700:group:27234224:metrics-2"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics response missing %s: %s", expected, body)
		}
	}
}

func TestInboundDedupeEndpointChecksAndListsRecords(t *testing.T) {
	store := memory.NewStore()
	dedupe := appservice.NewInboundDedupeService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterInboundDedupeRoutes(mux, dedupe)

	body := []byte(`{
		"scope": "telegram:telegram",
		"message_key": "123:456",
		"ttl_seconds": 60,
		"timestamp": "2026-05-31T11:00:00Z",
		"metadata": {"message_kind": "text"}
	}`)
	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/v1/inbound-dedupe/check", bytes.NewReader(body)))
	if first.Code != http.StatusOK {
		t.Fatalf("expected first check 200, got %d: %s", first.Code, first.Body.String())
	}
	if !bytes.Contains(first.Body.Bytes(), []byte(`"duplicate":false`)) {
		t.Fatalf("first check should not be duplicate: %s", first.Body.String())
	}

	second := httptest.NewRecorder()
	mux.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/v1/inbound-dedupe/check", bytes.NewReader(body)))
	if second.Code != http.StatusOK {
		t.Fatalf("expected second check 200, got %d: %s", second.Code, second.Body.String())
	}
	if !bytes.Contains(second.Body.Bytes(), []byte(`"duplicate":true`)) ||
		!bytes.Contains(second.Body.Bytes(), []byte(`"seen_count":2`)) {
		t.Fatalf("second check should be duplicate: %s", second.Body.String())
	}

	records := httptest.NewRecorder()
	mux.ServeHTTP(records, httptest.NewRequest(http.MethodGet, "/v1/inbound-dedupe/records?scope=telegram:telegram&limit=10", nil))
	if records.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", records.Code, records.Body.String())
	}
	if !bytes.Contains(records.Body.Bytes(), []byte(`"records":1`)) ||
		!bytes.Contains(records.Body.Bytes(), []byte(`"side_effect":"runtime_state_only"`)) {
		t.Fatalf("records response missing summary: %s", records.Body.String())
	}

	metrics := httptest.NewRecorder()
	mux.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/v1/inbound-dedupe/metrics?scope=telegram:telegram&limit=10", nil))
	if metrics.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d: %s", metrics.Code, metrics.Body.String())
	}
	if !bytes.Contains(metrics.Body.Bytes(), []byte(`"duplicate_seen_total":1`)) ||
		!bytes.Contains(metrics.Body.Bytes(), []byte(`"side_effect":"none"`)) ||
		!bytes.Contains(metrics.Body.Bytes(), []byte(`"scope":"telegram:telegram"`)) {
		t.Fatalf("metrics response missing summary: %s", metrics.Body.String())
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

func mustHTTPInboxEvent(
	t *testing.T,
	eventID string,
	conversationID string,
	senderID string,
	attachments []model.Attachment,
	metadata map[string]string,
	timestamp time.Time,
) model.InboxEvent {
	t.Helper()
	event, err := model.NewInboxEvent(model.MessageEnvelope{
		EventID: eventID,
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   conversationID,
			ConversationType: model.ConversationTypeGroup,
		},
		Sender: model.SenderRef{
			ID:   senderID,
			Kind: model.SenderKindHuman,
		},
		Content:     "raw group observation",
		Attachments: attachments,
		Timestamp:   timestamp,
		Metadata:    metadata,
	}, model.LoopDecision{Action: model.LoopActionAllow, Reason: "fixture"}, timestamp)
	if err != nil {
		t.Fatalf("new inbox event: %v", err)
	}
	return event
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

func TestDeliveryDispatchReadinessEndpointChecksAdaptersWithoutSending(t *testing.T) {
	store := memory.NewStore()
	sender := appservice.NewMessageSendService(store, store, store, store)
	outbox := appservice.NewOutboxService(store, store)
	adapter := &fakeDeliveryAdapter{channel: "qq_2365524513"}
	deliveryDispatch := appservice.NewDeliveryDispatchServiceWithAdapters(store, adapter)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(
		mux,
		nil,
		nil,
		nil,
		sender,
		appservice.NewImageJobService(store, store),
		outbox,
		appservice.NewMediaAssetService(store),
		appservice.NewAgentJobService(store),
		appservice.NewSendLedgerService(store),
		appservice.NewInboxEventService(store),
	)
	httptrigger.RegisterDeliveryDispatchRoutes(mux, deliveryDispatch)

	body := map[string]any{
		"event_id": "outbox-dispatch-ready-1",
		"channel": map[string]any{
			"platform":          "qq",
			"account_id":        "2365524513",
			"conversation_id":   "1049511700",
			"conversation_type": "private",
		},
		"content":   "hello readiness",
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

	request := []byte(`{"event_id":"outbox-dispatch-ready-1","channel_by_account":{"2365524513":"qq_2365524513"}}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/delivery-dispatch/readiness", bytes.NewReader(request)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected readiness 200, got %d: %s", response.Code, response.Body.String())
	}
	bodyText := response.Body.String()
	for _, expected := range []string{
		`"event_id":"outbox-dispatch-ready-1"`,
		`"channel":"qq_2365524513"`,
		`"ready":true`,
		`"reason":"delivery_adapter_ready"`,
		`"side_effect":"none"`,
	} {
		if !strings.Contains(bodyText, expected) {
			t.Fatalf("readiness response missing %s: %s", expected, bodyText)
		}
	}
	if len(adapter.steps) != 0 {
		t.Fatalf("readiness must not dispatch adapter steps: %+v", adapter.steps)
	}
}

func TestDeliveryDispatchReadinessEndpointReportsMissingAdapter(t *testing.T) {
	store := memory.NewStore()
	sender := appservice.NewMessageSendService(store, store, store, store)
	deliveryDispatch := appservice.NewDeliveryDispatchService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(
		mux,
		nil,
		nil,
		nil,
		sender,
		appservice.NewImageJobService(store, store),
		appservice.NewOutboxService(store, store),
		appservice.NewMediaAssetService(store),
		appservice.NewAgentJobService(store),
		appservice.NewSendLedgerService(store),
		appservice.NewInboxEventService(store),
	)
	httptrigger.RegisterDeliveryDispatchRoutes(mux, deliveryDispatch)

	body := map[string]any{
		"event_id": "outbox-dispatch-not-ready-1",
		"channel": map[string]any{
			"platform":          "telegram",
			"account_id":        "telegram-bot",
			"conversation_id":   "8655199155",
			"conversation_type": "private",
		},
		"content":   "hello telegram",
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/delivery-dispatch/readiness", bytes.NewReader([]byte(`{"event_id":"outbox-dispatch-not-ready-1"}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected readiness 200, got %d: %s", response.Code, response.Body.String())
	}
	bodyText := response.Body.String()
	for _, expected := range []string{
		`"ready":false`,
		`"reason":"delivery_adapter_unavailable"`,
		`"missing_channels":["telegram"]`,
	} {
		if !strings.Contains(bodyText, expected) {
			t.Fatalf("readiness response missing %s: %s", expected, bodyText)
		}
	}
}

func TestDeliverySmokeReadinessEndpointChecksMatrixWithoutSending(t *testing.T) {
	adapter := &fakeDeliveryAdapter{channel: "qq_2365524513"}
	deliverySmoke := appservice.NewDeliverySmokeReadinessService([]outport.DeliveryAdapter{adapter}, appservice.DeliverySmokeReadinessConfig{
		ChannelByAccount: map[string]string{"2365524513": "qq_2365524513"},
	})
	mux := http.NewServeMux()
	httptrigger.RegisterDeliverySmokeRoutes(mux, deliverySmoke)

	request := []byte(`{"cases":[{"name":"qq_private_file_2365524513_to_1049511700","channel_kind":"qq","account_id":"2365524513","conversation_id":"1049511700","conversation_type":"private","content":"smoke file","attachments":[{"kind":"file","url":"base64://YXNoaWNhYw==","name":"smoke.txt","mime_type":"text/plain"}]}]}`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/delivery-smoke/readiness", bytes.NewReader(request)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected smoke readiness 200, got %d: %s", response.Code, response.Body.String())
	}
	bodyText := response.Body.String()
	for _, expected := range []string{
		`"ready":true`,
		`"reason":"delivery_smoke_ready"`,
		`"name":"qq_private_file_2365524513_to_1049511700"`,
		`"channel":"qq_2365524513"`,
		`"kind":"file"`,
		`"side_effect":"none"`,
	} {
		if !strings.Contains(bodyText, expected) {
			t.Fatalf("smoke readiness response missing %s: %s", expected, bodyText)
		}
	}
	if len(adapter.steps) != 0 {
		t.Fatalf("smoke readiness must not dispatch adapter steps: %+v", adapter.steps)
	}
}

func TestDeliveryDispatchSendEndpointUsesAdapter(t *testing.T) {
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
	adapter := &fakeDeliveryAdapter{channel: "telegram"}
	deliveryDispatch := appservice.NewDeliveryDispatchServiceWithAdapters(store, adapter)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))
	httptrigger.RegisterDeliveryDispatchRoutes(mux, deliveryDispatch)

	body := map[string]any{
		"event_id": "outbox-dispatch-send-1",
		"channel": map[string]any{
			"platform":          "telegram",
			"account_id":        "telegram-bot",
			"conversation_id":   "8655199155",
			"conversation_type": "private",
		},
		"content":   "hello telegram",
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/delivery-dispatch/send", bytes.NewReader([]byte(`{"event_id":"outbox-dispatch-send-1"}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected send 200, got %d: %s", response.Code, response.Body.String())
	}
	bodyText := response.Body.String()
	for _, expected := range []string{
		`"event_id":"outbox-dispatch-send-1"`,
		`"status":"sent"`,
		`"provider":"test-telegram"`,
		`"provider_message_id":"msg-1"`,
	} {
		if !strings.Contains(bodyText, expected) {
			t.Fatalf("send response missing %s: %s", expected, bodyText)
		}
	}
	if len(adapter.steps) != 1 || adapter.steps[0].Message != "hello telegram" {
		t.Fatalf("adapter did not receive expected step: %+v", adapter.steps)
	}
}

func TestDeliveryDispatchSendEndpointReturnsUnavailableForUnsupportedRoute(t *testing.T) {
	store := memory.NewStore()
	sender := appservice.NewMessageSendService(store, store, store, store)
	deliveryDispatch := appservice.NewDeliveryDispatchService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(
		mux,
		nil,
		nil,
		nil,
		sender,
		appservice.NewImageJobService(store, store),
		appservice.NewOutboxService(store, store),
		appservice.NewMediaAssetService(store),
		appservice.NewAgentJobService(store),
		appservice.NewSendLedgerService(store),
		appservice.NewInboxEventService(store),
	)
	httptrigger.RegisterDeliveryDispatchRoutes(mux, deliveryDispatch)

	body := map[string]any{
		"event_id": "outbox-dispatch-send-unavailable",
		"channel": map[string]any{
			"platform":          "telegram",
			"account_id":        "telegram-bot",
			"conversation_id":   "8655199155",
			"conversation_type": "private",
		},
		"content":   "hello telegram",
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/delivery-dispatch/send", bytes.NewReader([]byte(`{"event_id":"outbox-dispatch-send-unavailable"}`))))
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("expected send 501, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"error_kind":"sender_unavailable"`) {
		t.Fatalf("expected sender_unavailable response: %s", response.Body.String())
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

type fakeDeliveryAdapter struct {
	channel string
	steps   []model.DeliveryDispatchStep
}

func (a *fakeDeliveryAdapter) SupportsDeliveryChannel(channel string) bool {
	return strings.EqualFold(strings.TrimSpace(channel), a.channel)
}

func (a *fakeDeliveryAdapter) DispatchDeliveryStep(_ context.Context, step model.DeliveryDispatchStep) (model.DeliveryDispatchResult, error) {
	a.steps = append(a.steps, step)
	return model.DeliveryDispatchResult{
		StepIndex:         step.StepIndex,
		Kind:              step.Kind,
		Channel:           step.Channel,
		ChatID:            step.ChatID,
		Status:            model.DeliveryDispatchSent,
		Provider:          "test-telegram",
		ProviderMessageID: "msg-1",
	}, nil
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
		"dedupe_key":       "knowledge:rag_ingest:qq:1049511700:27234224:source",
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

	duplicateBody := map[string]any{}
	for key, value := range body {
		duplicateBody[key] = value
	}
	duplicateBody["job_id"] = "job-http-duplicate"
	raw, err = json.Marshal(duplicateBody)
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs", bytes.NewReader(raw)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected duplicate job accepted, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"job_id":"job-http-1"`)) ||
		bytes.Contains(response.Body.Bytes(), []byte(`"job_id":"job-http-duplicate"`)) {
		t.Fatalf("duplicate create should return active existing job: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/lease-next", bytes.NewReader([]byte(`{"worker_id":"worker-http","job_type":"rag_ingest","ttl_seconds":60}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected lease-next 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"leased"`)) {
		t.Fatalf("lease response missing leased status: %s", response.Body.String())
	}
	var leasePayload struct {
		Data struct {
			LeaseToken string `json:"lease_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &leasePayload); err != nil {
		t.Fatalf("decode lease response: %v", err)
	}
	if leasePayload.Data.LeaseToken == "" {
		t.Fatalf("lease response missing lease token: %s", response.Body.String())
	}

	renewBody := []byte(`{"lease_token":"` + leasePayload.Data.LeaseToken + `","ttl_seconds":120}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/job-http-1/renew", bytes.NewReader(renewBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected renew 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"leased"`)) {
		t.Fatalf("renew response should preserve leased status: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	runningBody := []byte(`{"lease_token":"` + leasePayload.Data.LeaseToken + `"}`)
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/job-http-1/running", bytes.NewReader(runningBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected running 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	succeededBody := []byte(`{"lease_token":"` + leasePayload.Data.LeaseToken + `","result":{"indexed":"true"}}`)
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/job-http-1/succeeded", bytes.NewReader(succeededBody)))
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

func TestAgentJobMetricsEndpointReturnsLifecycleSummary(t *testing.T) {
	store := memory.NewStore()
	agentJobs := appservice.NewAgentJobServiceWithEvents(store, store)
	metrics := appservice.NewAgentJobMetricsService(store, store)
	mux := http.NewServeMux()
	httptrigger.RegisterAgentJobMetricsRoutes(mux, metrics)

	now := time.Date(2026, 5, 30, 11, 30, 0, 0, time.UTC)
	ctx := context.Background()
	if _, err := agentJobs.Create(ctx, command.CreateAgentJobCommand{
		JobID:   "job-metrics-http-succeeded",
		JobType: "rag_ingest",
		AgentID: "main",
		Route: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		MaxAttempts: 2,
		Timestamp:   now,
	}); err != nil {
		t.Fatalf("create succeeded job: %v", err)
	}
	if _, err := agentJobs.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      "job-metrics-http-succeeded",
		WorkerID:   "worker-http",
		LeaseToken: "lease-http-succeeded",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease succeeded job: %v", err)
	}
	if _, err := agentJobs.Complete(ctx, command.CompleteAgentJobCommand{
		JobID:      "job-metrics-http-succeeded",
		LeaseToken: "lease-http-succeeded",
		Result:     map[string]string{"ok": "true"},
		Timestamp:  now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("complete succeeded job: %v", err)
	}
	if _, err := agentJobs.Create(ctx, command.CreateAgentJobCommand{
		JobID:   "job-metrics-http-dead",
		JobType: "group_memory_extract",
		AgentID: "main",
		Route: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "3219982",
			ConversationType: "group",
		},
		MaxAttempts: 1,
		Timestamp:   now.Add(3 * time.Second),
	}); err != nil {
		t.Fatalf("create dead-letter job: %v", err)
	}
	if _, err := agentJobs.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      "job-metrics-http-dead",
		WorkerID:   "worker-http",
		LeaseToken: "lease-http-dead",
		TTLSeconds: 60,
		Timestamp:  now.Add(4 * time.Second),
	}); err != nil {
		t.Fatalf("lease dead-letter job: %v", err)
	}
	if _, err := agentJobs.Fail(ctx, command.FailAgentJobCommand{
		JobID:        "job-metrics-http-dead",
		LeaseToken:   "lease-http-dead",
		ErrorMessage: "fixture failure",
		Timestamp:    now.Add(5 * time.Second),
	}); err != nil {
		t.Fatalf("fail dead-letter job: %v", err)
	}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/job-metrics?job_limit=10&event_limit=20", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{
		`"sampled_jobs":2`,
		`"sampled_events":6`,
		`"succeeded":1`,
		`"failed":1`,
		`"current_total":1`,
		`"job_id":"job-metrics-http-dead"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics response missing %s: %s", expected, body)
		}
	}
}

func TestAgentJobLeaseWorkEndpointLeasesExactQueueWorkID(t *testing.T) {
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

	body := []byte(`{
		"job_id":"job-lease-work-http-1",
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

	leaseBody := []byte(`{
		"work_kind":"agent_job",
		"work_id":"job-lease-work-http-1",
		"aggregate_id":"job-lease-work-http-1",
		"subject":"akashic.work.agent_job.rag_ingest",
		"worker_id":"queue-worker",
		"ttl_seconds":60
	}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/lease-work", bytes.NewReader(leaseBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected lease-work 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"leased"`)) {
		t.Fatalf("lease-work response missing leased status: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"lease_owner":"queue-worker"`)) {
		t.Fatalf("lease-work response missing queue worker owner: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"lease_token"`)) {
		t.Fatalf("lease-work response missing lease token: %s", response.Body.String())
	}
}

func TestAgentJobRecoverExpiredEndpointReturnsExpiredLeaseToPending(t *testing.T) {
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
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))

	now := time.Date(2026, 5, 30, 9, 10, 0, 0, time.UTC)
	body := []byte(`{
		"job_id":"job-recover-http-1",
		"job_type":"rag_ingest",
		"agent_id":"main",
		"route":{
			"platform":"qq",
			"account_id":"1049511700",
			"conversation_id":"27234224",
			"conversation_type":"group"
		},
		"payload":{"source":"group"},
		"max_attempts":2,
		"timestamp":"` + now.Format(time.RFC3339Nano) + `"
	}`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected job accepted, got %d: %s", response.Code, response.Body.String())
	}

	leaseBody := []byte(`{"worker_id":"worker-recover","job_type":"rag_ingest","ttl_seconds":60,"timestamp":"` + now.Add(time.Second).Format(time.RFC3339Nano) + `"}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/lease-next", bytes.NewReader(leaseBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected lease-next 200, got %d: %s", response.Code, response.Body.String())
	}

	recoverBody := []byte(`{"limit":10,"timestamp":"` + now.Add(2*time.Minute).Format(time.RFC3339Nano) + `"}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/jobs/recover-expired", bytes.NewReader(recoverBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected recover-expired 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"recovered":1`)) {
		t.Fatalf("recover response missing recovered count: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"pending"`)) {
		t.Fatalf("recover response missing pending job: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"action":"recovered"`)) {
		t.Fatalf("recover response missing action: %s", response.Body.String())
	}
}

func TestOutboxEventsEndpointListsLifecycleStream(t *testing.T) {
	store := memory.NewStore()
	ingestor := appservice.NewMessageIngestService(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
		domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
	)
	sender := appservice.NewMessageSendServiceWithOutboxEvents(store, store, store, store, store)
	imageJobs := appservice.NewImageJobService(store, store)
	outbox := appservice.NewOutboxServiceWithEvents(store, store, store)
	outboxEvents := appservice.NewOutboxDeliveryEventService(store)
	mediaAssets := appservice.NewMediaAssetService(store)
	agentJobs := appservice.NewAgentJobService(store)
	shadowQueries := appservice.NewShadowQueryService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(mux, ingestor, ingestor, shadowQueries, sender, imageJobs, outbox, mediaAssets, agentJobs, appservice.NewSendLedgerService(store), appservice.NewInboxEventService(store))
	httptrigger.RegisterOutboxEventRoutes(mux, outboxEvents)

	body := map[string]any{
		"event_id": "outbox-events-http-1",
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/lease-next", bytes.NewReader([]byte(`{"worker_id":"outbox-worker","ttl_seconds":60}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected lease-next 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/outbox/outbox-events-http-1/succeeded", bytes.NewReader([]byte(`{}`))))
	if response.Code != http.StatusOK {
		t.Fatalf("expected succeeded 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/outbox-events?delivery_id=outbox-events-http-1&limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected outbox events 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"event_type":"queued"`)) {
		t.Fatalf("events response missing queued event: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"event_type":"leased"`)) {
		t.Fatalf("events response missing leased event: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"event_type":"succeeded"`)) {
		t.Fatalf("events response missing succeeded event: %s", response.Body.String())
	}
}

func TestOutboxMetricsEndpointReturnsLifecycleSummary(t *testing.T) {
	store := memory.NewStore()
	outbox := appservice.NewOutboxServiceWithEvents(store, store, store)
	metrics := appservice.NewOutboxMetricsService(store, store)
	mux := http.NewServeMux()
	httptrigger.RegisterOutboxMetricsRoutes(mux, metrics)

	now := time.Date(2026, 5, 30, 11, 45, 0, 0, time.UTC)
	ctx := context.Background()
	succeededMessage := model.OutboundMessage{
		EventID: "outbox-metrics-http-succeeded",
		Channel: model.ChannelRef{
			Kind:             "telegram",
			AccountID:        "bot",
			ConversationID:   "123",
			ConversationType: "private",
		},
		Content:   "image ready",
		Timestamp: now,
	}
	succeeded, err := model.NewOutboxDelivery(succeededMessage, 2, now)
	if err != nil {
		t.Fatalf("new succeeded delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, succeeded); err != nil {
		t.Fatalf("save succeeded delivery: %v", err)
	}
	if _, err := outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    "outbox-metrics-http-succeeded",
		WorkerID:   "worker-http",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease succeeded delivery: %v", err)
	}
	if _, err := outbox.MarkSucceeded(ctx, command.MarkOutboxSucceededCommand{
		EventID:   "outbox-metrics-http-succeeded",
		Timestamp: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark succeeded delivery: %v", err)
	}

	deadMessage := model.OutboundMessage{
		EventID: "outbox-metrics-http-dead",
		Channel: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "2365524513",
			ConversationID:   "1049511700",
			ConversationType: "private",
		},
		Content:   "bad route",
		Timestamp: now.Add(3 * time.Second),
	}
	dead, err := model.NewOutboxDelivery(deadMessage, 1, now.Add(3*time.Second))
	if err != nil {
		t.Fatalf("new dead-letter delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, dead); err != nil {
		t.Fatalf("save dead-letter delivery: %v", err)
	}
	if _, err := outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    "outbox-metrics-http-dead",
		WorkerID:   "worker-http",
		TTLSeconds: 60,
		Timestamp:  now.Add(4 * time.Second),
	}); err != nil {
		t.Fatalf("lease dead-letter delivery: %v", err)
	}
	if _, err := outbox.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      "outbox-metrics-http-dead",
		ErrorKind:    string(model.DeliveryErrorRoute),
		ErrorMessage: "missing adapter",
		Timestamp:    now.Add(5 * time.Second),
	}); err != nil {
		t.Fatalf("fail dead-letter delivery: %v", err)
	}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/outbox-metrics?delivery_limit=10&event_limit=20", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{
		`"sampled_deliveries":2`,
		`"sampled_events":4`,
		`"succeeded":1`,
		`"failed":1`,
		`"dead_lettered":1`,
		`"current_total":1`,
		`"delivery_id":"outbox-metrics-http-dead"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics response missing %s: %s", expected, body)
		}
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

func TestSchedulerJobEndpointSnapshotsAndListsJobs(t *testing.T) {
	store := memory.NewStore()
	schedulerJobs := appservice.NewSchedulerJobService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterSchedulerJobRoutes(mux, schedulerJobs)

	body := []byte(`{
		"source":"python_scheduler",
		"jobs":[{
			"id":"job-1",
			"trigger":"after",
			"tier":"instant",
			"fire_at":"2026-06-01T09:00:00Z",
			"channel":"qq",
			"chat_id":"1049511700",
			"message":"提醒",
			"timezone":"Asia/Shanghai",
			"created_at":"2026-06-01T08:00:00Z",
			"enabled":true
		}]
	}`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/scheduler/jobs/snapshot", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected scheduler snapshot 202, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"side_effect":"runtime_state_write"`)) {
		t.Fatalf("snapshot response missing side effect: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/scheduler/jobs", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected scheduler list 200, got %d: %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		`"id":"job-1"`,
		`"channel":"qq"`,
		`"fire_at":"2026-06-01T09:00:00Z"`,
	} {
		if !bytes.Contains(response.Body.Bytes(), []byte(expected)) {
			t.Fatalf("scheduler list missing %s: %s", expected, response.Body.String())
		}
	}

	upsertBody := []byte(`{
		"source":"python_scheduler",
		"job":{
			"id":"job-2",
			"trigger":"after",
			"tier":"instant",
			"fire_at":"2026-06-01T10:00:00Z",
			"channel":"qq",
			"chat_id":"2365524513",
			"message":"第二个提醒",
			"timezone":"Asia/Shanghai",
			"created_at":"2026-06-01T08:30:00Z",
			"enabled":true
		}
	}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/scheduler/jobs/upsert", bytes.NewReader(upsertBody)))
	if response.Code != http.StatusOK ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"created":true`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"job_id":"job-2"`)) {
		t.Fatalf("expected scheduler upsert response, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/v1/scheduler/jobs/job-1?source=python_scheduler", nil))
	if response.Code != http.StatusOK ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"found":true`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"deleted":true`)) {
		t.Fatalf("expected scheduler delete response, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/scheduler/jobs", nil))
	if response.Code != http.StatusOK ||
		bytes.Contains(response.Body.Bytes(), []byte(`"id":"job-1"`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"id":"job-2"`)) {
		t.Fatalf("expected scheduler list after CRUD, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/scheduler/diagnostics?timestamp=2026-06-01T09:59:00Z&due_soon_seconds=120", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected scheduler diagnostics 200, got %d: %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		`"sampled_jobs":1`,
		`"due_soon_jobs":1`,
		`"side_effect":"none"`,
	} {
		if !bytes.Contains(response.Body.Bytes(), []byte(expected)) {
			t.Fatalf("scheduler diagnostics missing %s: %s", expected, response.Body.String())
		}
	}

	acquireBody := []byte(`{
		"job_id":"job-2",
		"holder_id":"scheduler:worker-a",
		"ttl_seconds":120,
		"timestamp":"2026-06-01T08:59:00Z",
		"metadata":{"source":"python_scheduler"}
	}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/scheduler/leases/acquire", bytes.NewReader(acquireBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected scheduler lease acquire 200, got %d: %s", response.Code, response.Body.String())
	}
	var acquirePayload struct {
		Data query.SchedulerExecutionLeaseView `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &acquirePayload); err != nil {
		t.Fatalf("decode acquire response: %v", err)
	}
	if acquirePayload.Data.Acquired == nil || !*acquirePayload.Data.Acquired || acquirePayload.Data.LeaseToken == "" {
		t.Fatalf("expected acquired lease token: %s", response.Body.String())
	}

	denyBody := []byte(`{
		"job_id":"job-2",
		"holder_id":"scheduler:worker-b",
		"ttl_seconds":120,
		"timestamp":"2026-06-01T08:59:01Z"
	}`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/scheduler/leases/acquire", bytes.NewReader(denyBody)))
	if response.Code != http.StatusOK ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"acquired":false`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"denied_reason":"active_lease_held"`)) {
		t.Fatalf("expected scheduler lease deny response, got %d: %s", response.Code, response.Body.String())
	}

	renewBody, _ := json.Marshal(map[string]any{
		"job_id":      "job-2",
		"holder_id":   "scheduler:worker-a",
		"lease_token": acquirePayload.Data.LeaseToken,
		"ttl_seconds": 300,
		"timestamp":   "2026-06-01T08:59:30Z",
	})
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/scheduler/leases/renew", bytes.NewReader(renewBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected scheduler lease renew 200, got %d: %s", response.Code, response.Body.String())
	}

	releaseBody, _ := json.Marshal(map[string]any{
		"job_id":      "job-2",
		"holder_id":   "scheduler:worker-a",
		"lease_token": acquirePayload.Data.LeaseToken,
		"timestamp":   "2026-06-01T09:00:00Z",
	})
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/scheduler/leases/release", bytes.NewReader(releaseBody)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected scheduler lease release 200, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/scheduler/leases", nil))
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"leases":0`)) {
		t.Fatalf("expected empty scheduler lease list, got %d: %s", response.Code, response.Body.String())
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

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/send-ledger/metrics?from_bot_id=1049511700&conversation_id=2365524513&limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d: %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		`"sampled_records":1`,
		`"unique_bots":1`,
		`"unique_conversations":1`,
		`"records_by_bot"`,
		`"1049511700/2365524513"`,
	} {
		if !bytes.Contains(response.Body.Bytes(), []byte(expected)) {
			t.Fatalf("metrics response missing %s: %s", expected, response.Body.String())
		}
	}
}

func TestSendLedgerPrivateEchoEndpointUsesImageMarker(t *testing.T) {
	store := memory.NewStore()
	sendLedger := appservice.NewSendLedgerService(store)
	mux := http.NewServeMux()
	httptrigger.RegisterRoutes(
		mux,
		appservice.NewMessageIngestService(
			store,
			store,
			store,
			store,
			domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
			domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
		),
		appservice.NewMessageIngestService(
			store,
			store,
			store,
			store,
			domainservice.NewProvenanceClassifier([]string{"1049511700", "2365524513"}),
			domainservice.NewLoopGuard([]string{"1049511700", "2365524513"}, 15*time.Second, 6),
		),
		appservice.NewShadowQueryService(store),
		appservice.NewMessageSendService(store, store, store, store),
		appservice.NewImageJobService(store, store),
		appservice.NewOutboxService(store, store),
		appservice.NewMediaAssetService(store),
		appservice.NewAgentJobService(store),
		sendLedger,
		appservice.NewInboxEventService(store),
	)
	body := []byte(`{"from_bot_id":"1049511700","conversation_id":"2365524513","content":"[图片]"}`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/send-ledger/records", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected record accepted, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/send-ledger/private-echo?from_user_id=1049511700&to_bot_id=2365524513&has_image=true&window_seconds=180", nil)
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected echo 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"echo":true`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"reason":"recent_image_echo"`)) {
		t.Fatalf("private echo response missing image echo: %s", response.Body.String())
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/proactive/seen-items", bytes.NewReader([]byte(`{"entries":[{"source_key":"mcp:news:feed","item_id":"item-a"}],"timestamp":"2026-05-30T10:00:00Z"}`))))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected seen item 202, got %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/v1/proactive/seen-items/seen?source_key=mcp:news:other&item_id=item-a&ttl_hours=24&timestamp=2026-05-30T11:00:00Z", nil)
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected seen check 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"seen":true`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"source_key":"mcp:news"`)) {
		t.Fatalf("seen response missing normalized hit: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/proactive/rejection-cooldowns", bytes.NewReader([]byte(`{"entries":[{"source_key":"qq:group:1","item_id":"item-b"}],"hours":2,"timestamp":"2026-05-30T10:00:00Z"}`))))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected rejection cooldown 202, got %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/v1/proactive/rejection-cooldowns/cooled?source_key=qq:group:1&item_id=item-b&ttl_hours=2&timestamp=2026-05-30T11:00:00Z", nil)
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected rejection cooldown check 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"cooled":true`)) {
		t.Fatalf("cooldown response missing true flag: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/proactive/cleanup", bytes.NewReader([]byte(`{"seen_ttl_hours":1,"delivery_ttl_hours":1,"context_only_ttl_hours":1,"rejection_cooldown_ttl_hours":1,"timestamp":"2026-05-30T12:30:00Z"}`))))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected cleanup 202, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"removed_seen_items":1`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"removed_rejection_cooldowns":1`)) {
		t.Fatalf("cleanup response missing removed counts: %s", response.Body.String())
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

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/proactive/bg-context/main", bytes.NewReader([]byte(`{"timestamp":"2026-05-30T12:30:00Z"}`))))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected bg context 202, got %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/proactive/bg-context/main/last", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected bg context last 200, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"key":"bg_context_last_main_at"`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"found":true`)) {
		t.Fatalf("bg context response missing marker: %s", response.Body.String())
	}
}

func queryQueueBackendViewForTest() query.QueueBackendView {
	return query.QueueBackendView{
		Provider:                "nats_jetstream",
		Mode:                    "shadow_publish",
		MigrationPhase:          "shadow_ready",
		Stream:                  "AKASHIC_WORK",
		SubjectPrefix:           "akashic.work",
		ExternalQueueConfigured: true,
		ExternalQueueActive:     false,
		StateStoreAuthoritative: true,
		LeaseOwner:              "go_state_store",
		ConsumerModel:           "goroutine_worker_pool",
		ConsumerConcurrency:     8,
		MaxInFlight:             64,
		OutboxQueueSource:       "outbox_state_store",
		AgentJobQueueSource:     "agent_job_state_store",
		DSNConfigured:           true,
		DSNRedacted:             "nats://redacted@127.0.0.1:4222",
		RecommendedFirstBackend: "nats_jetstream",
		SupportedProviders:      []string{"local", "nats_jetstream", "redis_streams", "rabbitmq"},
		Notes:                   []string{"diagnostic only"},
	}
}

type staticRuntimeOverviewViewer struct {
	view query.RuntimeOverviewView
}

func (s staticRuntimeOverviewViewer) Get(context.Context, query.RuntimeOverviewFilter) (query.RuntimeOverviewView, error) {
	return s.view, nil
}

type staticObserveCaptureDiagnostics struct {
	view query.ObserveCaptureDiagnosticsView
}

func (s staticObserveCaptureDiagnostics) GetObserveCaptureDiagnostics(context.Context, query.ObserveCaptureDiagnosticsFilter) (query.ObserveCaptureDiagnosticsView, error) {
	return s.view, nil
}

type staticHTTPDeliveryHealthProbe struct {
	items []query.DeliveryAdapterHealthView
}

func (s staticHTTPDeliveryHealthProbe) CheckDeliveryAdapterHealth(context.Context, query.DeliveryAdapterHealthFilter) ([]query.DeliveryAdapterHealthView, error) {
	return append([]query.DeliveryAdapterHealthView(nil), s.items...), nil
}
