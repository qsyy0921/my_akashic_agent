package service

import (
	"context"
	"errors"
	"strings"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	defaultObserveCaptureLimit = 200
	maxObserveCaptureLimit     = 1000
)

type observeCaptureTargetLister interface {
	ListObserveTargets(ctx context.Context) (query.ObserveTargetsView, error)
}

type observeCaptureReceiverLister interface {
	ListReceiverStatuses(ctx context.Context) (query.ReceiverStatusesView, error)
}

type ObserveCaptureDiagnosticsService struct {
	observeTargets observeCaptureTargetLister
	receiverStatus observeCaptureReceiverLister
	inboxEvents    outport.InboxEventRepository
	mediaAssets    outport.MediaAssetRepository
	contentReader  outport.MediaAssetContentReader
}

func NewObserveCaptureDiagnosticsService(
	observeTargets observeCaptureTargetLister,
	receiverStatus observeCaptureReceiverLister,
	inboxEvents outport.InboxEventRepository,
	mediaAssets outport.MediaAssetRepository,
	contentReader outport.MediaAssetContentReader,
) *ObserveCaptureDiagnosticsService {
	return &ObserveCaptureDiagnosticsService{
		observeTargets: observeTargets,
		receiverStatus: receiverStatus,
		inboxEvents:    inboxEvents,
		mediaAssets:    mediaAssets,
		contentReader:  contentReader,
	}
}

func (s *ObserveCaptureDiagnosticsService) GetObserveCaptureDiagnostics(
	ctx context.Context,
	filter query.ObserveCaptureDiagnosticsFilter,
) (query.ObserveCaptureDiagnosticsView, error) {
	if err := ctx.Err(); err != nil {
		return query.ObserveCaptureDiagnosticsView{}, err
	}
	if s == nil || s.observeTargets == nil || s.receiverStatus == nil || s.inboxEvents == nil || s.mediaAssets == nil {
		return query.ObserveCaptureDiagnosticsView{}, errors.New("observe capture diagnostics service requires observe targets, receiver status, inbox, and media repositories")
	}
	limit := boundedObserveCaptureLimit(filter.Limit)
	targetsView, err := s.observeTargets.ListObserveTargets(ctx)
	if err != nil {
		return query.ObserveCaptureDiagnosticsView{}, err
	}
	receiversView, err := s.receiverStatus.ListReceiverStatuses(ctx)
	if err != nil {
		return query.ObserveCaptureDiagnosticsView{}, err
	}
	receiversByAccount := observeCaptureReceiversByAccount(receiversView.Receivers)

	items := make([]query.ObserveCaptureTargetDiagnosticsView, 0, len(targetsView.Targets))
	for _, target := range targetsView.Targets {
		if strings.TrimSpace(target.Channel.Kind) != string(model.ChannelKindQQ) ||
			strings.TrimSpace(target.Channel.ConversationType) != string(model.ConversationTypeGroup) {
			continue
		}
		items = append(items, s.targetDiagnostics(ctx, target, receiversByAccount, limit))
	}

	return query.ObserveCaptureDiagnosticsView{
		Targets:    items,
		Totals:     observeCaptureTotals(items),
		Notes:      []string{"side_effect=none", "runtime_view_only", "content probe opens bounded local media samples only"},
		SideEffect: "none",
	}, nil
}

