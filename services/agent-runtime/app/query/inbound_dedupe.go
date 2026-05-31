package query

type InboundDedupeFilter struct {
	Limit int
	Scope string
}

type InboundDedupeView struct {
	Duplicate   bool              `json:"duplicate"`
	Scope       string            `json:"scope"`
	MessageKey  string            `json:"message_key"`
	FirstSeenAt string            `json:"first_seen_at"`
	LastSeenAt  string            `json:"last_seen_at"`
	ExpiresAt   string            `json:"expires_at"`
	SeenCount   int               `json:"seen_count"`
	TTLSeconds  int               `json:"ttl_seconds"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	SideEffect  string            `json:"side_effect"`
}

type InboundDedupeRecordsView struct {
	Records    []InboundDedupeView `json:"records"`
	Totals     map[string]int      `json:"totals"`
	Notes      []string            `json:"notes"`
	SideEffect string              `json:"side_effect"`
}
