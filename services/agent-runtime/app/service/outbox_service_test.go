package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/memory"
)

func TestOutboxServiceTransitionsAndRetry(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 4, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 2, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}

	service := appservice.NewOutboxService(store, store)
	running, err := service.MarkDispatching(ctx, command.MarkOutboxDispatchingCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	if running.Status != string(model.DeliveryDispatching) || running.Attempts != 1 {
		t.Fatalf("unexpected running state: %+v", running)
	}

	failed, err := service.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      "outbox-1",
		ErrorKind:    string(model.DeliveryErrorPlatformTimeout),
		ErrorMessage: "platform timeout",
		Timestamp:    now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if failed.Status != string(model.DeliveryFailed) {
		t.Fatalf("expected failed, got %s", failed.Status)
	}
	if failed.ErrorKind != string(model.DeliveryErrorPlatformTimeout) {
		t.Fatalf("expected timeout failure kind, got %s", failed.ErrorKind)
	}

	retried, err := service.Retry(ctx, command.RetryOutboxCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if retried.Status != string(model.DeliveryQueued) {
		t.Fatalf("expected queued, got %s", retried.Status)
	}
	if retried.ErrorKind != "" || retried.ErrorMessage != "" {
		t.Fatalf("expected retry to clear failure details, got kind=%q message=%q", retried.ErrorKind, retried.ErrorMessage)
	}
	if len(store.OutboxQueue()) != 1 {
		t.Fatalf("expected retried delivery to be enqueued, got %d", len(store.OutboxQueue()))
	}
}

func TestOutboxServiceLeaseNextMarksDeliveryDispatching(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 5, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 2, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("enqueue delivery: %v", err)
	}

	service := appservice.NewOutboxService(store, store)
	leased, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:   "qq-dispatcher",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.Status != string(model.DeliveryDispatching) {
		t.Fatalf("expected dispatching, got %s", leased.Status)
	}
	if leased.Attempts != 1 {
		t.Fatalf("expected attempts to increment, got %d", leased.Attempts)
	}
	if leased.LeaseOwner != "qq-dispatcher" {
		t.Fatalf("expected lease owner, got %q", leased.LeaseOwner)
	}
	if leased.LeaseExpiresAt == "" {
		t.Fatalf("expected lease expiry")
	}

	_, err = service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:   "qq-dispatcher-2",
		TTLSeconds: 60,
		Timestamp:  now.Add(2 * time.Second),
	})
	if err == nil {
		t.Fatalf("expected no leaseable delivery while lease is active")
	}
}

