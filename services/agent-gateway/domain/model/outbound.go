package model

import (
	"errors"
	"strings"
	"time"
)

type OutboundMessage struct {
	EventID     string
	Channel     ChannelRef
	Content     string
	Attachments []Attachment
	Timestamp   time.Time
	Metadata    map[string]string
}

func (m OutboundMessage) Validate() error {
	if strings.TrimSpace(m.EventID) == "" {
		return errors.New("outbound message requires event id")
	}
	if strings.TrimSpace(string(m.Channel.Kind)) == "" {
		return errors.New("outbound message requires channel kind")
	}
	if strings.TrimSpace(m.Channel.AccountID) == "" {
		return errors.New("outbound message requires channel account id")
	}
	if strings.TrimSpace(m.Channel.ConversationID) == "" {
		return errors.New("outbound message requires conversation id")
	}
	if strings.TrimSpace(m.Content) == "" && len(m.Attachments) == 0 {
		return errors.New("outbound message requires content or attachments")
	}
	if m.Timestamp.IsZero() {
		return errors.New("outbound message requires timestamp")
	}
	return nil
}

type SendRecord struct {
	FromBotID      string
	ConversationID string
	ContentHash    string
	Timestamp      time.Time
}

func (r SendRecord) Validate() error {
	if strings.TrimSpace(r.FromBotID) == "" {
		return errors.New("send record requires bot id")
	}
	if strings.TrimSpace(r.ConversationID) == "" {
		return errors.New("send record requires conversation id")
	}
	if strings.TrimSpace(r.ContentHash) == "" {
		return errors.New("send record requires content hash")
	}
	if r.Timestamp.IsZero() {
		return errors.New("send record requires timestamp")
	}
	return nil
}