func (s *ObserveCaptureDiagnosticsService) targetDiagnostics(
	ctx context.Context,
	target query.ObserveTargetView,
	receiversByAccount map[string]query.ReceiverStatusView,
	limit int,
) query.ObserveCaptureTargetDiagnosticsView {
	item := query.ObserveCaptureTargetDiagnosticsView{
		TargetID:    target.TargetID,
		Channel:     target.Channel,
		Enabled:     target.Enabled,
		ObserveOnly: target.ObserveOnly,
		Status:      "warn",
		Blockers:    make([]string, 0, 4),
	}
	if receiver, ok := receiversByAccount[target.Channel.AccountID]; ok {
		item.ReceiverID = receiver.ReceiverID
		item.ReceiverStatus = receiver.Status
		item.ReceiverConnected = receiver.Status == "connected"
	}
	if target.Enabled && !item.ReceiverConnected {
		item.Blockers = append(item.Blockers, "receiver_not_connected")
	}

	events, err := s.inboxEvents.ListInboxEvents(ctx, query.InboxEventFilter{
		Limit:            limit,
		ChannelKind:      target.Channel.Kind,
		AccountID:        target.Channel.AccountID,
		ConversationID:   target.Channel.ConversationID,
		ConversationType: target.Channel.ConversationType,
		ObserveOnly:      "true",
	})
	if err == nil {
		applyObserveCaptureInboxEvents(&item, events)
	} else {
		item.Blockers = append(item.Blockers, "inbox_query_failed")
	}

	assets, err := s.mediaAssets.ListMediaAssets(ctx, query.MediaAssetFilter{
		Limit:            limit,
		ChannelKind:      target.Channel.Kind,
		AccountID:        target.Channel.AccountID,
		ConversationID:   target.Channel.ConversationID,
		ConversationType: target.Channel.ConversationType,
	})
	if err == nil {
		applyObserveCaptureMediaAssets(&item, ctx, assets, s.contentReader)
	} else {
		item.Blockers = append(item.Blockers, "media_asset_query_failed")
	}

	item.Coverage = query.ObserveCaptureCoverage{
		TextSeen:          item.TextEvents > 0,
		AttachmentSeen:    item.AttachmentCount > 0 || item.MediaAssets > 0,
		ImageSeen:         item.ImageAssets > 0,
		FileSeen:          item.FileAssets > 0,
		MediaContentReady: item.MediaAssets == 0 || item.ContentReadyAssets == item.MediaAssets,
	}
	if target.Enabled && item.TextEvents == 0 {
		item.Blockers = append(item.Blockers, "text_not_seen")
	}
	if target.Enabled && item.ImageAssets == 0 {
		item.Blockers = append(item.Blockers, "image_not_seen")
	}
	if target.Enabled && item.FileAssets == 0 {
		item.Blockers = append(item.Blockers, "file_not_seen")
	}
	if target.Enabled && item.MediaAssets > 0 && item.ContentReadyAssets < item.MediaAssets {
		item.Blockers = append(item.Blockers, "media_content_not_fully_ready")
	}
	item.Status = observeCaptureTargetStatus(item)
	return item
}

func applyObserveCaptureInboxEvents(item *query.ObserveCaptureTargetDiagnosticsView, events []model.InboxEvent) {
	item.InboxEvents = len(events)
	latest := time.Time{}
	for _, event := range events {
		content := strings.TrimSpace(event.Envelope.Content)
		if content != "" && content != "[空消息]" {
			item.TextEvents++
		}
		attachmentCount := len(event.Envelope.Attachments)
		if attachmentCount > 0 {
			item.AttachmentEvents++
			item.AttachmentCount += attachmentCount
		}
		if event.ReceivedAt.After(latest) {
			latest = event.ReceivedAt
		}
	}
	item.LatestReceivedAt = observeCaptureFormatTime(latest)
}

func applyObserveCaptureMediaAssets(
	item *query.ObserveCaptureTargetDiagnosticsView,
	ctx context.Context,
	assets []model.MediaAsset,
	contentReader outport.MediaAssetContentReader,
) {
	item.MediaAssets = len(assets)
	latest := time.Time{}
	for _, asset := range assets {
		switch asset.Kind {
		case model.MediaAssetImage:
			item.ImageAssets++
		case model.MediaAssetFile:
			item.FileAssets++
		}
		if asset.UpdatedAt.After(latest) {
			latest = asset.UpdatedAt
		}
		if contentReader == nil {
			item.ContentDisabledAssets++
			continue
		}
		content, err := contentReader.OpenMediaAssetContent(ctx, asset)
		if err == nil {
			item.ContentReadyAssets++
			_ = content.Body.Close()
			continue
		}
		switch {
		case errors.Is(err, outport.ErrMediaAssetContentDisabled):
			item.ContentDisabledAssets++
		case errors.Is(err, outport.ErrMediaAssetContentForbidden):
			item.ContentForbiddenAssets++
		default:
			item.ContentUnavailableAssets++
		}
	}
	item.LatestAssetAt = observeCaptureFormatTime(latest)
}

