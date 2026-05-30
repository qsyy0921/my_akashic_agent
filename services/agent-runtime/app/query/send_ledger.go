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

type SendLedgerMetricsFilter struct {
	Limit          int
	FromBotID      string
	ConversationID string
}

type SendLedgerBotMetricsView struct {
	Total               int    `json:"total"`
	UniqueConversations int    `json:"unique_conversations"`
	UniqueContentHashes int    `json:"unique_content_hashes"`
	LatestTimestamp     string `json:"latest_timestamp,omitempty"`
}

type SendLedgerConversationMetricsView struct {
	FromBotID           string `json:"from_bot_id"`
	ConversationID      string `json:"conversation_id"`
	Total               int    `json:"total"`
	UniqueContentHashes int    `json:"unique_content_hashes"`
	RepeatedHashes      int    `json:"repeated_hashes"`
	LatestTimestamp     string `json:"latest_timestamp,omitempty"`
}

type SendLedgerRepeatedHashView struct {
	FromBotID       string `json:"from_bot_id"`
	ConversationID  string `json:"conversation_id"`
	ContentHash     string `json:"content_hash"`
	Count           int    `json:"count"`
	LatestTimestamp string `json:"latest_timestamp,omitempty"`
}

type SendLedgerMetricsView struct {
	SampledRecords        int                                          `json:"sampled_records"`
	UniqueBots            int                                          `json:"unique_bots"`
	UniqueConversations   int                                          `json:"unique_conversations"`
	UniqueContentHashes   int                                          `json:"unique_content_hashes"`
	RepeatedContentHashes int                                          `json:"repeated_content_hashes"`
	RecordsByBot          map[string]SendLedgerBotMetricsView          `json:"records_by_bot"`
	RecordsByConversation map[string]SendLedgerConversationMetricsView `json:"records_by_conversation"`
	RepeatedHashes        []SendLedgerRepeatedHashView                 `json:"repeated_hashes"`
	Recent                []SendRecordView                             `json:"recent"`
}

type RecentSendView struct {
	Recent         bool   `json:"recent"`
	FromBotID      string `json:"from_bot_id"`
	ConversationID string `json:"conversation_id"`
	ContentHash    string `json:"content_hash"`
	WindowSeconds  int    `json:"window_seconds"`
}

type PrivateEchoView struct {
	Echo          bool   `json:"echo"`
	FromUserID    string `json:"from_user_id"`
	ToBotID       string `json:"to_bot_id"`
	ContentHash   string `json:"content_hash,omitempty"`
	WindowSeconds int    `json:"window_seconds"`
	Reason        string `json:"reason"`
}
