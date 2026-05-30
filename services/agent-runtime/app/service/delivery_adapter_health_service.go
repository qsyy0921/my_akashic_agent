package service

import (
	"context"
	"errors"
	"sort"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type DeliveryAdapterHealthService struct {
	probes []outport.DeliveryAdapterHealthProbe
}

func NewDeliveryAdapterHealthService(probes ...outport.DeliveryAdapterHealthProbe) *DeliveryAdapterHealthService {
	cloned := make([]outport.DeliveryAdapterHealthProbe, 0, len(probes))
	for _, probe := range probes {
		if probe != nil {
			cloned = append(cloned, probe)
		}
	}
	return &DeliveryAdapterHealthService{probes: cloned}
}

func (s *DeliveryAdapterHealthService) CheckDeliveryAdapters(
	ctx context.Context,
	filter query.DeliveryAdapterHealthFilter,
) (query.DeliveryAdapterHealthSummaryView, error) {
	if err := ctx.Err(); err != nil {
		return query.DeliveryAdapterHealthSummaryView{}, err
	}
	if s == nil {
		return query.DeliveryAdapterHealthSummaryView{}, errors.New("delivery adapter health service is nil")
	}
	items := make([]query.DeliveryAdapterHealthView, 0)
	for _, probe := range s.probes {
		probeItems, err := probe.CheckDeliveryAdapterHealth(ctx, filter)
		if err != nil {
			return query.DeliveryAdapterHealthSummaryView{}, err
		}
		items = append(items, probeItems...)
	}
	sort.SliceStable(items, func(i int, j int) bool {
		if items[i].Provider != items[j].Provider {
			return items[i].Provider < items[j].Provider
		}
		return items[i].Channel < items[j].Channel
	})
	healthy, authenticated := countDeliveryAdapterHealth(items)
	return query.DeliveryAdapterHealthSummaryView{
		Items: items,
		Totals: map[string]int{
			"adapters":      len(items),
			"healthy":       healthy,
			"unhealthy":     len(items) - healthy,
			"authenticated": authenticated,
		},
		Notes: []string{"read-only adapter connectivity probe; no platform messages are sent"},
	}, nil
}

func countDeliveryAdapterHealth(items []query.DeliveryAdapterHealthView) (int, int) {
	healthy := 0
	authenticated := 0
	for _, item := range items {
		if item.Healthy {
			healthy++
		}
		if item.Authenticated {
			authenticated++
		}
	}
	return healthy, authenticated
}
