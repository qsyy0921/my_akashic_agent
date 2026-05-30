package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToReceiverStatusView(status model.ReceiverStatus) query.ReceiverStatusView {
	return query.ReceiverStatusView{
		ReceiverID:  status.ReceiverID,
		Kind:        string(status.Kind),
		ChannelName: status.ChannelName,
		AccountID:   status.AccountID,
		Endpoint:    status.Endpoint,
		Status:      string(status.Status),
		Reason:      status.Reason,
		LastError:   status.LastError,
		Source:      status.Source,
		Metadata:    cloneObserveTargetMap(status.Metadata),
		UpdatedAt:   status.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func ToReceiverStatusViews(items []model.ReceiverStatus) []query.ReceiverStatusView {
	sorted := model.SortedReceiverStatuses(items)
	views := make([]query.ReceiverStatusView, 0, len(sorted))
	for _, item := range sorted {
		views = append(views, ToReceiverStatusView(item))
	}
	return views
}
