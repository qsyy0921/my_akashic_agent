package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type agentJobExternalLeaseCutoverDiffBundleViewer interface {
	GetAgentJobExternalLeaseLauncherBundle(ctx context.Context, filter query.AgentJobExternalLeaseLauncherBundleFilter) (query.AgentJobExternalLeaseLauncherBundleView, error)
}

type agentJobExternalLeaseCutoverDiffRuntimeConfigViewer interface {
	GetRuntimeConfig(ctx context.Context) (query.RuntimeConfigView, error)
}

type agentJobExternalLeaseCutoverDiffQueueBackendViewer interface {
	Get(ctx context.Context) (query.QueueBackendView, error)
}

type agentJobExternalLeaseCutoverDiffQueueTopologyViewer interface {
	GetQueueTopology(ctx context.Context) (query.QueueTopologyView, error)
}

type AgentJobExternalLeaseCutoverDiffService struct {
	bundle        agentJobExternalLeaseCutoverDiffBundleViewer
	runtime       agentJobExternalLeaseCutoverDiffRuntimeConfigViewer
	queueBackend  agentJobExternalLeaseCutoverDiffQueueBackendViewer
	queueTopology agentJobExternalLeaseCutoverDiffQueueTopologyViewer
}

func NewAgentJobExternalLeaseCutoverDiffService(
	bundle agentJobExternalLeaseCutoverDiffBundleViewer,
	runtime agentJobExternalLeaseCutoverDiffRuntimeConfigViewer,
	queueBackend agentJobExternalLeaseCutoverDiffQueueBackendViewer,
	queueTopology agentJobExternalLeaseCutoverDiffQueueTopologyViewer,
) *AgentJobExternalLeaseCutoverDiffService {
	return &AgentJobExternalLeaseCutoverDiffService{
		bundle:        bundle,
		runtime:       runtime,
		queueBackend:  queueBackend,
		queueTopology: queueTopology,
	}
}

