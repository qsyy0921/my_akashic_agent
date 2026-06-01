package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type MediaAssetService struct {
	repository    outport.MediaAssetRepository
	contentReader outport.MediaAssetContentReader
}

func NewMediaAssetService(repository outport.MediaAssetRepository) *MediaAssetService {
	return &MediaAssetService{repository: repository}
}

func NewMediaAssetServiceWithContent(
	repository outport.MediaAssetRepository,
	contentReader outport.MediaAssetContentReader,
) *MediaAssetService {
	return &MediaAssetService{repository: repository, contentReader: contentReader}
}

func (s *MediaAssetService) Register(ctx context.Context, cmd command.RegisterMediaAssetCommand) (query.MediaAssetView, error) {
	if s == nil || s.repository == nil {
		return query.MediaAssetView{}, errors.New("media asset service requires repository")
	}
	if cmd.Timestamp.IsZero() {
		cmd.Timestamp = time.Now().UTC()
	}
	assetID := strings.TrimSpace(cmd.AssetID)
	if assetID == "" {
		assetID = generatedAssetID(cmd)
	}
	if existing, ok, err := s.repository.FindMediaAsset(ctx, assetID); err != nil {
		return query.MediaAssetView{}, err
	} else if ok {
		return assembler.ToMediaAssetView(existing), nil
	}

	asset, err := model.NewMediaAsset(assembler.ToChannelRef(cmd.Channel), model.MediaAssetSpec{
		AssetID:         assetID,
		SourceMessageID: cmd.SourceMessageID,
		SenderID:        cmd.SenderID,
		Kind:            model.MediaAssetKind(cmd.Kind),
		URL:             cmd.URL,
		MimeType:        cmd.MimeType,
		Name:            cmd.Name,
		SizeBytes:       cmd.SizeBytes,
		ContentHash:     cmd.ContentHash,
		Retention:       cmd.Retention,
		Metadata:        cmd.Metadata,
	}, cmd.Timestamp)
	if err != nil {
		return query.MediaAssetView{}, err
	}
	if err := s.repository.SaveMediaAsset(ctx, asset); err != nil {
		return query.MediaAssetView{}, err
	}
	return assembler.ToMediaAssetView(asset), nil
}

func (s *MediaAssetService) Get(ctx context.Context, assetID string) (query.MediaAssetView, error) {
	if s == nil || s.repository == nil {
		return query.MediaAssetView{}, errors.New("media asset service requires repository")
	}
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return query.MediaAssetView{}, errors.New("media asset id required")
	}
	asset, ok, err := s.repository.FindMediaAsset(ctx, assetID)
	if err != nil {
		return query.MediaAssetView{}, err
	}
	if !ok {
		return query.MediaAssetView{}, errors.New("media asset not found")
	}
	return assembler.ToMediaAssetView(asset), nil
}

func (s *MediaAssetService) List(ctx context.Context, filter query.MediaAssetFilter) ([]query.MediaAssetView, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("media asset service requires repository")
	}
	items, err := s.repository.ListMediaAssets(ctx, filter)
	if err != nil {
		return nil, err
	}
	return assembler.ToMediaAssetViews(items), nil
}

func (s *MediaAssetService) OpenContent(ctx context.Context, assetID string) (outport.MediaAssetContent, error) {
	if s == nil || s.repository == nil {
		return outport.MediaAssetContent{}, errors.New("media asset service requires repository")
	}
	if s.contentReader == nil {
		return outport.MediaAssetContent{}, outport.ErrMediaAssetContentDisabled
	}
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return outport.MediaAssetContent{}, errors.New("media asset id required")
	}
	asset, ok, err := s.repository.FindMediaAsset(ctx, assetID)
	if err != nil {
		return outport.MediaAssetContent{}, err
	}
	if !ok {
		return outport.MediaAssetContent{}, errors.New("media asset not found")
	}
	return s.contentReader.OpenMediaAssetContent(ctx, asset)
}

