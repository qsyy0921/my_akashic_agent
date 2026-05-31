package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ProactiveStateManager interface {
	RecordDelivery(ctx context.Context, cmd command.RecordProactiveDeliveryCommand) (query.ProactiveDeliveryView, error)
	IsDeliveryDuplicate(ctx context.Context, cmd command.CheckProactiveDeliveryDuplicateCommand) (query.ProactiveDuplicateView, error)
	CountDeliveries(ctx context.Context, cmd command.CountProactiveDeliveriesCommand) (query.ProactiveCountView, error)
	ListDeliveries(ctx context.Context, filter query.ProactiveDeliveryFilter) ([]query.ProactiveDeliveryView, error)

	IsItemSeen(ctx context.Context, cmd command.CheckProactiveItemSeenCommand) (query.ProactiveSeenView, error)
	MarkItemsSeen(ctx context.Context, cmd command.MarkProactiveItemsSeenCommand) (query.ProactiveMarkItemsView, error)

	IsRejectionCooled(ctx context.Context, cmd command.CheckProactiveRejectionCooldownCommand) (query.ProactiveRejectionCooldownView, error)
	MarkRejectionCooldown(ctx context.Context, cmd command.MarkProactiveRejectionCooldownCommand) (query.ProactiveMarkItemsView, error)

	RecordContextOnly(ctx context.Context, cmd command.RecordProactiveContextOnlyCommand) (query.ProactiveTimestampView, error)
	LastContextOnly(ctx context.Context, sessionKey string) (query.ProactiveTimestampView, error)
	CountContextOnly(ctx context.Context, cmd command.CountProactiveContextOnlyCommand) (query.ProactiveCountView, error)

	RecordDriftRun(ctx context.Context, cmd command.RecordProactiveDriftRunCommand) (query.ProactiveTimestampView, error)
	LastDriftRun(ctx context.Context, sessionKey string) (query.ProactiveTimestampView, error)
	RecordDriftFinish(ctx context.Context, cmd command.RecordProactiveDriftFinishCommand) (query.ProactiveDriftFinishView, error)
	DriftSummary(ctx context.Context, limit int) (query.ProactiveDriftSummaryView, error)
	DriftSkillState(ctx context.Context, skillName string) (query.ProactiveDriftSkillStateView, error)

	RecordBGContextMain(ctx context.Context, cmd command.RecordProactiveBGContextMainCommand) (query.ProactiveTimestampView, error)
	LastBGContextMain(ctx context.Context) (query.ProactiveTimestampView, error)

	SnapshotAnyActionQuota(ctx context.Context, cmd command.SnapshotProactiveAnyActionQuotaCommand) (query.ProactiveAnyActionQuotaView, error)
	RecordAnyAction(ctx context.Context, cmd command.RecordProactiveAnyActionCommand) (query.ProactiveAnyActionQuotaView, error)

	Cleanup(ctx context.Context, cmd command.CleanupProactiveStateCommand) (query.ProactiveCleanupView, error)
}
