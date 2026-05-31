package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
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
	ragIngestByConversationDataset := bucketAgentJobsByConversationDataset(ragIngestJobs)
	memoryCheckpointByConversation := latestMemoryCheckpointByConversation(memoryCheckpoints)
	ragCheckpointByConversation := ragCheckpointsByConversation(ragCheckpoints)
	ragCheckpointByConversationDataset := ragCheckpointsByConversationDataset(ragCheckpoints)

	pipelines := make([]query.KnowledgePipelineView, 0, len(targetsView.Targets))
	for _, target := range targetsView.Targets {
		if strings.TrimSpace(target.Channel.Kind) != string(model.ChannelKindQQ) ||
			strings.TrimSpace(target.Channel.ConversationType) != string(model.ConversationTypeGroup) ||
			!target.ObserveOnly {
			continue
		}
		pipelines = append(pipelines, buildKnowledgePipelineView(
			now,
			staleAfterSeconds,
			s.latestSourceState(ctx, target.Channel, limit),
			target,
			captureByTarget[target.TargetID],
			groupMemoryByConversation[target.Channel.ConversationID],
			ragIngestByConversation[target.Channel.ConversationID],
			knowledgePipelineConfiguredDatasetIDs(target),
			ragIngestByConversationDataset[target.Channel.ConversationID],
			memoryCheckpointByConversation[target.Channel.ConversationID],
			ragCheckpointByConversation[target.Channel.ConversationID],
			ragCheckpointByConversationDataset[target.Channel.ConversationID],
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
		Totals:                 knowledgePipelineTotals(pipelines, staleAfterSeconds),
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

func bucketAgentJobsByConversationDataset(items []model.AgentJob) map[string]map[string][]model.AgentJob {
	grouped := make(map[string]map[string][]model.AgentJob)
	for _, item := range items {
		conversationID := strings.TrimSpace(item.Route.ConversationID)
		datasetID := knowledgePipelineJobDatasetID(item)
		if conversationID == "" || datasetID == "" {
			continue
		}
		byDataset := grouped[conversationID]
		if byDataset == nil {
			byDataset = make(map[string][]model.AgentJob)
			grouped[conversationID] = byDataset
		}
		byDataset[datasetID] = append(byDataset[datasetID], item)
	}
	for _, byDataset := range grouped {
		for key := range byDataset {
			sort.SliceStable(byDataset[key], func(i, j int) bool {
				if byDataset[key][i].UpdatedAt.Equal(byDataset[key][j].UpdatedAt) {
					return byDataset[key][i].JobID > byDataset[key][j].JobID
				}
				return byDataset[key][i].UpdatedAt.After(byDataset[key][j].UpdatedAt)
			})
		}
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

func ragCheckpointsByConversationDataset(items []model.KnowledgeCheckpoint) map[string]map[string][]query.KnowledgeCheckpointView {
	result := make(map[string]map[string][]query.KnowledgeCheckpointView)
	for _, item := range items {
		groupID := knowledgeCheckpointConversationID(item)
		datasetID := knowledgeCheckpointDatasetID(item)
		if groupID == "" || datasetID == "" {
			continue
		}
		byDataset := result[groupID]
		if byDataset == nil {
			byDataset = make(map[string][]query.KnowledgeCheckpointView)
			result[groupID] = byDataset
		}
		byDataset[datasetID] = append(byDataset[datasetID], assembler.ToKnowledgeCheckpointView(item))
	}
	for _, byDataset := range result {
		for key := range byDataset {
			sort.SliceStable(byDataset[key], func(i, j int) bool {
				if byDataset[key][i].UpdatedAt == byDataset[key][j].UpdatedAt {
					return byDataset[key][i].CheckpointID > byDataset[key][j].CheckpointID
				}
				return byDataset[key][i].UpdatedAt > byDataset[key][j].UpdatedAt
			})
		}
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

func knowledgeCheckpointDatasetID(item model.KnowledgeCheckpoint) string {
	if datasetID := strings.TrimSpace(item.Metadata["dataset_id"]); datasetID != "" {
		return datasetID
	}
	parts := strings.Split(strings.TrimSpace(item.CheckpointID), ":")
	if len(parts) >= 4 && parts[0] == "ragflow" {
		return strings.Join(parts[3:], ":")
	}
	return ""
}

func knowledgePipelineJobDatasetID(item model.AgentJob) string {
	if datasetID := strings.TrimSpace(item.Payload["dataset_id"]); datasetID != "" {
		return datasetID
	}
	if datasetID := strings.TrimSpace(item.Metadata["dataset_id"]); datasetID != "" {
		return datasetID
	}
	parts := strings.Split(strings.TrimSpace(item.JobID), ":")
	if len(parts) >= 4 && parts[0] == "rag_ingest" {
		return parts[3]
	}
	return ""
}

func buildKnowledgePipelineView(
	now time.Time,
	staleAfterSeconds int,
	source knowledgePipelineSourceState,
	target query.ObserveTargetView,
	capture query.ObserveCaptureTargetDiagnosticsView,
	groupMemoryJobs []model.AgentJob,
	ragIngestJobs []model.AgentJob,
	configuredDatasetIDs []string,
	ragIngestJobsByDataset map[string][]model.AgentJob,
	memoryCheckpoint *query.KnowledgeCheckpointView,
	ragCheckpoints []query.KnowledgeCheckpointView,
	ragCheckpointsByDataset map[string][]query.KnowledgeCheckpointView,
	workers query.AgentWorkerStatusesView,
) query.KnowledgePipelineView {
	groupMemory := knowledgePipelineJobStage(string(model.AgentJobGroupMemoryExtract), groupMemoryJobs, now, staleAfterSeconds)
	ragIngest := knowledgePipelineJobStage(string(model.AgentJobRagIngest), ragIngestJobs, now, staleAfterSeconds)
	ragDatasets := knowledgePipelineRagDatasets(
		now,
		staleAfterSeconds,
		source,
		configuredDatasetIDs,
		ragIngestJobsByDataset,
		ragCheckpointsByDataset,
	)
	memoryLag := knowledgePipelineCheckpointLag(memoryCheckpoint, source, now)
	ragLagMax := knowledgePipelineCheckpointLagMax(ragCheckpoints, source, now)
	coverage := agentJobWorkerCoverageFromPressure([]query.AgentJobTypePressureView{
		knowledgePipelinePressure(groupMemory),
		knowledgePipelinePressure(ragIngest),
	}, workers)
	status, reasons := knowledgePipelineStatus(target, capture, source, groupMemory, ragIngest, memoryLag, ragLagMax, coverage, staleAfterSeconds)
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
		RagDatasets:         ragDatasets,
		MemoryCheckpointLag: memoryLag,
		RagCheckpointLagMax: ragLagMax,
		WorkerCoverage:      coverage,
		Status:              status,
		Reasons:             reasons,
	}
}

func knowledgePipelineRagDatasets(
	now time.Time,
	staleAfterSeconds int,
	source knowledgePipelineSourceState,
	configuredDatasetIDs []string,
	jobsByDataset map[string][]model.AgentJob,
	checkpointsByDataset map[string][]query.KnowledgeCheckpointView,
) []query.KnowledgePipelineRagDatasetView {
	keys := make(map[string]struct{})
	configured := make(map[string]bool)
	for _, key := range configuredDatasetIDs {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
		configured[key] = true
	}
	for key := range jobsByDataset {
		if strings.TrimSpace(key) != "" {
			keys[key] = struct{}{}
		}
	}
	for key := range checkpointsByDataset {
		if strings.TrimSpace(key) != "" {
			keys[key] = struct{}{}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	items := make([]query.KnowledgePipelineRagDatasetView, 0, len(keys))
	for datasetID := range keys {
		runtimeObserved := len(jobsByDataset[datasetID]) > 0 || len(checkpointsByDataset[datasetID]) > 0
		if !runtimeObserved && configured[datasetID] {
			items = append(items, query.KnowledgePipelineRagDatasetView{
				DatasetID:       datasetID,
				Configured:      true,
				RuntimeObserved: false,
				JobStage:        knowledgePipelineJobStage(string(model.AgentJobRagIngest), nil, now, staleAfterSeconds),
				Status:          "muted",
				Reasons:         []string{"configured_dataset_not_started"},
			})
			continue
		}
		stage := knowledgePipelineJobStage(string(model.AgentJobRagIngest), jobsByDataset[datasetID], now, staleAfterSeconds)
		checkpoint := latestKnowledgeCheckpointView(checkpointsByDataset[datasetID])
		lag := knowledgePipelineCheckpointLag(checkpoint, source, now)
		status, reasons := knowledgePipelineRagDatasetStatus(stage, lag, staleAfterSeconds)
		items = append(items, query.KnowledgePipelineRagDatasetView{
			DatasetID:       datasetID,
			DisplayName:     knowledgePipelineDatasetDisplayName(checkpoint),
			Configured:      configured[datasetID],
			RuntimeObserved: runtimeObserved,
			JobStage:        stage,
			Checkpoint:      checkpoint,
			CheckpointLag:   lag,
			IngestSnapshot:  knowledgePipelineRagIngestSnapshot(checkpoint),
			Status:          status,
			Reasons:         reasons,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return knowledgePipelineStatusRank(items[i].Status) > knowledgePipelineStatusRank(items[j].Status)
		}
		return items[i].DatasetID < items[j].DatasetID
	})
	return items
}

func knowledgePipelineConfiguredDatasetIDs(target query.ObserveTargetView) []string {
	if target.Metadata == nil {
		return nil
	}
	return splitCommaSeparatedList(target.Metadata["ragflow_dataset_ids"])
}

func splitCommaSeparatedList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		result = append(result, part)
	}
	return result
}

func latestKnowledgeCheckpointView(items []query.KnowledgeCheckpointView) *query.KnowledgeCheckpointView {
	if len(items) == 0 {
		return nil
	}
	item := items[0]
	return &item
}

func knowledgePipelineDatasetDisplayName(checkpoint *query.KnowledgeCheckpointView) string {
	if checkpoint == nil || checkpoint.Metadata == nil {
		return ""
	}
	return strings.TrimSpace(checkpoint.Metadata["display_name"])
}

func knowledgePipelineRagIngestSnapshot(
	checkpoint *query.KnowledgeCheckpointView,
) *query.KnowledgePipelineRagIngestSnapshotView {
	if checkpoint == nil || checkpoint.Metadata == nil {
		return nil
	}
	messageCount, okMessage := parseKnowledgePipelineMetadataInt(checkpoint.Metadata["last_message_count"])
	documentCount, okDocument := parseKnowledgePipelineMetadataInt(checkpoint.Metadata["last_document_count"])
	startSeq, okStart := parseKnowledgePipelineMetadataInt(checkpoint.Metadata["last_start_seq"])
	endSeq, okEnd := parseKnowledgePipelineMetadataInt(checkpoint.Metadata["last_end_seq"])
	parseRequested, okParse := parseKnowledgePipelineMetadataBool(checkpoint.Metadata["last_parse_requested"])
	if !okMessage || !okDocument || !okStart || !okEnd || !okParse {
		return nil
	}
	return &query.KnowledgePipelineRagIngestSnapshotView{
		MessageCount:   messageCount,
		DocumentCount:  documentCount,
		StartSeq:       startSeq,
		EndSeq:         endSeq,
		ParseRequested: parseRequested,
		UpdatedAt:      checkpoint.UpdatedAt,
		DisplayName:    knowledgePipelineDatasetDisplayName(checkpoint),
	}
}

func parseKnowledgePipelineMetadataInt(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func parseKnowledgePipelineMetadataBool(value string) (bool, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "true", "1", "yes", "y", "on":
		return true, true
	case "false", "0", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

func knowledgePipelineCheckpointLag(
	checkpoint *query.KnowledgeCheckpointView,
	source knowledgePipelineSourceState,
	now time.Time,
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
	if updatedAt, ok := parseKnowledgePipelineCheckpointTime(checkpoint.UpdatedAt); ok && !now.IsZero() && !updatedAt.IsZero() {
		age := int(now.Sub(updatedAt).Seconds())
		if age < 0 {
			age = 0
		}
		view.AgeSeconds = age
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
	now time.Time,
) *query.KnowledgePipelineCheckpointLagView {
	var best *query.KnowledgePipelineCheckpointLagView
	for _, checkpoint := range checkpoints {
		item := knowledgePipelineCheckpointLag(&checkpoint, source, now)
		if item == nil {
			continue
		}
		if best == nil || item.Lag > best.Lag || (item.Lag == best.Lag && item.CheckpointID > best.CheckpointID) {
			best = item
		}
	}
	return best
}

func knowledgePipelineJobStage(
	jobType string,
	jobs []model.AgentJob,
	now time.Time,
	staleAfterSeconds int,
) query.KnowledgePipelineJobStageView {
	stage := query.KnowledgePipelineJobStageView{JobType: jobType, SampledJobs: len(jobs)}
	for _, job := range jobs {
		switch job.Status {
		case model.AgentJobPending:
			stage.Pending++
			age := knowledgePipelineJobAgeSeconds(now, job.CreatedAt)
			if age > stage.OldestPendingAgeSeconds {
				stage.OldestPendingAgeSeconds = age
			}
		case model.AgentJobLeased:
			stage.Leased++
			stage = accumulateKnowledgePipelineActiveLease(stage, job, now, staleAfterSeconds)
		case model.AgentJobRunning:
			stage.Running++
			stage = accumulateKnowledgePipelineActiveLease(stage, job, now, staleAfterSeconds)
		}
	}
	stage.Active = stage.Leased + stage.Running
	pressure := knowledgePipelinePressure(stage)
	stage.HighPressure = pressure.HighPressure
	stage.PressureReason = pressure.PressureReason
	stage.FreshnessStatus, stage.FreshnessReason = knowledgePipelineStageFreshness(stage, staleAfterSeconds)
	if len(jobs) > 0 {
		view := assembler.ToAgentJobView(jobs[0])
		stage.LatestJob = &view
	}
	return stage
}

func accumulateKnowledgePipelineActiveLease(
	stage query.KnowledgePipelineJobStageView,
	job model.AgentJob,
	now time.Time,
	staleAfterSeconds int,
) query.KnowledgePipelineJobStageView {
	age := knowledgePipelineJobAgeSeconds(now, job.UpdatedAt)
	if age > stage.OldestActiveAgeSeconds {
		stage.OldestActiveAgeSeconds = age
	}
	if job.LeaseExpired(now) {
		stage.ExpiredActiveLeases++
		return stage
	}
	if staleAfterSeconds > 0 && age >= staleAfterSeconds {
		stage.StaleActiveLeases++
	}
	return stage
}

func knowledgePipelineJobAgeSeconds(now time.Time, timestamp time.Time) int {
	if now.IsZero() || timestamp.IsZero() || now.Before(timestamp) {
		return 0
	}
	return int(now.Sub(timestamp) / time.Second)
}

func knowledgePipelineStageFreshness(
	stage query.KnowledgePipelineJobStageView,
	staleAfterSeconds int,
) (string, string) {
	switch {
	case stage.ExpiredActiveLeases > 0:
		return "danger", "expired_active_lease"
	case stage.StaleActiveLeases > 0:
		return "warn", "stale_active_lease"
	case staleAfterSeconds > 0 && stage.Pending > 0 && stage.OldestPendingAgeSeconds >= staleAfterSeconds:
		return "warn", "old_pending_backlog"
	case stage.Active > 0:
		return "ok", "active_lease_fresh"
	case stage.Pending > 0:
		return "ok", "pending_backlog_recent"
	default:
		return "muted", ""
	}
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
	staleAfterSeconds int,
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
		if lagReason, nextStatus := knowledgePipelineLagReason("memory", memoryLag, groupMemory, staleAfterSeconds); lagReason != "" {
			reasons = append(reasons, lagReason)
			status = mergeKnowledgePipelineStatus(status, nextStatus)
		}
		if lagReason, nextStatus := knowledgePipelineLagReason("rag", ragLagMax, ragIngest, staleAfterSeconds); lagReason != "" {
			reasons = append(reasons, lagReason)
			status = mergeKnowledgePipelineStatus(status, nextStatus)
		}
	}
	if stageReason, nextStatus := knowledgePipelineStageReason("group_memory", groupMemory); stageReason != "" {
		reasons = append(reasons, stageReason)
		status = mergeKnowledgePipelineStatus(status, nextStatus)
	}
	if stageReason, nextStatus := knowledgePipelineStageReason("rag_ingest", ragIngest); stageReason != "" {
		reasons = append(reasons, stageReason)
		status = mergeKnowledgePipelineStatus(status, nextStatus)
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "pipeline_ready")
	}
	return status, reasons
}

func knowledgePipelineStageReason(prefix string, stage query.KnowledgePipelineJobStageView) (string, string) {
	switch stage.FreshnessReason {
	case "expired_active_lease":
		return prefix + "_lease_expired", "blocked"
	case "stale_active_lease":
		return prefix + "_lease_stale", "warn"
	case "old_pending_backlog":
		return prefix + "_pending_old", "warn"
	default:
		return "", ""
	}
}

func knowledgePipelineRagDatasetStatus(
	stage query.KnowledgePipelineJobStageView,
	lag *query.KnowledgePipelineCheckpointLagView,
	staleAfterSeconds int,
) (string, []string) {
	status := "ok"
	reasons := make([]string, 0, 4)
	if stageReason, nextStatus := knowledgePipelineStageReason("rag_ingest", stage); stageReason != "" {
		reasons = append(reasons, stageReason)
		status = mergeKnowledgePipelineStatus(status, nextStatus)
	}
	if lagReason, nextStatus := knowledgePipelineLagReason("rag", lag, stage, staleAfterSeconds); lagReason != "" {
		reasons = append(reasons, lagReason)
		status = mergeKnowledgePipelineStatus(status, nextStatus)
	}
	if status == "ok" && (stage.Pending > 0 || stage.Active > 0) {
		status = "warn"
		if stage.Pending > 0 {
			reasons = append(reasons, "rag_ingest_pending_recent")
		} else if stage.Active > 0 {
			reasons = append(reasons, "rag_ingest_active_recent")
		}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "dataset_ready")
	}
	return status, reasons
}

func knowledgePipelineLagReason(
	prefix string,
	lag *query.KnowledgePipelineCheckpointLagView,
	stage query.KnowledgePipelineJobStageView,
	staleAfterSeconds int,
) (string, string) {
	if lag == nil {
		return "", ""
	}
	stale := knowledgePipelineCheckpointIsStale(lag, staleAfterSeconds)
	if lag.Status == "danger" && stage.HighPressure && stale {
		return prefix + "_checkpoint_stalled_under_pressure", "blocked"
	}
	if stale && (lag.Status == "danger" || lag.Status == "warn") {
		return prefix + "_checkpoint_stagnant", "warn"
	}
	if lag.Status == "danger" || lag.Status == "warn" {
		return prefix + "_checkpoint_lagging", "warn"
	}
	return "", ""
}

func knowledgePipelineCheckpointIsStale(
	lag *query.KnowledgePipelineCheckpointLagView,
	staleAfterSeconds int,
) bool {
	if lag == nil || staleAfterSeconds <= 0 {
		return false
	}
	return lag.AgeSeconds >= staleAfterSeconds
}

func mergeKnowledgePipelineStatus(current string, next string) string {
	if knowledgePipelineStatusRank(next) > knowledgePipelineStatusRank(current) {
		return next
	}
	return current
}

func knowledgePipelineTotals(items []query.KnowledgePipelineView, staleAfterSeconds int) map[string]int {
	totals := map[string]int{
		"targets":                            len(items),
		"enabled":                            0,
		"ready":                              0,
		"warning":                            0,
		"blocked":                            0,
		"muted":                              0,
		"receiver_connected":                 0,
		"group_memory_pending":               0,
		"rag_ingest_pending":                 0,
		"high_pressure":                      0,
		"memory_checkpoints":                 0,
		"rag_checkpoints":                    0,
		"rag_datasets":                       0,
		"rag_dataset_ingest_snapshots":       0,
		"configured_rag_datasets":            0,
		"configured_rag_dataset_not_started": 0,
		"rag_dataset_warning":                0,
		"rag_dataset_blocked":                0,
		"lagging":                            0,
		"stale_checkpoints":                  0,
		"expired_active_leases":              0,
		"stale_active_leases":                0,
		"stalled":                            0,
		"stagnant":                           0,
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
		if item.GroupMemory.ExpiredActiveLeases > 0 || item.RagIngest.ExpiredActiveLeases > 0 {
			totals["expired_active_leases"]++
		}
		if item.GroupMemory.StaleActiveLeases > 0 || item.RagIngest.StaleActiveLeases > 0 {
			totals["stale_active_leases"]++
		}
		if item.MemoryCheckpoint != nil {
			totals["memory_checkpoints"]++
		}
		totals["rag_checkpoints"] += len(item.RagCheckpoints)
		totals["rag_datasets"] += len(item.RagDatasets)
		for _, dataset := range item.RagDatasets {
			if dataset.IngestSnapshot != nil {
				totals["rag_dataset_ingest_snapshots"]++
			}
			if dataset.Configured {
				totals["configured_rag_datasets"]++
			}
			if knowledgePipelineContainsReason(dataset.Reasons, "configured_dataset_not_started") {
				totals["configured_rag_dataset_not_started"]++
			}
			switch dataset.Status {
			case "warn":
				totals["rag_dataset_warning"]++
			case "blocked":
				totals["rag_dataset_blocked"]++
			}
		}
		if (item.MemoryCheckpointLag != nil && (item.MemoryCheckpointLag.Status == "warn" || item.MemoryCheckpointLag.Status == "danger")) ||
			(item.RagCheckpointLagMax != nil && (item.RagCheckpointLagMax.Status == "warn" || item.RagCheckpointLagMax.Status == "danger")) {
			totals["lagging"]++
		}
		if knowledgePipelineCheckpointIsStale(item.MemoryCheckpointLag, staleAfterSeconds) ||
			knowledgePipelineCheckpointIsStale(item.RagCheckpointLagMax, staleAfterSeconds) {
			totals["stale_checkpoints"]++
		}
		if knowledgePipelineContainsReason(item.Reasons, "memory_checkpoint_stalled_under_pressure") ||
			knowledgePipelineContainsReason(item.Reasons, "rag_checkpoint_stalled_under_pressure") {
			totals["stalled"]++
		}
		if knowledgePipelineContainsReason(item.Reasons, "memory_checkpoint_stagnant") ||
			knowledgePipelineContainsReason(item.Reasons, "rag_checkpoint_stagnant") {
			totals["stagnant"]++
		}
	}
	return totals
}

func parseKnowledgePipelineCheckpointTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
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