func (s *MediaAssetService) ContentDiagnostics(
	ctx context.Context,
	filter query.MediaAssetContentDiagnosticsFilter,
) (query.MediaAssetContentDiagnosticsView, error) {
	if s == nil || s.repository == nil {
		return query.MediaAssetContentDiagnosticsView{}, errors.New("media asset service requires repository")
	}
	if err := ctx.Err(); err != nil {
		return query.MediaAssetContentDiagnosticsView{}, err
	}
	assets, err := s.contentDiagnosticAssets(ctx, filter)
	if err != nil {
		return query.MediaAssetContentDiagnosticsView{}, err
	}
	items := make([]query.MediaAssetContentDiagnosticItemView, 0, len(assets))
	totals := map[string]int{
		"assets":      len(assets),
		"ready":       0,
		"forbidden":   0,
		"unavailable": 0,
		"disabled":    0,
		"error":       0,
	}
	for _, asset := range assets {
		item := s.contentDiagnosticItem(ctx, asset)
		totals[item.ContentStatus]++
		items = append(items, item)
	}
	return query.MediaAssetContentDiagnosticsView{
		Items:      items,
		Totals:     totals,
		SideEffect: "none",
		Notes: []string{
			"read-only media asset content diagnostics; opened content is closed immediately",
			"Go checks deterministic content access only; Python remains responsible for OCR, VLM, file parsing and semantic extraction",
		},
	}, nil
}

func (s *MediaAssetService) ContentAccessPlan(
	ctx context.Context,
	assetID string,
) (query.MediaAssetContentAccessPlanView, error) {
	if s == nil || s.repository == nil {
		return query.MediaAssetContentAccessPlanView{}, errors.New("media asset service requires repository")
	}
	if err := ctx.Err(); err != nil {
		return query.MediaAssetContentAccessPlanView{}, err
	}
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return mediaAssetContentAccessPlan("", nil, "", "", 0, "media_asset_content_asset_id_required", []string{
			"asset_id query parameter is required",
		}), nil
	}
	asset, ok, err := s.repository.FindMediaAsset(ctx, assetID)
	if err != nil {
		return query.MediaAssetContentAccessPlanView{}, err
	}
	if !ok {
		return mediaAssetContentAccessPlan(assetID, nil, "", "", 0, "media_asset_content_asset_not_found", []string{
			"media asset metadata was not found",
		}), nil
	}
	view := assembler.ToMediaAssetView(asset)
	endpoint := mediaAssetContentEndpoint(view.AssetID)
	if s.contentReader == nil {
		return mediaAssetContentAccessPlan(assetID, &view, endpoint, "", 0, "media_asset_content_disabled", []string{
			"media asset content reader is disabled",
		}), nil
	}
	content, err := s.contentReader.OpenMediaAssetContent(ctx, asset)
	switch {
	case err == nil:
		_ = content.Body.Close()
		return mediaAssetContentAccessPlan(assetID, &view, endpoint, content.MimeType, content.SizeBytes, "media_asset_content_ready", nil), nil
	case errors.Is(err, outport.ErrMediaAssetContentDisabled):
		return mediaAssetContentAccessPlan(assetID, &view, endpoint, "", 0, "media_asset_content_disabled", []string{
			"media asset content reader is disabled",
		}), nil
	case errors.Is(err, outport.ErrMediaAssetContentForbidden):
		return mediaAssetContentAccessPlan(assetID, &view, endpoint, "", 0, "media_asset_content_forbidden", []string{
			"media asset local path is outside configured content roots",
		}), nil
	case errors.Is(err, outport.ErrMediaAssetContentUnavailable):
		return mediaAssetContentAccessPlan(assetID, &view, endpoint, "", 0, "media_asset_content_unavailable", []string{
			"media asset local content is unavailable",
		}), nil
	default:
		return mediaAssetContentAccessPlan(assetID, &view, endpoint, "", 0, "media_asset_content_error", []string{
			"media asset content probe failed: " + err.Error(),
		}), nil
	}
}

