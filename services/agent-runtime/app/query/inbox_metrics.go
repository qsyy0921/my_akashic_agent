package query

type InboxMetricsFilter struct {
	Limit            int
	ChannelKind      string
	AccountID        string
	ConversationID   string
	ConversationType string
	DecisionAction   string
	ObserveOnly      string
}

type InboxChannelKindMetricsView struct {
	Total              int            `json:"total"`
	ObserveOnly        int            `json:"observe_only"`
	ReplyEligible      int            `json:"reply_eligible"`
	WithAttachments    int            `json:"with_attachments"`
	AttachmentCount    int            `json:"attachment_count"`
	ByConversationType map[string]int `json:"by_conversation_type"`
}

type InboxConversationMetricsView struct {
	Channel          InboxChannelView `json:"channel"`
	Total            int              `json:"total"`
	ObserveOnly      int              `json:"observe_only"`
	ReplyEligible    int              `json:"reply_eligible"`
	WithAttachments  int              `json:"with_attachments"`
	AttachmentCount  int              `json:"attachment_count"`
	UniqueSenders    int              `json:"unique_senders"`
	SequencedEvents  int              `json:"sequenced_events"`
	LatestSeq        int              `json:"latest_seq,omitempty"`
	LatestReceivedAt string           `json:"latest_received_at,omitempty"`
}

type InboxRecentSampleView struct {
	EventID         string           `json:"event_id"`
	Channel         InboxChannelView `json:"channel"`
	SenderID        string           `json:"sender_id"`
	SenderKind      string           `json:"sender_kind,omitempty"`
	DecisionAction  string           `json:"decision_action"`
	ObserveOnly     bool             `json:"observe_only"`
	AttachmentCount int              `json:"attachment_count"`
	Seq             int              `json:"seq,omitempty"`
	ReceivedAt      string           `json:"received_at"`
}

type InboxMetricsView struct {
	SampledEvents          int                                     `json:"sampled_events"`
	ObserveOnlyTotal       int                                     `json:"observe_only_total"`
	ReplyEligibleTotal     int                                     `json:"reply_eligible_total"`
	WithAttachments        int                                     `json:"with_attachments"`
	AttachmentCount        int                                     `json:"attachment_count"`
	UniqueSenders          int                                     `json:"unique_senders"`
	EventsByChannelKind    map[string]InboxChannelKindMetricsView  `json:"events_by_channel_kind"`
	EventsByConversation   map[string]InboxConversationMetricsView `json:"events_by_conversation"`
	EventsByDecisionAction map[string]int                          `json:"events_by_decision_action"`
	EventsBySenderKind     map[string]int                          `json:"events_by_sender_kind"`
	Recent                 []InboxRecentSampleView                 `json:"recent"`
}
