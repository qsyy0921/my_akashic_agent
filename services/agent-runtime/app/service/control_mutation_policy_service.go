package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

type ControlMutationPolicyService struct {
	policy domainservice.ControlMutationPolicy
}

func NewControlMutationPolicyService() *ControlMutationPolicyService {
	return &ControlMutationPolicyService{policy: domainservice.NewControlMutationPolicy()}
}

func (s *ControlMutationPolicyService) GetControlMutationPolicy(ctx context.Context, filter query.ControlMutationPolicyFilter) (query.ControlMutationPolicyView, error) {
	if err := ctx.Err(); err != nil {
		return query.ControlMutationPolicyView{}, err
	}
	if s == nil {
		return query.ControlMutationPolicyView{}, errors.New("control mutation policy service is nil")
	}

	targetKind := strings.TrimSpace(filter.TargetKind)
	intents := make([]query.ControlMutationPolicyIntentView, 0)
	if targetKind == "" {
		for _, intent := range s.policy.SupportedIntents() {
			intents = append(intents, query.ControlMutationPolicyIntentView{
				TargetKind: intent.TargetKind,
				Actions:    append([]string(nil), intent.Actions...),
			})
		}
		return query.ControlMutationPolicyView{
			Allowed:    true,
			Reason:     "control_mutation_policy_listed",
			Intents:    intents,
			Notes:      []string{"policy query only; no runtime configuration is changed"},
			SideEffect: "none",
		}, nil
	}

	intent, ok := s.policy.SupportedIntent(targetKind)
	if !ok {
		return query.ControlMutationPolicyView{
			Allowed:    false,
			Reason:     "unsupported_control_mutation_target",
			Blockers:   []string{"unsupported_control_mutation_target"},
			TargetKind: targetKind,
			Intents:    intents,
			Notes:      []string{"policy query only; no runtime configuration is changed"},
			SideEffect: "none",
		}, nil
	}

	intents = append(intents, query.ControlMutationPolicyIntentView{
		TargetKind: intent.TargetKind,
		Actions:    append([]string(nil), intent.Actions...),
	})
	return query.ControlMutationPolicyView{
		Allowed:    true,
		Reason:     "control_mutation_policy_listed",
		TargetKind: targetKind,
		Intents:    intents,
		Notes:      []string{"policy query only; no runtime configuration is changed"},
		SideEffect: "none",
	}, nil
}
