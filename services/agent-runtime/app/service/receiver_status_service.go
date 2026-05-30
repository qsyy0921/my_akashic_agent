package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ReceiverStatusService struct {
	mu        sync.RWMutex
	receivers map[string]model.ReceiverStatus
}

func NewReceiverStatusService() *ReceiverStatusService {
	return &ReceiverStatusService{receivers: make(map[string]model.ReceiverStatus)}
}

func (s *ReceiverStatusService) ReportReceiverStatus(ctx context.Context, cmd command.ReportReceiverStatusCommand) (query.ReceiverStatusesView, error) {
	if err := ctx.Err(); err != nil {
		return query.ReceiverStatusesView{}, err
	}
	if s == nil {
		return query.ReceiverStatusesView{}, errors.New("receiver status service is nil")
	}
	timestamp := cmd.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	status, err := model.NewReceiverStatus(model.ReceiverStatusSpec{
		ReceiverID:  cmd.ReceiverID,
		Kind:        model.ChannelKind(cmd.Kind),
		ChannelName: cmd.ChannelName,
		AccountID:   cmd.AccountID,
		Endpoint:    cmd.Endpoint,
		Status:      cmd.Status,
		Reason:      cmd.Reason,
		LastError:   cmd.LastError,
		Source:      cmd.Source,
		Metadata:    cmd.Metadata,
	}, timestamp)
	if err != nil {
		return query.ReceiverStatusesView{}, err
	}

	s.mu.Lock()
	if s.receivers == nil {
		s.receivers = make(map[string]model.ReceiverStatus)
	}
	s.receivers[status.ReceiverID] = status
	items := s.snapshotLocked()
	s.mu.Unlock()

	view := receiverStatusesView(items)
	view.Notes = append(view.Notes, "reported")
	return view, nil
}

func (s *ReceiverStatusService) ListReceiverStatuses(ctx context.Context) (query.ReceiverStatusesView, error) {
	if err := ctx.Err(); err != nil {
		return query.ReceiverStatusesView{}, err
	}
	if s == nil {
		return query.ReceiverStatusesView{}, errors.New("receiver status service is nil")
	}
	s.mu.RLock()
	items := s.snapshotLocked()
	s.mu.RUnlock()
	return receiverStatusesView(items), nil
}

func (s *ReceiverStatusService) snapshotLocked() []model.ReceiverStatus {
	items := make([]model.ReceiverStatus, 0, len(s.receivers))
	for _, item := range s.receivers {
		items = append(items, item)
	}
	return model.SortedReceiverStatuses(items)
}

func receiverStatusesView(items []model.ReceiverStatus) query.ReceiverStatusesView {
	totals := map[string]int{
		"receivers":  0,
		"starting":   0,
		"connected":  0,
		"suspended":  0,
		"failed":     0,
		"stopped":    0,
		"qq":         0,
		"telegram":   0,
		"other_kind": 0,
	}
	totals["receivers"] = len(items)
	for _, item := range items {
		switch item.Status {
		case model.ReceiverStatusStarting:
			totals["starting"]++
		case model.ReceiverStatusConnected:
			totals["connected"]++
		case model.ReceiverStatusSuspended:
			totals["suspended"]++
		case model.ReceiverStatusFailed:
			totals["failed"]++
		case model.ReceiverStatusStopped:
			totals["stopped"]++
		}
		switch item.Kind {
		case model.ChannelKindQQ:
			totals["qq"]++
		case model.ChannelKindTelegram:
			totals["telegram"]++
		default:
			totals["other_kind"]++
		}
	}
	return query.ReceiverStatusesView{
		Receivers:  assembler.ToReceiverStatusViews(items),
		Totals:     totals,
		Notes:      []string{"side_effect=none", "runtime_view_only"},
		SideEffect: "none",
	}
}
