package service

import (
	"context"
	"sort"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type DeliveryAdapterDiagnosticsService struct {
	items []query.DeliveryAdapterDiagnosticsView
}

func NewDeliveryAdapterDiagnosticsService(items []query.DeliveryAdapterDiagnosticsView) *DeliveryAdapterDiagnosticsService {
	cloned := cloneDeliveryAdapterDiagnostics(items)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Provider == cloned[j].Provider {
			return cloned[i].Channel < cloned[j].Channel
		}
		return cloned[i].Provider < cloned[j].Provider
	})
	return &DeliveryAdapterDiagnosticsService{items: cloned}
}

func (s *DeliveryAdapterDiagnosticsService) ListDeliveryAdapters(context.Context) ([]query.DeliveryAdapterDiagnosticsView, error) {
	if s == nil {
		return nil, nil
	}
	return cloneDeliveryAdapterDiagnostics(s.items), nil
}

func cloneDeliveryAdapterDiagnostics(items []query.DeliveryAdapterDiagnosticsView) []query.DeliveryAdapterDiagnosticsView {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]query.DeliveryAdapterDiagnosticsView, len(items))
	for index, item := range items {
		item.Notes = append([]string(nil), item.Notes...)
		cloned[index] = item
	}
	return cloned
}