func (s *AgentJobExternalLeaseCutoverDiffService) GetAgentJobExternalLeaseCutoverDiff(
	ctx context.Context,
	filter query.AgentJobExternalLeaseCutoverDiffFilter,
) (query.AgentJobExternalLeaseCutoverDiffView, error) {
	if err := ctx.Err(); err != nil {
		return query.AgentJobExternalLeaseCutoverDiffView{}, err
	}
	if s == nil || s.bundle == nil || s.runtime == nil || s.queueBackend == nil || s.queueTopology == nil {
		return query.AgentJobExternalLeaseCutoverDiffView{}, errors.New("agent job external lease cutover diff requires bundle, runtime config, queue backend, and queue topology")
	}

	bundle, err := s.bundle.GetAgentJobExternalLeaseLauncherBundle(ctx, query.AgentJobExternalLeaseLauncherBundleFilter{
		DesiredExecutionOwner: filter.DesiredExecutionOwner,
		JobLimit:              filter.JobLimit,
		EventLimit:            filter.EventLimit,
		StaleAfterSeconds:     filter.StaleAfterSeconds,
	})
	if err != nil {
		return query.AgentJobExternalLeaseCutoverDiffView{}, err
	}
	runtimeConfig, err := s.runtime.GetRuntimeConfig(ctx)
	if err != nil {
		return query.AgentJobExternalLeaseCutoverDiffView{}, err
	}
	queueBackend, err := s.queueBackend.Get(ctx)
	if err != nil {
		return query.AgentJobExternalLeaseCutoverDiffView{}, err
	}
	topology, err := s.queueTopology.GetQueueTopology(ctx)
	if err != nil {
		return query.AgentJobExternalLeaseCutoverDiffView{}, err
	}

	matching := make([]query.AgentJobExternalLeaseCutoverDiffItemView, 0)
	drift := make([]query.AgentJobExternalLeaseCutoverDiffItemView, 0)
	runtimeEnv := runtimeConfigEnvIndex(runtimeConfig.Environment)

	addDiff := func(item query.AgentJobExternalLeaseCutoverDiffItemView) {
		if item.Status == "match" {
			matching = append(matching, item)
			return
		}
		drift = append(drift, item)
	}

	for runtimeKey, expected := range bundle.EnvironmentOverrides {
		actualItem, ok := runtimeEnv[runtimeKey]
		if !ok || !actualItem.Present {
			addDiff(query.AgentJobExternalLeaseCutoverDiffItemView{
				Name:       runtimeKey,
				Kind:       "environment_override",
				Expected:   expected,
				Status:     "missing",
				Detail:     "runtime config does not currently expose this required external-lease/result-ack flag",
				RuntimeKey: runtimeKey,
				Endpoint:   "/v1/runtime-config",
			})
			continue
		}
		actual := strings.TrimSpace(actualItem.ValueRedacted)
		status := "match"
		detail := "runtime config already matches the canonical launcher bundle"
		if actual != expected {
			status = "mismatch"
			detail = "runtime config differs from the canonical launcher bundle"
		}
		addDiff(query.AgentJobExternalLeaseCutoverDiffItemView{
			Name:       runtimeKey,
			Kind:       "environment_override",
			Expected:   expected,
			Actual:     actual,
			Status:     status,
			Detail:     detail,
			RuntimeKey: runtimeKey,
			Endpoint:   "/v1/runtime-config",
		})
	}

	for _, input := range bundle.RequiredExternalInputs {
		if strings.TrimSpace(input.Parameter) != "QueueDSN" {
			continue
		}
		status := "match"
		detail := "runtime queue backend already reports an external queue DSN"
		actual := "configured"
		if !queueBackend.DSNConfigured {
			status = "missing"
			detail = "runtime queue backend still reports no external queue DSN"
			actual = "missing"
		}
		addDiff(query.AgentJobExternalLeaseCutoverDiffItemView{
			Name:       input.Parameter,
			Kind:       "external_input",
			Expected:   "configured",
			Actual:     actual,
			Status:     status,
			Detail:     detail,
			RuntimeKey: input.EnvironmentKey,
			Endpoint:   "/v1/queue-backend",
		})
	}

	expectedQueueProvider := strings.TrimSpace(bundle.LauncherParameters["QueueBackend"])
	if expectedQueueProvider != "" {
		status := "match"
		detail := "queue backend provider already matches the canonical launcher bundle"
		if strings.TrimSpace(queueBackend.Provider) != expectedQueueProvider {
			status = "mismatch"
			detail = "queue backend provider still differs from the canonical launcher bundle"
		}
		addDiff(query.AgentJobExternalLeaseCutoverDiffItemView{
			Name:     "queue_provider",
			Kind:     "queue_backend",
			Expected: expectedQueueProvider,
			Actual:   strings.TrimSpace(queueBackend.Provider),
			Status:   status,
			Detail:   detail,
			Endpoint: "/v1/queue-backend",
		})
	}

	expectedQueueMode := strings.TrimSpace(bundle.LauncherParameters["QueueMode"])
	if expectedQueueMode != "" {
		status := "match"
		detail := "queue backend mode already matches the canonical launcher bundle"
		if strings.TrimSpace(queueBackend.Mode) != expectedQueueMode {
			status = "mismatch"
			detail = "queue backend mode still differs from the canonical launcher bundle"
		}
		addDiff(query.AgentJobExternalLeaseCutoverDiffItemView{
			Name:     "queue_mode",
			Kind:     "queue_backend",
			Expected: expectedQueueMode,
			Actual:   strings.TrimSpace(queueBackend.Mode),
			Status:   status,
			Detail:   detail,
			Endpoint: "/v1/queue-backend",
		})
	}

	currentExecutionOwner, currentAckOwner := topologyAgentJobOwnership(topology)
	expectedAckOwner := expectedAgentJobAckOwner(bundle.DesiredExecutionOwner)
	addDiff(compareCutoverStringField(
		"agent_job_execution_owner",
		"queue_topology",
		bundle.DesiredExecutionOwner,
		currentExecutionOwner,
		"queue topology already routes agent_job execution to the desired owner",
		"queue topology still routes agent_job execution to a different owner",
		"/v1/queue-topology",
	))
	addDiff(compareCutoverStringField(
		"agent_job_ack_owner",
		"queue_topology",
		expectedAckOwner,
		currentAckOwner,
		"queue topology already routes agent_job acknowledgements to the expected owner",
		"queue topology still routes agent_job acknowledgements to a different owner",
		"/v1/queue-topology",
	))

	expectedExternalLeaseReady := bundle.DesiredExecutionOwner == agentJobExternalLeaseNATSOwner
	externalLeaseActual := "false"
	if topology.ExternalLeaseReady {
		externalLeaseActual = "true"
	}
	externalLeaseExpected := "false"
	if expectedExternalLeaseReady {
		externalLeaseExpected = "true"
	}
	status := "match"
	detail := "queue topology external-lease readiness already matches the desired cutover state"
	if topology.ExternalLeaseReady != expectedExternalLeaseReady {
		status = "mismatch"
		detail = "queue topology external-lease readiness still differs from the desired cutover state"
	}
	addDiff(query.AgentJobExternalLeaseCutoverDiffItemView{
		Name:     "external_lease_ready",
		Kind:     "queue_topology",
		Expected: externalLeaseExpected,
		Actual:   externalLeaseActual,
		Status:   status,
		Detail:   detail,
		Endpoint: "/v1/queue-topology",
	})

	blockers := append([]string(nil), bundle.Blockers...)
	for _, item := range drift {
		blockers = append(blockers, "runtime_drift:"+item.Name)
	}
	blockers = sortedUniqueSmokeStrings(blockers)
	ready := bundle.Ready && len(drift) == 0
	reason := "agent_job_external_lease_cutover_diff_ready"
	if !ready {
		reason = "agent_job_external_lease_cutover_diff_blocked"
	}

	return query.AgentJobExternalLeaseCutoverDiffView{
		Ready:                      ready,
		Reason:                     reason,
		Blockers:                   blockers,
		DesiredExecutionOwner:      bundle.DesiredExecutionOwner,
		RecommendedExecutionOwner:  bundle.RecommendedExecutionOwner,
		CurrentExecutionOwner:      currentExecutionOwner,
		CurrentAckOwner:            currentAckOwner,
		ExpectedAckOwner:           expectedAckOwner,
		CurrentQueueProvider:       strings.TrimSpace(queueBackend.Provider),
		ExpectedQueueProvider:      expectedQueueProvider,
		CurrentQueueMode:           strings.TrimSpace(queueBackend.Mode),
		ExpectedQueueMode:          expectedQueueMode,
		CurrentExternalLeaseReady:  topology.ExternalLeaseReady,
		ExpectedExternalLeaseReady: expectedExternalLeaseReady,
		Bundle:                     bundle,
		Matching:                   matching,
		Drift:                      drift,
		Notes: []string{
			"read-only diff between the current runtime snapshot and the canonical launcher bundle for agent_job external lease result-ack cutover",
			"diff items isolate runtime config and ownership drift from plan/readiness blockers so production preflight can separate missing flags from missing worker coverage",
		},
		SideEffect: "none",
	}, nil
}

