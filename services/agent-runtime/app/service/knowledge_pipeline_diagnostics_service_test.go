package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestKnowledgePipelineDiagnosticsServiceReportsReadyPipeline(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	agentWorkers := NewAgentWorkerStatusService()
	syncObserveTarget(t, observeTargets, "27234224")
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessageForGroupWithSeq(t, store, "27234224", "msg-ready-text", "ready text", nil, 62)
	ingestObserveMessageForGroupWithSeq(t, store, "27234224", "msg-ready-image", "ready image", []command.AttachmentCommand{{
		ID:       "asset:ready:image:1",
		Kind:     "image",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/ready-image.png",
		MimeType: "image/png",
		Name:     "ready-image.png",
	}}, 63)
	ingestObserveMessageForGroupWithSeq(t, store, "27234224", "msg-ready-file", "ready file", []command.AttachmentCommand{{
		ID:       "asset:ready:file:1",
		Kind:     "file",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/ready-file.txt",
		MimeType: "text/plain",
		Name:     "ready-file.txt",
	}}, 64)

	if _, err := agentWorkers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:        "knowledge-worker-main",
		InstanceID:      "instance-1",
		WorkerType:      "knowledge",
		Status:          "idle",
		LeaseTTLSeconds: 120,
		Timestamp:       now,
		Source:          "test",
	}); err != nil {
		t.Fatalf("report knowledge worker: %v", err)
	}

	checkpoints := NewKnowledgeCheckpointService(store)
	if _, err := checkpoints.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "memory:qq:27234224",
		Cursor:       64,
		Metadata:     map[string]string{"group_id": "27234224"},
		Timestamp:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("upsert memory checkpoint: %v", err)
	}
	if _, err := checkpoints.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "ragflow:qq:27234224:ds-main",
		Cursor:       64,
		Metadata: map[string]string{
			"group_id":             "27234224",
			"dataset_id":           "ds-main",
			"display_name":         "qq_group_27234224_seq62_64.txt",
			"last_message_count":   "3",
			"last_document_count":  "1",
			"last_start_seq":       "62",
			"last_end_seq":         "64",
			"last_parse_requested": "true",
		},
		Timestamp: now.Add(-2 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert rag checkpoint: %v", err)
	}

	service := newKnowledgePipelineDiagnosticsServiceForTest(store, observeTargets, receiverStatuses, agentWorkers, map[string]bool{
		"asset:ready:image:1": true,
		"asset:ready:file:1":  true,
	})
	view, err := service.GetKnowledgePipelineDiagnostics(ctx, query.KnowledgePipelineDiagnosticsFilter{
		Limit:             50,
		StaleAfterSeconds: 300,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("knowledge pipeline diagnostics: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["ready"] != 1 || view.Totals["blocked"] != 0 {
		t.Fatalf("unexpected totals: %#v", view.Totals)
	}
	pipeline := view.Pipelines[0]
	if pipeline.Status != "ok" || pipeline.CaptureStatus != "ok" {
		t.Fatalf("unexpected ready pipeline: %#v", pipeline)
	}
	if pipeline.MemoryCheckpoint == nil || pipeline.MemoryCheckpoint.CheckpointID != "memory:qq:27234224" {
		t.Fatalf("unexpected memory checkpoint: %#v", pipeline.MemoryCheckpoint)
	}
	if len(pipeline.RagCheckpoints) != 1 || pipeline.RagCheckpoints[0].CheckpointID != "ragflow:qq:27234224:ds-main" {
		t.Fatalf("unexpected rag checkpoints: %#v", pipeline.RagCheckpoints)
	}
	if len(pipeline.RagDatasets) != 1 || pipeline.RagDatasets[0].DatasetID != "ds-main" || pipeline.RagDatasets[0].Status != "ok" {
		t.Fatalf("unexpected rag datasets: %#v", pipeline.RagDatasets)
	}
	if pipeline.RagDatasets[0].IngestSnapshot == nil ||
		pipeline.RagDatasets[0].IngestSnapshot.MessageCount != 3 ||
		pipeline.RagDatasets[0].IngestSnapshot.DocumentCount != 1 ||
		pipeline.RagDatasets[0].IngestSnapshot.StartSeq != 62 ||
		pipeline.RagDatasets[0].IngestSnapshot.EndSeq != 64 ||
		!pipeline.RagDatasets[0].IngestSnapshot.ParseRequested {
		t.Fatalf("unexpected rag ingest snapshot: %#v", pipeline.RagDatasets[0].IngestSnapshot)
	}
	if len(pipeline.WorkerCoverage) != 2 || pipeline.WorkerCoverage[0].CoverageStatus != "ok" || pipeline.WorkerCoverage[1].CoverageStatus != "ok" {
		t.Fatalf("unexpected worker coverage: %#v", pipeline.WorkerCoverage)
	}
	if pipeline.GroupMemory.FreshnessStatus != "muted" || pipeline.GroupMemory.FreshnessReason != "" {
		t.Fatalf("unexpected group memory freshness: %#v", pipeline.GroupMemory)
	}
	if pipeline.RagIngest.FreshnessStatus != "muted" || pipeline.RagIngest.FreshnessReason != "" {
		t.Fatalf("unexpected rag ingest freshness: %#v", pipeline.RagIngest)
	}
	if !pipeline.SourceSeqKnown || pipeline.LatestSourceSeq != 64 || pipeline.SequencedEvents != 3 {
		t.Fatalf("unexpected source seq state: %#v", pipeline)
	}
	if pipeline.MemoryCheckpointLag == nil || pipeline.MemoryCheckpointLag.Lag != 0 || pipeline.MemoryCheckpointLag.Status != "ok" {
		t.Fatalf("unexpected memory lag: %#v", pipeline.MemoryCheckpointLag)
	}
	if pipeline.MemoryCheckpointLag.AgeSeconds != 60 {
		t.Fatalf("unexpected memory checkpoint age: %#v", pipeline.MemoryCheckpointLag)
	}
	if pipeline.RagCheckpointLagMax == nil || pipeline.RagCheckpointLagMax.Lag != 0 || pipeline.RagCheckpointLagMax.Status != "ok" {
		t.Fatalf("unexpected rag lag: %#v", pipeline.RagCheckpointLagMax)
	}
	if pipeline.RagCheckpointLagMax.AgeSeconds != 120 {
		t.Fatalf("unexpected rag checkpoint age: %#v", pipeline.RagCheckpointLagMax)
	}
}

func TestKnowledgePipelineDiagnosticsServiceShowsConfiguredDatasetBeforeRuntimeObservation(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	agentWorkers := NewAgentWorkerStatusService()
	syncObserveTargetWithMetadata(t, observeTargets, "27234224", map[string]string{
		"ragflow_dataset_ids":   "ds-configured,ds-configured",
		"ragflow_dataset_count": "1",
	})
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessageForGroupWithSeq(t, store, "27234224", "msg-configured-text", "configured dataset only", nil, 12)

	if _, err := agentWorkers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:        "knowledge-worker-main",
		InstanceID:      "instance-1",
		WorkerType:      "knowledge",
		Status:          "idle",
		LeaseTTLSeconds: 120,
		Timestamp:       now,
		Source:          "test",
	}); err != nil {
		t.Fatalf("report knowledge worker: %v", err)
	}

	service := newKnowledgePipelineDiagnosticsServiceForTest(store, observeTargets, receiverStatuses, agentWorkers, nil)
	view, err := service.GetKnowledgePipelineDiagnostics(ctx, query.KnowledgePipelineDiagnosticsFilter{
		Limit:             50,
		StaleAfterSeconds: 300,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("knowledge pipeline diagnostics: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["rag_datasets"] != 1 || view.Totals["configured_rag_datasets"] != 1 || view.Totals["configured_rag_dataset_not_started"] != 1 {
		t.Fatalf("unexpected configured dataset totals: %#v", view.Totals)
	}
	pipeline := view.Pipelines[0]
	if len(pipeline.RagDatasets) != 1 {
		t.Fatalf("expected one configured dataset, got %#v", pipeline.RagDatasets)
	}
	dataset := pipeline.RagDatasets[0]
	if dataset.DatasetID != "ds-configured" || !dataset.Configured || dataset.RuntimeObserved || dataset.Status != "muted" || !containsString(dataset.Reasons, "configured_dataset_not_started") {
		t.Fatalf("unexpected configured dataset view: %#v", dataset)
	}
	if dataset.Checkpoint != nil || dataset.CheckpointLag != nil {
		t.Fatalf("configured-only dataset should not have checkpoint state: %#v", dataset)
	}
	if dataset.JobStage.JobType != "rag_ingest" || dataset.JobStage.FreshnessStatus != "muted" {
		t.Fatalf("unexpected configured-only dataset job stage: %#v", dataset.JobStage)
	}
}

func TestKnowledgePipelineDiagnosticsServiceBlocksCaptureFailure(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	agentWorkers := NewAgentWorkerStatusService()
	syncObserveTarget(t, observeTargets, "3219982")

	service := newKnowledgePipelineDiagnosticsServiceForTest(store, observeTargets, receiverStatuses, agentWorkers, nil)
	view, err := service.GetKnowledgePipelineDiagnostics(ctx, query.KnowledgePipelineDiagnosticsFilter{
		Limit:             50,
		StaleAfterSeconds: 300,
	})
	if err != nil {
		t.Fatalf("knowledge pipeline diagnostics: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["blocked"] != 1 {
		t.Fatalf("unexpected capture-blocked totals: %#v", view.Totals)
	}
	pipeline := view.Pipelines[0]
	if pipeline.Status != "blocked" || pipeline.CaptureStatus != "danger" || !containsString(pipeline.Reasons, "capture_blocked") {
		t.Fatalf("unexpected capture-blocked pipeline: %#v", pipeline)
	}
}

func TestKnowledgePipelineDiagnosticsServiceWarnsOnCheckpointLag(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	agentWorkers := NewAgentWorkerStatusService()
	syncObserveTarget(t, observeTargets, "3219982")
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessageForGroupWithSeq(t, store, "3219982", "msg-lag-text", "lag text", nil, 50)
	ingestObserveMessageForGroupWithSeq(t, store, "3219982", "msg-lag-image", "lag image", []command.AttachmentCommand{{
		ID:       "asset:lag:image:1",
		Kind:     "image",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/lag-image.png",
		MimeType: "image/png",
		Name:     "lag-image.png",
	}}, 51)
	ingestObserveMessageForGroupWithSeq(t, store, "3219982", "msg-lag-file", "lag file", []command.AttachmentCommand{{
		ID:       "asset:lag:file:1",
		Kind:     "file",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/lag-file.txt",
		MimeType: "text/plain",
		Name:     "lag-file.txt",
	}}, 52)

	if _, err := agentWorkers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:        "knowledge-worker-main",
		InstanceID:      "instance-1",
		WorkerType:      "knowledge",
		Status:          "idle",
		LeaseTTLSeconds: 120,
		Timestamp:       now,
		Source:          "test",
	}); err != nil {
		t.Fatalf("report knowledge worker: %v", err)
	}

	checkpoints := NewKnowledgeCheckpointService(store)
	if _, err := checkpoints.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "memory:qq:3219982",
		Cursor:       25,
		Metadata:     map[string]string{"group_id": "3219982"},
		Timestamp:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("upsert memory checkpoint: %v", err)
	}

	service := newKnowledgePipelineDiagnosticsServiceForTest(store, observeTargets, receiverStatuses, agentWorkers, map[string]bool{
		"asset:lag:image:1": true,
		"asset:lag:file:1":  true,
	})
	view, err := service.GetKnowledgePipelineDiagnostics(ctx, query.KnowledgePipelineDiagnosticsFilter{
		Limit:             50,
		StaleAfterSeconds: 300,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("knowledge pipeline diagnostics: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["warning"] != 1 || view.Totals["lagging"] != 1 || view.Totals["stalled"] != 0 {
		t.Fatalf("unexpected lagging totals: %#v", view.Totals)
	}
	pipeline := view.Pipelines[0]
	if pipeline.Status != "warn" || !containsString(pipeline.Reasons, "memory_checkpoint_lagging") {
		t.Fatalf("unexpected lagging pipeline: %#v", pipeline)
	}
	if pipeline.MemoryCheckpointLag == nil || pipeline.MemoryCheckpointLag.Lag != 27 || pipeline.MemoryCheckpointLag.Status != "warn" {
		t.Fatalf("unexpected memory lag state: %#v", pipeline.MemoryCheckpointLag)
	}
	if pipeline.MemoryCheckpointLag.AgeSeconds != 60 {
		t.Fatalf("unexpected memory lag age: %#v", pipeline.MemoryCheckpointLag)
	}
}

func TestKnowledgePipelineDiagnosticsServiceMarksCheckpointStagnant(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	agentWorkers := NewAgentWorkerStatusService()
	syncObserveTarget(t, observeTargets, "164369633")
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessageForGroupWithSeq(t, store, "164369633", "msg-stagnant-text", "stagnant text", nil, 80)
	ingestObserveMessageForGroupWithSeq(t, store, "164369633", "msg-stagnant-image", "stagnant image", []command.AttachmentCommand{{
		ID:       "asset:stagnant:image:1",
		Kind:     "image",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/stagnant-image.png",
		MimeType: "image/png",
		Name:     "stagnant-image.png",
	}}, 81)
	ingestObserveMessageForGroupWithSeq(t, store, "164369633", "msg-stagnant-file", "stagnant file", []command.AttachmentCommand{{
		ID:       "asset:stagnant:file:1",
		Kind:     "file",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/stagnant-file.txt",
		MimeType: "text/plain",
		Name:     "stagnant-file.txt",
	}}, 82)

	if _, err := agentWorkers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:        "knowledge-worker-main",
		InstanceID:      "instance-1",
		WorkerType:      "knowledge",
		Status:          "idle",
		LeaseTTLSeconds: 120,
		Timestamp:       now,
		Source:          "test",
	}); err != nil {
		t.Fatalf("report knowledge worker: %v", err)
	}

	checkpoints := NewKnowledgeCheckpointService(store)
	if _, err := checkpoints.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "memory:qq:164369633",
		Cursor:       55,
		Metadata:     map[string]string{"group_id": "164369633"},
		Timestamp:    now.Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert stagnant memory checkpoint: %v", err)
	}

	service := newKnowledgePipelineDiagnosticsServiceForTest(store, observeTargets, receiverStatuses, agentWorkers, map[string]bool{
		"asset:stagnant:image:1": true,
		"asset:stagnant:file:1":  true,
	})
	view, err := service.GetKnowledgePipelineDiagnostics(ctx, query.KnowledgePipelineDiagnosticsFilter{
		Limit:             50,
		StaleAfterSeconds: 300,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("knowledge pipeline diagnostics: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["warning"] != 1 || view.Totals["lagging"] != 1 || view.Totals["stale_checkpoints"] != 1 || view.Totals["stagnant"] != 1 || view.Totals["stalled"] != 0 {
		t.Fatalf("unexpected stagnant totals: %#v", view.Totals)
	}
	pipeline := view.Pipelines[0]
	if pipeline.Status != "warn" || !containsString(pipeline.Reasons, "memory_checkpoint_stagnant") {
		t.Fatalf("unexpected stagnant pipeline: %#v", pipeline)
	}
	if pipeline.MemoryCheckpointLag == nil || pipeline.MemoryCheckpointLag.Lag != 27 || pipeline.MemoryCheckpointLag.Status != "warn" || pipeline.MemoryCheckpointLag.AgeSeconds != 600 {
		t.Fatalf("unexpected stagnant lag state: %#v", pipeline.MemoryCheckpointLag)
	}
}

func TestKnowledgePipelineDiagnosticsServiceWarnsOnStaleActiveLease(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	agentWorkers := NewAgentWorkerStatusService()
	syncObserveTarget(t, observeTargets, "27234224")
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessageForGroupWithSeq(t, store, "27234224", "msg-stale-lease", "stale lease", nil, 32)

	if _, err := agentWorkers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:        "knowledge-worker-main",
		InstanceID:      "instance-1",
		WorkerType:      "knowledge",
		Status:          "running",
		LeaseTTLSeconds: 120,
		Timestamp:       now,
		Source:          "test",
	}); err != nil {
		t.Fatalf("report knowledge worker: %v", err)
	}

	jobs := NewAgentJobService(store)
	created, err := jobs.Create(ctx, knowledgePipelineJobCommand(
		"group_memory_extract:qq:27234224:stale",
		"group_memory_extract",
		"27234224",
		map[string]string{"group_id": "27234224"},
		now.Add(-20*time.Minute),
	))
	if err != nil {
		t.Fatalf("create stale lease job: %v", err)
	}
	leased, err := jobs.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      created.JobID,
		WorkerID:   "knowledge-worker",
		TTLSeconds: 30 * 60,
		LeaseToken: "lease-stale",
		Timestamp:  now.Add(-10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("lease stale job: %v", err)
	}
	if _, err := jobs.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      leased.JobID,
		LeaseToken: leased.LeaseToken,
		Timestamp:  now.Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("mark stale job running: %v", err)
	}

	service := newKnowledgePipelineDiagnosticsServiceForTest(store, observeTargets, receiverStatuses, agentWorkers, nil)
	view, err := service.GetKnowledgePipelineDiagnostics(ctx, query.KnowledgePipelineDiagnosticsFilter{
		Limit:             50,
		StaleAfterSeconds: 300,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("knowledge pipeline diagnostics: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["warning"] != 1 || view.Totals["stale_active_leases"] != 1 || view.Totals["expired_active_leases"] != 0 {
		t.Fatalf("unexpected stale lease totals: %#v", view.Totals)
	}
	pipeline := view.Pipelines[0]
	if pipeline.Status != "warn" || !containsString(pipeline.Reasons, "group_memory_lease_stale") {
		t.Fatalf("unexpected stale lease pipeline: %#v", pipeline)
	}
	if pipeline.GroupMemory.FreshnessStatus != "warn" || pipeline.GroupMemory.FreshnessReason != "stale_active_lease" || pipeline.GroupMemory.StaleActiveLeases != 1 || pipeline.GroupMemory.OldestActiveAgeSeconds != 600 {
		t.Fatalf("unexpected stale lease stage: %#v", pipeline.GroupMemory)
	}
}

func TestKnowledgePipelineDiagnosticsServiceBlocksExpiredActiveLease(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	agentWorkers := NewAgentWorkerStatusService()
	syncObserveTarget(t, observeTargets, "3219982")
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessageForGroupWithSeq(t, store, "3219982", "msg-expired-lease", "expired lease", nil, 48)

	if _, err := agentWorkers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:        "knowledge-worker-main",
		InstanceID:      "instance-1",
		WorkerType:      "knowledge",
		Status:          "running",
		LeaseTTLSeconds: 120,
		Timestamp:       now,
		Source:          "test",
	}); err != nil {
		t.Fatalf("report knowledge worker: %v", err)
	}

	jobs := NewAgentJobService(store)
	created, err := jobs.Create(ctx, knowledgePipelineJobCommand(
		"rag_ingest:qq:3219982:expired",
		"rag_ingest",
		"3219982",
		map[string]string{"group_id": "3219982", "dataset_id": "expired"},
		now.Add(-20*time.Minute),
	))
	if err != nil {
		t.Fatalf("create expired lease job: %v", err)
	}
	leased, err := jobs.Lease(ctx, command.AgentJobLeaseCommand{
		JobID:      created.JobID,
		WorkerID:   "knowledge-worker",
		TTLSeconds: 60,
		LeaseToken: "lease-expired",
		Timestamp:  now.Add(-10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("lease expired job: %v", err)
	}
	if _, err := jobs.MarkRunning(ctx, command.MarkAgentJobRunningCommand{
		JobID:      leased.JobID,
		LeaseToken: leased.LeaseToken,
		Timestamp:  now.Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("mark expired job running: %v", err)
	}

	service := newKnowledgePipelineDiagnosticsServiceForTest(store, observeTargets, receiverStatuses, agentWorkers, nil)
	view, err := service.GetKnowledgePipelineDiagnostics(ctx, query.KnowledgePipelineDiagnosticsFilter{
		Limit:             50,
		StaleAfterSeconds: 300,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("knowledge pipeline diagnostics: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["blocked"] != 1 || view.Totals["expired_active_leases"] != 1 {
		t.Fatalf("unexpected expired lease totals: %#v", view.Totals)
	}
	pipeline := view.Pipelines[0]
	if pipeline.Status != "blocked" || !containsString(pipeline.Reasons, "rag_ingest_lease_expired") {
		t.Fatalf("unexpected expired lease pipeline: %#v", pipeline)
	}
	if pipeline.RagIngest.FreshnessStatus != "danger" || pipeline.RagIngest.FreshnessReason != "expired_active_lease" || pipeline.RagIngest.ExpiredActiveLeases != 1 {
		t.Fatalf("unexpected expired lease stage: %#v", pipeline.RagIngest)
	}
	if len(pipeline.RagDatasets) != 1 || pipeline.RagDatasets[0].DatasetID != "expired" || pipeline.RagDatasets[0].Status != "blocked" || !containsString(pipeline.RagDatasets[0].Reasons, "rag_ingest_lease_expired") {
		t.Fatalf("unexpected expired rag dataset: %#v", pipeline.RagDatasets)
	}
}

func TestKnowledgePipelineDiagnosticsServiceBlocksHighPressureCheckpointStall(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	observeTargets := NewObserveTargetService()
	receiverStatuses := NewReceiverStatusService()
	agentWorkers := NewAgentWorkerStatusService()
	syncObserveTarget(t, observeTargets, "3219982")
	reportQQReceiver(t, receiverStatuses, "1049511700", "connected")
	ingestObserveMessageForGroupWithSeq(t, store, "3219982", "msg-blocked-text", "blocked text", nil, 118)
	ingestObserveMessageForGroupWithSeq(t, store, "3219982", "msg-blocked-image", "blocked image", []command.AttachmentCommand{{
		ID:       "asset:blocked:image:1",
		Kind:     "image",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/blocked-image.png",
		MimeType: "image/png",
		Name:     "blocked-image.png",
	}}, 119)
	ingestObserveMessageForGroupWithSeq(t, store, "3219982", "msg-blocked-file", "blocked file", []command.AttachmentCommand{{
		ID:       "asset:blocked:file:1",
		Kind:     "file",
		URL:      "E:/agent/akashic/.akashic-workspace/uploads/blocked-file.txt",
		MimeType: "text/plain",
		Name:     "blocked-file.txt",
	}}, 120)

	if _, err := agentWorkers.ReportAgentWorkerStatus(ctx, command.ReportAgentWorkerStatusCommand{
		WorkerID:        "knowledge-worker-main",
		InstanceID:      "instance-1",
		WorkerType:      "knowledge",
		Status:          "running",
		LeaseTTLSeconds: 120,
		Timestamp:       now,
		Source:          "test",
	}); err != nil {
		t.Fatalf("report knowledge worker: %v", err)
	}

	jobs := NewAgentJobService(store)
	for index := 0; index < 10; index++ {
		if _, err := jobs.Create(ctx, knowledgePipelineJobCommand(
			"rag_ingest:qq:3219982:blocked:"+string(rune('a'+index)),
			"rag_ingest",
			"3219982",
			map[string]string{"group_id": "3219982", "dataset_id": "ds-stalled"},
			now.Add(-10*time.Minute).Add(time.Duration(index)*time.Second),
		)); err != nil {
			t.Fatalf("create blocked rag ingest job %d: %v", index, err)
		}
	}
	checkpoints := NewKnowledgeCheckpointService(store)
	if _, err := checkpoints.Upsert(ctx, command.UpsertKnowledgeCheckpointCommand{
		CheckpointID: "ragflow:qq:3219982:ds-stalled",
		Cursor:       0,
		Metadata:     map[string]string{"group_id": "3219982", "dataset_id": "ds-stalled"},
		Timestamp:    now.Add(-5 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert stalled rag checkpoint: %v", err)
	}

	service := newKnowledgePipelineDiagnosticsServiceForTest(store, observeTargets, receiverStatuses, agentWorkers, map[string]bool{
		"asset:blocked:image:1": true,
		"asset:blocked:file:1":  true,
	})
	view, err := service.GetKnowledgePipelineDiagnostics(ctx, query.KnowledgePipelineDiagnosticsFilter{
		Limit:             50,
		StaleAfterSeconds: 300,
		Now:               now,
	})
	if err != nil {
		t.Fatalf("knowledge pipeline diagnostics: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["blocked"] != 1 || view.Totals["high_pressure"] != 1 || view.Totals["lagging"] != 1 || view.Totals["stale_checkpoints"] != 1 || view.Totals["stalled"] != 1 || view.Totals["stagnant"] != 0 || view.Totals["rag_datasets"] != 1 || view.Totals["rag_dataset_blocked"] != 1 {
		t.Fatalf("unexpected high-pressure totals: %#v", view.Totals)
	}
	pipeline := view.Pipelines[0]
	if pipeline.Status != "blocked" || pipeline.RagIngest.Pending != 10 || !pipeline.RagIngest.HighPressure {
		t.Fatalf("unexpected high-pressure pipeline: %#v", pipeline)
	}
	if len(pipeline.WorkerCoverage) != 2 || pipeline.WorkerCoverage[1].JobType != "rag_ingest" || pipeline.WorkerCoverage[1].CoverageStatus != "ok" {
		t.Fatalf("unexpected worker coverage: %#v", pipeline.WorkerCoverage)
	}
	if pipeline.RagCheckpointLagMax == nil || pipeline.RagCheckpointLagMax.Lag != 120 || pipeline.RagCheckpointLagMax.Status != "danger" {
		t.Fatalf("unexpected rag checkpoint lag: %#v", pipeline.RagCheckpointLagMax)
	}
	if pipeline.RagCheckpointLagMax.AgeSeconds != 300 {
		t.Fatalf("unexpected rag checkpoint lag age: %#v", pipeline.RagCheckpointLagMax)
	}
	if len(pipeline.RagDatasets) != 1 || pipeline.RagDatasets[0].DatasetID != "ds-stalled" || pipeline.RagDatasets[0].Status != "blocked" || !containsString(pipeline.RagDatasets[0].Reasons, "rag_checkpoint_stalled_under_pressure") {
		t.Fatalf("unexpected blocked rag dataset: %#v", pipeline.RagDatasets)
	}
	if !containsString(pipeline.Reasons, "rag_checkpoint_stalled_under_pressure") {
		t.Fatalf("expected rag_checkpoint_stalled_under_pressure reason: %#v", pipeline.Reasons)
	}
}

func newKnowledgePipelineDiagnosticsServiceForTest(
	store *memory.Store,
	observeTargets *ObserveTargetService,
	receiverStatuses *ReceiverStatusService,
	agentWorkers *AgentWorkerStatusService,
	ready map[string]bool,
) *KnowledgePipelineDiagnosticsService {
	capture := NewObserveCaptureDiagnosticsService(
		observeTargets,
		receiverStatuses,
		store,
		store,
		fakeObserveCaptureContentReader{ready: ready},
	)
	return NewKnowledgePipelineDiagnosticsService(
		observeTargets,
		capture,
		agentWorkers,
		store,
		store,
		store,
	)
}

func ingestObserveMessageForGroup(t *testing.T, store *memory.Store, groupID string, suffix string, content string, attachments []command.AttachmentCommand) {
	t.Helper()
	ingestObserveMessageForGroupWithSeq(t, store, groupID, suffix, content, attachments, 0)
}

func ingestObserveMessageForGroupWithSeq(t *testing.T, store *memory.Store, groupID string, suffix string, content string, attachments []command.AttachmentCommand, seq int) {
	t.Helper()
	ingestor := NewMessageIngestServiceWithRuntimeStores(
		store,
		store,
		store,
		store,
		domainservice.NewProvenanceClassifier([]string{"1049511700"}),
		domainservice.NewLoopGuard([]string{"1049511700"}, 15*time.Second, 6),
		store,
		store,
	)
	metadata := map[string]string{
		"observe_only": "true",
		"session_key":  "qq:gqq:" + groupID,
	}
	if seq > 0 {
		metadata["seq"] = strconv.Itoa(seq)
	}
	_, err := ingestor.ShadowIngest(context.Background(), command.IngestMessageCommand{
		EventID: "qq:1049511700:group:" + groupID + ":" + suffix,
		Channel: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   groupID,
			ConversationType: "group",
		},
		Sender: command.SenderCommand{
			ID:   "2948770636",
			Kind: "human",
		},
		Content:     content,
		Attachments: attachments,
		Timestamp:   time.Now().UTC(),
		Metadata:    metadata,
	})
	if err != nil {
		t.Fatalf("ingest observe message for group %s: %v", groupID, err)
	}
}

func knowledgePipelineJobCommand(jobID string, jobType string, groupID string, payload map[string]string, timestamp time.Time) command.CreateAgentJobCommand {
	return command.CreateAgentJobCommand{
		JobID:   jobID,
		JobType: jobType,
		AgentID: "knowledge-worker",
		Route: command.ChannelCommand{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   groupID,
			ConversationType: "group",
		},
		Payload:     payload,
		MaxAttempts: 2,
		Timestamp:   timestamp,
	}
}
