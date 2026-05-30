package model

import (
	"testing"
	"time"
)

func TestNewObserveTargetDefaultsAndValidatesObserveOnly(t *testing.T) {
	target, err := NewObserveTarget(ObserveTargetSpec{
		Channel: ChannelRef{
			Kind:             ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   "27234224",
			ConversationType: ConversationTypeGroup,
		},
		ObserveOnly:  true,
		ReplyAllowed: true,
		Enabled:      true,
		Source:       "python_config",
	}, time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new observe target: %v", err)
	}
	if target.TargetID != "qq:1049511700:group:27234224" {
		t.Fatalf("unexpected target id: %s", target.TargetID)
	}
	if target.ReplyAllowed {
		t.Fatalf("observe-only target must force reply_allowed=false")
	}
}

func TestNewObserveTargetRejectsMissingConversation(t *testing.T) {
	_, err := NewObserveTarget(ObserveTargetSpec{
		Channel: ChannelRef{
			Kind:             ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationType: ConversationTypeGroup,
		},
		Enabled: true,
	}, time.Now())
	if err == nil {
		t.Fatal("expected validation error")
	}
}