func (s *MediaAssetService) RetentionDiagnostics(
	ctx context.Context,
	filter query.MediaAssetRetentionDiagnosticsFilter,
) (query.MediaAssetRetentionDiagnosticsView, error) {
	if s == nil || s.repository == nil {
		return query.MediaAssetRetentionDiagnosticsView{}, errors.New("media asset service requires repository")
	}
	if err := ctx.Err(); err != nil {
		return query.MediaAssetRetentionDiagnosticsView{}, err
	}
	now := time.Now().UTC()
	if strings.TrimSpace(filter.Timestamp) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, filter.Timestamp)
		if err != nil {
			return query.MediaAssetRetentionDiagnosticsView{}, err
		}
		now = parsed.UTC()
	}
	assets, err := s.retentionDiagnosticAssets(ctx, filter)
	if err != nil {
		return query.MediaAssetRetentionDiagnosticsView{}, err
	}
	items := make([]query.MediaAssetRetentionDiagnosticItemView, 0, len(assets))
	totals := map[string]int{
		"assets":      len(assets),
		"cleanup_due": 0,
		"permanent":   0,
		"default":     0,
		"ephemeral":   0,
		"unknown":     0,
	}
	for _, asset := range assets {
		item := mediaAssetRetentionDiagnosticItem(asset, now, filter)
		if _, ok := totals[item.RetentionClass]; ok {
			totals[item.RetentionClass]++
		} else {
			totals["unknown"]++
		}
		if item.CleanupDue {
			totals["cleanup_due"]++
		}
		items = append(items, item)
	}
	return query.MediaAssetRetentionDiagnosticsView{
		Items:      items,
		Totals:     totals,
		SideEffect: "none",
		Notes: []string{
			"read-only media asset retention diagnostics; no metadata or file content is deleted",
			"Go owns deterministic retention visibility; Python remains responsible for OCR, VLM, file parsing and semantic extraction",
		},
	}, nil
}

func (s *MediaAssetService) RetentionPlan(
	ctx context.Context,
	filter query.MediaAssetRetentionDiagnosticsFilter,
) (query.MediaAssetRetentionPlanView, error) {
	diagnostics, err := s.RetentionDiagnostics(ctx, filter)
	if err != nil {
		return query.MediaAssetRetentionPlanView{}, err
	}
	candidates := make([]query.MediaAssetRetentionDiagnosticItemView, 0, len(diagnostics.Items))
	for _, item := range diagnostics.Items {
		if item.CleanupDue {
			candidates = append(candidates, item)
		}
	}
	blockers := []string{}
	ready := len(candidates) > 0
	reason := "media_asset_retention_cleanup_candidates_ready"
	if !ready {
		reason = "media_asset_retention_no_cleanup_candidates"
		blockers = append(blockers, "no media assets are past retention ttl under current filter")
	}
	return query.MediaAssetRetentionPlanView{
		Ready:          ready,
		Reason:         reason,
		Blockers:       blockers,
		AssetCount:     intFromTotals(diagnostics.Totals, "assets"),
		CandidateCount: len(candidates),
		Candidates:     candidates,
		RequiredSteps:  mediaAssetRetentionPlanRequiredSteps(),
		VerifySteps:    mediaAssetRetentionPlanVerifySteps(),
		RollbackSteps:  mediaAssetRetentionPlanRollbackSteps(),
		Diagnostics:    diagnostics,
		SideEffect:     "none",
		Notes: []string{
			"read-only media asset retention cleanup plan; no metadata or file content is deleted",
			"destructive cleanup must be implemented in a separate SDD slice with operator approval and control mutation audit binding",
		},
	}, nil
}

func mediaAssetRetentionPlanRequiredSteps() []query.MediaAssetRetentionPlanStep {
	return []query.MediaAssetRetentionPlanStep{
		{
			Name:        "inspect-retention-diagnostics",
			Description: "Review retention diagnostics and confirm only expired non-permanent media assets are selected.",
			Endpoint:    "/v1/media-assets/retention-diagnostics",
			Method:      "GET",
		},
		{
			Name:        "record-operator-approval",
			Description: "Record an explicit operator approval before any future destructive cleanup executor is allowed to run.",
			Endpoint:    "/v1/operator-approvals",
			Method:      "POST",
			Metadata: map[string]string{
				"target_kind": "media_asset_retention",
				"action":      "cleanup_expired",
			},
		},
		{
			Name:        "record-planned-control-mutation",
			Description: "Record a planned control mutation audit bound to the approval id before deletion is implemented.",
			Endpoint:    "/v1/control-mutations",
			Method:      "POST",
			Metadata: map[string]string{
				"target_kind": "media_asset_retention",
				"action":      "cleanup_expired",
				"status":      "planned",
			},
		},
	}
}

