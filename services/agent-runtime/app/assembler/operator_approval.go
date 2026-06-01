package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToOperatorApprovalView(item model.OperatorApproval, now time.Time) query.OperatorApprovalView {
	view := query.OperatorApprovalView{
		ApprovalID: item.ApprovalID,
		TargetKind: item.TargetKind,
		TargetID:   item.TargetID,
		Decision:   string(item.Decision),
		OperatorID: item.OperatorID,
		Reason:     item.Reason,
		Active:     item.ActiveAt(now),
		CreatedAt:  item.CreatedAt.Format(time.RFC3339Nano),
		Metadata:   cloneOperatorApprovalMetadata(item.Metadata),
	}
	if !item.ExpiresAt.IsZero() {
		view.ExpiresAt = item.ExpiresAt.Format(time.RFC3339Nano)
	}
	return view
}

func ToOperatorApprovalViews(items []model.OperatorApproval, now time.Time) []query.OperatorApprovalView {
	views := make([]query.OperatorApprovalView, 0, len(items))
	for _, item := range items {
		views = append(views, ToOperatorApprovalView(item, now))
	}
	return views
}

func cloneOperatorApprovalMetadata(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(items))
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}
