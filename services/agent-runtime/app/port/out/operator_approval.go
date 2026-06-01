package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type OperatorApprovalRepository interface {
	SaveOperatorApproval(ctx context.Context, approval model.OperatorApproval) error
	ListOperatorApprovals(ctx context.Context) ([]model.OperatorApproval, error)
}
