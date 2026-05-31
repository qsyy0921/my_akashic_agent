package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ObserveTargetService struct {
	mu         sync.RWMutex
	targets    map[string]model.ObserveTarget
	repository outport.ObserveTargetRepository
}

func NewObserveTargetService() *ObserveTargetService {
	return &ObserveTargetService{targets: make(map[string]model.ObserveTarget)}
}

func NewObserveTargetServiceWithRepository(ctx context.Context, repository outport.ObserveTargetRepository) (*ObserveTargetService, error) {
	if repository == nil {
		return NewObserveTargetService(), nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	targets, err := repository.ListObserveTargets(ctx)
	if err != nil {
		return nil, err
	}
	service := &ObserveTargetService{
		targets:    make(map[string]model.ObserveTarget, len(targets)),
		repository: repository,
	}
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			continue
		}
		service.targets[target.TargetID] = target
	}
	return service, nil
}

func (s *ObserveTargetService) SyncObserveTargets(ctx context.Context, cmd command.SyncObserveTargetsCommand) (query.ObserveTargetsView, error) {
	if err := ctx.Err(); err != nil {
		return query.ObserveTargetsView{}, err
	}
	if s == nil {
		return query.ObserveTargetsView{}, errors.New("observe target service is nil")
	}
	source := strings.TrimSpace(cmd.Source)
	if source == "" {
		source = "python_config"
	}
	timestamp := cmd.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	incoming := make(map[string]model.ObserveTarget, len(cmd.Targets))
	for _, item := range cmd.Targets {
		if strings.TrimSpace(item.Source) == "" {
			item.Source = source
		}
		target, err := model.NewObserveTarget(model.ObserveTargetSpec{
			TargetID:     item.TargetID,
			Channel:      assembler.ToChannelRef(item.Channel),
			ObserveOnly:  item.ObserveOnly,
			ReplyAllowed: item.ReplyAllowed,
			RequireAt:    item.RequireAt,
			AllowFrom:    item.AllowFrom,
			Enabled:      item.Enabled,
			Source:       item.Source,
			Metadata:     item.Metadata,
		}, timestamp)
		if err != nil {
			return query.ObserveTargetsView{}, err
		}
		incoming[target.TargetID] = target
	}

	s.mu.Lock()
	if s.targets == nil {
		s.targets = make(map[string]model.ObserveTarget)
	}
	next := make(map[string]model.ObserveTarget, len(s.targets)+len(incoming))
	for targetID, target := range s.targets {
		if target.Source != source {
			next[targetID] = target
		}
	}
	for targetID, target := range incoming {
		next[targetID] = target
	}
	items := observeTargetMapItems(next)
	if s.repository != nil {
		if err := s.repository.SaveObserveTargets(ctx, items); err != nil {
			s.mu.Unlock()
			return query.ObserveTargetsView{}, err
		}
	}
	s.targets = next
	s.mu.Unlock()

	view := observeTargetsView(items)
	view.Notes = append(view.Notes, "source_bound_sync")
	return view, nil
}

func (s *ObserveTargetService) ListObserveTargets(ctx context.Context) (query.ObserveTargetsView, error) {
	if err := ctx.Err(); err != nil {
		return query.ObserveTargetsView{}, err
	}
	if s == nil {
		return query.ObserveTargetsView{}, errors.New("observe target service is nil")
	}
	s.mu.RLock()
	items := s.snapshotLocked()
	s.mu.RUnlock()
	return observeTargetsView(items), nil
}

func (s *ObserveTargetService) snapshotLocked() []model.ObserveTarget {
	return observeTargetMapItems(s.targets)
}

func observeTargetMapItems(targets map[string]model.ObserveTarget) []model.ObserveTarget {
	items := make([]model.ObserveTarget, 0, len(targets))
	for _, target := range targets {
		items = append(items, target)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Channel.Kind != items[j].Channel.Kind {
			return items[i].Channel.Kind < items[j].Channel.Kind
		}
		if items[i].Channel.AccountID != items[j].Channel.AccountID {
			return items[i].Channel.AccountID < items[j].Channel.AccountID
		}
		if items[i].Channel.ConversationType != items[j].Channel.ConversationType {
			return items[i].Channel.ConversationType < items[j].Channel.ConversationType
		}
		if items[i].Channel.ConversationID != items[j].Channel.ConversationID {
			return items[i].Channel.ConversationID < items[j].Channel.ConversationID
		}
		return items[i].TargetID < items[j].TargetID
	})
	return items
}

func observeTargetsView(items []model.ObserveTarget) query.ObserveTargetsView {
	totals := map[string]int{
		"targets":       len(items),
		"enabled":       0,
		"disabled":      0,
		"observe_only":  0,
		"reply_allowed": 0,
		"groups":        0,
		"private":       0,
		"qq":            0,
		"telegram":      0,
	}
	for _, item := range items {
		if item.Enabled {
			totals["enabled"]++
		} else {
			totals["disabled"]++
		}
		if item.ObserveOnly {
			totals["observe_only"]++
		}
		if item.ReplyAllowed {
			totals["reply_allowed"]++
		}
		switch item.Channel.ConversationType {
		case model.ConversationTypeGroup:
			totals["groups"]++
		case model.ConversationTypePrivate:
			totals["private"]++
		}
		switch item.Channel.Kind {
		case model.ChannelKindQQ:
			totals["qq"]++
		case model.ChannelKindTelegram:
			totals["telegram"]++
		}
	}
	return query.ObserveTargetsView{
		Targets:    assembler.ToObserveTargetViews(items),
		Totals:     totals,
		Notes:      []string{"side_effect=none", "runtime_view_only"},
		SideEffect: "none",
	}
}
