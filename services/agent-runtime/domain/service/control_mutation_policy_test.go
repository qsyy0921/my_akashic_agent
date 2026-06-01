package service_test

import (
	"testing"

	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

func TestControlMutationPolicyAllowsSupportedIntent(t *testing.T) {
	policy := domainservice.NewControlMutationPolicy()

	result := policy.Check("outbound_cutover", "enable")
	if !result.Allowed || result.Reason != "control_mutation_supported" {
		t.Fatalf("expected supported mutation intent, got %+v", result)
	}
	if len(result.SupportedActions) != 2 || result.SupportedActions[0] != "enable" || result.SupportedActions[1] != "rollback" {
		t.Fatalf("unexpected supported actions: %+v", result.SupportedActions)
	}

	mediaRecovery := policy.Check("media_asset_content", "recover_content")
	if !mediaRecovery.Allowed || mediaRecovery.Reason != "control_mutation_supported" {
		t.Fatalf("expected media content recovery intent, got %+v", mediaRecovery)
	}
}

func TestControlMutationPolicyListsSupportedIntentsDeterministically(t *testing.T) {
	policy := domainservice.NewControlMutationPolicy()

	intents := policy.SupportedIntents()
	if len(intents) == 0 {
		t.Fatal("expected supported intents")
	}
	for i := 1; i < len(intents); i++ {
		if intents[i-1].TargetKind > intents[i].TargetKind {
			t.Fatalf("expected sorted targets, got %+v", intents)
		}
	}
	for _, intent := range intents {
		for i := 1; i < len(intent.Actions); i++ {
			if intent.Actions[i-1] > intent.Actions[i] {
				t.Fatalf("expected sorted actions for %s: %+v", intent.TargetKind, intent.Actions)
			}
		}
	}
}

func TestControlMutationPolicyFindsSupportedIntentByTarget(t *testing.T) {
	policy := domainservice.NewControlMutationPolicy()

	intent, ok := policy.SupportedIntent("outbound_cutover")
	if !ok || intent.TargetKind != "outbound_cutover" || len(intent.Actions) != 2 {
		t.Fatalf("expected outbound cutover intent, got ok=%t intent=%+v", ok, intent)
	}

	if _, ok := policy.SupportedIntent("model_provider_config"); ok {
		t.Fatal("unexpected unsupported target")
	}
}

func TestControlMutationPolicyBlocksUnsupportedIntent(t *testing.T) {
	policy := domainservice.NewControlMutationPolicy()

	unsupportedTarget := policy.Check("model_provider_config", "enable")
	if unsupportedTarget.Allowed || unsupportedTarget.Reason != "unsupported_control_mutation_target" {
		t.Fatalf("expected unsupported target, got %+v", unsupportedTarget)
	}

	unsupportedAction := policy.Check("outbound_cutover", "apply")
	if unsupportedAction.Allowed || unsupportedAction.Reason != "unsupported_control_mutation_action" {
		t.Fatalf("expected unsupported action, got %+v", unsupportedAction)
	}
	if len(unsupportedAction.SupportedActions) != 2 {
		t.Fatalf("expected supported action hints, got %+v", unsupportedAction.SupportedActions)
	}
}
