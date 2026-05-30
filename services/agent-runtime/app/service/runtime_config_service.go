package service

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type RuntimeConfigService struct {
	view query.RuntimeConfigView
}

func NewRuntimeConfigService(view query.RuntimeConfigView) *RuntimeConfigService {
	return &RuntimeConfigService{view: cloneRuntimeConfig(view)}
}

func (s *RuntimeConfigService) GetRuntimeConfig(context.Context) (query.RuntimeConfigView, error) {
	if s == nil {
		return query.RuntimeConfigView{}, nil
	}
	return cloneRuntimeConfig(s.view), nil
}

func cloneRuntimeConfig(view query.RuntimeConfigView) query.RuntimeConfigView {
	view.Runtime.BotIDs = append([]string(nil), view.Runtime.BotIDs...)
	view.Delivery.TelegramChannels = append([]string(nil), view.Delivery.TelegramChannels...)
	view.Delivery.OneBotExpectedChannels = append([]string(nil), view.Delivery.OneBotExpectedChannels...)
	view.Delivery.OneBotMissingChannels = append([]string(nil), view.Delivery.OneBotMissingChannels...)
	view.Delivery.OneBotPerChannelTokenChannels = append([]string(nil), view.Delivery.OneBotPerChannelTokenChannels...)
	view.Delivery.OneBotEndpoints = cloneRuntimeOneBotEndpoints(view.Delivery.OneBotEndpoints)
	view.Environment = append([]query.RuntimeEnvVarView(nil), view.Environment...)
	view.Readiness.Blockers = append([]string(nil), view.Readiness.Blockers...)
	view.Notes = append([]string(nil), view.Notes...)
	return view
}

func cloneRuntimeOneBotEndpoints(items []query.RuntimeOneBotEndpointConfigView) []query.RuntimeOneBotEndpointConfigView {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]query.RuntimeOneBotEndpointConfigView, len(items))
	for index, item := range items {
		item.Notes = append([]string(nil), item.Notes...)
		cloned[index] = item
	}
	return cloned
}
