package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestDeliveryAdapterHealthServiceAggregatesSortedTotals(t *testing.T) {
	service := NewDeliveryAdapterHealthService(
		staticDeliveryHealthProbe{items: []query.DeliveryAdapterHealthView{
			{Provider: "onebot", Channel: "qq_2365524513", Healthy: true, Authenticated: true},
		}},
		staticDeliveryHealthProbe{items: []query.DeliveryAdapterHealthView{
			{Provider: "telegram", Channel: "telegram", Healthy: false, Authenticated: false},
			{Provider: "onebot", Channel: "qq_1049511700", Healthy: true, Authenticated: true},
		}},
	)

	view, err := service.CheckDeliveryAdapters(context.Background(), query.DeliveryAdapterHealthFilter{})
	if err != nil {
		t.Fatalf("check adapter health: %v", err)
	}
	if view.Totals["adapters"] != 3 || view.Totals["healthy"] != 2 || view.Totals["unhealthy"] != 1 || view.Totals["authenticated"] != 2 {
		t.Fatalf("unexpected totals: %#v", view.Totals)
	}
	if view.Items[0].Channel != "qq_1049511700" || view.Items[1].Channel != "qq_2365524513" || view.Items[2].Provider != "telegram" {
		t.Fatalf("items not sorted by provider/channel: %#v", view.Items)
	}
}

type staticDeliveryHealthProbe struct {
	items []query.DeliveryAdapterHealthView
}

func (s staticDeliveryHealthProbe) CheckDeliveryAdapterHealth(context.Context, query.DeliveryAdapterHealthFilter) ([]query.DeliveryAdapterHealthView, error) {
	return append([]query.DeliveryAdapterHealthView(nil), s.items...), nil
}
