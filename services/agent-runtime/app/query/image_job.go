package query

import "time"

type ChannelView struct {
	Kind             string
	AccountID        string
	ConversationID   string
	ConversationType string
}

type AttachmentView struct {
	ID        string
	Kind      string
	URL       string
	MimeType  string
	Name      string
	SizeBytes int64
}

type ImageJobView struct {
	JobID        string
	RequestID    string
	Requester    ChannelView
	RequesterID  string
	Prompt       string
	Provider     string
	Model        string
	Size         string
	Count        int
	Status       string
	Attempts     int
	MaxAttempts  int
	Results      []AttachmentView
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Metadata     map[string]string
}

