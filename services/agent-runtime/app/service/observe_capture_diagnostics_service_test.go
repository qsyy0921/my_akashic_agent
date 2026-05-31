package service

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestObserveCaptureDiagnosticsServiceReportsTextImageFileCoverage(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	syncObserveTarget(t, observeTargets, "27234224")
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessage(t, store, "msg-text", "hello hardware group", nil)
	ingestObserveMessage(t, store, "msg-image", "image evidence", []command.AttachmentCommand{{
		ID:       "asset:image:1",
		Kind:     "image",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/image.png",
		MimeType: "image/png",
		Name:     "image.png",
	}})
	ingestObserveMessage(t, store, "msg-file", "file evidence", []command.AttachmentCommand{{
		ID:       "asset:file:1",
		Kind:     "file",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/guide.txt",
		MimeType: "text/plain",
		Name:     "guide.txt",
	}})

	service := NewObserveCaptureDiagnosticsService(
		observeTargets,
		receiverStatuses,
		store,
		store,
		fakeObserveCaptureContentReader{ready: map[string]bool{
			"asset:image:1": true,
			"asset:file:1":  true,
		}},
	)
	view, err := service.GetObserveCaptureDiagnostics(ctx, query.ObserveCaptureDiagnosticsFilter{Limit: 50})
	if err != nil {
		t.Fatalf("observe capture diagnostics: %v", err)
	}
	if view.Totals["ready"] != 1 || view.Totals["text_covered"] != 1 || view.Totals["image_covered"] != 1 || view.Totals["file_covered"] != 1 {
		t.Fatalf("unexpected coverage totals: %#v", view.Totals)
	}
	if len(view.Targets) != 1 {
		t.Fatalf("expected one target, got %#v", view.Targets)
	}
	target := view.Targets[0]
	if target.Status != "ok" || !target.ReceiverConnected || len(target.Blockers) != 0 {
		t.Fatalf("unexpected target diagnostics: %#v", target)
	}
	if target.ContentReadyAssets != 2 || target.MediaAssets != 2 {
		t.Fatalf("unexpected media readiness: %#v", target)
	}
}

func TestObserveCaptureDiagnosticsServiceWarnsOnMissingMediaCoverage(t *testing.T) {
	store := memory.NewStore()
	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	syncObserveTarget(t, observeTargets, "27234224")
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessage(t, store, "msg-text", "only text so far", nil)

	service := NewObserveCaptureDiagnosticsService(
		observeTargets,
		receiverStatuses,
		store,
		store,
		fakeObserveCaptureContentReader{},
	)
	view, err := service.GetObserveCaptureDiagnostics(context.Background(), query.ObserveCaptureDiagnosticsFilter{Limit: 50})
	if err != nil {
		t.Fatalf("observe capture diagnostics: %v", err)
	}
	if view.Totals["ready"] != 0 || view.Totals["warning"] != 1 {
		t.Fatalf("unexpected warning totals: %#v", view.Totals)
	}
	target := view.Targets[0]
	if target.Status != "warn" || !containsString(target.Blockers, "image_not_seen") || !containsString(target.Blockers, "file_not_seen") {
		t.Fatalf("expected missing media blockers, got %#v", target)
	}
}

func TestObserveCaptureDiagnosticsServiceInfersReceiverFromRecentInboxActivity(t *testing.T) {
	store := memory.NewStore()
	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	syncObserveTarget(t, observeTargets, "27234224")
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)
	saveObserveInboxEventAt(t, store, "msg-recent", "recent group activity", now.Add(-2*time.Minute))

	service := NewObserveCaptureDiagnosticsService(
		observeTargets,
		receiverStatuses,
		store,
		store,
		fakeObserveCaptureContentReader{},
	)
	service.clock = func() time.Time { return now }
	view, err := service.GetObserveCaptureDiagnostics(context.Background(), query.ObserveCaptureDiagnosticsFilter{Limit: 50})
	if err != nil {
		t.Fatalf("observe capture diagnostics: %v", err)
	}
	target := view.Targets[0]
	if !target.ReceiverConnected || !target.ReceiverActivityRecent || target.ReceiverStatusConnected {
		t.Fatalf("expected recent inbox activity to infer receiver connectivity: %#v", target)
	}
	if target.ReceiverConnectionSource != "recent_inbox_activity" {
		t.Fatalf("unexpected connection source: %#v", target)
	}
	if containsString(target.Blockers, "receiver_not_connected") {
		t.Fatalf("recent activity should not report receiver_not_connected: %#v", target.Blockers)
	}
	if view.Totals["receiver_connected"] != 1 || view.Totals["receiver_activity_recent"] != 1 || view.Totals["receiver_status_connected"] != 0 {
		t.Fatalf("unexpected receiver totals: %#v", view.Totals)
	}
}

