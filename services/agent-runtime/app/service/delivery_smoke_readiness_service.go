package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

type DeliverySmokeReadinessConfig struct {
	DefaultCases     []command.DeliverySmokeCaseCommand
	ChannelByAccount map[string]string
}

type DeliverySmokeReadinessService struct {
	planner          domainservice.DeliveryPlanner
	adapters         []outport.DeliveryAdapter
	defaultCases     []command.DeliverySmokeCaseCommand
	channelByAccount map[string]string
}

func NewDeliverySmokeReadinessService(
	adapters []outport.DeliveryAdapter,
	config DeliverySmokeReadinessConfig,
) *DeliverySmokeReadinessService {
	filteredAdapters := make([]outport.DeliveryAdapter, 0, len(adapters))
	for _, adapter := range adapters {
		if adapter != nil {
			filteredAdapters = append(filteredAdapters, adapter)
		}
	}
	return &DeliverySmokeReadinessService{
		planner:          domainservice.NewDeliveryPlanner(),
		adapters:         filteredAdapters,
		defaultCases:     cloneDeliverySmokeCases(config.DefaultCases),
		channelByAccount: cloneDeliverySmokeStringMap(config.ChannelByAccount),
	}
}

func (s *DeliverySmokeReadinessService) CheckDeliverySmokeReadiness(
	ctx context.Context,
	cmd command.CheckDeliverySmokeReadinessCommand,
) (query.DeliverySmokeReadinessView, error) {
	if err := ctx.Err(); err != nil {
		return query.DeliverySmokeReadinessView{}, err
	}
	if s == nil {
		return query.DeliverySmokeReadinessView{}, errors.New("delivery smoke readiness service is nil")
	}

	cases := cloneDeliverySmokeCases(cmd.Cases)
	if len(cases) == 0 {
		cases = cloneDeliverySmokeCases(s.defaultCases)
	}
	if len(cmd.GroupIDs) > 0 {
		cases = append(cases, deliverySmokeGroupCases(accountIDsFromSmokeCases(cases), cmd.GroupIDs, cmd.IncludeSyntheticMedia)...)
	}
	if len(cases) == 0 {
		return query.DeliverySmokeReadinessView{
			Ready:      false,
			Reason:     "no_smoke_cases",
			Cases:      []query.DeliverySmokeCaseReadinessView{},
			Totals:     map[string]int{"cases": 0, "ready": 0, "not_ready": 0},
			Blockers:   []string{"delivery_smoke_cases_missing"},
			Attributes: map[string]string{"checked_by": "agent_runtime_delivery_smoke_readiness"},
			Notes:      []string{"read-only smoke readiness matrix; no platform messages are sent"},
			SideEffect: "none",
		}, nil
	}

	channelByAccount := cloneDeliverySmokeStringMap(s.channelByAccount)
	for accountID, channel := range cmd.ChannelByAccount {
		accountID = strings.TrimSpace(accountID)
		channel = strings.TrimSpace(channel)
		if accountID != "" && channel != "" {
			channelByAccount[accountID] = channel
		}
	}

	caseViews := make([]query.DeliverySmokeCaseReadinessView, 0, len(cases))
	blockers := make([]string, 0)
	readyCount := 0
	for _, smokeCase := range cases {
		view := s.checkCase(smokeCase, channelByAccount)
		if view.Ready {
			readyCount++
		} else {
			blockers = append(blockers, smokeCaseBlockers(view)...)
		}
		caseViews = append(caseViews, view)
	}

	ready := readyCount == len(caseViews)
	reason := "delivery_smoke_ready"
	if !ready {
		reason = "delivery_smoke_not_ready"
	}
	return query.DeliverySmokeReadinessView{
		Ready:    ready,
		Reason:   reason,
		Cases:    caseViews,
		Totals:   map[string]int{"cases": len(caseViews), "ready": readyCount, "not_ready": len(caseViews) - readyCount},
		Blockers: sortedUniqueSmokeStrings(blockers),
		Attributes: map[string]string{
			"checked_by":  "agent_runtime_delivery_smoke_readiness",
			"side_effect": "none",
		},
		Notes: []string{
			"read-only smoke readiness matrix; no platform messages are sent",
			"synthetic media cases verify dispatch planning and adapter routing only; they do not validate remote platform media upload behavior",
		},
		SideEffect: "none",
	}, nil
}

