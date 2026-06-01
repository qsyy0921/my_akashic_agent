package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/localmedia"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestMediaAssetServiceContentDiagnosticsClassifiesContentAccess(t *testing.T) {
	ctx := context.Background()
	assetRoot := t.TempDir()
	outsideRoot := t.TempDir()
	readyPath := filepath.Join(assetRoot, "ready.txt")
	if err := writeTestFile(readyPath, "ready bytes"); err != nil {
		t.Fatal(err)
	}
	forbiddenPath := filepath.Join(outsideRoot, "forbidden.txt")
	if err := writeTestFile(forbiddenPath, "secret bytes"); err != nil {
		t.Fatal(err)
	}

	reader, err := localmedia.NewReader([]string{assetRoot})
	if err != nil {
		t.Fatal(err)
	}
	store := memory.NewStore()
	service := appservice.NewMediaAssetServiceWithContent(store, reader)
	registerTestMediaAsset(t, service, "asset:ready", readyPath, "ready.txt")
	registerTestMediaAsset(t, service, "asset:forbidden", forbiddenPath, "forbidden.txt")
	registerTestMediaAsset(t, service, "asset:missing", filepath.Join(assetRoot, "missing.txt"), "missing.txt")

	view, err := service.ContentDiagnostics(ctx, query.MediaAssetContentDiagnosticsFilter{Limit: 10})
	if err != nil {
		t.Fatalf("content diagnostics: %v", err)
	}
	if view.Totals["assets"] != 3 ||
		view.Totals["ready"] != 1 ||
		view.Totals["forbidden"] != 1 ||
		view.Totals["unavailable"] != 1 ||
		view.Totals["disabled"] != 0 ||
		view.Totals["error"] != 0 {
		t.Fatalf("unexpected totals: %+v", view.Totals)
	}
	ready := findMediaAssetContentDiagnostic(t, view, "asset:ready")
	if ready.ContentStatus != "ready" ||
		ready.ContentReason != "media_asset_content_ready" ||
		ready.ContentSizeBytes <= 0 ||
		ready.ContentEndpoint != "/v1/media-assets/asset:ready/content" ||
		ready.ContentAccessPlanEndpoint != "/v1/media-assets/content-access-plan?asset_id=asset%3Aready" {
		t.Fatalf("unexpected ready item: %+v", ready)
	}
	forbidden := findMediaAssetContentDiagnostic(t, view, "asset:forbidden")
	if forbidden.ContentStatus != "forbidden" || forbidden.ContentReason != "media_asset_content_forbidden" {
		t.Fatalf("unexpected forbidden item: %+v", forbidden)
	}
	missing := findMediaAssetContentDiagnostic(t, view, "asset:missing")
	if missing.ContentStatus != "unavailable" || missing.ContentReason != "media_asset_content_unavailable" {
		t.Fatalf("unexpected missing item: %+v", missing)
	}
}

func TestMediaAssetServiceContentDiagnosticsReportsDisabledReader(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewMediaAssetService(store)
	registerTestMediaAsset(t, service, "asset:disabled", filepath.Join(t.TempDir(), "disabled.txt"), "disabled.txt")

	view, err := service.ContentDiagnostics(ctx, query.MediaAssetContentDiagnosticsFilter{AssetID: "asset:disabled"})
	if err != nil {
		t.Fatalf("content diagnostics: %v", err)
	}
	if view.Totals["assets"] != 1 || view.Totals["disabled"] != 1 {
		t.Fatalf("unexpected disabled totals: %+v", view.Totals)
	}
	item := findMediaAssetContentDiagnostic(t, view, "asset:disabled")
	if item.ContentStatus != "disabled" || item.ContentReason != "media_asset_content_disabled" {
		t.Fatalf("unexpected disabled item: %+v", item)
	}
	if view.SideEffect != "none" {
		t.Fatalf("content diagnostics must be read-only: %+v", view)
	}
}

