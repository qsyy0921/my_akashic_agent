package service

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
)

func TestReceiverStatusServiceReportsLatestStatus(t *testing.T) {
	service := NewReceiverStatusService()
	_, err := service.ReportReceiverStatus(context.Background(), command.ReportReceiverStatusCommand{
		Kind:        "telegram",
		ChannelName: "telegram",
		AccountID:   "7689386159",
		Status:      "connected",
		Source:      "python_channel",
		Timestamp:   time.Date(2026, 5, 31, 1, 2, 3, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("report connected: %v", err)
	}
	view, err := service.ReportReceiverStatus(context.Background(), command.ReportReceiverStatusCommand{
		ReceiverID:  "telegram:7689386159:telegram",
		Kind:        "telegram",
		ChannelName: "telegram",
		AccountID:   "7689386159",
		Status:      "suspended",
		Reason:      "getupdates_conflict",
		Source:      "python_channel",
	})
	if err != nil {
		t.Fatalf("report suspended: %v", err)
	}
	if got := view.Totals["receivers"]; got != 1 {
		t.Fatalf("receivers = %d, want 1", got)
	}
	if got := view.Totals["suspended"]; got != 1 {
		t.Fatalf("suspended = %d, want 1", got)
	}
	if view.Receivers[0].Reason != "getupdates_conflict" {
		t.Fatalf("unexpected reason: %#v", view.Receivers[0])
	}
}

func TestReceiverStatusServiceAggregatesMultipleKinds(t *testing.T) {
	service := NewReceiverStatusService()
	for _, item := range []command.ReportReceiverStatusCommand{
		{Kind: "qq", ChannelName: "qq_1049511700", AccountID: "1049511700", Status: "connected"},
		{Kind: "qq", ChannelName: "qq_2365524513", AccountID: "2365524513", Status: "failed"},
		{Kind: "telegram", ChannelName: "telegram", AccountID: "7689386159", Status: "suspended"},
	} {
		if _, err := service.ReportReceiverStatus(context.Background(), item); err != nil {
			t.Fatalf("report status: %v", err)
		}
	}
	view, err := service.ListReceiverStatuses(context.Background())
	if err != nil {
		t.Fatalf("list statuses: %v", err)
	}
	if view.Totals["receivers"] != 3 || view.Totals["qq"] != 2 || view.Totals["telegram"] != 1 {
		t.Fatalf("unexpected totals: %#v", view.Totals)
	}
	if view.Totals["connected"] != 1 || view.Totals["failed"] != 1 || view.Totals["suspended"] != 1 {
		t.Fatalf("unexpected status totals: %#v", view.Totals)
	}
}
