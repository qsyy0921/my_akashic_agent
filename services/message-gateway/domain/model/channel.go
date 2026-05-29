package model

type ChannelKind string

const (
	ChannelKindQQ       ChannelKind = "qq"
	ChannelKindTelegram ChannelKind = "telegram"
)

type ConversationType string

const (
	ConversationTypePrivate ConversationType = "private"
	ConversationTypeGroup   ConversationType = "group"
)

type ChannelRef struct {
	Kind             ChannelKind
	AccountID        string
	ConversationID   string
	ConversationType ConversationType
}
