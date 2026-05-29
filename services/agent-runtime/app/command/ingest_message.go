package command

import "time"

type IngestMessageCommand struct {
	EventID     string
	Channel     ChannelCommand
	Sender      SenderCommand
	Content     string
	Attachments []AttachmentCommand
	Timestamp   time.Time
	Metadata    map[string]string
}

type ChannelCommand struct {
	Kind             string
	AccountID        string
	ConversationID   string
	ConversationType string
}

type SenderCommand struct {
	ID          string
	DisplayName string
	Kind        string
}

type AttachmentCommand struct {
	ID        string
	Kind      string
	URL       string
	MimeType  string
	Name      string
	SizeBytes int64
}