func TestObserveCaptureDiagnosticsServiceDoesNotInferReceiverFromStaleInboxActivity(t *testing.T) {
	store := memory.NewStore()
	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	syncObserveTarget(t, observeTargets, "27234224")
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)
	saveObserveInboxEventAt(t, store, "msg-stale", "old group activity", now.Add(-30*time.Minute))

	service := NewObserveCaptureDiagnosticsService(
		observeTargets,
		receiverStatuses,
		store,
		store,
		fakeObserveCaptureContentReader{},
	)
	service.clock = func() time.Time { return now }
	view, err := service.GetObserveCaptureDiagnostics(context.Background(), query.ObserveCaptureDiagnosticsFilter{Limit: 50})
	if err != nil {
		t.Fatalf("observe capture diagnostics: %v", err)
	}
	target := view.Targets[0]
	if target.ReceiverConnected || target.ReceiverActivityRecent || !containsString(target.Blockers, "receiver_not_connected") {
		t.Fatalf("expected stale activity to keep receiver disconnected blocker: %#v", target)
	}
}

func syncObserveTarget(t *testing.T, service *ObserveTargetService, groupID string) {
	syncObserveTargetWithMetadata(t, service, groupID, nil)
}

func syncObserveTargetWithMetadata(t *testing.T, service *ObserveTargetService, groupID string, metadata map[string]string) {
	t.Helper()
	_, err := service.SyncObserveTargets(context.Background(), command.SyncObserveTargetsCommand{
		Source: "test",
		Targets: []command.ObserveTargetCommand{{
			TargetID: "qq:1049511700:group:" + groupID,
			Channel: command.ChannelCommand{
				Kind:             "qq",
				AccountID:        "1049511700",
				ConversationID:   groupID,
				ConversationType: "group",
			},
			ObserveOnly: true,
			Enabled:     true,
			Source:      "test",
			Metadata:    metadata,
		}},
	})
	if err != nil {
		t.Fatalf("sync observe target: %v", err)
	}
}

func reportQQReceiver(t *testing.T, service *ReceiverStatusService, accountID string, status string) {
	t.Helper()
	_, err := service.ReportReceiverStatus(context.Background(), command.ReportReceiverStatusCommand{
		Kind:        "qq",
		ChannelName: "qq",
		AccountID:   accountID,
		Endpoint:    "ws://127.0.0.1:3001",
		Status:      status,
		Source:      "test",
	})
	if err != nil {
		t.Fatalf("report receiver: %v", err)
	}
}

func ingestObserveMessage(t *testing.T, store *memory.Store, suffix string, content string, attachments []command.AttachmentCommand) {
	t.Helper()
	ingestor := NewMessageIngestServiceWithRuntimeStores(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700"}),
		domainservice.NewLoopGuard([]string{"1049511700"}, 15*time.Second, 6),
		store,
		store,
	)
	_, err := ingestor.ShadowIngest(context.Background(), command.IngestMessageCommand{
		EventID: "qq:1049511700:group:27234224:" + suffix,
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		Sender: command.SenderCommand{
			ID:   "2948770636",
			Kind: string(model.SenderKindHuman),
		},
		Content:     content,
		Attachments: attachments,
		Timestamp:   time.Now().UTC(),
		Metadata: map[string]string{
			"observe_only": "true",
			"session_key":  "qq:gqq:27234224",
		},
	})
	if err != nil {
		t.Fatalf("ingest observe message: %v", err)
	}
}

func saveObserveInboxEventAt(t *testing.T, store *memory.Store, suffix string, content string, receivedAt time.Time) {
	t.Helper()
	event, err := model.NewInboxEvent(model.MessageEnvelope{
		EventID: "qq:1049511700:group:27234224:" + suffix,
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
		Content:   content,
		Timestamp: receivedAt,
		Metadata: map[string]string{
			"observe_only": "true",
			"session_key":  "qq:gqq:27234224",
		},
	}, model.LoopDecision{Action: model.LoopActionObserveOnly, Reason: "test"}, receivedAt)
	if err != nil {
		t.Fatalf("new inbox event: %v", err)
	}
	if err := store.SaveInboxEvent(context.Background(), event); err != nil {
		t.Fatalf("save inbox event: %v", err)
	}
}

type fakeObserveCaptureContentReader struct {
	ready map[string]bool
}

func (r fakeObserveCaptureContentReader) OpenMediaAssetContent(_ context.Context, asset model.MediaAsset) (outport.MediaAssetContent, error) {
	if r.ready[asset.AssetID] {
		return outport.MediaAssetContent{
			Name:      asset.Name,
			MimeType:  asset.MimeType,
			SizeBytes: asset.SizeBytes,
			Body:      io.NopCloser(strings.NewReader("ok")),
		}, nil
	}
	return outport.MediaAssetContent{}, outport.ErrMediaAssetContentUnavailable
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
