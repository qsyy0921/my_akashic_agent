package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type knowledgeJobPlannerObserveTargetLister interface {
	ListObserveTargets(ctx context.Context) (query.ObserveTargetsView, error)
}

type knowledgeJobPlannerAgentJobCreator interface {
	Create(ctx context.Context, cmd command.CreateAgentJobCommand) (query.AgentJobView, error)
}

type KnowledgeJobPlannerService struct {
	observeTargets knowledgeJobPlannerObserveTargetLister
	agentJobs      knowledgeJobPlannerAgentJobCreator
	clock          func() time.Time
}

func NewKnowledgeJobPlannerService(
	observeTargets knowledgeJobPlannerObserveTargetLister,
	agentJobs knowledgeJobPlannerAgentJobCreator,
) *KnowledgeJobPlannerService {
	return &KnowledgeJobPlannerService{
		observeTargets: observeTargets,
		agentJobs:      agentJobs,
		clock:          time.Now,
	}
}

func (s *KnowledgeJobPlannerService) PlanKnowledgeJobs(
	ctx context.Context,
	cmd command.PlanKnowledgeJobsCommand,
) (query.KnowledgeJobPlannerRunView, error) {
	if err := ctx.Err(); err != nil {
		return query.KnowledgeJobPlannerRunView{}, err
	}
	if s == nil || s.observeTargets == nil || s.agentJobs == nil {
		return query.KnowledgeJobPlannerRunView{}, errors.New("knowledge job planner requires observe targets and agent jobs")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = s.now()
	}
	now = now.UTC()
	intervalSeconds := cmd.IntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	bucket := now.Unix() / int64(intervalSeconds)
	agentID := strings.TrimSpace(cmd.AgentID)
	if agentID == "" {
		agentID = "akashic-python-worker"
	}
	plannerID := strings.TrimSpace(cmd.PlannerID)
	if plannerID == "" {
		plannerID = "agent-runtime-knowledge-job-planner"
	}
	maxAttempts := cmd.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 2
	}
	ragMaxMessages := cmd.RagMaxMessages
	if ragMaxMessages <= 0 {
		ragMaxMessages = 1000
	}

	targets, err := s.observeTargets.ListObserveTargets(ctx)
	if err != nil {
		return query.KnowledgeJobPlannerRunView{}, err
	}
	view := query.KnowledgeJobPlannerRunView{
		Timestamp:  now.Format(time.RFC3339Nano),
		Bucket:     bucket,
		SideEffect: "agent_job_create",
	}
	groups := make(map[string]struct{})
	for _, target := range targets.Targets {
		if !knowledgeJobPlannerTargetEligible(target) {
			view.SkippedTargets++
			continue
		}
		view.Targets++
		groupID := strings.TrimSpace(target.Channel.ConversationID)
		groups[groupID] = struct{}{}
		if suppressed, err := s.createGroupMemoryJob(ctx, target, agentID, plannerID, maxAttempts, bucket, now); err != nil {
			return query.KnowledgeJobPlannerRunView{}, err
		} else {
			view.CreatedOrExisting++
			view.GroupMemoryJobs++
			if suppressed {
				view.SuppressedByDedupe++
			}
		}
		for _, datasetID := range splitCommaSeparatedList(target.Metadata["ragflow_dataset_ids"]) {
			if suppressed, err := s.createRagIngestJob(ctx, target, agentID, plannerID, maxAttempts, ragMaxMessages, cmd.RagParse, bucket, now, datasetID); err != nil {
				return query.KnowledgeJobPlannerRunView{}, err
			} else {
				view.CreatedOrExisting++
				view.RagIngestJobs++
				if suppressed {
					view.SuppressedByDedupe++
				}
			}
		}
	}
	view.Groups = sortedKnowledgePlannerGroups(groups)
	return view, nil
}

