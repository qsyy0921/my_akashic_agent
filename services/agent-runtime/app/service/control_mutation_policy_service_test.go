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
	mediaRetention := findControlMutationPolicyIntent(t, view.Intents, "media_asset_retention")
	if len(mediaRetention.Actions) != 1 || mediaRetention.Actions[0] != "cleanup_expired" {
		t.Fatalf("unexpected media retention actions: %+v", mediaRetention.Actions)
	}
	mediaContent := findControlMutationPolicyIntent(t, view.Intents, "media_asset_content")
	if len(mediaContent.Actions) != 1 || mediaContent.Actions[0] != "recover_content" {
		t.Fatalf("unexpected media content actions: %+v", mediaContent.Actions)
	}
}

func TestControlMutationPolicyServiceFiltersMediaRetentionTarget(t *testing.T) {
	service := appservice.NewControlMutationPolicyService()

	view, err := service.GetControlMutationPolicy(context.Background(), query.ControlMutationPolicyFilter{
		TargetKind: "media_asset_retention",
	})
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if !view.Allowed || view.TargetKind != "media_asset_retention" || len(view.Intents) != 1 {
		t.Fatalf("unexpected media retention policy view: %+v", view)
	}
	if len(view.Intents[0].Actions) != 1 || view.Intents[0].Actions[0] != "cleanup_expired" {
		t.Fatalf("unexpected media retention actions: %+v", view.Intents[0].Actions)
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

func findControlMutationPolicyIntent(
	t *testing.T,
	intents []query.ControlMutationPolicyIntentView,
	targetKind string,
) query.ControlMutationPolicyIntentView {
	t.Helper()
	for _, intent := range intents {
		if intent.TargetKind == targetKind {
			return intent
		}
	}
	t.Fatalf("policy intent not found for %s: %+v", targetKind, intents)
	return query.ControlMutationPolicyIntentView{}
}
