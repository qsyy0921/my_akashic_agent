package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
		ready.ContentAccessPlanEndpoint != "/v1/media-assets/content-access-plan?asset_id=asset%3Aready" ||
		ready.ContentRecoveryPlanEndpoint != "/v1/media-assets/content-recovery-plan?asset_id=asset%3Aready" ||
		ready.ContentRecoveryPreflightEndpoint != "/v1/media-assets/content-recovery/preflight?asset_id=asset%3Aready" {
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
		ready.RuntimePath != "/v1/media-assets/content-access-plan?asset_id=asset%3Aready" ||
		ready.DashboardPath != "/api/dashboard/media-assets/content-access-plan?asset_id=asset%3Aready" ||
		ready.ContentEndpoint != "/v1/media-assets/asset:ready/content" ||
		ready.ContentURL != "/api/dashboard/media-assets/content?asset_id=asset%3Aready" ||
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
		forbidden.RuntimePath != "/v1/media-assets/content-access-plan?asset_id=asset%3Aforbidden" ||
		forbidden.DashboardPath != "/api/dashboard/media-assets/content-access-plan?asset_id=asset%3Aforbidden" ||
		forbidden.ContentURL != "/api/dashboard/media-assets/content?asset_id=asset%3Aforbidden" ||
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

func TestMediaAssetServiceContentRecoveryPlanExplainsRecoveryPaths(t *testing.T) {
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

	ready, err := service.ContentRecoveryPlan(ctx, "asset:ready")
	if err != nil {
		t.Fatalf("content recovery plan ready: %v", err)
	}
	if !ready.Ready ||
		ready.Reason != "media_asset_content_recovery_not_required" ||
		ready.RuntimePath != "/v1/media-assets/content-recovery-plan?asset_id=asset%3Aready" ||
		ready.DashboardPath != "/api/dashboard/media-assets/content-recovery-plan?asset_id=asset%3Aready" ||
		ready.ContentURL != "/api/dashboard/media-assets/content?asset_id=asset%3Aready" ||
		ready.AccessPlan.Reason != "media_asset_content_ready" ||
		ready.SideEffect != "none" ||
		ready.FutureExecutorScope != "" {
		t.Fatalf("unexpected ready recovery plan: %+v", ready)
	}

	forbidden, err := service.ContentRecoveryPlan(ctx, "asset:forbidden")
	if err != nil {
		t.Fatalf("content recovery plan forbidden: %v", err)
	}
	if forbidden.Ready ||
		forbidden.Reason != "media_asset_content_recovery_fix_content_roots" ||
		len(forbidden.Blockers) == 0 ||
		forbidden.FutureExecutorScope != "operator_runtime_config" ||
		!mediaAssetStepNamesContain(forbidden.FallbackSteps, "add-intended-media-root") {
		t.Fatalf("unexpected forbidden recovery plan: %+v", forbidden)
	}

	missingContent, err := service.ContentRecoveryPlan(ctx, "asset:missing")
	if err != nil {
		t.Fatalf("content recovery plan unavailable: %v", err)
	}
	if missingContent.Ready ||
		missingContent.Reason != "media_asset_content_recovery_restore_or_redownload" ||
		missingContent.FutureExecutorScope != "media_content_cache_executor" ||
		!mediaAssetStepNamesContain(missingContent.FallbackSteps, "future-redownload-executor") {
		t.Fatalf("unexpected unavailable recovery plan: %+v", missingContent)
	}
}

func TestMediaAssetContentRecoveryPreflightRequiresApprovedControlMutation(t *testing.T) {
	ctx := context.Background()
	assetRoot := t.TempDir()
	reader, err := localmedia.NewReader([]string{assetRoot})
	if err != nil {
		t.Fatal(err)
	}
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetServiceWithContent(store, reader)
	registerTestMediaAsset(t, mediaAssets, "asset:missing", filepath.Join(assetRoot, "missing.txt"), "missing.txt")

	approvals := appservice.NewOperatorApprovalService()
	controlPreflight := appservice.NewControlMutationPreflightService(approvals)
	service := appservice.NewMediaAssetContentRecoveryPreflightService(mediaAssets, controlPreflight)

	blocked, err := service.CheckMediaAssetContentRecoveryPreflight(ctx, query.MediaAssetContentRecoveryPreflightFilter{
		AssetID:    "asset:missing",
		TargetID:   "asset:missing",
		OperatorID: "qsyy",
	})
	if err != nil {
		t.Fatalf("content recovery preflight blocked: %v", err)
	}
	if blocked.Ready ||
		blocked.Reason != "missing_approval_id" ||
		blocked.TargetKind != "media_asset_content" ||
		blocked.Action != "recover_content" ||
		!blocked.RecoveryNeeded ||
		blocked.ExecutorScope != "media_content_cache_executor" ||
		blocked.SideEffect != "none" {
		t.Fatalf("unexpected blocked preflight: %+v", blocked)
	}

	approval, err := approvals.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "media_asset_content",
		TargetID:   "asset:missing",
		Decision:   "approved",
		OperatorID: "qsyy",
		Timestamp:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	ready, err := service.CheckMediaAssetContentRecoveryPreflight(ctx, query.MediaAssetContentRecoveryPreflightFilter{
		AssetID:    "asset:missing",
		OperatorID: "qsyy",
		ApprovalID: approval.ApprovalID,
	})
	if err != nil {
		t.Fatalf("content recovery preflight ready: %v", err)
	}
	if !ready.Ready ||
		ready.Reason != "media_asset_content_recovery_preflight_ready" ||
		ready.SuggestedAudit == nil ||
		ready.SuggestedAudit.TargetKind != "media_asset_content" ||
		ready.SuggestedAudit.Action != "recover_content" ||
		ready.SuggestedAudit.Status != "planned" ||
		ready.SideEffect != "none" {
		t.Fatalf("unexpected ready preflight: %+v", ready)
	}
}

func TestMediaAssetContentRecoveryPreflightBlocksWhenRecoveryNotRequired(t *testing.T) {
	ctx := context.Background()
	assetRoot := t.TempDir()
	readyPath := filepath.Join(assetRoot, "ready.txt")
	if err := writeTestFile(readyPath, "ready bytes"); err != nil {
		t.Fatal(err)
	}
	reader, err := localmedia.NewReader([]string{assetRoot})
	if err != nil {
		t.Fatal(err)
	}
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetServiceWithContent(store, reader)
	registerTestMediaAsset(t, mediaAssets, "asset:ready", readyPath, "ready.txt")
	service := appservice.NewMediaAssetContentRecoveryPreflightService(mediaAssets, nil)

	view, err := service.CheckMediaAssetContentRecoveryPreflight(ctx, query.MediaAssetContentRecoveryPreflightFilter{
		AssetID: "asset:ready",
	})
	if err != nil {
		t.Fatalf("content recovery preflight: %v", err)
	}
	if view.Ready ||
		view.Reason != "media_asset_content_recovery_not_required" ||
		view.RecoveryNeeded ||
		len(view.Blockers) == 0 ||
		view.SideEffect != "none" {
		t.Fatalf("unexpected not-required preflight: %+v", view)
	}
}

func TestMediaAssetContentRecoveryExecutorDownloadsCachesAndAudits(t *testing.T) {
	ctx := context.Background()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png bytes"))
	}))
	defer source.Close()

	assetRoot := t.TempDir()
	cacheRoot := filepath.Join(assetRoot, "recovered")
	reader, err := localmedia.NewReader([]string{assetRoot})
	if err != nil {
		t.Fatal(err)
	}
	downloader, err := localmedia.NewDownloader(cacheRoot, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetServiceWithContent(store, reader)
	registerTestMediaAsset(t, mediaAssets, "asset:remote", source.URL+"/image.png", "image.png")

	approvals := appservice.NewOperatorApprovalService()
	approval, err := approvals.RecordOperatorApproval(ctx, command.RecordOperatorApprovalCommand{
		TargetKind: "media_asset_content",
		TargetID:   "asset:remote",
		Decision:   "approved",
		OperatorID: "qsyy",
		Timestamp:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("record approval: %v", err)
	}
	preflight := appservice.NewMediaAssetContentRecoveryPreflightService(mediaAssets, appservice.NewControlMutationPreflightService(approvals))
	audits := appservice.NewControlMutationAuditService()
	recoverer := appservice.NewMediaAssetContentRecoveryService(store, preflight, downloader, audits)

	dryRun, err := recoverer.RecoverMediaAssetContent(ctx, command.RecoverMediaAssetContentCommand{
		AssetID:    "asset:remote",
		OperatorID: "qsyy",
		ApprovalID: approval.ApprovalID,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("dry-run content recovery: %v", err)
	}
	if !dryRun.Ready || dryRun.Applied || dryRun.Reason != "media_asset_content_recovery_dry_run" || dryRun.SideEffect != "none" {
		t.Fatalf("unexpected dry-run recovery: %+v", dryRun)
	}
	if _, err := os.Stat(filepath.Join(cacheRoot, "asset_remote.png")); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create cache file, stat err=%v", err)
	}

	view, err := recoverer.RecoverMediaAssetContent(ctx, command.RecoverMediaAssetContentCommand{
		AssetID:    "asset:remote",
		OperatorID: "qsyy",
		ApprovalID: approval.ApprovalID,
		MutationID: "mutation-recover-remote",
	})
	if err != nil {
		t.Fatalf("content recovery: %v", err)
	}
	if !view.Ready || !view.Applied ||
		view.Reason != "media_asset_content_recovery_applied" ||
		view.LocalPath == "" ||
		view.ContentMimeType != "image/png" ||
		view.ContentSizeBytes != int64(len("png bytes")) ||
		!strings.HasPrefix(view.ContentHash, "sha256:") ||
		view.AppliedAudit == nil ||
		view.SideEffect != "local_cache_write_and_runtime_state_update" {
		t.Fatalf("unexpected recovery view: %+v", view)
	}
	if raw, err := os.ReadFile(view.LocalPath); err != nil || string(raw) != "png bytes" {
		t.Fatalf("unexpected recovered file content raw=%q err=%v", string(raw), err)
	}
	access, err := mediaAssets.ContentAccessPlan(ctx, "asset:remote")
	if err != nil {
		t.Fatalf("content access after recovery: %v", err)
	}
	if !access.Ready || access.Reason != "media_asset_content_ready" || access.ContentSizeBytes != int64(len("png bytes")) {
		t.Fatalf("expected recovered content ready: %+v", access)
	}
	asset, ok, err := store.FindMediaAsset(ctx, "asset:remote")
	if err != nil || !ok {
		t.Fatalf("find recovered asset ok=%t err=%v", ok, err)
	}
	if asset.Metadata["local_path"] != view.LocalPath ||
		asset.Metadata["recovered_from_url"] != source.URL+"/image.png" ||
		asset.Metadata["recovery_scope"] != "download_to_local_cache" ||
		asset.Metadata["recovery_approval_id"] != approval.ApprovalID ||
		asset.Metadata["recovery_mutation_id"] != "mutation-recover-remote" {
		t.Fatalf("unexpected recovered metadata: %+v", asset.Metadata)
	}
}

func TestMediaAssetContentRecoveryExecutorBlocksWithoutApprovalBeforeDownload(t *testing.T) {
	ctx := context.Background()
	calls := 0
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte("should not download"))
	}))
	defer source.Close()

	assetRoot := t.TempDir()
	reader, err := localmedia.NewReader([]string{assetRoot})
	if err != nil {
		t.Fatal(err)
	}
	downloader, err := localmedia.NewDownloader(filepath.Join(assetRoot, "recovered"), 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	store := memory.NewStore()
	mediaAssets := appservice.NewMediaAssetServiceWithContent(store, reader)
	registerTestMediaAsset(t, mediaAssets, "asset:no-approval", source.URL+"/file.bin", "file.bin")
	preflight := appservice.NewMediaAssetContentRecoveryPreflightService(mediaAssets, appservice.NewControlMutationPreflightService(appservice.NewOperatorApprovalService()))
	recoverer := appservice.NewMediaAssetContentRecoveryService(store, preflight, downloader, appservice.NewControlMutationAuditService())

	view, err := recoverer.RecoverMediaAssetContent(ctx, command.RecoverMediaAssetContentCommand{
		AssetID:    "asset:no-approval",
		OperatorID: "qsyy",
	})
	if err != nil {
		t.Fatalf("blocked content recovery: %v", err)
	}
	if view.Ready || view.Applied || view.Reason != "missing_approval_id" || len(view.Blockers) == 0 || calls != 0 {
		t.Fatalf("expected approval blocker before download, calls=%d view=%+v", calls, view)
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
	if missingID.RuntimePath != "" || missingID.DashboardPath != "" || missingID.ContentURL != "" {
		t.Fatalf("missing id plan should not expose URL hints: %+v", missingID)
	}

	disabled, err := service.ContentAccessPlan(ctx, "asset:disabled")
	if err != nil {
		t.Fatalf("content access plan disabled: %v", err)
	}
	if disabled.Ready ||
		disabled.Reason != "media_asset_content_disabled" ||
		disabled.Asset == nil ||
		disabled.RuntimePath != "/v1/media-assets/content-access-plan?asset_id=asset%3Adisabled" ||
		disabled.DashboardPath != "/api/dashboard/media-assets/content-access-plan?asset_id=asset%3Adisabled" ||
		disabled.ContentURL != "/api/dashboard/media-assets/content?asset_id=asset%3Adisabled" ||
		disabled.ContentEndpoint != "/v1/media-assets/asset:disabled/content" ||
		disabled.SideEffect != "none" {
		t.Fatalf("unexpected disabled plan: %+v", disabled)
	}

	disabledRecovery, err := service.ContentRecoveryPlan(ctx, "asset:disabled")
	if err != nil {
		t.Fatalf("content recovery plan disabled: %v", err)
	}
	if disabledRecovery.Ready ||
		disabledRecovery.Reason != "media_asset_content_recovery_enable_reader" ||
		disabledRecovery.FutureExecutorScope != "operator_runtime_config" ||
		!mediaAssetStepNamesContain(disabledRecovery.FallbackSteps, "enable-content-reader") {
		t.Fatalf("unexpected disabled recovery plan: %+v", disabledRecovery)
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

func mediaAssetStepNamesContain(steps []query.MediaAssetContentAccessStep, name string) bool {
	for _, step := range steps {
		if step.Name == name {
			return true
		}
	}
	return false
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