func (s *KnowledgeJobPlannerService) createGroupMemoryJob(
	ctx context.Context,
	target query.ObserveTargetView,
	agentID string,
	plannerID string,
	maxAttempts int,
	bucket int64,
	now time.Time,
) (bool, error) {
	accountID := strings.TrimSpace(target.Channel.AccountID)
	groupID := strings.TrimSpace(target.Channel.ConversationID)
	jobID := fmt.Sprintf("group_memory_extract:qq:%s:%d", groupID, bucket)
	dedupeKey := fmt.Sprintf("knowledge:group_memory_extract:qq:%s:%s", accountID, groupID)
	job, err := s.agentJobs.Create(ctx, command.CreateAgentJobCommand{
		JobID:       jobID,
		JobType:     string(model.AgentJobGroupMemoryExtract),
		AgentID:     agentID,
		Route:       knowledgeJobPlannerRoute(target),
		Payload:     knowledgeJobPlannerGroupMemoryPayload(groupID),
		DedupeKey:   dedupeKey,
		MaxAttempts: maxAttempts,
		Timestamp:   now,
		Metadata: map[string]string{
			"dedupe_key": dedupeKey,
			"scheduler":  plannerID,
		},
	})
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(job.JobID) != jobID, nil
}

func (s *KnowledgeJobPlannerService) createRagIngestJob(
	ctx context.Context,
	target query.ObserveTargetView,
	agentID string,
	plannerID string,
	maxAttempts int,
	maxMessages int,
	parse bool,
	bucket int64,
	now time.Time,
	datasetID string,
) (bool, error) {
	accountID := strings.TrimSpace(target.Channel.AccountID)
	groupID := strings.TrimSpace(target.Channel.ConversationID)
	datasetID = strings.TrimSpace(datasetID)
	jobID := fmt.Sprintf("rag_ingest:qq:%s:%s:%d", groupID, datasetID, bucket)
	dedupeKey := fmt.Sprintf("knowledge:rag_ingest:qq:%s:%s:%s", accountID, groupID, datasetID)
	job, err := s.agentJobs.Create(ctx, command.CreateAgentJobCommand{
		JobID:       jobID,
		JobType:     string(model.AgentJobRagIngest),
		AgentID:     agentID,
		Route:       knowledgeJobPlannerRoute(target),
		Payload:     knowledgeJobPlannerRagPayload(groupID, datasetID, maxMessages, parse),
		DedupeKey:   dedupeKey,
		MaxAttempts: maxAttempts,
		Timestamp:   now,
		Metadata: map[string]string{
			"dedupe_key": dedupeKey,
			"dataset_id": datasetID,
			"scheduler":  plannerID,
		},
	})
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(job.JobID) != jobID, nil
}

func knowledgeJobPlannerTargetEligible(target query.ObserveTargetView) bool {
	return target.Enabled &&
		target.ObserveOnly &&
		strings.TrimSpace(target.Channel.Kind) == string(model.ChannelKindQQ) &&
		strings.TrimSpace(target.Channel.ConversationType) == string(model.ConversationTypeGroup) &&
		strings.TrimSpace(target.Channel.AccountID) != "" &&
		strings.TrimSpace(target.Channel.ConversationID) != ""
}

func knowledgeJobPlannerRoute(target query.ObserveTargetView) command.ChannelCommand {
	return command.ChannelCommand{
		Kind:             strings.TrimSpace(target.Channel.Kind),
		AccountID:        strings.TrimSpace(target.Channel.AccountID),
		ConversationID:   strings.TrimSpace(target.Channel.ConversationID),
		ConversationType: strings.TrimSpace(target.Channel.ConversationType),
	}
}

func knowledgeJobPlannerGroupMemoryPayload(groupID string) map[string]string {
	return map[string]string{
		"group_id":     strings.TrimSpace(groupID),
		"session_key":  "qq:gqq:" + strings.TrimSpace(groupID),
		"observe_only": "true",
	}
}

func knowledgeJobPlannerRagPayload(groupID string, datasetID string, maxMessages int, parse bool) map[string]string {
	return map[string]string{
		"group_id":     strings.TrimSpace(groupID),
		"session_key":  "qq:gqq:" + strings.TrimSpace(groupID),
		"dataset_id":   strings.TrimSpace(datasetID),
		"max_messages": fmt.Sprintf("%d", maxMessages),
		"parse":        boolText(parse),
		"observe_only": "true",
	}
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (s *KnowledgeJobPlannerService) now() time.Time {
	if s != nil && s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
}

func sortedKnowledgePlannerGroups(groups map[string]struct{}) []string {
	if len(groups) == 0 {
		return nil
	}
	items := make([]string, 0, len(groups))
	for groupID := range groups {
		items = append(items, groupID)
	}
	sort.Strings(items)
	return items
}
