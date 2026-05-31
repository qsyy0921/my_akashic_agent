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
		AssetID:          view.AssetID,
		Channel:          view.Channel,
		SourceMessageID:  view.SourceMessageID,
		SenderID:         view.SenderID,
		Kind:             view.Kind,
		MimeType:         view.MimeType,
		Name:             view.Name,
		SizeBytes:        view.SizeBytes,
		ContentStatus:    status,
		ContentReason:    reason,
		ContentEndpoint:  "/v1/media-assets/" + url.PathEscape(view.AssetID) + "/content",
		ContentMimeType:  contentMimeType,
		ContentSizeBytes: contentSizeBytes,
		UpdatedAt:        view.UpdatedAt,
	}
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
