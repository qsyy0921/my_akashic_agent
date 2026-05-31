package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestProactiveDeliveryRecordWithinWindow(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	record, err := model.NewProactiveDeliveryRecord("telegram:1", "abc", now.Add(-30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !record.WithinWindow(now, time.Hour) {
		t.Fatal("expected record to be inside one hour window")
	}
	if record.WithinWindow(now, 10*time.Minute) {
		t.Fatal("expected record to be outside ten minute window")
	}
}

func TestProactiveSourceItemRecordsNormalizeMCPSourceKeys(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	seen, err := model.NewProactiveSeenItemRecord("mcp:news:feed-a", "item-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if seen.SourceKey != "mcp:news" {
		t.Fatalf("unexpected source key: %s", seen.SourceKey)
	}
	if !seen.WithinWindow(now.Add(30*time.Minute), time.Hour) {
		t.Fatal("expected seen item to be inside window")
	}

	rejection, err := model.NewProactiveRejectionCooldownRecord("mcp:news:feed-b", "item-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if rejection.SourceKey != "mcp:news" {
		t.Fatalf("unexpected rejection source key: %s", rejection.SourceKey)
	}
	if rejection.WithinWindow(now.Add(2*time.Hour), time.Hour) {
		t.Fatal("expected rejection cooldown to expire")
	}
}

func TestProactiveSessionMarkRejectsUnknownKey(t *testing.T) {
	if _, err := model.NewProactiveSessionMark("telegram:1", "unknown", time.Now().UTC()); err == nil {
		t.Fatal("expected invalid key error")
	}
}
