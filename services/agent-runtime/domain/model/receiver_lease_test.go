package model

import (
	"testing"
	"time"
)

func TestNewReceiverLeaseBuildsStableIDAndActiveState(t *testing.T) {
	now := time.Date(2026, 5, 31, 1, 2, 3, 0, time.UTC)
	lease, err := NewReceiverLease(ReceiverLeaseSpec{
		Kind:        ChannelKindTelegram,
		ChannelName: "telegram",
		AccountID:   "7689386159",
		HolderID:    "python:1",
		LeaseToken:  "tok-1",
		ExpiresAt:   now.Add(time.Minute),
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("new receiver lease: %v", err)
	}
	if lease.ReceiverID != "telegram:7689386159:telegram" {
		t.Fatalf("receiver id = %s", lease.ReceiverID)
	}
	if !lease.ActiveAt(now) || lease.ActiveAt(now.Add(2*time.Minute)) {
		t.Fatalf("unexpected active state")
	}
	if !lease.Matches("python:1", "tok-1") || lease.Matches("python:2", "tok-1") {
		t.Fatalf("unexpected match state")
	}
}

func TestReceiverLeaseRequiresToken(t *testing.T) {
	_, err := NewReceiverLease(ReceiverLeaseSpec{
		Kind:        ChannelKindQQ,
		ChannelName: "qq",
		HolderID:    "python:1",
		ExpiresAt:   time.Now().Add(time.Minute),
	})
	if err == nil {
		t.Fatal("expected token error")
	}
}
