package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToControlMutationAuditView(item model.ControlMutationAudit) query.ControlMutationAuditView {
	return query.ControlMutationAuditView{
		MutationID:  item.MutationID,
		TargetKind:  item.TargetKind,
		TargetID:    item.TargetID,
		Action:      item.Action,
		Status:      string(item.Status),
		OperatorID:  item.OperatorID,
		ApprovalID:  item.ApprovalID,
		Reason:      item.Reason,
		RollbackOf:  item.RollbackOf,
		RollbackRef: item.RollbackRef,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339Nano),
		Metadata:    cloneControlMutationMetadata(item.Metadata),
	}
}

func ToControlMutationAuditViews(items []model.ControlMutationAudit) []query.ControlMutationAuditView {
	views := make([]query.ControlMutationAuditView, 0, len(items))
	for _, item := range items {
		views = append(views, ToControlMutationAuditView(item))
	}
	return views
}

func cloneControlMutationMetadata(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(items))
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}
