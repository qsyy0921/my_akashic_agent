package assembler

import (
	"sort"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToObserveTargetView(target model.ObserveTarget) query.ObserveTargetView {
	return query.ObserveTargetView{
		TargetID: target.TargetID,
		Channel: query.ObserveTargetChannelView{
			Kind:             string(target.Channel.Kind),
			AccountID:        target.Channel.AccountID,
			ConversationID:   target.Channel.ConversationID,
			ConversationType: string(target.Channel.ConversationType),
		},
		ObserveOnly:  target.ObserveOnly,
		ReplyAllowed: target.ReplyAllowed,
		RequireAt:    target.RequireAt,
		AllowFrom:    append([]string(nil), target.AllowFrom...),
		Enabled:      target.Enabled,
		Source:       target.Source,
		Metadata:     cloneObserveTargetMap(target.Metadata),
		UpdatedAt:    target.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func ToObserveTargetViews(items []model.ObserveTarget) []query.ObserveTargetView {
	views := make([]query.ObserveTargetView, 0, len(items))
	for _, item := range items {
		views = append(views, ToObserveTargetView(item))
	}
	sort.SliceStable(views, func(i, j int) bool {
		left := views[i].Channel
		right := views[j].Channel
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.AccountID != right.AccountID {
			return left.AccountID < right.AccountID
		}
		if left.ConversationType != right.ConversationType {
			return left.ConversationType < right.ConversationType
		}
		if left.ConversationID != right.ConversationID {
			return left.ConversationID < right.ConversationID
		}
		return views[i].TargetID < views[j].TargetID
	})
	return views
}

func cloneObserveTargetMap(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(items))
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}
