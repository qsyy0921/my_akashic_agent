package service

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type DeliveryPlanner struct{}

type DeliveryPlannerConfig struct {
	ChannelByAccount map[string]string
}

type DeliveryPlanError struct {
	Kind    model.DeliveryErrorKind
	Message string
}

func (e DeliveryPlanError) Error() string {
	return e.Message
}

func (e DeliveryPlanError) DeliveryErrorKind() string {
	return string(model.NormalizeDeliveryErrorKind(string(e.Kind)))
}

func NewDeliveryPlanner() DeliveryPlanner {
	return DeliveryPlanner{}
}

func (DeliveryPlanner) Plan(delivery model.OutboxDelivery, config DeliveryPlannerConfig) (model.DeliveryDispatchPlan, error) {
	channel := resolveDeliveryChannel(delivery.Message.Channel, config.ChannelByAccount)
	if channel == "" {
		return model.DeliveryDispatchPlan{}, DeliveryPlanError{
			Kind:    model.DeliveryErrorRoute,
			Message: "outbox delivery missing channel kind",
		}
	}
	chatID := strings.TrimSpace(delivery.Message.Channel.ConversationID)
	if chatID == "" {
		return model.DeliveryDispatchPlan{}, DeliveryPlanError{
			Kind:    model.DeliveryErrorRoute,
			Message: "outbox delivery missing conversation_id",
		}
	}
	if err := delivery.Validate(); err != nil {
		return model.DeliveryDispatchPlan{}, DeliveryPlanError{
			Kind:    model.DeliveryErrorValidation,
			Message: err.Error(),
		}
	}

	content := delivery.Message.Content
	images := make([]string, 0)
	files := make([]string, 0)
	for _, attachment := range delivery.Message.Attachments {
		if strings.TrimSpace(attachment.URL) == "" {
			continue
		}
		uri := normaliseDeliveryAttachmentURI(attachment.URL)
		if attachment.Kind == model.AttachmentKindImage {
			images = append(images, uri)
			continue
		}
		files = append(files, uri)
	}

	steps := make([]model.DeliveryDispatchStep, 0, len(images)+len(files)+1)
	message := content
	for _, image := range images {
		steps = append(steps, model.DeliveryDispatchStep{
			Kind:    model.DeliveryDispatchStepImage,
			Channel: channel,
			ChatID:  chatID,
			Message: message,
			Image:   image,
		})
		message = ""
	}
	for _, file := range files {
		steps = append(steps, model.DeliveryDispatchStep{
			Kind:    model.DeliveryDispatchStepFile,
			Channel: channel,
			ChatID:  chatID,
			Message: message,
			File:    file,
		})
		message = ""
	}
	if message != "" || len(steps) == 0 {
		steps = append(steps, model.DeliveryDispatchStep{
			Kind:    model.DeliveryDispatchStepText,
			Channel: channel,
			ChatID:  chatID,
			Message: message,
		})
	}
	for index := range steps {
		steps[index].StepIndex = index + 1
	}

	return model.DeliveryDispatchPlan{
		EventID:   delivery.Message.EventID,
		Channel:   channel,
		ChatID:    chatID,
		StepCount: len(steps),
		Steps:     steps,
		Attributes: map[string]string{
			"planned_by": "agent_runtime_delivery_planner",
		},
	}, nil
}

func resolveDeliveryChannel(channel model.ChannelRef, channelByAccount map[string]string) string {
	accountID := strings.TrimSpace(channel.AccountID)
	if accountID != "" {
		if mapped := strings.TrimSpace(channelByAccount[accountID]); mapped != "" {
			return mapped
		}
	}
	return strings.TrimSpace(string(channel.Kind))
}

func normaliseDeliveryAttachmentURI(raw string) string {
	value := strings.TrimSpace(raw)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "file" {
		return value
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return value
	}
	if parsed.Host != "" {
		path = `\\` + parsed.Host + filepath.FromSlash(path)
	} else if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return path
}
