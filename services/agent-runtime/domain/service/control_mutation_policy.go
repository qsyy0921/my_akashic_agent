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
		},
	}
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
