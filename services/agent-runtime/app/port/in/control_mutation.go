package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type ControlMutationAuditManager interface {
	RecordControlMutationAudit(ctx context.Context, cmd command.RecordControlMutationAuditCommand) (query.ControlMutationAuditView, error)
	ListControlMutationAudits(ctx context.Context, filter query.ControlMutationAuditFilter) (query.ControlMutationAuditsView, error)
}