func observeCaptureReceiversByAccount(items []query.ReceiverStatusView) map[string]query.ReceiverStatusView {
	result := make(map[string]query.ReceiverStatusView)
	for _, item := range items {
		if strings.TrimSpace(item.Kind) != string(model.ChannelKindQQ) {
			continue
		}
		accountID := strings.TrimSpace(item.AccountID)
		if accountID == "" {
			continue
		}
		if existing, ok := result[accountID]; !ok || receiverStatusRank(item.Status) > receiverStatusRank(existing.Status) {
			result[accountID] = item
		}
	}
	return result
}

func receiverStatusRank(status string) int {
	switch strings.TrimSpace(status) {
	case "connected":
		return 3
	case "starting":
		return 2
	case "suspended", "stopped":
		return 1
	default:
		return 0
	}
}

func observeCaptureTargetStatus(item query.ObserveCaptureTargetDiagnosticsView) string {
	if !item.Enabled {
		return "muted"
	}
	if !item.ReceiverConnected {
		return "danger"
	}
	if len(item.Blockers) > 0 {
		return "warn"
	}
	return "ok"
}

func observeCaptureTotals(items []query.ObserveCaptureTargetDiagnosticsView) map[string]int {
	totals := map[string]int{
		"targets":                    len(items),
		"enabled":                    0,
		"ready":                      0,
		"warning":                    0,
		"blocked":                    0,
		"receiver_connected":         0,
		"text_covered":               0,
		"attachment_covered":         0,
		"image_covered":              0,
		"file_covered":               0,
		"inbox_events":               0,
		"text_events":                0,
		"attachment_events":          0,
		"attachment_count":           0,
		"media_assets":               0,
		"image_assets":               0,
		"file_assets":                0,
		"content_ready_assets":       0,
		"content_unavailable_assets": 0,
		"content_forbidden_assets":   0,
		"content_disabled_assets":    0,
	}
	for _, item := range items {
		if item.Enabled {
			totals["enabled"]++
		}
		switch item.Status {
		case "ok":
			totals["ready"]++
		case "danger":
			totals["blocked"]++
		case "warn":
			totals["warning"]++
		}
		if item.ReceiverConnected {
			totals["receiver_connected"]++
		}
		if item.Coverage.TextSeen {
			totals["text_covered"]++
		}
		if item.Coverage.AttachmentSeen {
			totals["attachment_covered"]++
		}
		if item.Coverage.ImageSeen {
			totals["image_covered"]++
		}
		if item.Coverage.FileSeen {
			totals["file_covered"]++
		}
		totals["inbox_events"] += item.InboxEvents
		totals["text_events"] += item.TextEvents
		totals["attachment_events"] += item.AttachmentEvents
		totals["attachment_count"] += item.AttachmentCount
		totals["media_assets"] += item.MediaAssets
		totals["image_assets"] += item.ImageAssets
		totals["file_assets"] += item.FileAssets
		totals["content_ready_assets"] += item.ContentReadyAssets
		totals["content_unavailable_assets"] += item.ContentUnavailableAssets
		totals["content_forbidden_assets"] += item.ContentForbiddenAssets
		totals["content_disabled_assets"] += item.ContentDisabledAssets
	}
	return totals
}

func boundedObserveCaptureLimit(value int) int {
	if value <= 0 {
		return defaultObserveCaptureLimit
	}
	if value > maxObserveCaptureLimit {
		return maxObserveCaptureLimit
	}
	return value
}

func observeCaptureFormatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
