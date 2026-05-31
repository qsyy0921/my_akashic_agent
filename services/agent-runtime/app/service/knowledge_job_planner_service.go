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
	if s == nil || s.agentJobs == nil {
		return query.KnowledgeJobPlannerRunView{}, errors.New("knowledge job planner requires observe targets and agent jobs")
	}
	preview, err := s.PreviewKnowledgeJobs(ctx, cmd)
	if err != nil {
		return query.KnowledgeJobPlannerRunView{}, err
	}
	view := query.KnowledgeJobPlannerRunView{
		Timestamp:       preview.Timestamp,
		Bucket:          preview.Bucket,
		Targets:         preview.Targets,
		GroupMemoryJobs: preview.GroupMemoryJobs,
		RagIngestJobs:   preview.RagIngestJobs,
		SkippedTargets:  preview.SkippedTargets,
		Groups:          append([]string(nil), preview.Groups...),
		SideEffect:      "agent_job_create",
	}
	timestamp, err := time.Parse(time.RFC3339Nano, preview.Timestamp)
	if err != nil {
		return query.KnowledgeJobPlannerRunView{}, err
	}
	for _, targetPlan := range preview.Plans {
		for _, jobPlan := range targetPlan.Jobs {
			job, err := s.agentJobs.Create(ctx, command.CreateAgentJobCommand{
				JobID:       jobPlan.JobID,
				JobType:     jobPlan.JobType,
				AgentID:     jobPlan.AgentID,
				Route:       knowledgeJobPlannerCommandRoute(jobPlan.Route),
				Payload:     copyKnowledgePlannerStringMap(jobPlan.Payload),
				DedupeKey:   jobPlan.DedupeKey,
				MaxAttempts: jobPlan.MaxAttempts,
				Timestamp:   timestamp,
				Metadata:    copyKnowledgePlannerStringMap(jobPlan.Metadata),
			})
			if err != nil {
				return query.KnowledgeJobPlannerRunView{}, err
			}
			view.CreatedOrExisting++
			if strings.TrimSpace(job.JobID) != jobPlan.JobID {
				view.SuppressedByDedupe++
			}
		}
	}
	return view, nil
}

func (s *KnowledgeJobPlannerService) PreviewKnowledgeJobs(
	ctx context.Context,
	cmd command.PlanKnowledgeJobsCommand,
) (query.KnowledgeJobPlannerPreviewView, error) {
	if err := ctx.Err(); err != nil {
		return query.KnowledgeJobPlannerPreviewView{}, err
	}
	if s == nil || s.observeTargets == nil {
		return query.KnowledgeJobPlannerPreviewView{}, errors.New("knowledge job planner requires observe targets")
	}
	resolved := s.resolvePlanCommand(cmd)
	targets, err := s.observeTargets.ListObserveTargets(ctx)
	if err != nil {
		return query.KnowledgeJobPlannerPreviewView{}, err
	}
	view := query.KnowledgeJobPlannerPreviewView{
		Timestamp:       resolved.timestamp.Format(time.RFC3339Nano),
		Bucket:          resolved.bucket,
		IntervalSeconds: resolved.intervalSeconds,
		Plans:           []query.KnowledgeJobPlannerTargetPlanView{},
		SideEffect:      "none",
	}
	groups := make(map[string]struct{})
	for _, target := range targets.Targets {
		if reason := knowledgeJobPlannerTargetSkipReason(target); reason != "" {
			view.SkippedTargets++
			view.Skipped = append(view.Skipped, query.KnowledgeJobPlannerSkippedView{
				TargetID: strings.TrimSpace(target.TargetID),
				Channel:  target.Channel,
				Reason:   reason,
			})
			continue
		}
		groupID := strings.TrimSpace(target.Channel.ConversationID)
		accountID := strings.TrimSpace(target.Channel.AccountID)
		groups[groupID] = struct{}{}
		datasets := splitCommaSeparatedList(target.Metadata["ragflow_dataset_ids"])
		plan := query.KnowledgeJobPlannerTargetPlanView{
			TargetID: strings.TrimSpace(target.TargetID),
			Channel:  target.Channel,
			Datasets: append([]string(nil), datasets...),
			Jobs:     []query.KnowledgeJobPlannerJobPlanView{},
			Metadata: copyKnowledgePlannerStringMap(target.Metadata),
		}
		groupMemoryDedupeKey := fmt.Sprintf("knowledge:group_memory_extract:qq:%s:%s", accountID, groupID)
		plan.Jobs = append(plan.Jobs, query.KnowledgeJobPlannerJobPlanView{
			JobID:       fmt.Sprintf("group_memory_extract:qq:%s:%d", groupID, resolved.bucket),
			JobType:     string(model.AgentJobGroupMemoryExtract),
			AgentID:     resolved.agentID,
			Route:       knowledgeJobPlannerRouteView(target),
			Payload:     knowledgeJobPlannerGroupMemoryPayload(groupID),
			DedupeKey:   groupMemoryDedupeKey,
			MaxAttempts: resolved.maxAttempts,
			Metadata: map[string]string{
				"dedupe_key": groupMemoryDedupeKey,
				"scheduler":  resolved.plannerID,
			},
		})
		view.GroupMemoryJobs++
		for _, datasetID := range datasets {
			ragDedupeKey := fmt.Sprintf("knowledge:rag_ingest:qq:%s:%s:%s", accountID, groupID, datasetID)
			plan.Jobs = append(plan.Jobs, query.KnowledgeJobPlannerJobPlanView{
				JobID:       fmt.Sprintf("rag_ingest:qq:%s:%s:%d", groupID, datasetID, resolved.bucket),
				JobType:     string(model.AgentJobRagIngest),
				AgentID:     resolved.agentID,
				Route:       knowledgeJobPlannerRouteView(target),
				Payload:     knowledgeJobPlannerRagPayload(groupID, datasetID, resolved.ragMaxMessages, cmd.RagParse),
				DedupeKey:   ragDedupeKey,
				MaxAttempts: resolved.maxAttempts,
				Metadata: map[string]string{
					"dedupe_key": ragDedupeKey,
					"dataset_id": datasetID,
					"scheduler":  resolved.plannerID,
				},
			})
			view.RagIngestJobs++
		}
		view.Targets++
		view.TotalJobs += len(plan.Jobs)
		view.Plans = append(view.Plans, plan)
	}
	view.Groups = sortedKnowledgePlannerGroups(groups)
	return view, nil
}

