package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	inport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/in"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const defaultKnowledgePipelineDiagnosticsLimit = 50
const (
	knowledgePipelineLagWarn   = 20
	knowledgePipelineLagDanger = 100
)

type knowledgePipelineObserveTargetLister interface {
	ListObserveTargets(ctx context.Context) (query.ObserveTargetsView, error)
}

type KnowledgePipelineDiagnosticsService struct {
	observeTargets knowledgePipelineObserveTargetLister
	observeCapture inport.ObserveCaptureDiagnosticsViewer
	agentWorkers   inport.AgentWorkerStatusManager
	inboxEvents    outport.InboxEventRepository
	agentJobs      outport.AgentJobRepository
	checkpoints    outport.KnowledgeCheckpointRepository
	clock          func() time.Time
}

func NewKnowledgePipelineDiagnosticsService(
	observeTargets knowledgePipelineObserveTargetLister,
	observeCapture inport.ObserveCaptureDiagnosticsViewer,
	agentWorkers inport.AgentWorkerStatusManager,
	inboxEvents outport.InboxEventRepository,
	agentJobs outport.AgentJobRepository,
	checkpoints outport.KnowledgeCheckpointRepository,
) *KnowledgePipelineDiagnosticsService {
	return &KnowledgePipelineDiagnosticsService{
		observeTargets: observeTargets,
		observeCapture: observeCapture,
		agentWorkers:   agentWorkers,
		inboxEvents:    inboxEvents,
		agentJobs:      agentJobs,
		checkpoints:    checkpoints,
		clock:          time.Now,
	}
}

func (s *KnowledgePipelineDiagnosticsService) GetKnowledgePipelineDiagnostics(
	ctx context.Context,
	filter query.KnowledgePipelineDiagnosticsFilter,
) (query.KnowledgePipelineDiagnosticsView, error) {
	if err := ctx.Err(); err != nil {
		return query.KnowledgePipelineDiagnosticsView{}, err
	}
	if s == nil || s.observeTargets == nil || s.observeCapture == nil || s.agentWorkers == nil || s.inboxEvents == nil || s.agentJobs == nil || s.checkpoints == nil {
		return query.KnowledgePipelineDiagnosticsView{}, errors.New("knowledge pipeline diagnostics service requires observe targets, capture diagnostics, agent workers, inbox events, agent jobs, and checkpoints")
	}
	now := filter.Now
	if now.IsZero() {
		now = s.now()
	}
	limit := boundedKnowledgePipelineDiagnosticsLimit(filter.Limit)
	staleAfterSeconds := filter.StaleAfterSeconds
	if staleAfterSeconds <= 0 {
		staleAfterSeconds = defaultKnowledgeDiagnosticsStaleAfterSeconds
	}

	targetsView, err := s.observeTargets.ListObserveTargets(ctx)
	if err != nil {
		return query.KnowledgePipelineDiagnosticsView{}, err
	}
	captureView, err := s.observeCapture.GetObserveCaptureDiagnostics(ctx, query.ObserveCaptureDiagnosticsFilter{Limit: limit})
	if err != nil {
		return query.KnowledgePipelineDiagnosticsView{}, err
	}
	workersView, err := s.agentWorkers.ListAgentWorkerStatuses(ctx, query.AgentWorkerStatusFilter{StaleAfterSeconds: staleAfterSeconds})
	if err != nil {
		return query.KnowledgePipelineDiagnosticsView{}, err
	}

	groupMemoryJobs, err := s.agentJobs.ListAgentJobs(ctx, query.AgentJobFilter{
		JobType: string(model.AgentJobGroupMemoryExtract),
		Limit:   limit,
	})
	if err != nil {
		return query.KnowledgePipelineDiagnosticsView{}, err
	}
	ragIngestJobs, err := s.agentJobs.ListAgentJobs(ctx, query.AgentJobFilter{
		JobType: string(model.AgentJobRagIngest),
		Limit:   limit,
	})
	if err != nil {
		return query.KnowledgePipelineDiagnosticsView{}, err
	}
	memoryCheckpoints, err := s.checkpoints.ListKnowledgeCheckpoints(ctx, query.KnowledgeCheckpointFilter{
		Limit:  limit,
		Prefix: "memory:",
	})
	if err != nil {
		return query.KnowledgePipelineDiagnosticsView{}, err
	}
	ragCheckpoints, err := s.checkpoints.ListKnowledgeCheckpoints(ctx, query.KnowledgeCheckpointFilter{
		Limit:  limit,
		Prefix: "ragflow:",
	})
	if err != nil {
		return query.KnowledgePipelineDiagnosticsView{}, err
	}

	captureByTarget := make(map[string]query.ObserveCaptureTargetDiagnosticsView, len(captureView.Targets))
	for _, item := range captureView.Targets {
		captureByTarget[item.TargetID] = item
	}
	groupMemoryByConversation := bucketAgentJobsByConversation(groupMemoryJobs)
	ragIngestByConversation := bucketAgentJobsByConversation(ragIngestJobs)
	memoryCheckpointByConversation := latestMemoryCheckpointByConversation(memoryCheckpoints)
	ragCheckpointByConversation := ragCheckpointsByConversation(ragCheckpoints)

	pipelines := make([]query.KnowledgePipelineView, 0, len(targetsView.Targets))
	for _, target := range targetsView.Targets {
		if strings.TrimSpace(target.Channel.Kind) != string(model.ChannelKindQQ) ||
			strings.TrimSpace(target.Channel.ConversationType) != string(model.ConversationTypeGroup) ||
			!target.ObserveOnly {
			continue
		}
		pipelines = append(pipelines, buildKnowledgePipelineView(
			s.latestSourceState(ctx, target.Channel, limit),
			target,
			captureByTarget[target.TargetID],
			groupMemoryByConversation[target.Channel.ConversationID],
			ragIngestByConversation[target.Channel.ConversationID],
			memoryCheckpointByConversation[target.Channel.ConversationID],
			ragCheckpointByConversation[target.Channel.ConversationID],
			workersView,
		))
	}
	sort.SliceStable(pipelines, func(i, j int) bool {
		if pipelines[i].Status != pipelines[j].Status {
			return knowledgePipelineStatusRank(pipelines[i].Status) > knowledgePipelineStatusRank(pipelines[j].Status)
		}
		return pipelines[i].TargetID < pipelines[j].TargetID
	})
	return query.KnowledgePipelineDiagnosticsView{
		GeneratedAt:            formatKnowledgeDiagnosticsTime(now),
		StaleAfterSeconds:      staleAfterSeconds,
		SampledJobLimit:        limit,
		SampledCheckpointLimit: limit,
		Totals:                 knowledgePipelineTotals(pipelines),
		Pipelines:              pipelines,
		Notes: []string{
			"side_effect=none",
			"group_level_knowledge_control_plane_view",
			"bounded_samples_for_jobs_and_checkpoints",
		},
		SideEffect: "none",
	}, nil
}

