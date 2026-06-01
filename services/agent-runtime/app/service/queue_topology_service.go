package service

import (
	"context"
	"errors"

	inport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/in"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type QueueTopologyService struct {
	backend inport.QueueBackendViewer
}

func NewQueueTopologyService(backend inport.QueueBackendViewer) *QueueTopologyService {
	return &QueueTopologyService{backend: backend}
}

func (s *QueueTopologyService) GetQueueTopology(ctx context.Context) (query.QueueTopologyView, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueTopologyView{}, err
	}
	if s == nil || s.backend == nil {
		return query.QueueTopologyView{}, errors.New("queue topology service requires queue backend viewer")
	}
	backend, err := s.backend.Get(ctx)
	if err != nil {
		return query.QueueTopologyView{}, err
	}
	return buildQueueTopologyView(backend), nil
}

func buildQueueTopologyView(backend query.QueueBackendView) query.QueueTopologyView {
	selectedProvider := backend.Provider
	if backend.SelectedProviderCapability != nil && backend.SelectedProviderCapability.Provider != "" {
		selectedProvider = backend.SelectedProviderCapability.Provider
	}
	externalLeaseReady := backend.ExternalLease != nil && backend.ExternalLease.AllowExecution
	executionScope := ""
	if backend.ExternalLease != nil {
		executionScope = backend.ExternalLease.ExecutionScope
	}
	workKinds := []query.QueueTopologyWorkKind{
		queueTopologyWorkKind(backend, "outbox_delivery", backend.OutboxQueueSource, backend.OutboxExecutionOwner),
		queueTopologyWorkKind(backend, "agent_job", backend.AgentJobQueueSource, backend.AgentJobExecutionOwner),
	}
	blockers := queueTopologyBlockers(backend, workKinds)
	return query.QueueTopologyView{
		Provider:                backend.Provider,
		Mode:                    backend.Mode,
		MigrationPhase:          backend.MigrationPhase,
		SelectedProvider:        selectedProvider,
		RecommendedProvider:     backend.RecommendedFirstBackend,
		StateStoreAuthoritative: backend.StateStoreAuthoritative,
		ExternalQueueActive:     backend.ExternalQueueActive,
		ExternalLeaseReady:      externalLeaseReady,
		ExecutionScope:          executionScope,
		Nodes:                   queueTopologyNodes(backend),
		Edges:                   queueTopologyEdges(workKinds),
		WorkKinds:               workKinds,
		Blockers:                blockers,
		Notes:                   queueTopologyNotes(backend),
		SideEffect:              "none",
	}
}

func queueTopologyNodes(backend query.QueueBackendView) []query.QueueTopologyNodeView {
	nodes := []query.QueueTopologyNodeView{
		{
			ID:     "state_store",
			Label:  "Go State Store",
			Kind:   "state_store",
			Status: boolStatus(backend.StateStoreAuthoritative),
			Owner:  backend.LeaseOwner,
			Active: backend.StateStoreAuthoritative,
		},
		{
			ID:     "external_queue",
			Label:  backend.Provider,
			Kind:   "external_queue",
			Status: boolStatus(backend.ExternalQueueActive),
			Active: backend.ExternalQueueActive,
			Attributes: map[string]string{
				"mode":            backend.Mode,
				"migration_phase": backend.MigrationPhase,
			},
		},
		{
			ID:     "go_outbox_executor",
			Label:  "Go Outbox Executor",
			Kind:   "go_runtime_worker",
			Status: ownerNodeStatus(backend.OutboxExecutionOwner, "go"),
			Owner:  backend.OutboxExecutionOwner,
			Active: backend.OutboxExecutionOwner == "go_local_outbox_worker" || backend.OutboxExecutionOwner == "go_state_store_api",
		},
		{
			ID:     "python_ai_worker",
			Label:  "Python AI Worker",
			Kind:   "python_worker",
			Status: ownerNodeStatus(backend.AgentJobExecutionOwner, "python"),
			Owner:  backend.AgentJobExecutionOwner,
			Active: backend.AgentJobExecutionOwner != "",
		},
	}
	if backend.ExternalLease != nil {
		nodes = append(nodes, query.QueueTopologyNodeView{
			ID:     "external_lease_executor",
			Label:  "NATS External Lease Executor",
			Kind:   "external_lease_executor",
			Status: boolStatus(backend.ExternalLease.AllowExecution),
			Owner:  backend.ExternalLease.ExecutionScope,
			Active: backend.ExternalLease.AllowExecution,
			Attributes: map[string]string{
				"gate_state": backend.ExternalLease.GateState,
				"ack_policy": backend.ExternalLease.AckPolicy,
			},
		})
	}
	return nodes
}

