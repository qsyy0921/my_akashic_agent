package service_test

import (
	"context"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
)

func TestControlMutationPolicyServiceListsPolicy(t *testing.T) {
	service := appservice.NewControlMutationPolicyService()

	view, err := service.GetControlMutationPolicy(context.Background(), query.ControlMutationPolicyFilter{})
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if !view.Allowed || view.Reason != "control_mutation_policy_listed" || view.SideEffect != "none" {
		t.Fatalf("unexpected policy view: %+v", view)
	}
	if len(view.Intents) == 0 {
		t.Fatal("expected policy intents")
	}
}

func TestControlMutationPolicyServiceFiltersUnsupportedTarget(t *testing.T) {
	service := appservice.NewControlMutationPolicyService()

	view, err := service.GetControlMutationPolicy(context.Background(), query.ControlMutationPolicyFilter{
		TargetKind: "model_provider_config",
	})
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if view.Allowed || view.Reason != "unsupported_control_mutation_target" || view.Blockers[0] != "unsupported_control_mutation_target" {
		t.Fatalf("unexpected blocked policy view: %+v", view)
	}
	if len(view.Intents) != 0 {
		t.Fatalf("unsupported target should not return intents: %+v", view.Intents)
	}
}
