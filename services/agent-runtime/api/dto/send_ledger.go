package dto

type RecordSendRequest struct {
	FromBotID      string `json:"from_bot_id"`
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content,omitempty"`
	ContentHash    string `json:"content_hash,omitempty"`
	Timestamp      string `json:"timestamp,omitempty"`
}
