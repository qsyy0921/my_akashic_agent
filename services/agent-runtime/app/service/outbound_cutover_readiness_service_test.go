package service

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func TestOutboundCutoverReadinessReportsReadyLocalWorker(t *testing.T) {
	adapter := &recordingSmokeDeliveryAdapter{channels: map[string]bool{"qq_1049511700": true}}
	deliverySmoke := NewDeliverySmokeReadinessService([]outport.DeliveryAdapter{adapter}, DeliverySmokeReadinessConfig{
		ChannelByAccount: map[string]string{"1049511700": "qq_1049511700"},
	})
	service := NewOutboundCutoverReadinessService(OutboundCutoverReadinessDeps{
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Delivery: query.RuntimeDeliveryConfigView{
				OneBotExpectedChannels: []string{"qq_1049511700"},
			},
			Readiness: query.RuntimeConfigReadinessView{
				OneBotConfigured:         true,
				OneBotExpectedChannelsOK: true,
				OneBotHealthProbeReady:   true,
			},
			SideEffect: "none",
		}},
		DeliverySmoke: deliverySmoke,
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:             "local",
			Mode:                 "local_state_store",
			OutboxExecutionOwner: "go_local_outbox_worker",
		}},
		RuntimeWorkers: staticRuntimeWorkerDiagnostics{view: query.RuntimeWorkerDiagnosticsView{
			Workers: []query.RuntimeWorkerView{{
				Name:    "outbox_delivery_worker",
				Kind:    "outbox_delivery",
				Enabled: true,
				Running: true,
			}},
		}},
	})

	view, err := service.CheckOutboundCutoverReadiness(context.Background(), command.CheckOutboundCutoverReadinessCommand{
		Smoke: command.CheckDeliverySmokeReadinessCommand{Cases: []command.DeliverySmokeCaseCommand{{
			Name:             "qq_private_text_1049511700_to_2365524513",
			ChannelKind:      "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
			Content:          "smoke text",
		}}},
	})
	if err != nil {
		t.Fatalf("cutover readiness: %v", err)
	}
	if !view.Ready || view.Reason != "outbound_cutover_ready" {
		t.Fatalf("expected ready cutover: %#v", view)
	}
	if !view.OneBotReady || !view.SmokeReady || !view.ExecutionReady || !view.LocalOutboxWorkerReady || view.ExternalLeaseOutboxReady {
		t.Fatalf("unexpected readiness flags: %#v", view)
	}
	if view.ExecutionOwner != "go_local_outbox_worker" || len(view.Blockers) != 0 {
		t.Fatalf("unexpected ready owner/blockers: %#v", view)
	}
	if len(adapter.steps) != 0 {
		t.Fatalf("readiness must not dispatch adapter steps: %+v", adapter.steps)
	}
}

func TestOutboundCutoverReadinessBlocksWithoutExecutionPath(t *testing.T) {
	adapter := &recordingSmokeDeliveryAdapter{channels: map[string]bool{"qq_1049511700": true}}
	deliverySmoke := NewDeliverySmokeReadinessService([]outport.DeliveryAdapter{adapter}, DeliverySmokeReadinessConfig{
		ChannelByAccount: map[string]string{"1049511700": "qq_1049511700"},
	})
	service := NewOutboundCutoverReadinessService(OutboundCutoverReadinessDeps{
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Delivery: query.RuntimeDeliveryConfigView{OneBotExpectedChannels: []string{"qq_1049511700"}},
			Readiness: query.RuntimeConfigReadinessView{
				OneBotConfigured:         true,
				OneBotExpectedChannelsOK: true,
			},
			SideEffect: "none",
		}},
		DeliverySmoke: deliverySmoke,
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:             "local",
			Mode:                 "local_state_store",
			OutboxExecutionOwner: "go_state_store_api",
		}},
		RuntimeWorkers: staticRuntimeWorkerDiagnostics{view: query.RuntimeWorkerDiagnosticsView{
			Workers: []query.RuntimeWorkerView{{Name: "outbox_delivery_worker", Enabled: false, Running: false}},
		}},
	})

	view, err := service.CheckOutboundCutoverReadiness(context.Background(), command.CheckOutboundCutoverReadinessCommand{
		Smoke: command.CheckDeliverySmokeReadinessCommand{Cases: []command.DeliverySmokeCaseCommand{{
			Name:             "qq_private_text_1049511700_to_2365524513",
			ChannelKind:      "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
			Content:          "smoke text",
		}}},
	})
	if err != nil {
		t.Fatalf("cutover readiness: %v", err)
	}
	if view.Ready || view.ExecutionReady {
		t.Fatalf("expected blocked cutover: %#v", view)
	}
	if !containsString(view.Blockers, "outbox_execution_path_not_ready") {
		t.Fatalf("expected execution path blocker: %#v", view.Blockers)
	}
}

func TestOutboundCutoverReadinessBlocksMissingAdapter(t *testing.T) {
	deliverySmoke := NewDeliverySmokeReadinessService(nil, DeliverySmokeReadinessConfig{
		ChannelByAccount: map[string]string{"1049511700": "qq_1049511700"},
	})
	service := NewOutboundCutoverReadinessService(OutboundCutoverReadinessDeps{
		RuntimeConfig: staticRuntimeConfig{view: query.RuntimeConfigView{
			Delivery: query.RuntimeDeliveryConfigView{OneBotExpectedChannels: []string{"qq_1049511700"}},
			Readiness: query.RuntimeConfigReadinessView{
				OneBotConfigured:         true,
				OneBotExpectedChannelsOK: true,
			},
			SideEffect: "none",
		}},
		DeliverySmoke: deliverySmoke,
		QueueBackend: staticRuntimeQueueBackend{view: query.QueueBackendView{
			Provider:             "local",
			Mode:                 "local_state_store",
			OutboxExecutionOwner: "go_local_outbox_worker",
		}},
		RuntimeWorkers: staticRuntimeWorkerDiagnostics{view: query.RuntimeWorkerDiagnosticsView{
			Workers: []query.RuntimeWorkerView{{Name: "outbox_delivery_worker", Enabled: true, Running: true}},
		}},
	})

	view, err := service.CheckOutboundCutoverReadiness(context.Background(), command.CheckOutboundCutoverReadinessCommand{
		Smoke: command.CheckDeliverySmokeReadinessCommand{Cases: []command.DeliverySmokeCaseCommand{{
			Name:             "qq_private_text_1049511700_to_2365524513",
			ChannelKind:      "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
			Content:          "smoke text",
		}}},
	})
	if err != nil {
		t.Fatalf("cutover readiness: %v", err)
	}
	if view.Ready || view.SmokeReady {
		t.Fatalf("expected missing-adapter cutover blocker: %#v", view)
	}
	if !containsString(view.Blockers, "delivery_smoke_not_ready") {
		t.Fatalf("expected smoke blocker: %#v", view.Blockers)
	}
}