func runtimeConfigEnvIndex(items []query.RuntimeEnvVarView) map[string]query.RuntimeEnvVarView {
	index := make(map[string]query.RuntimeEnvVarView, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		index[key] = item
	}
	return index
}

func topologyAgentJobOwnership(topology query.QueueTopologyView) (string, string) {
	for _, item := range topology.WorkKinds {
		if strings.TrimSpace(item.WorkKind) != "agent_job" {
			continue
		}
		return strings.TrimSpace(item.ExecutionOwner), strings.TrimSpace(item.AckOwner)
	}
	return "", ""
}

func expectedAgentJobAckOwner(desiredExecutionOwner string) string {
	if strings.TrimSpace(desiredExecutionOwner) == agentJobExternalLeaseNATSOwner {
		return "nats_external_lease_result_ack"
	}
	return "go_state_store_api"
}

func compareCutoverStringField(
	name string,
	kind string,
	expected string,
	actual string,
	matchDetail string,
	mismatchDetail string,
	endpoint string,
) query.AgentJobExternalLeaseCutoverDiffItemView {
	status := "match"
	detail := matchDetail
	if strings.TrimSpace(actual) != strings.TrimSpace(expected) {
		status = "mismatch"
		detail = mismatchDetail
	}
	return query.AgentJobExternalLeaseCutoverDiffItemView{
		Name:     name,
		Kind:     kind,
		Expected: expected,
		Actual:   actual,
		Status:   status,
		Detail:   detail,
		Endpoint: endpoint,
	}
}