func (s *DeliverySmokeReadinessService) checkCase(
	smokeCase command.DeliverySmokeCaseCommand,
	channelByAccount map[string]string,
) query.DeliverySmokeCaseReadinessView {
	name := strings.TrimSpace(smokeCase.Name)
	if name == "" {
		name = deliverySmokeCaseName(smokeCase)
	}
	plan, err := s.planCase(smokeCase, name, channelByAccount)
	if err != nil {
		return query.DeliverySmokeCaseReadinessView{
			Name:   name,
			Ready:  false,
			Reason: "delivery_smoke_plan_error",
			Attributes: map[string]string{
				"error":       err.Error(),
				"checked_by":  "agent_runtime_delivery_smoke_readiness",
				"side_effect": "none",
			},
		}
	}

	missingChannels := make([]string, 0)
	seen := make(map[string]struct{})
	for _, step := range plan.Steps {
		channel := strings.TrimSpace(step.Channel)
		if channel == "" {
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		if !s.adapterAvailable(channel) {
			missingChannels = append(missingChannels, channel)
		}
	}
	sort.Strings(missingChannels)
	ready := len(missingChannels) == 0
	reason := "delivery_adapter_ready"
	if !ready {
		reason = "delivery_adapter_unavailable"
	}
	return query.DeliverySmokeCaseReadinessView{
		Name:            name,
		Ready:           ready,
		Reason:          reason,
		MissingChannels: missingChannels,
		Plan:            assembler.ToDeliveryDispatchPlanView(plan),
		Attributes: map[string]string{
			"checked_by":  "agent_runtime_delivery_smoke_readiness",
			"side_effect": "none",
		},
	}
}

func (s *DeliverySmokeReadinessService) planCase(
	smokeCase command.DeliverySmokeCaseCommand,
	name string,
	channelByAccount map[string]string,
) (model.DeliveryDispatchPlan, error) {
	now := time.Now().UTC()
	delivery, err := model.NewOutboxDelivery(model.OutboundMessage{
		EventID: deliverySmokeEventID(name),
		Channel: model.ChannelRef{
			Kind:             model.ChannelKind(strings.TrimSpace(smokeCase.ChannelKind)),
			AccountID:        strings.TrimSpace(smokeCase.AccountID),
			ConversationID:   strings.TrimSpace(smokeCase.ConversationID),
			ConversationType: model.ConversationType(strings.TrimSpace(smokeCase.ConversationType)),
		},
		Content:     strings.TrimSpace(smokeCase.Content),
		Attachments: deliverySmokeAttachments(smokeCase.Attachments),
		Timestamp:   now,
		Metadata:    cloneDeliverySmokeStringMap(smokeCase.Metadata),
	}, 1, now)
	if err != nil {
		return model.DeliveryDispatchPlan{}, err
	}
	return s.planner.Plan(delivery, domainservice.DeliveryPlannerConfig{ChannelByAccount: channelByAccount})
}

func (s *DeliverySmokeReadinessService) adapterAvailable(channel string) bool {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return false
	}
	for _, adapter := range s.adapters {
		if adapter != nil && adapter.SupportsDeliveryChannel(channel) {
			return true
		}
	}
	return false
}

func deliverySmokeAttachments(items []command.DeliverySmokeAttachmentCommand) []model.Attachment {
	attachments := make([]model.Attachment, 0, len(items))
	for _, item := range items {
		kind := model.AttachmentKind(strings.TrimSpace(item.Kind))
		if kind != model.AttachmentKindImage {
			kind = model.AttachmentKindFile
		}
		url := strings.TrimSpace(item.URL)
		if url == "" {
			continue
		}
		attachments = append(attachments, model.Attachment{
			Kind:     kind,
			URL:      url,
			Name:     strings.TrimSpace(item.Name),
			MimeType: strings.TrimSpace(item.MimeType),
		})
	}
	return attachments
}

