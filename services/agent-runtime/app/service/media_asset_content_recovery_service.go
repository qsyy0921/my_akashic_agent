package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type mediaAssetContentRecoveryAuditRecorder interface {
	RecordControlMutationAudit(ctx context.Context, cmd command.RecordControlMutationAuditCommand) (query.ControlMutationAuditView, error)
}

type MediaAssetContentRecoveryService struct {
	repository outport.MediaAssetRepository
	preflight  *MediaAssetContentRecoveryPreflightService
	downloader outport.MediaAssetContentDownloader
	audits     mediaAssetContentRecoveryAuditRecorder
}

func NewMediaAssetContentRecoveryService(
	repository outport.MediaAssetRepository,
	preflight *MediaAssetContentRecoveryPreflightService,
	downloader outport.MediaAssetContentDownloader,
	audits mediaAssetContentRecoveryAuditRecorder,
) *MediaAssetContentRecoveryService {
	return &MediaAssetContentRecoveryService{
		repository: repository,
		preflight:  preflight,
		downloader: downloader,
		audits:     audits,
	}
}

func (s *MediaAssetContentRecoveryService) RecoverMediaAssetContent(
	ctx context.Context,
	cmd command.RecoverMediaAssetContentCommand,
) (query.MediaAssetContentRecoveryView, error) {
	if err := ctx.Err(); err != nil {
		return query.MediaAssetContentRecoveryView{}, err
	}
	if s == nil || s.repository == nil || s.preflight == nil {
		return query.MediaAssetContentRecoveryView{}, errors.New("media asset content recovery requires repository and preflight")
	}
	assetID := strings.TrimSpace(cmd.AssetID)
	targetID := strings.TrimSpace(cmd.TargetID)
	if targetID == "" {
		targetID = assetID
	}
	preflight, err := s.preflight.CheckMediaAssetContentRecoveryPreflight(ctx, query.MediaAssetContentRecoveryPreflightFilter{
		AssetID:    assetID,
		TargetID:   targetID,
		OperatorID: cmd.OperatorID,
		ApprovalID: cmd.ApprovalID,
	})
	if err != nil {
		return query.MediaAssetContentRecoveryView{}, err
	}
	view := query.MediaAssetContentRecoveryView{
		Ready:      preflight.Ready,
		DryRun:     cmd.DryRun,
		Reason:     preflight.Reason,
		Blockers:   append([]string(nil), preflight.Blockers...),
		TargetKind: preflight.TargetKind,
		TargetID:   preflight.TargetID,
		Action:     preflight.Action,
		OperatorID: strings.TrimSpace(cmd.OperatorID),
		ApprovalID: strings.TrimSpace(cmd.ApprovalID),
		MutationID: strings.TrimSpace(cmd.MutationID),
		AssetID:    assetID,
		Preflight:  preflight,
		SideEffect: "none",
		Notes: []string{
			"approval-bound media content recovery executor; writes only to configured local cache roots",
			"Python remains responsible for OCR, VLM, file parsing, semantic memory and RAG extraction",
		},
	}
	if cmd.DryRun {
		view.Reason = "media_asset_content_recovery_dry_run"
		return view, nil
	}
	if !preflight.Ready {
		return view, nil
	}
	if s.downloader == nil {
		view.Ready = false
		view.Reason = "media_asset_content_recovery_downloader_unavailable"
		view.Blockers = []string{"media_asset_content_recovery_downloader_unavailable"}
		return view, nil
	}
	if s.audits == nil {
		view.Ready = false
		view.Reason = "control_mutation_audit_unavailable"
		view.Blockers = []string{"control_mutation_audit_unavailable"}
		return view, nil
	}
	asset, ok, err := s.repository.FindMediaAsset(ctx, assetID)
	if err != nil {
		return query.MediaAssetContentRecoveryView{}, err
	}
	if !ok {
		view.Ready = false
		view.Reason = "media_asset_content_recovery_asset_not_found"
		view.Blockers = []string{"media_asset_content_recovery_asset_not_found"}
		return view, nil
	}
	recovered, err := s.downloader.RecoverMediaAssetContent(ctx, asset)
	if err != nil {
		failed := s.recordMediaAssetContentRecoveryAudit(ctx, cmd, "failed", contentRecoveryFailureReason(err), recovered)
		view.FailedAudit = failed
		view.Ready = false
		view.Reason = contentRecoveryFailureReason(err)
		view.Blockers = []string{view.Reason}
		view.SideEffect = "none"
		return view, nil
	}

	now := time.Now().UTC()
	metadata := copyMediaAssetContentRecoveryMetadata(asset.Metadata)
	metadata["local_path"] = recovered.LocalPath
	metadata["recovered_from_url"] = recovered.SourceURL
	metadata["recovered_at"] = now.Format(time.RFC3339Nano)
	metadata["recovery_scope"] = "download_to_local_cache"
	metadata["recovery_approval_id"] = strings.TrimSpace(cmd.ApprovalID)
	if mutationID := strings.TrimSpace(cmd.MutationID); mutationID != "" {
		metadata["recovery_mutation_id"] = mutationID
	}
	asset.Metadata = metadata
	asset.URL = recovered.LocalPath
	if recovered.Name != "" {
		asset.Name = recovered.Name
	}
	if recovered.MimeType != "" {
		asset.MimeType = recovered.MimeType
	}
	if recovered.SizeBytes >= 0 {
		asset.SizeBytes = recovered.SizeBytes
	}
	if recovered.ContentHash != "" {
		asset.ContentHash = recovered.ContentHash
	}
	asset.UpdatedAt = now
	if err := s.repository.SaveMediaAsset(ctx, asset); err != nil {
		failed := s.recordMediaAssetContentRecoveryAudit(ctx, cmd, "failed", "media_asset_content_recovery_registry_update_failed: "+err.Error(), recovered)
		view.FailedAudit = failed
		view.Ready = false
		view.Reason = "media_asset_content_recovery_registry_update_failed"
		view.Blockers = []string{"media_asset_content_recovery_registry_update_failed"}
		view.SideEffect = "local_cache_write"
		return view, nil
	}
	applied := s.recordMediaAssetContentRecoveryAudit(ctx, cmd, "applied", "media content recovered to local cache", recovered)
	recoveredView := assembler.ToMediaAssetView(asset)
	view.Applied = true
	view.Reason = "media_asset_content_recovery_applied"
	view.LocalPath = recovered.LocalPath
	view.ContentMimeType = recovered.MimeType
	view.ContentSizeBytes = recovered.SizeBytes
	view.ContentHash = recovered.ContentHash
	view.RecoveredAsset = &recoveredView
	view.AppliedAudit = applied
	view.SideEffect = "local_cache_write_and_runtime_state_update"
	return view, nil
}

