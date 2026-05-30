package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type SendLedgerManager interface {
	Record(ctx context.Context, cmd command.RecordSendCommand) (query.SendRecordView, error)
	RecentlySent(ctx context.Context, cmd command.CheckRecentSendCommand) (query.RecentSendView, error)
	CheckPrivateEcho(ctx context.Context, cmd command.CheckPrivateEchoCommand) (query.PrivateEchoView, error)
	List(ctx context.Context, filter query.SendRecordFilter) ([]query.SendRecordView, error)
}
