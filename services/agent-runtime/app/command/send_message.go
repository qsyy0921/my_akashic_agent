package command

import "time"

type SendMessageCommand struct {
	EventID         string
	Channel         ChannelCommand
	Content         string
	Attachments     []AttachmentCommand
	Timestamp       time.Time
	Metadata        map[string]string
	WithBotProtocol bool
	ProtocolFromBot string
	ProtocolNonce   string
	ProtocolNextHop int
}

