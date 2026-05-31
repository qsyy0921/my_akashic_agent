package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/receiverleasestore"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/receiverstatusstore"
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

func TestReceiverStatusServiceLoadsFileBackedStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receiver-statuses.json")
	store, err := receiverstatusstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewReceiverStatusServiceWithRepository(context.Background(), store, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReportReceiverStatus(context.Background(), command.ReportReceiverStatusCommand{
		Kind:        "qq",
		ChannelName: "qq",
		AccountID:   "1049511700",
		Status:      "connected",
		Reason:      "ncatbot_started",
		Source:      "python_channel",
		Timestamp:   time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	reopenedStore, err := receiverstatusstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := NewReceiverStatusServiceWithRepository(context.Background(), reopenedStore, 0)
	if err != nil {
		t.Fatal(err)
	}
	view, err := reopened.ListReceiverStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if view.Totals["connected"] != 1 || view.Totals["qq"] != 1 {
		t.Fatalf("expected persisted receiver status, got %#v", view)
	}
}

func TestReceiverStatusServiceMarksStaleHeartbeatsStopped(t *testing.T) {
	service := NewReceiverStatusService()
	service.statusStaleAfter = 2 * time.Minute
	service.statusClock = func() time.Time {
		return time.Date(2026, 5, 31, 8, 5, 0, 0, time.UTC)
	}
	_, err := service.ReportReceiverStatus(context.Background(), command.ReportReceiverStatusCommand{
		Kind:        "qq",
		ChannelName: "qq",
		AccountID:   "1049511700",
		Status:      "connected",
		Reason:      "heartbeat",
		Source:      "python_channel",
		Timestamp:   time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	view, err := service.ListReceiverStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if view.Totals["connected"] != 0 || view.Totals["stopped"] != 1 {
		t.Fatalf("expected stale connected status to be stopped: %#v", view.Totals)
	}
	if view.Receivers[0].Reason != "heartbeat_stale" {
		t.Fatalf("expected heartbeat stale reason: %#v", view.Receivers[0])
	}
	if view.Receivers[0].Metadata["last_status"] != "connected" {
		t.Fatalf("expected stale metadata to preserve last status: %#v", view.Receivers[0].Metadata)
	}
}

func TestReceiverStatusServiceReceiverLeaseAcquireDenyRenewRelease(t *testing.T) {
	service := NewReceiverStatusService()
	now := time.Date(2026, 5, 31, 1, 2, 3, 0, time.UTC)
	acquired, err := service.AcquireReceiverLease(context.Background(), command.AcquireReceiverLeaseCommand{
		Kind:        "telegram",
		ChannelName: "telegram",
		AccountID:   "7689386159",
		HolderID:    "python:1",
		TTLSeconds:  60,
		Timestamp:   now,
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if acquired.Acquired == nil || !*acquired.Acquired || acquired.LeaseToken == "" || !acquired.Active {
		t.Fatalf("unexpected acquire view: %#v", acquired)
	}

	denied, err := service.AcquireReceiverLease(context.Background(), command.AcquireReceiverLeaseCommand{
		Kind:        "telegram",
		ChannelName: "telegram",
		AccountID:   "7689386159",
		HolderID:    "python:2",
		TTLSeconds:  60,
		Timestamp:   now.Add(10 * time.Second),
	})
	if err != nil {
		t.Fatalf("deny acquire: %v", err)
	}
	if denied.Acquired == nil || *denied.Acquired || denied.DeniedReason != "active_lease_held" || denied.LeaseTokenPresent != true || denied.LeaseToken != "" {
		t.Fatalf("unexpected denied view: %#v", denied)
	}
	listedActive, err := service.ListReceiverLeases(context.Background())
	if err != nil {
		t.Fatalf("list active leases: %v", err)
	}
	if len(listedActive.Leases) != 1 || listedActive.Leases[0].Acquired != nil {
		t.Fatalf("list response should omit acquired control result: %#v", listedActive.Leases)
	}

	renewed, err := service.RenewReceiverLease(context.Background(), command.RenewReceiverLeaseCommand{
		ReceiverID: acquired.ReceiverID,
		HolderID:   "python:1",
		LeaseToken: acquired.LeaseToken,
		TTLSeconds: 120,
		Timestamp:  now.Add(20 * time.Second),
	})
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if renewed.Acquired == nil || !*renewed.Acquired || renewed.LeaseToken != acquired.LeaseToken {
		t.Fatalf("unexpected renewed view: %#v", renewed)
	}

	released, err := service.ReleaseReceiverLease(context.Background(), command.ReleaseReceiverLeaseCommand{
		ReceiverID: acquired.ReceiverID,
		HolderID:   "python:1",
		LeaseToken: acquired.LeaseToken,
		Timestamp:  now.Add(30 * time.Second),
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if released.Active {
		t.Fatalf("release should return inactive view: %#v", released)
	}
	listed, err := service.ListReceiverLeases(context.Background())
	if err != nil {
		t.Fatalf("list leases: %v", err)
	}
	if listed.Totals["leases"] != 0 || listed.Totals["active"] != 0 {
		t.Fatalf("unexpected lease totals: %#v", listed.Totals)
	}
}

func TestReceiverStatusServiceLoadsFileBackedLeases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receiver-leases.json")
	store, err := receiverleasestore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewReceiverStatusServiceWithRepositories(context.Background(), nil, store, 0)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 5, 31, 9, 0, 0, 0, time.UTC)
	acquired, err := service.AcquireReceiverLease(context.Background(), command.AcquireReceiverLeaseCommand{
		Kind:        "telegram",
		ChannelName: "telegram",
		AccountID:   "7689386159",
		HolderID:    "python:1",
		TTLSeconds:  120,
		Timestamp:   now,
	})
	if err != nil {
		t.Fatal(err)
	}

	reopenedStore, err := receiverleasestore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := NewReceiverStatusServiceWithRepositories(context.Background(), nil, reopenedStore, 0)
	if err != nil {
		t.Fatal(err)
	}
	renewed, err := reopened.RenewReceiverLease(context.Background(), command.RenewReceiverLeaseCommand{
		ReceiverID: acquired.ReceiverID,
		HolderID:   "python:1",
		LeaseToken: acquired.LeaseToken,
		TTLSeconds: 120,
		Timestamp:  now.Add(30 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if renewed.LeaseToken != acquired.LeaseToken || renewed.Active != true {
		t.Fatalf("expected persisted lease to renew after reopen: %#v", renewed)
	}
}

func TestReceiverStatusServiceRejectsExpiredReceiverLeaseRenewal(t *testing.T) {
	service := NewReceiverStatusService()
	now := time.Date(2026, 5, 31, 9, 0, 0, 0, time.UTC)
	acquired, err := service.AcquireReceiverLease(context.Background(), command.AcquireReceiverLeaseCommand{
		Kind:        "telegram",
		ChannelName: "telegram",
		AccountID:   "7689386159",
		HolderID:    "python:1",
		TTLSeconds:  30,
		Timestamp:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.RenewReceiverLease(context.Background(), command.RenewReceiverLeaseCommand{
		ReceiverID: acquired.ReceiverID,
		HolderID:   "python:1",
		LeaseToken: acquired.LeaseToken,
		TTLSeconds: 120,
		Timestamp:  now.Add(31 * time.Second),
	})
	if err == nil || err.Error() != "receiver lease expired" {
		t.Fatalf("expected expired lease error, got %v", err)
	}
}
