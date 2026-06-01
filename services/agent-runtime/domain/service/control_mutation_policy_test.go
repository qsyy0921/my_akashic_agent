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
