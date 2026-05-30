package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestDeliverySmokeReadinessChecksAdaptersWithoutDispatch(t *testing.T) {
	adapter := &recordingSmokeDeliveryAdapter{channels: map[string]bool{
		"qq_1049511700": true,
		"qq_2365524513": true,
	}}
	service := NewDeliverySmokeReadinessService([]outport.DeliveryAdapter{adapter}, DeliverySmokeReadinessConfig{
		ChannelByAccount: map[string]string{
			"1049511700": "qq_1049511700",
			"2365524513": "qq_2365524513",
		},
	})

	view, err := service.CheckDeliverySmokeReadiness(context.Background(), command.CheckDeliverySmokeReadinessCommand{
		Cases: []command.DeliverySmokeCaseCommand{{
			Name:             "qq_private_image_1049511700_to_2365524513",
			ChannelKind:      "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
			Content:          "smoke image",
			Attachments: []command.DeliverySmokeAttachmentCommand{{
				Kind: "image",
				URL:  "base64://YXNoaWNhYw==",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("delivery smoke readiness: %v", err)
	}
	if !view.Ready || view.Reason != "delivery_smoke_ready" {
		t.Fatalf("expected ready smoke view: %#v", view)
	}
	if got := view.Totals["cases"]; got != 1 {
		t.Fatalf("expected one case, got totals %#v", view.Totals)
	}
	if len(view.Cases) != 1 || !view.Cases[0].Ready || view.Cases[0].Plan.Channel != "qq_1049511700" {
		t.Fatalf("unexpected case readiness: %#v", view.Cases)
	}
	if got := len(adapter.steps); got != 0 {
		t.Fatalf("readiness must not dispatch adapter steps, got %d", got)
	}
}

func TestDeliverySmokeReadinessReportsMissingAdapter(t *testing.T) {
	service := NewDeliverySmokeReadinessService(nil, DeliverySmokeReadinessConfig{
		ChannelByAccount: map[string]string{"2365524513": "qq_2365524513"},
	})

	view, err := service.CheckDeliverySmokeReadiness(context.Background(), command.CheckDeliverySmokeReadinessCommand{
		Cases: []command.DeliverySmokeCaseCommand{{
			Name:             "qq_private_text_2365524513_to_1049511700",
			ChannelKind:      "qq",
			AccountID:        "2365524513",
			ConversationID:   "1049511700",
			ConversationType: "private",
			Content:          "smoke text",
		}},
	})
	if err != nil {
		t.Fatalf("delivery smoke readiness: %v", err)
	}
	if view.Ready || view.Reason != "delivery_smoke_not_ready" {
		t.Fatalf("expected not-ready smoke view: %#v", view)
	}
	if len(view.Cases) != 1 || view.Cases[0].Reason != "delivery_adapter_unavailable" {
		t.Fatalf("unexpected case readiness: %#v", view.Cases)
	}
	if len(view.Blockers) != 1 || !strings.Contains(view.Blockers[0], "missing_channel:qq_2365524513") {
		t.Fatalf("expected missing channel blocker, got %#v", view.Blockers)
	}
}

type recordingSmokeDeliveryAdapter struct {
	channels map[string]bool
	steps    []model.DeliveryDispatchStep
}

func (a *recordingSmokeDeliveryAdapter) SupportsDeliveryChannel(channel string) bool {
	return a.channels[strings.ToLower(strings.TrimSpace(channel))]
}

func (a *recordingSmokeDeliveryAdapter) DispatchDeliveryStep(_ context.Context, step model.DeliveryDispatchStep) (model.DeliveryDispatchResult, error) {
	a.steps = append(a.steps, step)
	return model.DeliveryDispatchResult{}, nil
}
