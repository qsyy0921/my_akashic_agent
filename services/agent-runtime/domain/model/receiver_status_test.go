package model

import (
	"testing"
	"time"
)

func TestNewReceiverStatusNormalizesAliasesAndStableID(t *testing.T) {
	status, err := NewReceiverStatus(ReceiverStatusSpec{
		Kind:        ChannelKindTelegram,
		ChannelName: "telegram",
		AccountID:   "7689386159",
		Status:      "paused",
		Source:      "python_channel",
		Metadata:    map[string]string{" reason ": " conflict "},
	}, time.Date(2026, 5, 31, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatalf("new receiver status: %v", err)
	}
	if status.ReceiverID != "telegram:7689386159:telegram" {
		t.Fatalf("unexpected receiver id: %s", status.ReceiverID)
	}
	if status.Status != ReceiverStatusSuspended {
		t.Fatalf("unexpected normalized status: %s", status.Status)
	}
	if status.Metadata["reason"] != "conflict" {
		t.Fatalf("metadata was not trimmed: %#v", status.Metadata)
	}
}

func TestNewReceiverStatusRejectsUnknownStatus(t *testing.T) {
	_, err := NewReceiverStatus(ReceiverStatusSpec{
		Kind:        ChannelKindQQ,
		ChannelName: "qq",
		Status:      "mystery",
	}, time.Now())
	if err == nil {
		t.Fatal("expected unknown status error")
	}
}