func TestOutboxServiceLeaseNextSkipsBlockedAccountKeys(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewOutboxService(store, store)
	now := time.Date(2026, 5, 31, 9, 0, 0, 0, time.UTC)

	blockedMessage := sampleOutboxMessage(now)
	blockedMessage.EventID = "outbox-blocked-account"
	blockedMessage.Channel.AccountID = "1049511700"
	blockedMessage.Channel.ConversationID = "2365524513"
	blocked, err := model.NewOutboxDelivery(blockedMessage, 2, now)
	if err != nil {
		t.Fatalf("new blocked delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, blocked); err != nil {
		t.Fatalf("save blocked delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, blocked); err != nil {
		t.Fatalf("enqueue blocked delivery: %v", err)
	}

	allowedMessage := sampleOutboxMessage(now.Add(time.Second))
	allowedMessage.EventID = "outbox-allowed-account"
	allowedMessage.Channel.AccountID = "2365524513"
	allowedMessage.Channel.ConversationID = "1049511700"
	allowed, err := model.NewOutboxDelivery(allowedMessage, 2, allowedMessage.Timestamp)
	if err != nil {
		t.Fatalf("new allowed delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, allowed); err != nil {
		t.Fatalf("save allowed delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, allowed); err != nil {
		t.Fatalf("enqueue allowed delivery: %v", err)
	}

	leased, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:           "qq-dispatcher",
		TTLSeconds:         60,
		Timestamp:          now.Add(2 * time.Second),
		BlockedAccountKeys: []string{"qq:1049511700"},
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.EventID != "outbox-allowed-account" || leased.Channel.AccountID != "2365524513" {
		t.Fatalf("expected allowed account delivery, got %+v", leased)
	}

	stillQueued, err := service.Get(ctx, "outbox-blocked-account")
	if err != nil {
		t.Fatalf("get blocked delivery: %v", err)
	}
	if stillQueued.Status != string(model.DeliveryQueued) || stillQueued.Attempts != 0 {
		t.Fatalf("blocked account delivery should remain queued without attempts, got %+v", stillQueued)
	}
}

func TestOutboxServiceLeaseNextSkipsUnsupportedStepKinds(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewOutboxService(store, store)
	now := time.Date(2026, 6, 2, 13, 0, 0, 0, time.UTC)

	imageMessage := sampleOutboxMessage(now)
	imageMessage.EventID = "outbox-image"
	imageMessage.Content = "caption"
	imageMessage.Attachments = []model.Attachment{{
		Kind: model.AttachmentKindImage,
		URL:  "file:///tmp/smoke.png",
		Name: "smoke.png",
	}}
	imageDelivery, err := model.NewOutboxDelivery(imageMessage, 2, now)
	if err != nil {
		t.Fatalf("new image delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, imageDelivery); err != nil {
		t.Fatalf("save image delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, imageDelivery); err != nil {
		t.Fatalf("enqueue image delivery: %v", err)
	}

	textMessage := sampleOutboxMessage(now.Add(time.Second))
	textMessage.EventID = "outbox-text"
	textDelivery, err := model.NewOutboxDelivery(textMessage, 2, textMessage.Timestamp)
	if err != nil {
		t.Fatalf("new text delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, textDelivery); err != nil {
		t.Fatalf("save text delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, textDelivery); err != nil {
		t.Fatalf("enqueue text delivery: %v", err)
	}

	leased, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:         "qq-dispatcher",
		TTLSeconds:       60,
		Timestamp:        now.Add(2 * time.Second),
		AllowedStepKinds: []string{"text"},
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.EventID != "outbox-text" {
		t.Fatalf("expected text-only lease, got %+v", leased)
	}

	stillQueued, err := service.Get(ctx, "outbox-image")
	if err != nil {
		t.Fatalf("get image delivery: %v", err)
	}
	if stillQueued.Status != string(model.DeliveryQueued) || stillQueued.Attempts != 0 {
		t.Fatalf("unsupported image delivery should remain queued, got %+v", stillQueued)
	}
}

func TestOutboxServiceLeaseNextFiltersByAllowedStepKindsPerAccount(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewOutboxService(store, store)
	now := time.Date(2026, 6, 2, 13, 5, 0, 0, time.UTC)

	firstFileMessage := sampleOutboxMessage(now)
	firstFileMessage.EventID = "outbox-file-first-account"
	firstFileMessage.Content = ""
	firstFileMessage.Attachments = []model.Attachment{{
		Kind: model.AttachmentKindFile,
		URL:  "file:///tmp/first.txt",
		Name: "first.txt",
	}}
	firstFileDelivery, err := model.NewOutboxDelivery(firstFileMessage, 2, now)
	if err != nil {
		t.Fatalf("new first-account file delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, firstFileDelivery); err != nil {
		t.Fatalf("save first-account file delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, firstFileDelivery); err != nil {
		t.Fatalf("enqueue first-account file delivery: %v", err)
	}

	secondFileMessage := sampleOutboxMessage(now.Add(time.Second))
	secondFileMessage.EventID = "outbox-file-second-account"
	secondFileMessage.Content = ""
	secondFileMessage.Channel.AccountID = "2365524513"
	secondFileMessage.Attachments = []model.Attachment{{
		Kind: model.AttachmentKindFile,
		URL:  "file:///tmp/second.txt",
		Name: "second.txt",
	}}
	secondFileDelivery, err := model.NewOutboxDelivery(secondFileMessage, 2, secondFileMessage.Timestamp)
	if err != nil {
		t.Fatalf("new second-account file delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, secondFileDelivery); err != nil {
		t.Fatalf("save second-account file delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, secondFileDelivery); err != nil {
		t.Fatalf("enqueue second-account file delivery: %v", err)
	}

	leased, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:         "qq-dispatcher",
		TTLSeconds:       60,
		Timestamp:        now.Add(2 * time.Second),
		AllowedStepKinds: []string{"text"},
		AllowedStepKindsByAccount: map[string][]string{
			"2365524513": {"text", "file"},
		},
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.EventID != "outbox-file-second-account" {
		t.Fatalf("expected second-account file lease, got %+v", leased)
	}

	stillQueued, err := service.Get(ctx, "outbox-file-first-account")
	if err != nil {
		t.Fatalf("get first-account file delivery: %v", err)
	}
	if stillQueued.Status != string(model.DeliveryQueued) || stillQueued.Attempts != 0 {
		t.Fatalf("first-account file delivery should remain queued, got %+v", stillQueued)
	}
}

func TestOutboxServiceLeaseNextFiltersByAllowedStepKindsPerAccountConversationType(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewOutboxService(store, store)
	now := time.Date(2026, 6, 2, 13, 40, 0, 0, time.UTC)

	firstGroupFileMessage := sampleOutboxMessage(now)
	firstGroupFileMessage.EventID = "outbox-file-first-account-group"
	firstGroupFileMessage.Content = ""
	firstGroupFileMessage.Channel.ConversationType = model.ConversationTypeGroup
	firstGroupFileMessage.Channel.ConversationID = "3219982"
	firstGroupFileMessage.Attachments = []model.Attachment{{
		Kind: model.AttachmentKindFile,
		URL:  "file:///tmp/first-group.txt",
		Name: "first-group.txt",
	}}
	firstGroupFileDelivery, err := model.NewOutboxDelivery(firstGroupFileMessage, 2, now)
	if err != nil {
		t.Fatalf("new first-account group file delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, firstGroupFileDelivery); err != nil {
		t.Fatalf("save first-account group file delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, firstGroupFileDelivery); err != nil {
		t.Fatalf("enqueue first-account group file delivery: %v", err)
	}

	firstPrivateFileMessage := sampleOutboxMessage(now.Add(time.Second))
	firstPrivateFileMessage.EventID = "outbox-file-first-account-private"
	firstPrivateFileMessage.Content = ""
	firstPrivateFileMessage.Channel.ConversationType = model.ConversationTypePrivate
	firstPrivateFileMessage.Channel.ConversationID = "2365524513"
	firstPrivateFileMessage.Attachments = []model.Attachment{{
		Kind: model.AttachmentKindFile,
		URL:  "file:///tmp/first-private.txt",
		Name: "first-private.txt",
	}}
	firstPrivateFileDelivery, err := model.NewOutboxDelivery(firstPrivateFileMessage, 2, firstPrivateFileMessage.Timestamp)
	if err != nil {
		t.Fatalf("new first-account private file delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, firstPrivateFileDelivery); err != nil {
		t.Fatalf("save first-account private file delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, firstPrivateFileDelivery); err != nil {
		t.Fatalf("enqueue first-account private file delivery: %v", err)
	}

	leased, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:         "qq-dispatcher",
		TTLSeconds:       60,
		Timestamp:        now.Add(2 * time.Second),
		AllowedStepKinds: []string{"text"},
		AllowedStepKindsByAccountConversationType: map[string]map[string][]string{
			"1049511700": {
				"private": {"text", "file"},
			},
		},
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.EventID != "outbox-file-first-account-private" {
		t.Fatalf("expected first-account private file lease, got %+v", leased)
	}

	stillQueued, err := service.Get(ctx, "outbox-file-first-account-group")
	if err != nil {
		t.Fatalf("get first-account group file delivery: %v", err)
	}
	if stillQueued.Status != string(model.DeliveryQueued) || stillQueued.Attempts != 0 {
		t.Fatalf("first-account group file delivery should remain queued, got %+v", stillQueued)
	}
}

func TestOutboxServiceLeaseNextFiltersByAllowedStepKindsPerAccountConversationID(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	service := appservice.NewOutboxService(store, store)
	now := time.Date(2026, 6, 2, 14, 5, 0, 0, time.UTC)

	blockedGroupFileMessage := sampleOutboxMessage(now)
	blockedGroupFileMessage.EventID = "outbox-file-first-account-group-blocked"
	blockedGroupFileMessage.Content = ""
	blockedGroupFileMessage.Channel.ConversationType = model.ConversationTypeGroup
	blockedGroupFileMessage.Channel.ConversationID = "3219982"
	blockedGroupFileMessage.Attachments = []model.Attachment{{
		Kind: model.AttachmentKindFile,
		URL:  "file:///tmp/blocked-group.txt",
		Name: "blocked-group.txt",
	}}
	blockedGroupFileDelivery, err := model.NewOutboxDelivery(blockedGroupFileMessage, 2, now)
	if err != nil {
		t.Fatalf("new blocked first-account group file delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, blockedGroupFileDelivery); err != nil {
		t.Fatalf("save blocked first-account group file delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, blockedGroupFileDelivery); err != nil {
		t.Fatalf("enqueue blocked first-account group file delivery: %v", err)
	}

	allowedGroupFileMessage := sampleOutboxMessage(now.Add(time.Second))
	allowedGroupFileMessage.EventID = "outbox-file-first-account-group-allowed"
	allowedGroupFileMessage.Content = ""
	allowedGroupFileMessage.Channel.ConversationType = model.ConversationTypeGroup
	allowedGroupFileMessage.Channel.ConversationID = "391289439"
	allowedGroupFileMessage.Attachments = []model.Attachment{{
		Kind: model.AttachmentKindFile,
		URL:  "file:///tmp/allowed-group.txt",
		Name: "allowed-group.txt",
	}}
	allowedGroupFileDelivery, err := model.NewOutboxDelivery(allowedGroupFileMessage, 2, allowedGroupFileMessage.Timestamp)
	if err != nil {
		t.Fatalf("new allowed first-account group file delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, allowedGroupFileDelivery); err != nil {
		t.Fatalf("save allowed first-account group file delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, allowedGroupFileDelivery); err != nil {
		t.Fatalf("enqueue allowed first-account group file delivery: %v", err)
	}

	leased, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:         "qq-dispatcher",
		TTLSeconds:       60,
		Timestamp:        now.Add(2 * time.Second),
		AllowedStepKinds: []string{"text"},
		AllowedStepKindsByAccountConversationID: map[string]map[string]map[string][]string{
			"1049511700": {
				"group": {
					"391289439": {"text", "file"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if leased.EventID != "outbox-file-first-account-group-allowed" {
		t.Fatalf("expected allowed first-account group file lease, got %+v", leased)
	}

	stillQueued, err := service.Get(ctx, "outbox-file-first-account-group-blocked")
	if err != nil {
		t.Fatalf("get blocked first-account group file delivery: %v", err)
	}
	if stillQueued.Status != string(model.DeliveryQueued) || stillQueued.Attempts != 0 {
		t.Fatalf("blocked first-account group file delivery should remain queued, got %+v", stillQueued)
	}
}

func TestOutboxServiceDeadLettersNonRetryableFailureKind(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 5, 30, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 3, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}

	service := appservice.NewOutboxService(store, store)
	if _, err := service.MarkDispatching(ctx, command.MarkOutboxDispatchingCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	dead, err := service.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      "outbox-1",
		ErrorKind:    string(model.DeliveryErrorRoute),
		ErrorMessage: "route missing",
		Timestamp:    now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if dead.Status != string(model.DeliveryDeadLettered) {
		t.Fatalf("expected dead-lettered, got %s", dead.Status)
	}
	if dead.ErrorKind != string(model.DeliveryErrorRoute) {
		t.Fatalf("expected route error kind, got %s", dead.ErrorKind)
	}
	if _, err := service.Retry(ctx, command.RetryOutboxCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(3 * time.Second),
	}); err == nil {
		t.Fatalf("expected retry to fail for dead-lettered delivery")
	}
}

func TestOutboxServiceRecordsLifecycleEvents(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	now := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	delivery, err := model.NewOutboxDelivery(sampleOutboxMessage(now), 3, now)
	if err != nil {
		t.Fatalf("new delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("save delivery: %v", err)
	}
	if err := store.EnqueueOutboxDelivery(ctx, delivery); err != nil {
		t.Fatalf("enqueue delivery: %v", err)
	}

	service := appservice.NewOutboxServiceWithEvents(store, store, store)
	if _, err := service.LeaseNext(ctx, command.LeaseNextOutboxCommand{
		WorkerID:   "outbox-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease next: %v", err)
	}
	if _, err := service.MarkSucceeded(ctx, command.MarkOutboxSucceededCommand{
		EventID:   "outbox-1",
		Timestamp: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark succeeded: %v", err)
	}

	events, err := store.ListOutboxDeliveryEvents(ctx, query.OutboxDeliveryEventFilter{
		DeliveryID: "outbox-1",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("list outbox events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 outbox events, got %d", len(events))
	}
	if events[0].EventType != model.OutboxDeliveryEventSucceeded || events[1].EventType != model.OutboxDeliveryEventLeased {
		t.Fatalf("unexpected event order/types: %#v", events)
	}
}

func TestOutboxMetricsServiceSummarizesThroughputAndDeadLetters(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	outbox := appservice.NewOutboxServiceWithEvents(store, store, store)
	metrics := appservice.NewOutboxMetricsService(store, store)
	now := time.Date(2026, 5, 30, 6, 30, 0, 0, time.UTC)

	succeededMessage := sampleOutboxMessage(now)
	succeededMessage.EventID = "outbox-metrics-succeeded"
	succeededMessage.Channel.Kind = "telegram"
	succeeded, err := model.NewOutboxDelivery(succeededMessage, 2, now)
	if err != nil {
		t.Fatalf("new succeeded delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, succeeded); err != nil {
		t.Fatalf("save succeeded delivery: %v", err)
	}
	if _, err := outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    "outbox-metrics-succeeded",
		WorkerID:   "outbox-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("lease succeeded delivery: %v", err)
	}
	if _, err := outbox.MarkSucceeded(ctx, command.MarkOutboxSucceededCommand{
		EventID:   "outbox-metrics-succeeded",
		Timestamp: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("mark succeeded: %v", err)
	}

	deadMessage := sampleOutboxMessage(now.Add(3 * time.Second))
	deadMessage.EventID = "outbox-metrics-dead"
	deadMessage.Channel.Kind = "qq"
	dead, err := model.NewOutboxDelivery(deadMessage, 1, now.Add(3*time.Second))
	if err != nil {
		t.Fatalf("new dead-letter delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, dead); err != nil {
		t.Fatalf("save dead-letter delivery: %v", err)
	}
	if _, err := outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    "outbox-metrics-dead",
		WorkerID:   "outbox-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(4 * time.Second),
	}); err != nil {
		t.Fatalf("lease dead-letter delivery: %v", err)
	}
	if _, err := outbox.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      "outbox-metrics-dead",
		ErrorKind:    string(model.DeliveryErrorRoute),
		ErrorMessage: "missing adapter",
		Timestamp:    now.Add(5 * time.Second),
	}); err != nil {
		t.Fatalf("fail dead-letter delivery: %v", err)
	}

	view, err := metrics.Get(ctx, query.OutboxMetricsFilter{
		DeliveryLimit: 10,
		EventLimit:    20,
	})
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	if view.SampledDeliveries != 2 || view.SampledEvents != 4 {
		t.Fatalf("unexpected sample counts: %+v", view)
	}
	if view.DeliveriesByStatus[string(model.DeliverySucceeded)] != 1 ||
		view.DeliveriesByStatus[string(model.DeliveryDeadLettered)] != 1 {
		t.Fatalf("unexpected deliveries by status: %+v", view.DeliveriesByStatus)
	}
	if view.DeliveriesByChannelKind["telegram"].ByStatus[string(model.DeliverySucceeded)] != 1 ||
		view.DeliveriesByChannelKind["qq"].ByStatus[string(model.DeliveryDeadLettered)] != 1 {
		t.Fatalf("unexpected deliveries by channel kind: %+v", view.DeliveriesByChannelKind)
	}
	if view.Throughput.Leased != 2 || view.Throughput.Succeeded != 1 || view.Throughput.Failed != 1 ||
		view.Throughput.DeadLettered != 1 || view.Throughput.TerminalEvents != 2 {
		t.Fatalf("unexpected throughput metrics: %+v", view.Throughput)
	}
	if view.DeadLetters.CurrentTotal != 1 || view.DeadLetters.ByChannelKind["qq"] != 1 {
		t.Fatalf("unexpected dead-letter metrics: %+v", view.DeadLetters)
	}
	if len(view.DeadLetters.Recent) != 1 ||
		view.DeadLetters.Recent[0].DeliveryID != "outbox-metrics-dead" ||
		view.DeadLetters.Recent[0].EventType != string(model.OutboxDeliveryEventFailed) {
		t.Fatalf("unexpected recent dead letters: %+v", view.DeadLetters.Recent)
	}
}

func TestOutboxMetricsServiceSummarizesAccountPressure(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()
	outbox := appservice.NewOutboxServiceWithEvents(store, store, store)
	metrics := appservice.NewOutboxMetricsService(store, store)
	now := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)

	for i := 0; i < 10; i++ {
		message := sampleOutboxMessage(now.Add(time.Duration(i) * time.Second))
		message.EventID = "pressure-high-queued-" + string(rune('a'+i))
		message.Channel.AccountID = "1049511700"
		message.Channel.ConversationID = "2365524513"
		delivery, err := model.NewOutboxDelivery(message, 3, message.Timestamp)
		if err != nil {
			t.Fatalf("new high-pressure delivery %d: %v", i, err)
		}
		if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
			t.Fatalf("save high-pressure delivery %d: %v", i, err)
		}
	}

	for i := 0; i < 2; i++ {
		message := sampleOutboxMessage(now.Add(time.Duration(20+i) * time.Second))
		message.EventID = "pressure-normal-queued-" + string(rune('a'+i))
		message.Channel.AccountID = "2365524513"
		message.Channel.ConversationID = "1049511700"
		delivery, err := model.NewOutboxDelivery(message, 3, message.Timestamp)
		if err != nil {
			t.Fatalf("new normal-pressure delivery %d: %v", i, err)
		}
		if err := store.SaveOutboxDelivery(ctx, delivery); err != nil {
			t.Fatalf("save normal-pressure delivery %d: %v", i, err)
		}
	}

	deadMessage := sampleOutboxMessage(now.Add(30 * time.Second))
	deadMessage.EventID = "pressure-normal-dead"
	deadMessage.Channel.AccountID = "2365524513"
	deadMessage.Channel.ConversationID = "1049511700"
	dead, err := model.NewOutboxDelivery(deadMessage, 1, deadMessage.Timestamp)
	if err != nil {
		t.Fatalf("new dead pressure delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, dead); err != nil {
		t.Fatalf("save dead pressure delivery: %v", err)
	}
	if _, err := outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    deadMessage.EventID,
		WorkerID:   "outbox-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(31 * time.Second),
	}); err != nil {
		t.Fatalf("lease dead pressure delivery: %v", err)
	}
	if _, err := outbox.MarkFailed(ctx, command.MarkOutboxFailedCommand{
		EventID:      deadMessage.EventID,
		ErrorKind:    string(model.DeliveryErrorRoute),
		ErrorMessage: "missing adapter",
		Timestamp:    now.Add(32 * time.Second),
	}); err != nil {
		t.Fatalf("dead-letter pressure delivery: %v", err)
	}

	dispatchingMessage := sampleOutboxMessage(now.Add(40 * time.Second))
	dispatchingMessage.EventID = "pressure-dispatching"
	dispatchingMessage.Channel.AccountID = "2365524513"
	dispatchingMessage.Channel.ConversationID = "1049511700"
	dispatching, err := model.NewOutboxDelivery(dispatchingMessage, 3, dispatchingMessage.Timestamp)
	if err != nil {
		t.Fatalf("new dispatching pressure delivery: %v", err)
	}
	if err := store.SaveOutboxDelivery(ctx, dispatching); err != nil {
		t.Fatalf("save dispatching pressure delivery: %v", err)
	}
	if _, err := outbox.Lease(ctx, command.LeaseOutboxDeliveryCommand{
		EventID:    dispatchingMessage.EventID,
		WorkerID:   "outbox-worker",
		TTLSeconds: 60,
		Timestamp:  now.Add(41 * time.Second),
	}); err != nil {
		t.Fatalf("lease dispatching pressure delivery: %v", err)
	}

	view, err := metrics.Get(ctx, query.OutboxMetricsFilter{
		DeliveryLimit: 50,
		EventLimit:    50,
	})
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	if view.Pressure.Accounts != 2 {
		t.Fatalf("expected 2 pressure accounts, got %+v", view.Pressure)
	}
	if view.Pressure.HighPressureAccounts != 1 {
		t.Fatalf("expected 1 high-pressure account, got %+v", view.Pressure)
	}
	if view.Pressure.MaxActive != 10 || view.Pressure.MaxQueued != 10 {
		t.Fatalf("unexpected pressure max values: %+v", view.Pressure)
	}
	if len(view.Pressure.ByAccount) != 2 {
		t.Fatalf("expected pressure details for 2 accounts, got %+v", view.Pressure.ByAccount)
	}
	high := view.Pressure.ByAccount[0]
	if high.AccountKey != "qq:1049511700" || !high.HighPressure || high.PressureReason != "active>=10" {
		t.Fatalf("unexpected high-pressure account: %+v", high)
	}
	if high.Queued != 10 || high.Dispatching != 0 || high.Active != 10 || high.DeadLettered != 0 {
		t.Fatalf("unexpected high-pressure counts: %+v", high)
	}
	normal := view.Pressure.ByAccount[1]
	if normal.AccountKey != "qq:2365524513" || normal.HighPressure {
		t.Fatalf("unexpected normal-pressure account: %+v", normal)
	}
	if normal.Queued != 2 || normal.Dispatching != 1 || normal.Active != 3 || normal.DeadLettered != 1 {
		t.Fatalf("unexpected normal-pressure counts: %+v", normal)
	}
}

func sampleOutboxMessage(timestamp time.Time) model.OutboundMessage {
	return model.OutboundMessage{
		EventID: "outbox-1",
		Channel: model.ChannelRef{
			Kind:             "qq",
			AccountID:        "1049511700",
			ConversationID:   "2365524513",
			ConversationType: "private",
		},
		Content:   "hello",
		Timestamp: timestamp,
	}
}
