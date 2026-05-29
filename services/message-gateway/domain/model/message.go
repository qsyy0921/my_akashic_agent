package model

import (
	"errors"
	"strings"
	"time"
)

type MessageEnvelope struct {
	EventID     string
	Channel     ChannelRef
	Sender      SenderRef
	Content     string
	Attachments []Attachment
	Timestamp   time.Time
	Provenance  Provenance
	Metadata    map[string]string
}

type SenderKind string

const (
	SenderKindHuman SenderKind = "human"
	SenderKindBot   SenderKind = "bot"
	SenderKindSelf  SenderKind = "self"
)

type SenderRef struct {
	ID          string
	DisplayName string
	Kind        SenderKind
}

type AttachmentKind string

const (
	AttachmentKindImage AttachmentKind = "image"
	AttachmentKindFile  AttachmentKind = "file"
)

type Attachment struct {
	ID        string
	Kind      AttachmentKind
	URL       string
	MimeType  string
	Name      string
	SizeBytes int64
}

func (m MessageEnvelope) Validate() error {
	if strings.TrimSpace(m.EventID) == "" {
		return errors.New("message envelope requires event id")
	}
	if strings.TrimSpace(string(m.Channel.Kind)) == "" {
		return errors.New("message envelope requires channel kind")
	}
	if strings.TrimSpace(m.Channel.AccountID) == "" {
		return errors.New("message envelope requires channel account id")
	}
	if strings.TrimSpace(m.Channel.ConversationID) == "" {
		return errors.New("message envelope requires conversation id")
	}
	if strings.TrimSpace(m.Sender.ID) == "" {
		return errors.New("message envelope requires sender id")
	}
	if m.Timestamp.IsZero() {
		return errors.New("message envelope requires timestamp")
	}
	return nil
}
