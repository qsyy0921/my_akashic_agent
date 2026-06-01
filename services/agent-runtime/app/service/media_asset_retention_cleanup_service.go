package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type mediaAssetRetentionCleanupAuditRecorder interface {
	RecordControlMutationAudit(ctx context.Context, cmd command.RecordControlMutationAuditCommand) (query.ControlMutationAuditView, error)
}

type MediaAssetRetentionCleanupService struct {
	repository outport.MediaAssetRepository
	preflight  *MediaAssetRetentionCleanupPreflightService
	audits     mediaAssetRetentionCleanupAuditRecorder
}

func NewMediaAssetRetentionCleanupService(
	repository outport.MediaAssetRepository,
	preflight *MediaAssetRetentionCleanupPreflightService,
	audits mediaAssetRetentionCleanupAuditRecorder,
) *MediaAssetRetentionCleanupService {
	return &MediaAssetRetentionCleanupService{
		repository: repository,
		preflight:  preflight,
		audits:     audits,
	}
}

func (s *MediaAssetRetentionCleanupService) CleanupMediaAssetRetention(
	ctx context.Context,
	cmd command.CleanupMediaAssetRetentionCommand,
) (query.MediaAssetRetentionCleanupView, error) {
	if err := ctx.Err(); err != nil {
		return query.MediaAssetRetentionCleanupView{}, err
	}
	if s == nil || s.repository == nil || s.preflight == nil {
		return query.MediaAssetRetentionCleanupView{}, errors.New("media asset retention cleanup requires repository and preflight")
	}
	filter := mediaAssetRetentionFilterFromCleanupCommand(cmd)
	preflight, err := s.preflight.CheckMediaAssetRetentionCleanupPreflight(ctx, query.MediaAssetRetentionCleanupPreflightFilter{
		RetentionFilter: filter,
		TargetID:        cmd.TargetID,
		OperatorID:      cmd.OperatorID,
		ApprovalID:      cmd.ApprovalID,
	})
	if err != nil {
		return query.MediaAssetRetentionCleanupView{}, err
	}

	view := query.MediaAssetRetentionCleanupView{
		Ready:          preflight.Ready,
		DryRun:         cmd.DryRun,
		Reason:         preflight.Reason,
		Blockers:       append([]string(nil), preflight.Blockers...),
		TargetKind:     preflight.TargetKind,
		TargetID:       preflight.TargetID,
		Action:         preflight.Action,
		OperatorID:     preflight.OperatorID,
		ApprovalID:     preflight.ApprovalID,
		MutationID:     strings.TrimSpace(cmd.MutationID),
		CandidateCount: preflight.CandidateCount,
		Candidates:     append([]query.MediaAssetRetentionDiagnosticItemView(nil), preflight.Plan.Candidates...),
		Preflight:      preflight,
		SideEffect:     "none",
		Notes: []string{
			"metadata-only media retention cleanup; local files and provider caches are not deleted",
			"Python remains responsible for OCR, VLM, file parsing, semantic memory and RAG extraction",
		},
	}
	if cmd.DryRun {
		view.Reason = "media_asset_retention_cleanup_dry_run"
		return view, nil
	}
	if !preflight.Ready {
		return view, nil
	}
	if s.audits == nil {
		view.Ready = false
		view.Reason = "control_mutation_audit_unavailable"
		view.Blockers = []string{"control_mutation_audit_unavailable"}
		return view, nil
	}

	deletedIDs := make([]string, 0, len(preflight.Plan.Candidates))
	for _, candidate := range preflight.Plan.Candidates {
		deleted, err := s.repository.DeleteMediaAsset(ctx, candidate.AssetID)
		if err != nil {
			failed := s.recordMediaAssetRetentionCleanupAudit(ctx, cmd, "failed", "delete media asset metadata "+candidate.AssetID+": "+err.Error(), deletedIDs)
			view.FailedAudit = failed
			view.Ready = false
			view.Reason = "media_asset_retention_cleanup_failed"
			view.Blockers = []string{"media_asset_retention_cleanup_failed"}
			view.DeletedAssetIDs = deletedIDs
			view.DeletedCount = len(deletedIDs)
			view.SideEffect = "runtime_state_cleanup"
			return view, nil
		}
		if deleted {
			deletedIDs = append(deletedIDs, candidate.AssetID)
		}
	}
	applied := s.recordMediaAssetRetentionCleanupAudit(ctx, cmd, "applied", "metadata cleanup applied", deletedIDs)
	view.Applied = true
	view.Reason = "media_asset_retention_metadata_cleanup_applied"
	view.DeletedAssetIDs = deletedIDs
	view.DeletedCount = len(deletedIDs)
	view.AppliedAudit = applied
	view.SideEffect = "runtime_state_cleanup"
	return view, nil
}

func (s *MediaAssetRetentionCleanupService) recordMediaAssetRetentionCleanupAudit(
	ctx context.Context,
	cmd command.CleanupMediaAssetRetentionCommand,
	status string,
	reason string,
	deletedIDs []string,
) *query.ControlMutationAuditView {
	if s == nil || s.audits == nil {
		return nil
	}
	now := time.Now().UTC()
	audit, err := s.audits.RecordControlMutationAudit(ctx, command.RecordControlMutationAuditCommand{
		MutationID: strings.TrimSpace(cmd.MutationID),
		TargetKind: mediaAssetRetentionCleanupTargetKind,
		TargetID:   strings.TrimSpace(cmd.TargetID),
		Action:     mediaAssetRetentionCleanupAction,
		Status:     status,
		OperatorID: strings.TrimSpace(cmd.OperatorID),
		ApprovalID: strings.TrimSpace(cmd.ApprovalID),
		Reason:     reason,
		Metadata: map[string]string{
			"candidate_count": strconv.Itoa(len(deletedIDs)),
			"deleted_count":   strconv.Itoa(len(deletedIDs)),
			"deleted_ids":     strings.Join(deletedIDs, ","),
			"cleanup_scope":   "metadata_only",
		},
		Timestamp: now,
	})
	if err != nil {
		return nil
	}
	return &audit
}

func mediaAssetRetentionFilterFromCleanupCommand(cmd command.CleanupMediaAssetRetentionCommand) query.MediaAssetRetentionDiagnosticsFilter {
	return query.MediaAssetRetentionDiagnosticsFilter{
		AssetID:               strings.TrimSpace(cmd.AssetID),
		Limit:                 cmd.Limit,
		ChannelKind:           strings.TrimSpace(cmd.ChannelKind),
		AccountID:             strings.TrimSpace(cmd.AccountID),
		ConversationID:        strings.TrimSpace(cmd.ConversationID),
		ConversationType:      strings.TrimSpace(cmd.ConversationType),
		SourceMessageID:       strings.TrimSpace(cmd.SourceMessageID),
		SourceMessageIDSuffix: strings.TrimSpace(cmd.SourceMessageIDSuffix),
		Kind:                  strings.TrimSpace(cmd.Kind),
		Timestamp:             strings.TrimSpace(cmd.Timestamp),
		DefaultTTLHours:       cmd.DefaultTTLHours,
		EphemeralTTLHours:     cmd.EphemeralTTLHours,
	}
}
