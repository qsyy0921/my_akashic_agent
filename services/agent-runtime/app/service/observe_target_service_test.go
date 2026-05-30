package service

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
)

func TestObserveTargetServiceSyncsSourceBoundTargets(t *testing.T) {
	service := NewObserveTargetService()
	now := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)

	view, err := service.SyncObserveTargets(context.Background(), command.SyncObserveTargetsCommand{
		Source:    "python_config",
		Timestamp: now,
		Targets: []command.ObserveTargetCommand{{
			Channel: command.ChannelCommand{
				Kind:             "qq",
				AccountID:        "1049511700",
				ConversationID:   "27234224",
				ConversationType: "group",
			},
			ObserveOnly:  true,
			ReplyAllowed: true,
			RequireAt:    false,
			AllowFrom:    []string{"2948770636", "2948770636"},
			Enabled:      true,
			Metadata:     map[string]string{"channel_name": "qq"},
		}},
	})
	if err != nil {
		t.Fatalf("sync observe targets: %v", err)
	}
	if view.Totals["targets"] != 1 || view.Totals["observe_only"] != 1 {
		t.Fatalf("unexpected totals: %#v", view.Totals)
	}
	if view.Targets[0].TargetID != "qq:1049511700:group:27234224" {
		t.Fatalf("unexpected generated target id: %#v", view.Targets[0])
	}
	if view.Targets[0].ReplyAllowed {
		t.Fatalf("observe-only target must not allow replies: %#v", view.Targets[0])
	}
	if len(view.Targets[0].AllowFrom) != 1 {
		t.Fatalf("allow_from should be normalized: %#v", view.Targets[0].AllowFrom)
	}

	manual, err := service.SyncObserveTargets(context.Background(), command.SyncObserveTargetsCommand{
		Source:    "manual",
		Timestamp: now.Add(time.Minute),
		Targets: []command.ObserveTargetCommand{{
			TargetID: "manual:telegram:group:123",
			Channel: command.ChannelCommand{
				Kind:             "telegram",
				AccountID:        "telegram",
				ConversationID:   "123",
				ConversationType: "group",
			},
			ReplyAllowed: true,
			Enabled:      true,
		}},
	})
	if err != nil {
		t.Fatalf("sync manual observe target: %v", err)
	}
	if manual.Totals["targets"] != 2 {
		t.Fatalf("manual sync should preserve python source targets: %#v", manual.Totals)
	}

	replaced, err := service.SyncObserveTargets(context.Background(), command.SyncObserveTargetsCommand{
		Source:    "python_config",
		Timestamp: now.Add(2 * time.Minute),
		Targets: []command.ObserveTargetCommand{{
			Channel: command.ChannelCommand{
				Kind:             "qq",
				AccountID:        "1049511700",
				ConversationID:   "3219982",
				ConversationType: "group",
			},
			ObserveOnly: true,
			Enabled:     true,
		}},
	})
	if err != nil {
		t.Fatalf("replace python source observe targets: %v", err)
	}
	if replaced.Totals["targets"] != 2 || replaced.Totals["qq"] != 1 || replaced.Totals["telegram"] != 1 {
		t.Fatalf("source-bound replace kept wrong targets: %#v", replaced.Totals)
	}
	if replaced.SideEffect != "none" {
		t.Fatalf("observe target sync must declare side effect none: %#v", replaced)
	}
}