type knowledgePipelineSourceState struct {
	Known           bool
	LatestSeq       int
	SequencedEvents int
}

func (s *KnowledgePipelineDiagnosticsService) latestSourceState(
	ctx context.Context,
	channel query.ObserveTargetChannelView,
	limit int,
) knowledgePipelineSourceState {
	events, err := s.inboxEvents.ListInboxEvents(ctx, query.InboxEventFilter{
		Limit:            limit,
		ChannelKind:      channel.Kind,
		AccountID:        channel.AccountID,
		ConversationID:   channel.ConversationID,
		ConversationType: channel.ConversationType,
		ObserveOnly:      "true",
	})
	if err != nil {
		return knowledgePipelineSourceState{}
	}
	state := knowledgePipelineSourceState{}
	for _, event := range events {
		seq, ok := inboxMetricSeq(event)
		if !ok {
			continue
		}
		state.SequencedEvents++
		if !state.Known || seq > state.LatestSeq {
			state.Known = true
			state.LatestSeq = seq
		}
	}
	return state
}

func (s *KnowledgePipelineDiagnosticsService) now() time.Time {
	if s != nil && s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
}

func boundedKnowledgePipelineDiagnosticsLimit(limit int) int {
	if limit <= 0 {
		return defaultKnowledgePipelineDiagnosticsLimit
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func bucketAgentJobsByConversation(items []model.AgentJob) map[string][]model.AgentJob {
	grouped := make(map[string][]model.AgentJob)
	for _, item := range items {
		if item.Route.ConversationID == "" {
			continue
		}
		grouped[item.Route.ConversationID] = append(grouped[item.Route.ConversationID], item)
	}
	for key := range grouped {
		sort.SliceStable(grouped[key], func(i, j int) bool {
			if grouped[key][i].UpdatedAt.Equal(grouped[key][j].UpdatedAt) {
				return grouped[key][i].JobID > grouped[key][j].JobID
			}
			return grouped[key][i].UpdatedAt.After(grouped[key][j].UpdatedAt)
		})
	}
	return grouped
}

func latestMemoryCheckpointByConversation(items []model.KnowledgeCheckpoint) map[string]*query.KnowledgeCheckpointView {
	result := make(map[string]*query.KnowledgeCheckpointView)
	for _, item := range items {
		groupID := knowledgeCheckpointConversationID(item)
		if groupID == "" {
			continue
		}
		view := assembler.ToKnowledgeCheckpointView(item)
		existing := result[groupID]
		if existing == nil || view.UpdatedAt > existing.UpdatedAt || (view.UpdatedAt == existing.UpdatedAt && view.CheckpointID > existing.CheckpointID) {
			cp := view
			result[groupID] = &cp
		}
	}
	return result
}

func ragCheckpointsByConversation(items []model.KnowledgeCheckpoint) map[string][]query.KnowledgeCheckpointView {
	result := make(map[string][]query.KnowledgeCheckpointView)
	for _, item := range items {
		groupID := knowledgeCheckpointConversationID(item)
		if groupID == "" {
			continue
		}
		result[groupID] = append(result[groupID], assembler.ToKnowledgeCheckpointView(item))
	}
	for key := range result {
		sort.SliceStable(result[key], func(i, j int) bool {
			if result[key][i].UpdatedAt == result[key][j].UpdatedAt {
				return result[key][i].CheckpointID > result[key][j].CheckpointID
			}
			return result[key][i].UpdatedAt > result[key][j].UpdatedAt
		})
	}
	return result
}

func knowledgeCheckpointConversationID(item model.KnowledgeCheckpoint) string {
	if groupID := strings.TrimSpace(item.Metadata["group_id"]); groupID != "" {
		return groupID
	}
	parts := strings.Split(strings.TrimSpace(item.CheckpointID), ":")
	if len(parts) >= 3 && parts[1] == "qq" {
		return parts[2]
	}
	return ""
}

func buildKnowledgePipelineView(
	source knowledgePipelineSourceState,
	target query.ObserveTargetView,
	capture query.ObserveCaptureTargetDiagnosticsView,
	groupMemoryJobs []model.AgentJob,
	ragIngestJobs []model.AgentJob,
	memoryCheckpoint *query.KnowledgeCheckpointView,
	ragCheckpoints []query.KnowledgeCheckpointView,
	workers query.AgentWorkerStatusesView,
) query.KnowledgePipelineView {
	groupMemory := knowledgePipelineJobStage(string(model.AgentJobGroupMemoryExtract), groupMemoryJobs)
	ragIngest := knowledgePipelineJobStage(string(model.AgentJobRagIngest), ragIngestJobs)
	memoryLag := knowledgePipelineCheckpointLag(memoryCheckpoint, source)
	ragLagMax := knowledgePipelineCheckpointLagMax(ragCheckpoints, source)
	coverage := agentJobWorkerCoverageFromPressure([]query.AgentJobTypePressureView{
		knowledgePipelinePressure(groupMemory),
		knowledgePipelinePressure(ragIngest),
	}, workers)
	status, reasons := knowledgePipelineStatus(target, capture, source, groupMemory, ragIngest, memoryLag, ragLagMax, coverage)
	return query.KnowledgePipelineView{
		TargetID:            target.TargetID,
		Channel:             target.Channel,
		Enabled:             target.Enabled,
		ObserveOnly:         target.ObserveOnly,
		CaptureStatus:       capture.Status,
		ReceiverConnected:   capture.ReceiverConnected,
		CaptureBlockers:     append([]string(nil), capture.Blockers...),
		SequencedEvents:     source.SequencedEvents,
		SourceSeqKnown:      source.Known,
		LatestSourceSeq:     source.LatestSeq,
		GroupMemory:         groupMemory,
		RagIngest:           ragIngest,
		MemoryCheckpoint:    memoryCheckpoint,
		RagCheckpoints:      append([]query.KnowledgeCheckpointView(nil), ragCheckpoints...),
		MemoryCheckpointLag: memoryLag,
		RagCheckpointLagMax: ragLagMax,
		WorkerCoverage:      coverage,
		Status:              status,
		Reasons:             reasons,
	}
}

func knowledgePipelineCheckpointLag(
	checkpoint *query.KnowledgeCheckpointView,
	source knowledgePipelineSourceState,
) *query.KnowledgePipelineCheckpointLagView {
	if checkpoint == nil {
		return nil
	}
	view := &query.KnowledgePipelineCheckpointLagView{
		CheckpointID: checkpoint.CheckpointID,
		Cursor:       checkpoint.Cursor,
		UpdatedAt:    checkpoint.UpdatedAt,
		Status:       "muted",
	}
	if !source.Known {
		view.Reason = "source_seq_unknown"
		return view
	}
	view.LatestSourceSeq = source.LatestSeq
	lag := source.LatestSeq - checkpoint.Cursor
	if lag < 0 {
		lag = 0
	}
	view.Lag = lag
	switch {
	case lag >= knowledgePipelineLagDanger:
		view.Status = "danger"
		view.Reason = "lag>=100"
	case lag >= knowledgePipelineLagWarn:
		view.Status = "warn"
		view.Reason = "lag>=20"
	default:
		view.Status = "ok"
		view.Reason = "lag_within_threshold"
	}
	return view
}

func knowledgePipelineCheckpointLagMax(
	checkpoints []query.KnowledgeCheckpointView,
	source knowledgePipelineSourceState,
) *query.KnowledgePipelineCheckpointLagView {
	var best *query.KnowledgePipelineCheckpointLagView
	for _, checkpoint := range checkpoints {
		item := knowledgePipelineCheckpointLag(&checkpoint, source)
		if item == nil {
			continue
		}
		if best == nil || item.Lag > best.Lag || (item.Lag == best.Lag && item.CheckpointID > best.CheckpointID) {
			best = item
		}
	}
	return best
}

func knowledgePipelineJobStage(jobType string, jobs []model.AgentJob) query.KnowledgePipelineJobStageView {
	stage := query.KnowledgePipelineJobStageView{JobType: jobType, SampledJobs: len(jobs)}
	for _, job := range jobs {
		switch job.Status {
		case model.AgentJobPending:
			stage.Pending++
		case model.AgentJobLeased:
			stage.Leased++
		case model.AgentJobRunning:
			stage.Running++
		}
	}
	stage.Active = stage.Leased + stage.Running
	pressure := knowledgePipelinePressure(stage)
	stage.HighPressure = pressure.HighPressure
	stage.PressureReason = pressure.PressureReason
	if len(jobs) > 0 {
		view := assembler.ToAgentJobView(jobs[0])
		stage.LatestJob = &view
	}
	return stage
}

func knowledgePipelinePressure(stage query.KnowledgePipelineJobStageView) query.AgentJobTypePressureView {
	pressure := query.AgentJobTypePressureView{
		JobType: stage.JobType,
		Pending: stage.Pending,
		Leased:  stage.Leased,
		Running: stage.Running,
		Active:  stage.Active,
	}
	switch {
	case stage.Pending >= 10:
		pressure.HighPressure = true
		pressure.PressureReason = "pending>=10"
	case stage.Active >= 5:
		pressure.HighPressure = true
		pressure.PressureReason = "active>=5"
	}
	return pressure
}

func knowledgePipelineStatus(
	target query.ObserveTargetView,
	capture query.ObserveCaptureTargetDiagnosticsView,
	source knowledgePipelineSourceState,
	groupMemory query.KnowledgePipelineJobStageView,
	ragIngest query.KnowledgePipelineJobStageView,
	memoryLag *query.KnowledgePipelineCheckpointLagView,
	ragLagMax *query.KnowledgePipelineCheckpointLagView,
	coverage []query.AgentJobWorkerCoverageView,
) (string, []string) {
	if !target.Enabled {
		return "muted", []string{"target_disabled"}
	}
	reasons := make([]string, 0, 6)
	if capture.Status == "danger" {
		reasons = append(reasons, "capture_blocked")
	}
	if capture.Status == "warn" {
		reasons = append(reasons, "capture_warning")
	}
	if groupMemory.Pending > 0 {
		reasons = append(reasons, "group_memory_pending")
	}
	if ragIngest.Pending > 0 {
		reasons = append(reasons, "rag_ingest_pending")
	}
	status := "ok"
	if capture.Status == "danger" {
		status = "blocked"
	}
	for _, item := range coverage {
		switch item.CoverageStatus {
		case "danger":
			reasons = append(reasons, item.JobType+"_worker_blocked")
			status = "blocked"
		case "warn":
			reasons = append(reasons, item.JobType+"_worker_warning")
			if status != "blocked" {
				status = "warn"
			}
		}
	}
	if status != "blocked" && capture.Status == "warn" {
		status = "warn"
	}
	if status == "ok" && (groupMemory.Pending > 0 || ragIngest.Pending > 0 || groupMemory.Active > 0 || ragIngest.Active > 0) {
		status = "warn"
	}
	if source.Known {
		if lagReason, nextStatus := knowledgePipelineLagReason("memory", memoryLag, groupMemory); lagReason != "" {
			reasons = append(reasons, lagReason)
			status = mergeKnowledgePipelineStatus(status, nextStatus)
		}
		if lagReason, nextStatus := knowledgePipelineLagReason("rag", ragLagMax, ragIngest); lagReason != "" {
			reasons = append(reasons, lagReason)
			status = mergeKnowledgePipelineStatus(status, nextStatus)
		}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "pipeline_ready")
	}
	return status, reasons
}

func knowledgePipelineLagReason(
	prefix string,
	lag *query.KnowledgePipelineCheckpointLagView,
	stage query.KnowledgePipelineJobStageView,
) (string, string) {
	if lag == nil {
		return "", ""
	}
	if lag.Status == "danger" && stage.HighPressure {
		return prefix + "_checkpoint_stalled_under_pressure", "blocked"
	}
	if lag.Status == "danger" || lag.Status == "warn" {
		return prefix + "_checkpoint_lagging", "warn"
	}
	return "", ""
}

func mergeKnowledgePipelineStatus(current string, next string) string {
	if knowledgePipelineStatusRank(next) > knowledgePipelineStatusRank(current) {
		return next
	}
	return current
}

func knowledgePipelineTotals(items []query.KnowledgePipelineView) map[string]int {
	totals := map[string]int{
		"targets":              len(items),
		"enabled":              0,
		"ready":                0,
		"warning":              0,
		"blocked":              0,
		"muted":                0,
		"receiver_connected":   0,
		"group_memory_pending": 0,
		"rag_ingest_pending":   0,
		"high_pressure":        0,
		"memory_checkpoints":   0,
		"rag_checkpoints":      0,
		"lagging":              0,
		"stalled":              0,
	}
	for _, item := range items {
		if item.Enabled {
			totals["enabled"]++
		}
		switch item.Status {
		case "ok":
			totals["ready"]++
		case "warn":
			totals["warning"]++
		case "blocked":
			totals["blocked"]++
		default:
			totals["muted"]++
		}
		if item.ReceiverConnected {
			totals["receiver_connected"]++
		}
		if item.GroupMemory.Pending > 0 {
			totals["group_memory_pending"]++
		}
		if item.RagIngest.Pending > 0 {
			totals["rag_ingest_pending"]++
		}
		if item.GroupMemory.HighPressure || item.RagIngest.HighPressure {
			totals["high_pressure"]++
		}
		if item.MemoryCheckpoint != nil {
			totals["memory_checkpoints"]++
		}
		totals["rag_checkpoints"] += len(item.RagCheckpoints)
		if (item.MemoryCheckpointLag != nil && (item.MemoryCheckpointLag.Status == "warn" || item.MemoryCheckpointLag.Status == "danger")) ||
			(item.RagCheckpointLagMax != nil && (item.RagCheckpointLagMax.Status == "warn" || item.RagCheckpointLagMax.Status == "danger")) {
			totals["lagging"]++
		}
		if knowledgePipelineContainsReason(item.Reasons, "memory_checkpoint_stalled_under_pressure") ||
			knowledgePipelineContainsReason(item.Reasons, "rag_checkpoint_stalled_under_pressure") {
			totals["stalled"]++
		}
	}
	return totals
}

func knowledgePipelineContainsReason(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func knowledgePipelineStatusRank(status string) int {
	switch status {
	case "blocked":
		return 4
	case "warn":
		return 3
	case "ok":
		return 2
	case "muted":
		return 1
	default:
		return 0
	}
}
