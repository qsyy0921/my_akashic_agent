package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToReceiverLeaseView(lease model.ReceiverLease, now time.Time, includeToken bool) query.ReceiverLeaseView {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	token := ""
	if includeToken {
		token = lease.LeaseToken
	}
	return query.ReceiverLeaseView{
		ReceiverID:        lease.ReceiverID,
		Kind:              string(lease.Kind),
		ChannelName:       lease.ChannelName,
		AccountID:         lease.AccountID,
		HolderID:          lease.HolderID,
		LeaseToken:        token,
		LeaseTokenPresent: lease.LeaseToken != "",
		Active:            lease.ActiveAt(now),
		ExpiresAt:         lease.ExpiresAt.UTC().Format(time.RFC3339Nano),
		AcquiredAt:        lease.AcquiredAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         lease.UpdatedAt.UTC().Format(time.RFC3339Nano),
		Metadata:          cloneObserveTargetMap(lease.Metadata),
	}
}

func ToReceiverLeaseViews(items []model.ReceiverLease, now time.Time) []query.ReceiverLeaseView {
	views := make([]query.ReceiverLeaseView, 0, len(items))
	for _, item := range items {
		views = append(views, ToReceiverLeaseView(item, now, false))
	}
	return views
}