func queueTopologyWorkKind(backend query.QueueBackendView, workKind string, queueSource string, executionOwner string) query.QueueTopologyWorkKind {
	blockers := queueTopologyWorkKindBlockers(backend, workKind)
	allowed := len(blockers) == 0
	gateState := ""
	if backend.ExternalLease != nil {
		gateState = backend.ExternalLease.GateState
		if externalLeaseMentionsWorkKind(backend.ExternalLease.AllowedWorkKinds, workKind) {
			allowed = allowed && backend.ExternalLease.AllowExecution
		}
	}
	notes := []string{}
	if workKind == "agent_job" {
		notes = append(notes, "Python workers execute AI work; Go owns lifecycle and result acknowledgement state")
	}
	return query.QueueTopologyWorkKind{
		WorkKind:       workKind,
		QueueSource:    queueSource,
		ExecutionOwner: executionOwner,
		AckOwner:       queueTopologyAckOwner(backend, workKind, executionOwner),
		GateState:      gateState,
		Allowed:        allowed,
		Blockers:       blockers,
		Notes:          notes,
	}
}

func queueTopologyEdges(workKinds []query.QueueTopologyWorkKind) []query.QueueTopologyEdgeView {
	edges := make([]query.QueueTopologyEdgeView, 0, len(workKinds))
	for _, item := range workKinds {
		to := "python_ai_worker"
		if item.WorkKind == "outbox_delivery" {
			to = "go_outbox_executor"
			if item.ExecutionOwner == "nats_external_lease" {
				to = "external_lease_executor"
			}
		}
		status := "ok"
		if !item.Allowed {
			status = "blocked"
		}
		edges = append(edges, query.QueueTopologyEdgeView{
			From:           "state_store",
			To:             to,
			WorkKind:       item.WorkKind,
			QueueSource:    item.QueueSource,
			ExecutionOwner: item.ExecutionOwner,
			AckOwner:       item.AckOwner,
			Status:         status,
			GateState:      item.GateState,
			Blockers:       item.Blockers,
		})
	}
	return edges
}

func queueTopologyWorkKindBlockers(backend query.QueueBackendView, workKind string) []string {
	if backend.ExternalLease == nil {
		return nil
	}
	for _, item := range backend.ExternalLease.BlockedWorkKinds {
		if item.WorkKind != workKind {
			continue
		}
		if item.RequiredChange != "" {
			return []string{item.Reason + ":" + item.RequiredChange}
		}
		return []string{item.Reason}
	}
	if externalLeaseMentionsWorkKind(backend.ExternalLease.AllowedWorkKinds, workKind) && !backend.ExternalLease.AllowExecution {
		return append([]string{}, backend.ExternalLease.Blockers...)
	}
	return nil
}

func queueTopologyAckOwner(backend query.QueueBackendView, workKind string, executionOwner string) string {
	if backend.ExternalLease != nil && externalLeaseMentionsWorkKind(backend.ExternalLease.AllowedWorkKinds, workKind) {
		if workKind == "agent_job" {
			return "nats_external_lease_result_ack"
		}
		return "nats_external_lease"
	}
	if executionOwner == "python_ai_worker_with_nats_result_ack" {
		return "nats_external_lease_result_ack"
	}
	return "go_state_store_api"
}

func queueTopologyBlockers(backend query.QueueBackendView, workKinds []query.QueueTopologyWorkKind) []string {
	blockers := []string{}
	if backend.ExternalLease != nil {
		blockers = append(blockers, backend.ExternalLease.Blockers...)
	}
	for _, item := range workKinds {
		for _, blocker := range item.Blockers {
			blockers = append(blockers, item.WorkKind+":"+blocker)
		}
	}
	return blockers
}

func queueTopologyNotes(backend query.QueueBackendView) []string {
	notes := append([]string{
		"queue topology is read-only and does not lease, acknowledge, publish or execute work",
	}, backend.Notes...)
	if backend.ExternalLease != nil {
		notes = append(notes, backend.ExternalLease.Notes...)
	}
	return notes
}

func externalLeaseMentionsWorkKind(values []string, workKind string) bool {
	for _, value := range values {
		if value == workKind {
			return true
		}
	}
	return false
}

func boolStatus(value bool) string {
	if value {
		return "ok"
	}
	return "disabled"
}

func ownerNodeStatus(owner string, expectedPrefix string) string {
	if owner == "" {
		return "unknown"
	}
	if expectedPrefix == "go" && len(owner) >= 2 && owner[:2] == "go" {
		return "ok"
	}
	if expectedPrefix == "python" && len(owner) >= 6 && owner[:6] == "python" {
		return "ok"
	}
	return "standby"
}
