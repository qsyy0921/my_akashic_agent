package service

import (
	"sort"
	"strings"
)

type ControlMutationPolicy struct {
	supported map[string]map[string]struct{}
}

type ControlMutationPolicyResult struct {
	Allowed          bool
	Reason           string
	SupportedActions []string
}

type ControlMutationPolicyIntent struct {
	TargetKind string
	Actions    []string
}

func NewControlMutationPolicy() ControlMutationPolicy {
	return ControlMutationPolicy{
		supported: map[string]map[string]struct{}{
			"outbound_cutover": {
				"enable":   {},
				"rollback": {},
			},
			"knowledge_job_planner_cutover": {
				"enable":   {},
				"rollback": {},
			},
			"agent_job_priority": {
				"apply":    {},
				"rollback": {},
			},
			"agent_job_capacity": {
				"apply":    {},
				"rollback": {},
			},
			"agent_job_external_lease": {
				"enable":   {},
				"rollback": {},
			},
			"media_asset_retention": {
				"cleanup_expired": {},
			},
		},
	}
}

func (p ControlMutationPolicy) SupportedIntents() []ControlMutationPolicyIntent {
	targets := make([]string, 0, len(p.supported))
	for target := range p.supported {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	result := make([]ControlMutationPolicyIntent, 0, len(targets))
	for _, target := range targets {
		result = append(result, ControlMutationPolicyIntent{
			TargetKind: target,
			Actions:    sortedControlMutationActions(p.supported[target]),
		})
	}
	return result
}

func (p ControlMutationPolicy) SupportedIntent(targetKind string) (ControlMutationPolicyIntent, bool) {
	targetKind = strings.TrimSpace(targetKind)
	actions, ok := p.supported[targetKind]
	if !ok {
		return ControlMutationPolicyIntent{}, false
	}
	return ControlMutationPolicyIntent{
		TargetKind: targetKind,
		Actions:    sortedControlMutationActions(actions),
	}, true
}

func (p ControlMutationPolicy) Check(targetKind string, action string) ControlMutationPolicyResult {
	targetKind = strings.TrimSpace(targetKind)
	action = strings.TrimSpace(action)
	if targetKind == "" || action == "" {
		return ControlMutationPolicyResult{Reason: "missing_control_mutation_policy_input"}
	}
	actions, ok := p.supported[targetKind]
	if !ok {
		return ControlMutationPolicyResult{Reason: "unsupported_control_mutation_target"}
	}
	if _, ok := actions[action]; !ok {
		return ControlMutationPolicyResult{
			Reason:           "unsupported_control_mutation_action",
			SupportedActions: sortedControlMutationActions(actions),
		}
	}
	return ControlMutationPolicyResult{
		Allowed:          true,
		Reason:           "control_mutation_supported",
		SupportedActions: sortedControlMutationActions(actions),
	}
}

func sortedControlMutationActions(actions map[string]struct{}) []string {
	result := make([]string, 0, len(actions))
	for action := range actions {
		result = append(result, action)
	}
	sort.Strings(result)
	return result
}