func mediaAssetRetentionPlanVerifySteps() []query.MediaAssetRetentionPlanStep {
	return []query.MediaAssetRetentionPlanStep{
		{
			Name:        "rerun-retention-plan",
			Description: "After a future cleanup executor runs, rerun this plan and confirm candidate_count decreases as expected.",
			Endpoint:    "/v1/media-assets/retention-plan",
			Method:      "GET",
		},
		{
			Name:        "verify-content-access",
			Description: "Sample remaining assets through content diagnostics and confirm non-expired assets still resolve normally.",
			Endpoint:    "/v1/media-assets/content-diagnostics",
			Method:      "GET",
		},
	}
}

func mediaAssetRetentionPlanRollbackSteps() []query.MediaAssetRetentionPlanStep {
	return []query.MediaAssetRetentionPlanStep{
		{
			Name:        "restore-from-runtime-backup",
			Description: "Restore deleted metadata or local content from the operator-approved backup path if a future cleanup executor removes the wrong assets.",
		},
		{
			Name:        "record-rollback-control-mutation",
			Description: "Record rollback evidence in the control mutation audit ledger.",
			Endpoint:    "/v1/control-mutations",
			Method:      "POST",
			Metadata: map[string]string{
				"target_kind": "media_asset_retention",
				"action":      "cleanup_expired",
				"status":      "rolled_back",
			},
		},
	}
}

func intFromTotals(totals map[string]int, key string) int {
	if totals == nil {
		return 0
	}
	return totals[key]
}

func (s *MediaAssetService) retentionDiagnosticAssets(
	ctx context.Context,
	filter query.MediaAssetRetentionDiagnosticsFilter,
) ([]model.MediaAsset, error) {
	assetID := strings.TrimSpace(filter.AssetID)
	if assetID != "" {
		asset, ok, err := s.repository.FindMediaAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("media asset not found")
		}
		return []model.MediaAsset{asset}, nil
	}
	return s.repository.ListMediaAssets(ctx, query.MediaAssetFilter{
		Limit:                 filter.Limit,
		ChannelKind:           filter.ChannelKind,
		AccountID:             filter.AccountID,
		ConversationID:        filter.ConversationID,
		ConversationType:      filter.ConversationType,
		SourceMessageID:       filter.SourceMessageID,
		SourceMessageIDSuffix: filter.SourceMessageIDSuffix,
		Kind:                  filter.Kind,
	})
}

func (s *MediaAssetService) contentDiagnosticAssets(
	ctx context.Context,
	filter query.MediaAssetContentDiagnosticsFilter,
) ([]model.MediaAsset, error) {
	assetID := strings.TrimSpace(filter.AssetID)
	if assetID != "" {
		asset, ok, err := s.repository.FindMediaAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("media asset not found")
		}
		return []model.MediaAsset{asset}, nil
	}
	return s.repository.ListMediaAssets(ctx, query.MediaAssetFilter{
		Limit:                 filter.Limit,
		ChannelKind:           filter.ChannelKind,
		AccountID:             filter.AccountID,
		ConversationID:        filter.ConversationID,
		ConversationType:      filter.ConversationType,
		SourceMessageID:       filter.SourceMessageID,
		SourceMessageIDSuffix: filter.SourceMessageIDSuffix,
		Kind:                  filter.Kind,
	})
}

