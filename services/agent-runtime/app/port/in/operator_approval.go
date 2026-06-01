package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type OperatorApprovalManager interface {
	RecordOperatorApproval(ctx context.Context, cmd command.RecordOperatorApprovalCommand) (query.OperatorApprovalView, error)
	ListOperatorApprovals(ctx context.Context, filter query.OperatorApprovalFilter) (query.OperatorApprovalsView, error)
}
