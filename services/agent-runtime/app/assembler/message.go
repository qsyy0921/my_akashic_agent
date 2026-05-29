package assembler

import (
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToEnvelope(cmd command.IngestMessageCommand) model.MessageEnvelope {
	return model.MessageEnvelope{
		EventID: cmd.EventID,
		Channel: ToChannelRef(cmd.Channel),
		Sender: model.SenderRef{
			ID:          cmd.Sender.ID,
			DisplayName: cmd.Sender.DisplayName,
			Kind:        model.SenderKind(cmd.Sender.Kind),
		},
		Content:     cmd.Content,
		Attachments: ToAttachments(cmd.Attachments),
		Timestamp:   cmd.Timestamp,
		Metadata:    cmd.Metadata,
	}
}

func ToOutbound(cmd command.SendMessageCommand) model.OutboundMessage {
	return model.OutboundMessage{
		EventID:     cmd.EventID,
		Channel:     ToChannelRef(cmd.Channel),
		Content:     cmd.Content,
		Attachments: ToAttachments(cmd.Attachments),
		Timestamp:   cmd.Timestamp,
		Metadata:    cmd.Metadata,
	}
}

func ToChannelRef(cmd command.ChannelCommand) model.ChannelRef {
	return model.ChannelRef{
		Kind:             model.ChannelKind(cmd.Kind),
		AccountID:        cmd.AccountID,
		ConversationID:   cmd.ConversationID,
		ConversationType: model.ConversationType(cmd.ConversationType),
	}
}

func ToAttachments(items []command.AttachmentCommand) []model.Attachment {
	attachments := make([]model.Attachment, 0, len(items))
	for _, item := range items {
		attachments = append(attachments, model.Attachment{
			ID:        item.ID,
			Kind:      model.AttachmentKind(item.Kind),
			URL:       item.URL,
			MimeType:  item.MimeType,
			Name:      item.Name,
			SizeBytes: item.SizeBytes,
		})
	}
	return attachments
}