func mediaAssetRetentionDiagnosticItem(
	asset model.MediaAsset,
	now time.Time,
	filter query.MediaAssetRetentionDiagnosticsFilter,
) query.MediaAssetRetentionDiagnosticItemView {
	ttl, retentionClass := mediaAssetRetentionTTL(asset.Retention, filter)
	age := int(now.Sub(asset.CreatedAt.UTC()).Seconds())
	if age < 0 {
		age = 0
	}
	cleanupDue := false
	cleanupAfter := ""
	reason := "media_asset_retention_not_due"
	ttlSeconds := 0
	if ttl <= 0 {
		reason = "media_asset_retention_permanent"
	} else {
		ttlSeconds = int(ttl.Seconds())
		cutoff := asset.CreatedAt.UTC().Add(ttl)
		cleanupAfter = cutoff.Format(time.RFC3339Nano)
		if !now.Before(cutoff) {
			cleanupDue = true
			reason = "media_asset_retention_due"
		}
	}
	view := assembler.ToMediaAssetView(asset)
	return query.MediaAssetRetentionDiagnosticItemView{
		AssetID:         view.AssetID,
		Channel:         view.Channel,
		SourceMessageID: view.SourceMessageID,
		SenderID:        view.SenderID,
		Kind:            view.Kind,
		MimeType:        view.MimeType,
		Name:            view.Name,
		SizeBytes:       view.SizeBytes,
		Retention:       view.Retention,
		RetentionClass:  retentionClass,
		CleanupDue:      cleanupDue,
		AgeSeconds:      age,
		TTLSeconds:      ttlSeconds,
		CleanupAfter:    cleanupAfter,
		CleanupReason:   reason,
		CreatedAt:       view.CreatedAt,
		UpdatedAt:       view.UpdatedAt,
	}
}

func mediaAssetRetentionTTL(retention string, filter query.MediaAssetRetentionDiagnosticsFilter) (time.Duration, string) {
	retention = strings.ToLower(strings.TrimSpace(retention))
	switch retention {
	case "permanent", "keep", "never":
		return 0, "permanent"
	case "ephemeral", "temp", "temporary", "short":
		return time.Duration(positiveOrDefault(filter.EphemeralTTLHours, 24)) * time.Hour, "ephemeral"
	case "", "default", "default-observed-group":
		return time.Duration(positiveOrDefault(filter.DefaultTTLHours, 30*24)) * time.Hour, "default"
	default:
		return time.Duration(positiveOrDefault(filter.DefaultTTLHours, 30*24)) * time.Hour, "unknown"
	}
}

