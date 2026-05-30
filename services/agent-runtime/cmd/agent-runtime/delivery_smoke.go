package main

import (
	"os"
	"strings"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/onebotdelivery"
)

func deliverySmokeReadinessConfigFromEnv(
	botIDs []string,
	onebotEndpoints map[string]onebotdelivery.EndpointConfig,
) ([]command.DeliverySmokeCaseCommand, map[string]string) {
	channelByAccount := deliveryChannelByAccountFromEnv(botIDs, onebotEndpoints)
	cases := make([]command.DeliverySmokeCaseCommand, 0)
	includeMedia := boolEnvDefault("AKASHIC_DELIVERY_SMOKE_INCLUDE_MEDIA", true)
	for _, pair := range deliverySmokePrivatePairsFromEnv(botIDs) {
		base := command.DeliverySmokeCaseCommand{
			Name:             "qq_private_text_" + pair.From + "_to_" + pair.To,
			ChannelKind:      string(model.ChannelKindQQ),
			AccountID:        pair.From,
			ConversationID:   pair.To,
			ConversationType: string(model.ConversationTypePrivate),
			Content:          "[akashic smoke] delivery readiness private text",
			Metadata:         map[string]string{"synthetic": "true"},
		}
		cases = append(cases, base)
		if includeMedia {
			cases = append(cases, deliverySmokeSyntheticMediaCases(base)...)
		}
	}
	for _, groupID := range csvEnvOrDefault("AKASHIC_DELIVERY_SMOKE_GROUP_IDS", nil) {
		groupID = strings.TrimSpace(groupID)
		if groupID == "" {
			continue
		}
		for _, botID := range botIDs {
			botID = strings.TrimSpace(botID)
			if botID == "" {
				continue
			}
			base := command.DeliverySmokeCaseCommand{
				Name:             "qq_group_text_" + botID + "_to_" + groupID,
				ChannelKind:      string(model.ChannelKindQQ),
				AccountID:        botID,
				ConversationID:   groupID,
				ConversationType: string(model.ConversationTypeGroup),
				Content:          "[akashic smoke] delivery readiness group text",
				Metadata:         map[string]string{"synthetic": "true"},
			}
			cases = append(cases, base)
			if includeMedia {
				cases = append(cases, deliverySmokeSyntheticMediaCases(base)...)
			}
		}
	}
	return cases, channelByAccount
}

type deliverySmokePair struct {
	From string
	To   string
}

func deliverySmokePrivatePairsFromEnv(botIDs []string) []deliverySmokePair {
	if pairs := parseDeliverySmokePairs(strings.TrimSpace(os.Getenv("AKASHIC_DELIVERY_SMOKE_PRIVATE_PAIRS"))); len(pairs) > 0 {
		return pairs
	}
	ids := make([]string, 0, len(botIDs))
	for _, botID := range botIDs {
		botID = strings.TrimSpace(botID)
		if botID != "" {
			ids = append(ids, botID)
		}
	}
	if len(ids) < 2 {
		return nil
	}
	return []deliverySmokePair{
		{From: ids[0], To: ids[1]},
		{From: ids[1], To: ids[0]},
	}
}

func parseDeliverySmokePairs(raw string) []deliverySmokePair {
	pairs := make([]deliverySmokePair, 0)
	for _, item := range strings.Split(raw, ",") {
		from, to, ok := strings.Cut(strings.TrimSpace(item), ">")
		if !ok {
			continue
		}
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		if from == "" || to == "" {
			continue
		}
		pairs = append(pairs, deliverySmokePair{From: from, To: to})
	}
	return pairs
}

func deliveryChannelByAccountFromEnv(
	botIDs []string,
	onebotEndpoints map[string]onebotdelivery.EndpointConfig,
) map[string]string {
	channelByAccount := make(map[string]string)
	for index, botID := range botIDs {
		botID = strings.TrimSpace(botID)
		if botID == "" {
			continue
		}
		exact := "qq_" + botID
		if _, ok := onebotEndpoints[exact]; ok {
			channelByAccount[botID] = exact
			continue
		}
		if index == 0 {
			if _, ok := onebotEndpoints["qq"]; ok {
				channelByAccount[botID] = "qq"
			}
		}
	}
	for accountID, channel := range keyValueCSVEnv("AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT") {
		accountID = strings.TrimSpace(accountID)
		channel = strings.TrimSpace(channel)
		if accountID != "" && channel != "" {
			channelByAccount[accountID] = channel
		}
	}
	return channelByAccount
}

func deliverySmokeSyntheticMediaCases(base command.DeliverySmokeCaseCommand) []command.DeliverySmokeCaseCommand {
	image := base
	image.Name = strings.Replace(base.Name, "_text_", "_image_", 1)
	image.Content = "[akashic smoke] delivery readiness image"
	image.Attachments = []command.DeliverySmokeAttachmentCommand{{
		Kind:     string(model.AttachmentKindImage),
		URL:      "base64://YXNoaWNhYy1zbW9rZS1pbWFnZQ==",
		Name:     "akashic-smoke-image.txt",
		MimeType: "image/png",
	}}

	file := base
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
