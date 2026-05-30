package command

type CheckDeliverySmokeReadinessCommand struct {
	Cases                 []DeliverySmokeCaseCommand
	GroupIDs              []string
	ChannelByAccount      map[string]string
	IncludeSyntheticMedia bool
}

type DeliverySmokeCaseCommand struct {
	Name             string
	ChannelKind      string
	AccountID        string
	ConversationID   string
	ConversationType string
	Content          string
	Attachments      []DeliverySmokeAttachmentCommand
	Metadata         map[string]string
}

type DeliverySmokeAttachmentCommand struct {
	Kind     string
	URL      string
	Name     string
	MimeType string
}
