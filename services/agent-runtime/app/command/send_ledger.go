package command

import (
	"time"
)

type RecordSendCommand struct {
	FromBotID      string
	ConversationID string
	Content        string
	ContentHash    string
	Timestamp      time.Time
}

type CheckRecentSendCommand struct {
	FromBotID      string
	ConversationID string
	Content        string
	ContentHash    string
	Window         time.Duration
}

type CheckPrivateEchoCommand struct {
	FromUserID  string
	ToBotID     string
	Text        string
	HasImage    bool
	HasFile     bool
	HasForward  bool
	ContentHash string
	Window      time.Duration
}
