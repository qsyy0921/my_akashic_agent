package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestDeliveryAdapterDiagnosticsServiceReturnsSortedCopy(t *testing.T) {
	service := NewDeliveryAdapterDiagnosticsService([]query.DeliveryAdapterDiagnosticsView{
		{Provider: "telegram", Channel: "telegram", Notes: []string{"bot-api"}},
		{Provider: "onebot", Channel: "qq_2365524513", Notes: []string{"ws"}},
		{Provider: "onebot", Channel: "qq_1049511700", Notes: []string{"ws"}},
	})

	items, err := service.ListDeliveryAdapters(context.Background())
	if err != nil {
		t.Fatalf("list adapters: %v", err)
	}
	if got := []string{items[0].Channel, items[1].Channel, items[2].Channel}; got[0] != "qq_1049511700" || got[1] != "qq_2365524513" || got[2] != "telegram" {
		t.Fatalf("unexpected sorted channels: %#v", got)
	}

	items[0].Channel = "mutated"
	items[0].Notes[0] = "mutated"
	again, err := service.ListDeliveryAdapters(context.Background())
	if err != nil {
		t.Fatalf("list adapters again: %v", err)
	}
	if again[0].Channel != "qq_1049511700" || again[0].Notes[0] != "ws" {
		t.Fatalf("service returned mutable internal state: %#v", again[0])
	}
}
