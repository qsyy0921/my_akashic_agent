package dto

type RecordProactiveDeliveryRequest struct {
	SessionKey  string `json:"session_key"`
	DeliveryKey string `json:"delivery_key"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type RecordProactiveSessionRequest struct {
	SessionKey string `json:"session_key"`
	Timestamp  string `json:"timestamp,omitempty"`
}

type RecordProactiveDriftFinishRequest struct {
	SkillUsed     string `json:"skill_used"`
	OneLine       string `json:"one_line"`
	Next          string `json:"next"`
	MessageResult string `json:"message_result"`
	Note          string `json:"note,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
}

type RecordProactiveTickLogStartRequest struct {
	TickID     string `json:"tick_id"`
	SessionKey string `json:"session_key"`
	StartedAt  string `json:"started_at,omitempty"`
	GateExit   string `json:"gate_exit,omitempty"`
}

type RecordProactiveTickLogFinishRequest struct {
	TickID         string   `json:"tick_id"`
	SessionKey     string   `json:"session_key"`
	StartedAt      string   `json:"started_at,omitempty"`
	FinishedAt     string   `json:"finished_at,omitempty"`
	GateExit       string   `json:"gate_exit,omitempty"`
	TerminalAction string   `json:"terminal_action,omitempty"`
	SkipReason     string   `json:"skip_reason,omitempty"`
	StepsTaken     int      `json:"steps_taken,omitempty"`
	AlertCount     int      `json:"alert_count,omitempty"`
	ContentCount   int      `json:"content_count,omitempty"`
	ContextCount   int      `json:"context_count,omitempty"`
	InterestingIDs []string `json:"interesting_ids,omitempty"`
	DiscardedIDs   []string `json:"discarded_ids,omitempty"`
	CitedIDs       []string `json:"cited_ids,omitempty"`
	DriftEntered   bool     `json:"drift_entered,omitempty"`
	FinalMessage   string   `json:"final_message,omitempty"`
}

type RecordProactiveTickStepLogRequest struct {
	TickID              string         `json:"tick_id"`
	StepIndex           int            `json:"step_index"`
	Phase               string         `json:"phase"`
	ToolName            string         `json:"tool_name"`
	ToolCallID          string         `json:"tool_call_id"`
	ToolArgs            map[string]any `json:"tool_args,omitempty"`
	ToolResultText      string         `json:"tool_result_text,omitempty"`
	TerminalActionAfter string         `json:"terminal_action_after,omitempty"`
	SkipReasonAfter     string         `json:"skip_reason_after,omitempty"`
	InterestingIDsAfter []string       `json:"interesting_ids_after,omitempty"`
	DiscardedIDsAfter   []string       `json:"discarded_ids_after,omitempty"`
	CitedIDsAfter       []string       `json:"cited_ids_after,omitempty"`
	FinalMessageAfter   string         `json:"final_message_after,omitempty"`
}

type RecordProactiveTimestampRequest struct {
	Timestamp string `json:"timestamp,omitempty"`
}

type ProactiveAnyActionQuotaRequest struct {
	QuotaKey  string `json:"quota_key"`
	ResetHour int    `json:"reset_hour"`
	Timezone  string `json:"timezone"`
	Timestamp string `json:"timestamp,omitempty"`
}

type ProactiveSourceItemEntry struct {
	SourceKey string `json:"source_key"`
	ItemID    string `json:"item_id"`
}

type MarkProactiveItemsRequest struct {
	Entries   []ProactiveSourceItemEntry `json:"entries"`
	Timestamp string                     `json:"timestamp,omitempty"`
}

type MarkProactiveRejectionCooldownRequest struct {
	Entries   []ProactiveSourceItemEntry `json:"entries"`
	Hours     int                        `json:"hours"`
	Timestamp string                     `json:"timestamp,omitempty"`
}

type CleanupProactiveStateRequest struct {
	SeenTTLHours              int    `json:"seen_ttl_hours"`
	DeliveryTTLHours          int    `json:"delivery_ttl_hours"`
	ContextOnlyTTLHours       int    `json:"context_only_ttl_hours,omitempty"`
	RejectionCooldownTTLHours int    `json:"rejection_cooldown_ttl_hours,omitempty"`
	Timestamp                 string `json:"timestamp,omitempty"`
}
