package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ControlMutationAuditRepository interface {
	SaveControlMutationAudit(ctx context.Context, audit model.ControlMutationAudit) error
	ListControlMutationAudits(ctx context.Context) ([]model.ControlMutationAudit, error)
}