func TestMediaAssetServiceContentAccessPlanExplainsReadyAndBlockedStates(t *testing.T) {
	ctx := context.Background()
	assetRoot := t.TempDir()
	outsideRoot := t.TempDir()
	readyPath := filepath.Join(assetRoot, "ready.txt")
	if err := writeTestFile(readyPath, "ready bytes"); err != nil {
		t.Fatal(err)
	}
	forbiddenPath := filepath.Join(outsideRoot, "forbidden.txt")
	if err := writeTestFile(forbiddenPath, "secret bytes"); err != nil {
		t.Fatal(err)
	}
	reader, err := localmedia.NewReader([]string{assetRoot})
	if err != nil {
		t.Fatal(err)
	}
	store := memory.NewStore()
	service := appservice.NewMediaAssetServiceWithContent(store, reader)
	registerTestMediaAsset(t, service, "asset:ready", readyPath, "ready.txt")
	registerTestMediaAsset(t, service, "asset:forbidden", forbiddenPath, "forbidden.txt")
	registerTestMediaAsset(t, service, "asset:missing", filepath.Join(assetRoot, "missing.txt"), "missing.txt")

	ready, err := service.ContentAccessPlan(ctx, "asset:ready")
	if err != nil {
		t.Fatalf("content access plan ready: %v", err)
	}
	if !ready.Ready ||
		ready.Reason != "media_asset_content_ready" ||
		ready.ContentEndpoint != "/v1/media-assets/asset:ready/content" ||
		ready.ContentSizeBytes <= 0 ||
		ready.SideEffect != "none" ||
		len(ready.RequiredSteps) < 2 {
		t.Fatalf("unexpected ready plan: %+v", ready)
	}

	forbidden, err := service.ContentAccessPlan(ctx, "asset:forbidden")
	if err != nil {
		t.Fatalf("content access plan forbidden: %v", err)
	}
	if forbidden.Ready ||
		forbidden.Reason != "media_asset_content_forbidden" ||
		len(forbidden.Blockers) == 0 ||
		forbidden.ContentEndpoint != "/v1/media-assets/asset:forbidden/content" {
		t.Fatalf("unexpected forbidden plan: %+v", forbidden)
	}

	missingContent, err := service.ContentAccessPlan(ctx, "asset:missing")
	if err != nil {
		t.Fatalf("content access plan unavailable: %v", err)
	}
	if missingContent.Ready || missingContent.Reason != "media_asset_content_unavailable" || len(missingContent.Blockers) == 0 {
		t.Fatalf("unexpected unavailable plan: %+v", missingContent)
	}

	missingAsset, err := service.ContentAccessPlan(ctx, "asset:not-found")
	if err != nil {
		t.Fatalf("content access plan missing asset: %v", err)
	}
	if missingAsset.Ready || missingAsset.Reason != "media_asset_content_asset_not_found" || missingAsset.Asset != nil {
		t.Fatalf("unexpected missing asset plan: %+v", missingAsset)
	}
}

func TestMediaAssetServiceContentAccessPlanReportsMissingIDAndDisabledReader(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewMediaAssetService(store)
	registerTestMediaAsset(t, service, "asset:disabled", filepath.Join(t.TempDir(), "disabled.txt"), "disabled.txt")

	missingID, err := service.ContentAccessPlan(ctx, "")
	if err != nil {
		t.Fatalf("content access plan missing id: %v", err)
	}
	if missingID.Ready || missingID.Reason != "media_asset_content_asset_id_required" || len(missingID.Blockers) == 0 {
		t.Fatalf("unexpected missing id plan: %+v", missingID)
	}

	disabled, err := service.ContentAccessPlan(ctx, "asset:disabled")
	if err != nil {
		t.Fatalf("content access plan disabled: %v", err)
	}
	if disabled.Ready ||
		disabled.Reason != "media_asset_content_disabled" ||
		disabled.Asset == nil ||
		disabled.ContentEndpoint != "/v1/media-assets/asset:disabled/content" ||
		disabled.SideEffect != "none" {
		t.Fatalf("unexpected disabled plan: %+v", disabled)
	}
}

func TestMediaAssetServiceRetentionDiagnosticsClassifiesCleanupDue(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewMediaAssetService(store)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, service, "asset:permanent", "permanent.txt", "permanent", now.Add(-90*24*time.Hour))
	registerTestMediaAssetWithRetention(t, service, "asset:default", "default.txt", "default-observed-group", now.Add(-48*time.Hour))
	registerTestMediaAssetWithRetention(t, service, "asset:ephemeral", "ephemeral.txt", "ephemeral", now.Add(-2*time.Hour))

	view, err := service.RetentionDiagnostics(ctx, query.MediaAssetRetentionDiagnosticsFilter{
		Limit:             10,
		Timestamp:         now.Format(time.RFC3339Nano),
		DefaultTTLHours:   24,
		EphemeralTTLHours: 1,
	})
	if err != nil {
		t.Fatalf("retention diagnostics: %v", err)
	}
	if view.Totals["assets"] != 3 ||
		view.Totals["cleanup_due"] != 2 ||
		view.Totals["permanent"] != 1 ||
		view.Totals["default"] != 1 ||
		view.Totals["ephemeral"] != 1 {
		t.Fatalf("unexpected retention totals: %+v", view.Totals)
	}
	permanent := findMediaAssetRetentionDiagnostic(t, view, "asset:permanent")
	if permanent.CleanupDue || permanent.RetentionClass != "permanent" || permanent.CleanupReason != "media_asset_retention_permanent" {
		t.Fatalf("unexpected permanent item: %+v", permanent)
	}
	defaultItem := findMediaAssetRetentionDiagnostic(t, view, "asset:default")
	if !defaultItem.CleanupDue || defaultItem.RetentionClass != "default" || defaultItem.TTLSeconds != 24*60*60 {
		t.Fatalf("unexpected default item: %+v", defaultItem)
	}
	ephemeral := findMediaAssetRetentionDiagnostic(t, view, "asset:ephemeral")
	if !ephemeral.CleanupDue || ephemeral.RetentionClass != "ephemeral" || ephemeral.TTLSeconds != 60*60 {
		t.Fatalf("unexpected ephemeral item: %+v", ephemeral)
	}
	if view.SideEffect != "none" {
		t.Fatalf("retention diagnostics must be read-only: %+v", view)
	}
}

