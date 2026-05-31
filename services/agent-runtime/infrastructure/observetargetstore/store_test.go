package observetargetstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/observetargetstore"
)

func TestObserveTargetStorePersistsTargets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "observe-targets.json")
	store, err := observetargetstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	target := newObserveTarget(t, "27234224")
	if err := store.SaveObserveTargets(context.Background(), []model.ObserveTarget{target}); err != nil {
		t.Fatal(err)
	}

	reopened, err := observetargetstore.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items, err := reopened.ListObserveTargets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one persisted target, got %d", len(items))
	}
	if items[0].Channel.ConversationID != "27234224" || !items[0].ObserveOnly {
		t.Fatalf("unexpected target: %#v", items[0])
	}
}

func TestObserveTargetStoreReplacesState(t *testing.T) {
	store, err := observetargetstore.NewStore(filepath.Join(t.TempDir(), "observe-targets.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveObserveTargets(context.Background(), []model.ObserveTarget{
		newObserveTarget(t, "27234224"),
		newObserveTarget(t, "3219982"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveObserveTargets(context.Background(), []model.ObserveTarget{
		newObserveTarget(t, "164369633"),
	}); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListObserveTargets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Channel.ConversationID != "164369633" {
		t.Fatalf("expected replaced target state, got %#v", items)
	}
}

func newObserveTarget(t *testing.T, groupID string) model.ObserveTarget {
	t.Helper()
	target, err := model.NewObserveTarget(model.ObserveTargetSpec{
		Channel: model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "1049511700",
			ConversationID:   groupID,
			ConversationType: model.ConversationTypeGroup,
		},
		ObserveOnly: true,
		Enabled:     true,
		Source:      "python_config",
		Metadata:    map[string]string{"channel_name": "qq"},
	}, time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return target
}
