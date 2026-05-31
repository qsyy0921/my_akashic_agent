package assembler

import (
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func ToSchedulerExecutionLeaseView(lease model.SchedulerExecutionLease, now time.Time, includeToken bool) query.SchedulerExecutionLeaseView {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	view := query.SchedulerExecutionLeaseView{
		JobID:             lease.JobID,
		HolderID:          lease.HolderID,
		LeaseTokenPresent: lease.LeaseToken != "",
		Active:            lease.ActiveAt(now),
		ExpiresAt:         lease.ExpiresAt.UTC().Format(time.RFC3339Nano),
		AcquiredAt:        lease.AcquiredAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         lease.UpdatedAt.UTC().Format(time.RFC3339Nano),
		Metadata:          cloneAssemblerStringMap(lease.Metadata),
		SideEffect:        "runtime_state_write",
	}
	if includeToken {
		view.LeaseToken = lease.LeaseToken
	}
	return view
}

func ToSchedulerExecutionLeaseViews(items []model.SchedulerExecutionLease, now time.Time) []query.SchedulerExecutionLeaseView {
	views := make([]query.SchedulerExecutionLeaseView, 0, len(items))
	for _, item := range items {
		view := ToSchedulerExecutionLeaseView(item, now, false)
		view.SideEffect = ""
		views = append(views, view)
	}
	return views
}

func cloneAssemblerStringMap(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(items))
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}