func knowledgeJobPlannerTargetEligible(target query.ObserveTargetView) bool {
	return knowledgeJobPlannerTargetSkipReason(target) == ""
}

func knowledgeJobPlannerTargetSkipReason(target query.ObserveTargetView) string {
	if !target.Enabled {
		return "disabled"
	}
	if !target.ObserveOnly {
		return "not_observe_only"
	}
	if strings.TrimSpace(target.Channel.Kind) != string(model.ChannelKindQQ) {
		return "non_qq_channel"
	}
	if strings.TrimSpace(target.Channel.ConversationType) != string(model.ConversationTypeGroup) {
		return "non_group_conversation"
	}
	if strings.TrimSpace(target.Channel.AccountID) == "" {
		return "missing_account_id"
	}
	if strings.TrimSpace(target.Channel.ConversationID) == "" {
		return "missing_conversation_id"
	}
	return ""
}

func knowledgeJobPlannerRouteView(target query.ObserveTargetView) query.AgentJobRouteView {
	return query.AgentJobRouteView{
		Kind:             strings.TrimSpace(target.Channel.Kind),
		AccountID:        strings.TrimSpace(target.Channel.AccountID),
		ConversationID:   strings.TrimSpace(target.Channel.ConversationID),
		ConversationType: strings.TrimSpace(target.Channel.ConversationType),
	}
}

func knowledgeJobPlannerCommandRoute(route query.AgentJobRouteView) command.ChannelCommand {
	return command.ChannelCommand{
		Kind:             strings.TrimSpace(route.Kind),
		AccountID:        strings.TrimSpace(route.AccountID),
		ConversationID:   strings.TrimSpace(route.ConversationID),
		ConversationType: strings.TrimSpace(route.ConversationType),
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

type resolvedKnowledgeJobPlanCommand struct {
	timestamp       time.Time
	intervalSeconds int
	bucket          int64
	agentID         string
	plannerID       string
	maxAttempts     int
	ragMaxMessages  int
}

func (s *KnowledgeJobPlannerService) resolvePlanCommand(cmd command.PlanKnowledgeJobsCommand) resolvedKnowledgeJobPlanCommand {
	now := cmd.Timestamp
	if now.IsZero() {
		now = s.now()
	}
	now = now.UTC()
	intervalSeconds := cmd.IntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
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
	return resolvedKnowledgeJobPlanCommand{
		timestamp:       now,
		intervalSeconds: intervalSeconds,
		bucket:          now.Unix() / int64(intervalSeconds),
		agentID:         agentID,
		plannerID:       plannerID,
		maxAttempts:     maxAttempts,
		ragMaxMessages:  ragMaxMessages,
	}
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

func copyKnowledgePlannerStringMap(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