func TestMediaAssetServiceRetentionPlanBuildsDryRunCleanupPlan(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewMediaAssetService(store)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, service, "asset:keep", "keep.txt", "permanent", now.Add(-90*24*time.Hour))
	registerTestMediaAssetWithRetention(t, service, "asset:due", "due.txt", "default-observed-group", now.Add(-48*time.Hour))
	registerTestMediaAssetWithRetention(t, service, "asset:fresh", "fresh.txt", "ephemeral", now.Add(-30*time.Minute))

	view, err := service.RetentionPlan(ctx, query.MediaAssetRetentionDiagnosticsFilter{
		Limit:             10,
		Timestamp:         now.Format(time.RFC3339Nano),
		DefaultTTLHours:   24,
		EphemeralTTLHours: 1,
	})
	if err != nil {
		t.Fatalf("retention plan: %v", err)
	}
	if !view.Ready || view.Reason != "media_asset_retention_cleanup_candidates_ready" {
		t.Fatalf("unexpected readiness: %+v", view)
	}
	if view.AssetCount != 3 || view.CandidateCount != 1 || len(view.Candidates) != 1 ||
		view.Candidates[0].AssetID != "asset:due" {
		t.Fatalf("unexpected candidates: %+v", view)
	}
	if len(view.RequiredSteps) < 3 || len(view.VerifySteps) == 0 || len(view.RollbackSteps) == 0 {
		t.Fatalf("missing plan steps: %+v", view)
	}
	if view.SideEffect != "none" || view.Diagnostics.SideEffect != "none" {
		t.Fatalf("retention plan must be read-only: %+v", view)
	}
}

func TestMediaAssetServiceRetentionPlanBlocksWhenNoCandidates(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewMediaAssetService(store)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	registerTestMediaAssetWithRetention(t, service, "asset:keep", "keep.txt", "permanent", now.Add(-90*24*time.Hour))

	view, err := service.RetentionPlan(ctx, query.MediaAssetRetentionDiagnosticsFilter{
		Limit:     10,
		Timestamp: now.Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("retention plan: %v", err)
	}
	if view.Ready || view.Reason != "media_asset_retention_no_cleanup_candidates" ||
		view.CandidateCount != 0 || len(view.Blockers) == 0 {
		t.Fatalf("expected blocked no-candidate plan: %+v", view)
	}
	if view.SideEffect != "none" {
		t.Fatalf("retention plan must be read-only: %+v", view)
	}
}

func registerTestMediaAsset(
	t *testing.T,
	service *appservice.MediaAssetService,
	assetID string,
	path string,
	name string,
) {
	t.Helper()
	_, err := service.Register(context.Background(), command.RegisterMediaAssetCommand{
		AssetID: assetID,
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		SourceMessageID: "qq:gqq:27234224:1",
		SenderID:        "2948770636",
		Kind:            "image",
		URL:             path,
		MimeType:        "text/plain",
		Name:            name,
		Timestamp:       time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("register media asset %s: %v", assetID, err)
	}
}

func registerTestMediaAssetWithRetention(
	t *testing.T,
	service *appservice.MediaAssetService,
	assetID string,
	name string,
	retention string,
	timestamp time.Time,
) {
	t.Helper()
	_, err := service.Register(context.Background(), command.RegisterMediaAssetCommand{
		AssetID: assetID,
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: "group",
		},
		SourceMessageID: "qq:gqq:27234224:" + assetID,
		SenderID:        "2948770636",
		Kind:            "image",
		URL:             filepath.Join(t.TempDir(), name),
		MimeType:        "text/plain",
		Name:            name,
		Retention:       retention,
		Timestamp:       timestamp,
	})
	if err != nil {
		t.Fatalf("register media asset %s: %v", assetID, err)
	}
}

func findMediaAssetContentDiagnostic(
	t *testing.T,
	view query.MediaAssetContentDiagnosticsView,
	assetID string,
) query.MediaAssetContentDiagnosticItemView {
	t.Helper()
	for _, item := range view.Items {
		if item.AssetID == assetID {
			return item
		}
	}
	t.Fatalf("content diagnostic item not found for %s: %+v", assetID, view.Items)
	return query.MediaAssetContentDiagnosticItemView{}
}

func findMediaAssetRetentionDiagnostic(
	t *testing.T,
	view query.MediaAssetRetentionDiagnosticsView,
	assetID string,
) query.MediaAssetRetentionDiagnosticItemView {
	t.Helper()
	for _, item := range view.Items {
		if item.AssetID == assetID {
			return item
		}
	}
	t.Fatalf("retention diagnostic item not found for %s: %+v", assetID, view.Items)
	return query.MediaAssetRetentionDiagnosticItemView{}
}

func writeTestFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