func positiveOrDefault(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func (s *MediaAssetService) contentDiagnosticItem(
	ctx context.Context,
	asset model.MediaAsset,
) query.MediaAssetContentDiagnosticItemView {
	status := "disabled"
	reason := "media_asset_content_disabled"
	contentMimeType := ""
	contentSizeBytes := int64(0)
	if s.contentReader != nil {
		content, err := s.contentReader.OpenMediaAssetContent(ctx, asset)
		switch {
		case err == nil:
			status = "ready"
			reason = "media_asset_content_ready"
			contentMimeType = content.MimeType
			contentSizeBytes = content.SizeBytes
			_ = content.Body.Close()
		case errors.Is(err, outport.ErrMediaAssetContentDisabled):
			status = "disabled"
			reason = "media_asset_content_disabled"
		case errors.Is(err, outport.ErrMediaAssetContentForbidden):
			status = "forbidden"
			reason = "media_asset_content_forbidden"
		case errors.Is(err, outport.ErrMediaAssetContentUnavailable):
			status = "unavailable"
			reason = "media_asset_content_unavailable"
		default:
			status = "error"
			reason = "media_asset_content_error"
		}
	}
	view := assembler.ToMediaAssetView(asset)
	return query.MediaAssetContentDiagnosticItemView{
		AssetID:                   view.AssetID,
		Channel:                   view.Channel,
		SourceMessageID:           view.SourceMessageID,
		SenderID:                  view.SenderID,
		Kind:                      view.Kind,
		MimeType:                  view.MimeType,
		Name:                      view.Name,
		SizeBytes:                 view.SizeBytes,
		ContentStatus:             status,
		ContentReason:             reason,
		ContentEndpoint:           mediaAssetContentEndpoint(view.AssetID),
		ContentAccessPlanEndpoint: mediaAssetContentAccessPlanEndpoint(view.AssetID),
		ContentMimeType:           contentMimeType,
		ContentSizeBytes:          contentSizeBytes,
		UpdatedAt:                 view.UpdatedAt,
	}
}

func mediaAssetContentAccessPlan(
	assetID string,
	asset *query.MediaAssetView,
	endpoint string,
	contentMimeType string,
	contentSizeBytes int64,
	reason string,
	blockers []string,
) query.MediaAssetContentAccessPlanView {
	ready := reason == "media_asset_content_ready"
	return query.MediaAssetContentAccessPlanView{
		Ready:            ready,
		Reason:           reason,
		Blockers:         blockers,
		AssetID:          assetID,
		Asset:            asset,
		ContentEndpoint:  endpoint,
		ContentMimeType:  contentMimeType,
		ContentSizeBytes: contentSizeBytes,
		RequiredSteps:    mediaAssetContentAccessRequiredSteps(assetID, endpoint, ready),
		VerifySteps:      mediaAssetContentAccessVerifySteps(assetID, endpoint),
		FallbackSteps:    mediaAssetContentAccessFallbackSteps(assetID),
		SideEffect:       "none",
		Notes: []string{
			"read-only media asset content access plan; content probe is closed immediately",
			"Go checks deterministic content availability only; Python remains responsible for OCR, VLM, file parsing and semantic extraction",
		},
	}
}

func mediaAssetContentAccessRequiredSteps(
	assetID string,
	endpoint string,
	ready bool,
) []query.MediaAssetContentAccessStep {
	steps := []query.MediaAssetContentAccessStep{
		{
			Name:        "inspect-content-diagnostics",
			Description: "Inspect deterministic media content status before opening the attachment.",
			Endpoint:    "/v1/media-assets/content-diagnostics?asset_id=" + url.QueryEscape(assetID),
			Method:      "GET",
		},
	}
	if ready {
		steps = append(steps, query.MediaAssetContentAccessStep{
			Name:        "open-content-endpoint",
			Description: "Open the media asset content endpoint from the frontend or dashboard link.",
			Endpoint:    endpoint,
			Method:      "GET",
		})
	}
	return steps
}

func mediaAssetContentAccessVerifySteps(assetID string, endpoint string) []query.MediaAssetContentAccessStep {
	steps := []query.MediaAssetContentAccessStep{
		{
			Name:        "rerun-content-access-plan",
			Description: "Rerun this plan and confirm ready/reason matches the expected content access state.",
			Endpoint:    "/v1/media-assets/content-access-plan?asset_id=" + url.QueryEscape(assetID),
			Method:      "GET",
		},
	}
	if endpoint != "" {
		steps = append(steps, query.MediaAssetContentAccessStep{
			Name:        "verify-content-route",
			Description: "Open the content endpoint and confirm HTTP status and content headers are expected.",
			Endpoint:    endpoint,
			Method:      "GET",
		})
	}
	return steps
}

func mediaAssetContentAccessFallbackSteps(assetID string) []query.MediaAssetContentAccessStep {
	return []query.MediaAssetContentAccessStep{
		{
			Name:        "check-local-media-roots",
			Description: "If the plan is forbidden, add only the intended media directory to configured Go content roots.",
		},
		{
			Name:        "restore-or-redownload-content",
			Description: "If the plan is unavailable, restore the local file or let the platform media downloader refresh it.",
			Metadata: map[string]string{
				"asset_id": assetID,
			},
		},
	}
}

func mediaAssetContentEndpoint(assetID string) string {
	if strings.TrimSpace(assetID) == "" {
		return ""
	}
	return "/v1/media-assets/" + url.PathEscape(assetID) + "/content"
}

func mediaAssetContentAccessPlanEndpoint(assetID string) string {
	if strings.TrimSpace(assetID) == "" {
		return ""
	}
	return "/v1/media-assets/content-access-plan?asset_id=" + url.QueryEscape(assetID)
}

func generatedAssetID(cmd command.RegisterMediaAssetCommand) string {
	index := cmd.Index
	if index <= 0 {
		index = 1
	}
	return fmt.Sprintf(
		"asset:%s:%s:%s:%s:%s:%d",
		safeAssetPart(cmd.Channel.Kind),
		safeAssetPart(cmd.Channel.AccountID),
		safeAssetPart(cmd.Channel.ConversationType),
		safeAssetPart(cmd.Channel.ConversationID),
		safeAssetPart(cmd.SourceMessageID),
		index,
	)
}

func safeAssetPart(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "\t", "_", "\n", "_", "\r", "_")
	value = replacer.Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
