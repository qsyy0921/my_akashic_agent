package query

type SendRecordFilter struct {
	Limit          int
	FromBotID      string
	ConversationID string
	ContentHash    string
}

type SendRecordView struct {
	FromBotID      string `json:"from_bot_id"`
	ConversationID string `json:"conversation_id"`
	ContentHash    string `json:"content_hash"`
	Timestamp      string `json:"timestamp"`
}

type RecentSendView struct {
	Recent         bool   `json:"recent"`
	FromBotID      string `json:"from_bot_id"`
	ConversationID string `json:"conversation_id"`
	ContentHash    string `json:"content_hash"`
	WindowSeconds  int    `json:"window_seconds"`
}