func deliverySmokeGroupCases(
	accountIDs []string,
	groupIDs []string,
	includeSyntheticMedia bool,
) []command.DeliverySmokeCaseCommand {
	cases := make([]command.DeliverySmokeCaseCommand, 0)
	for _, accountID := range accountIDs {
		accountID = strings.TrimSpace(accountID)
		if accountID == "" {
			continue
		}
		for _, groupID := range groupIDs {
			groupID = strings.TrimSpace(groupID)
			if groupID == "" {
				continue
			}
			base := command.DeliverySmokeCaseCommand{
				Name:             "qq_group_text_" + accountID + "_to_" + groupID,
				ChannelKind:      "qq",
				AccountID:        accountID,
				ConversationID:   groupID,
				ConversationType: string(model.ConversationTypeGroup),
				Content:          "[akashic smoke] delivery readiness group text",
				Metadata:         map[string]string{"synthetic": "true"},
			}
			cases = append(cases, base)
			if includeSyntheticMedia {
				cases = append(cases, syntheticMediaSmokeCases(base)...)
			}
		}
	}
	return cases
}

func accountIDsFromSmokeCases(cases []command.DeliverySmokeCaseCommand) []string {
	accountIDs := make([]string, 0)
	seen := make(map[string]struct{})
	for _, item := range cases {
		accountID := strings.TrimSpace(item.AccountID)
		if accountID == "" {
			continue
		}
		if _, ok := seen[accountID]; ok {
			continue
		}
		seen[accountID] = struct{}{}
		accountIDs = append(accountIDs, accountID)
	}
	sort.Strings(accountIDs)
	return accountIDs
}

func syntheticMediaSmokeCases(base command.DeliverySmokeCaseCommand) []command.DeliverySmokeCaseCommand {
	image := cloneDeliverySmokeCase(base)
	image.Name = strings.Replace(base.Name, "_text_", "_image_", 1)
	image.Content = "[akashic smoke] delivery readiness image"
	image.Attachments = []command.DeliverySmokeAttachmentCommand{{
		Kind:     string(model.AttachmentKindImage),
		URL:      "base64://YXNoaWNhYy1zbW9rZS1pbWFnZQ==",
		Name:     "akashic-smoke-image.txt",
		MimeType: "image/png",
	}}

	file := cloneDeliverySmokeCase(base)
	file.Name = strings.Replace(base.Name, "_text_", "_file_", 1)
	file.Content = "[akashic smoke] delivery readiness file"
	file.Attachments = []command.DeliverySmokeAttachmentCommand{{
		Kind:     string(model.AttachmentKindFile),
		URL:      "base64://YXNoaWNhYy1zbW9rZS1maWxl",
		Name:     "akashic-smoke-file.txt",
		MimeType: "text/plain",
	}}
	return []command.DeliverySmokeCaseCommand{image, file}
}

func smokeCaseBlockers(view query.DeliverySmokeCaseReadinessView) []string {
	if len(view.MissingChannels) == 0 {
		return []string{view.Name + ":not_ready"}
	}
	blockers := make([]string, 0, len(view.MissingChannels))
	for _, channel := range view.MissingChannels {
		blockers = append(blockers, view.Name+":missing_channel:"+channel)
	}
	return blockers
}

func deliverySmokeCaseName(smokeCase command.DeliverySmokeCaseCommand) string {
	parts := []string{
		strings.TrimSpace(smokeCase.ChannelKind),
		strings.TrimSpace(smokeCase.ConversationType),
		strings.TrimSpace(smokeCase.AccountID),
		"to",
		strings.TrimSpace(smokeCase.ConversationID),
	}
	return strings.Join(nonEmptyStrings(parts), "_")
}

func deliverySmokeEventID(name string) string {
	id := strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", ">", "-to-", "<", "-")
	id = replacer.Replace(id)
	id = strings.Trim(id, "-")
	if id == "" {
		id = "delivery-smoke"
	}
	return "smoke-" + id
}

func nonEmptyStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func cloneDeliverySmokeCases(items []command.DeliverySmokeCaseCommand) []command.DeliverySmokeCaseCommand {
	cloned := make([]command.DeliverySmokeCaseCommand, len(items))
	for index, item := range items {
		cloned[index] = cloneDeliverySmokeCase(item)
	}
	return cloned
}

func cloneDeliverySmokeCase(item command.DeliverySmokeCaseCommand) command.DeliverySmokeCaseCommand {
	cloned := item
	cloned.Attachments = append([]command.DeliverySmokeAttachmentCommand(nil), item.Attachments...)
	cloned.Metadata = cloneDeliverySmokeStringMap(item.Metadata)
	return cloned
}

func cloneDeliverySmokeStringMap(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(items))
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}

func sortedUniqueSmokeStrings(items []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}