func (s *MediaAssetContentRecoveryService) recordMediaAssetContentRecoveryAudit(
	ctx context.Context,
	cmd command.RecoverMediaAssetContentCommand,
	status string,
	reason string,
	recovered outport.RecoveredMediaAssetContent,
) *query.ControlMutationAuditView {
	if s == nil || s.audits == nil {
		return nil
	}
	audit, err := s.audits.RecordControlMutationAudit(ctx, command.RecordControlMutationAuditCommand{
		MutationID: strings.TrimSpace(cmd.MutationID),
		TargetKind: mediaAssetContentRecoveryTargetKind,
		TargetID:   firstNonBlankMediaAssetContentRecovery(strings.TrimSpace(cmd.TargetID), strings.TrimSpace(cmd.AssetID)),
		Action:     mediaAssetContentRecoveryAction,
		Status:     status,
		OperatorID: strings.TrimSpace(cmd.OperatorID),
		ApprovalID: strings.TrimSpace(cmd.ApprovalID),
		Reason:     reason,
		Metadata: map[string]string{
			"asset_id":       strings.TrimSpace(cmd.AssetID),
			"local_path":     recovered.LocalPath,
			"source_url":     recovered.SourceURL,
			"content_hash":   recovered.ContentHash,
			"content_size":   strconv.FormatInt(recovered.SizeBytes, 10),
			"recovery_scope": "download_to_local_cache",
		},
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return nil
	}
	return &audit
}

func contentRecoveryFailureReason(err error) string {
	switch {
	case errors.Is(err, outport.ErrMediaAssetRecoveryUnsupported):
		return "media_asset_content_recovery_source_unsupported"
	case errors.Is(err, outport.ErrMediaAssetContentUnavailable):
		return "media_asset_content_recovery_source_unavailable"
	case errors.Is(err, outport.ErrMediaAssetContentForbidden):
		return "media_asset_content_recovery_forbidden"
	default:
		return "media_asset_content_recovery_failed"
	}
}

func firstNonBlankMediaAssetContentRecovery(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func copyMediaAssetContentRecoveryMetadata(input map[string]string) map[string]string {
	output := make(map[string]string, len(input)+6)
	for key, value := range input {
		output[key] = value
	}
	return output
}
