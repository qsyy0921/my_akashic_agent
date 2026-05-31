package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type SchedulerJobService struct {
	mu              sync.RWMutex
	repository      outport.SchedulerJobRepository
	leaseRepository outport.SchedulerExecutionLeaseRepository
	leases          map[string]model.SchedulerExecutionLease
}

func NewSchedulerJobService(repository outport.SchedulerJobRepository) *SchedulerJobService {
	return &SchedulerJobService{
		repository: repository,
		leases:     make(map[string]model.SchedulerExecutionLease),
	}
}

func NewSchedulerJobServiceWithLeaseRepository(
	ctx context.Context,
	repository outport.SchedulerJobRepository,
	leaseRepository outport.SchedulerExecutionLeaseRepository,
) (*SchedulerJobService, error) {
	service := NewSchedulerJobService(repository)
	service.leaseRepository = leaseRepository
	if leaseRepository == nil {
		return service, nil
	}
	items, err := leaseRepository.ListSchedulerExecutionLeases(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := item.Validate(); err != nil {
			continue
		}
		service.leases[item.JobID] = item
	}
	return service, nil
}

func (s *SchedulerJobService) ReplaceSchedulerJobs(ctx context.Context, cmd command.ReplaceSchedulerJobsCommand) (query.SchedulerJobSnapshotView, error) {
	if s == nil || s.repository == nil {
		return query.SchedulerJobSnapshotView{}, errors.New("scheduler job service requires repository")
	}
	jobs := make([]model.SchedulerJob, 0, len(cmd.Jobs))
	for _, item := range cmd.Jobs {
		job, err := schedulerJobFromCommand(item)
		if err != nil {
			return query.SchedulerJobSnapshotView{}, err
		}
		jobs = append(jobs, job)
	}
	if err := s.repository.ReplaceSchedulerJobs(ctx, jobs); err != nil {
		return query.SchedulerJobSnapshotView{}, err
	}
	return query.SchedulerJobSnapshotView{
		Count:      len(jobs),
		Source:     strings.TrimSpace(cmd.Source),
		SideEffect: "runtime_state_write",
	}, nil
}

func (s *SchedulerJobService) UpsertSchedulerJob(ctx context.Context, cmd command.UpsertSchedulerJobCommand) (query.SchedulerJobMutationView, error) {
	if s == nil || s.repository == nil {
		return query.SchedulerJobMutationView{}, errors.New("scheduler job service requires repository")
	}
	job, err := schedulerJobFromCommand(cmd.Job)
	if err != nil {
		return query.SchedulerJobMutationView{}, err
	}
	created, err := s.repository.UpsertSchedulerJob(ctx, job)
	if err != nil {
		return query.SchedulerJobMutationView{}, err
	}
	jobView := assembler.ToSchedulerJobView(job)
	return query.SchedulerJobMutationView{
		Job:        &jobView,
		JobID:      job.ID,
		Source:     strings.TrimSpace(cmd.Source),
		Created:    boolPtr(created),
		Deleted:    false,
		SideEffect: "runtime_state_write",
	}, nil
}

func (s *SchedulerJobService) DeleteSchedulerJob(ctx context.Context, cmd command.DeleteSchedulerJobCommand) (query.SchedulerJobMutationView, error) {
	if s == nil || s.repository == nil {
		return query.SchedulerJobMutationView{}, errors.New("scheduler job service requires repository")
	}
	jobID := strings.TrimSpace(cmd.ID)
	if jobID == "" {
		return query.SchedulerJobMutationView{}, errors.New("scheduler job delete requires id")
	}
	found, err := s.repository.DeleteSchedulerJob(ctx, jobID)
	if err != nil {
		return query.SchedulerJobMutationView{}, err
	}
	return query.SchedulerJobMutationView{
		JobID:      jobID,
		Source:     strings.TrimSpace(cmd.Source),
		Found:      boolPtr(found),
		Deleted:    found,
		SideEffect: "runtime_state_write",
	}, nil
}

func (s *SchedulerJobService) CompleteSchedulerJob(ctx context.Context, cmd command.CompleteSchedulerJobCommand) (query.SchedulerJobCompletionView, error) {
	if err := ctx.Err(); err != nil {
		return query.SchedulerJobCompletionView{}, err
	}
	if s == nil || s.repository == nil {
		return query.SchedulerJobCompletionView{}, errors.New("scheduler job service requires repository")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	jobID := strings.TrimSpace(cmd.ID)
	holderID := strings.TrimSpace(cmd.HolderID)
	leaseToken := strings.TrimSpace(cmd.LeaseToken)
	action := strings.TrimSpace(cmd.Action)
	if jobID == "" {
		return query.SchedulerJobCompletionView{}, errors.New("scheduler job complete requires id")
	}
	if holderID == "" {
		return query.SchedulerJobCompletionView{}, errors.New("scheduler job complete requires holder_id")
	}
	if leaseToken == "" {
		return query.SchedulerJobCompletionView{}, errors.New("scheduler job complete requires lease_token")
	}
	if action != "reschedule" && action != "delete" {
		return query.SchedulerJobCompletionView{}, errors.New("scheduler job complete action must be reschedule or delete")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureSchedulerLeaseMapLocked()
	current, ok := s.leases[jobID]
	if !ok {
		return query.SchedulerJobCompletionView{}, errors.New("scheduler execution lease not found")
	}
	if !current.ActiveAt(now) {
		return query.SchedulerJobCompletionView{}, errors.New("scheduler execution lease expired")
	}
	if !current.Matches(holderID, leaseToken) {
		return query.SchedulerJobCompletionView{}, errors.New("scheduler execution lease token mismatch")
	}

	var jobView *query.SchedulerJobView
	deleted := false
	switch action {
	case "reschedule":
		job, err := schedulerJobFromCommand(cmd.Job)
		if err != nil {
			return query.SchedulerJobCompletionView{}, err
		}
		if job.ID != jobID {
			return query.SchedulerJobCompletionView{}, errors.New("scheduler job complete path id does not match job id")
		}
		if _, err := s.repository.UpsertSchedulerJob(ctx, job); err != nil {
			return query.SchedulerJobCompletionView{}, err
		}
		view := assembler.ToSchedulerJobView(job)
		jobView = &view
	case "delete":
		if _, err := s.repository.DeleteSchedulerJob(ctx, jobID); err != nil {
			return query.SchedulerJobCompletionView{}, err
		}
		deleted = true
	}

	if s.leaseRepository != nil {
		if err := s.leaseRepository.DeleteSchedulerExecutionLease(ctx, jobID); err != nil {
			return query.SchedulerJobCompletionView{}, err
		}
	}
	delete(s.leases, jobID)
	return query.SchedulerJobCompletionView{
		Job:           jobView,
		JobID:         jobID,
		Source:        strings.TrimSpace(cmd.Source),
		Action:        action,
		Deleted:       deleted,
		LeaseReleased: true,
		SideEffect:    "runtime_state_write",
	}, nil
}

func (s *SchedulerJobService) ListSchedulerJobs(ctx context.Context) ([]query.SchedulerJobView, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("scheduler job service requires repository")
	}
	jobs, err := s.repository.ListSchedulerJobs(ctx)
	if err != nil {
		return nil, err
	}
	return assembler.ToSchedulerJobViews(jobs), nil
}

func schedulerJobFromCommand(item command.SchedulerJobCommand) (model.SchedulerJob, error) {
	job := model.SchedulerJob{
		ID:              strings.TrimSpace(item.ID),
		Trigger:         strings.TrimSpace(item.Trigger),
		Tier:            strings.TrimSpace(item.Tier),
		FireAt:          utcOrZero(item.FireAt),
		Channel:         strings.TrimSpace(item.Channel),
		ChatID:          strings.TrimSpace(item.ChatID),
		IntervalSeconds: item.IntervalSeconds,
		CronExpr:        strings.TrimSpace(item.CronExpr),
		Message:         item.Message,
		Prompt:          item.Prompt,
		Name:            strings.TrimSpace(item.Name),
		Timezone:        timezoneOrUTC(item.Timezone),
		CreatedAt:       createdAtOrNow(item.CreatedAt),
		RunCount:        item.RunCount,
		Enabled:         item.Enabled,
	}
	if err := job.Validate(); err != nil {
		return model.SchedulerJob{}, err
	}
	return job, nil
}

func (s *SchedulerJobService) GetSchedulerJobDiagnostics(ctx context.Context, filter query.SchedulerJobDiagnosticsFilter) (query.SchedulerJobDiagnosticsView, error) {
	if s == nil || s.repository == nil {
		return query.SchedulerJobDiagnosticsView{}, errors.New("scheduler job service requires repository")
	}
	jobs, err := s.repository.ListSchedulerJobs(ctx)
	if err != nil {
		return query.SchedulerJobDiagnosticsView{}, err
	}
	return schedulerJobDiagnostics(jobs, filter), nil
}

func (s *SchedulerJobService) AcquireSchedulerExecutionLease(ctx context.Context, cmd command.AcquireSchedulerExecutionLeaseCommand) (query.SchedulerExecutionLeaseView, error) {
	if err := ctx.Err(); err != nil {
		return query.SchedulerExecutionLeaseView{}, err
	}
	if s == nil {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler job service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	jobID := strings.TrimSpace(cmd.JobID)
	holderID := strings.TrimSpace(cmd.HolderID)
	if jobID == "" {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease requires job_id")
	}
	if holderID == "" {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease requires holder_id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureSchedulerLeaseMapLocked()
	if current, ok := s.leases[jobID]; ok && current.ActiveAt(now) && current.HolderID != holderID {
		view := assembler.ToSchedulerExecutionLeaseView(current, now, false)
		view.Acquired = boolPtr(false)
		view.DeniedReason = "active_lease_held"
		view.SideEffect = "none"
		return view, nil
	}
	token := ""
	acquiredAt := now
	if current, ok := s.leases[jobID]; ok && current.ActiveAt(now) && current.HolderID == holderID {
		token = current.LeaseToken
		acquiredAt = current.AcquiredAt
	} else {
		var err error
		token, err = randomSchedulerExecutionLeaseToken()
		if err != nil {
			return query.SchedulerExecutionLeaseView{}, err
		}
	}
	lease, err := model.NewSchedulerExecutionLease(model.SchedulerExecutionLeaseSpec{
		JobID:      jobID,
		HolderID:   holderID,
		LeaseToken: token,
		ExpiresAt:  now.Add(schedulerExecutionLeaseTTL(cmd.TTLSeconds)),
		AcquiredAt: acquiredAt,
		UpdatedAt:  now,
		Metadata:   cmd.Metadata,
	})
	if err != nil {
		return query.SchedulerExecutionLeaseView{}, err
	}
	if s.leaseRepository != nil {
		if err := s.leaseRepository.SaveSchedulerExecutionLease(ctx, lease); err != nil {
			return query.SchedulerExecutionLeaseView{}, err
		}
	}
	s.leases[jobID] = lease
	view := assembler.ToSchedulerExecutionLeaseView(lease, now, true)
	view.Acquired = boolPtr(true)
	return view, nil
}

func (s *SchedulerJobService) RenewSchedulerExecutionLease(ctx context.Context, cmd command.RenewSchedulerExecutionLeaseCommand) (query.SchedulerExecutionLeaseView, error) {
	if err := ctx.Err(); err != nil {
		return query.SchedulerExecutionLeaseView{}, err
	}
	if s == nil {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler job service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	jobID := strings.TrimSpace(cmd.JobID)
	if jobID == "" {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease renew requires job_id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureSchedulerLeaseMapLocked()
	current, ok := s.leases[jobID]
	if !ok {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease not found")
	}
	if !current.ActiveAt(now) {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease expired")
	}
	if !current.Matches(cmd.HolderID, cmd.LeaseToken) {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease token mismatch")
	}
	renewed, err := current.Renew(now.Add(schedulerExecutionLeaseTTL(cmd.TTLSeconds)), now)
	if err != nil {
		return query.SchedulerExecutionLeaseView{}, err
	}
	if s.leaseRepository != nil {
		if err := s.leaseRepository.SaveSchedulerExecutionLease(ctx, renewed); err != nil {
			return query.SchedulerExecutionLeaseView{}, err
		}
	}
	s.leases[jobID] = renewed
	view := assembler.ToSchedulerExecutionLeaseView(renewed, now, true)
	view.Acquired = boolPtr(true)
	return view, nil
}

func (s *SchedulerJobService) ReleaseSchedulerExecutionLease(ctx context.Context, cmd command.ReleaseSchedulerExecutionLeaseCommand) (query.SchedulerExecutionLeaseView, error) {
	if err := ctx.Err(); err != nil {
		return query.SchedulerExecutionLeaseView{}, err
	}
	if s == nil {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler job service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	jobID := strings.TrimSpace(cmd.JobID)
	if jobID == "" {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease release requires job_id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureSchedulerLeaseMapLocked()
	current, ok := s.leases[jobID]
	if !ok {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease not found")
	}
	if !current.Matches(cmd.HolderID, cmd.LeaseToken) {
		return query.SchedulerExecutionLeaseView{}, errors.New("scheduler execution lease token mismatch")
	}
	if s.leaseRepository != nil {
		if err := s.leaseRepository.DeleteSchedulerExecutionLease(ctx, jobID); err != nil {
			return query.SchedulerExecutionLeaseView{}, err
		}
	}
	delete(s.leases, jobID)
	view := assembler.ToSchedulerExecutionLeaseView(current, now, false)
	view.Active = false
	view.Acquired = boolPtr(false)
	return view, nil
}

func (s *SchedulerJobService) ListSchedulerExecutionLeases(ctx context.Context) (query.SchedulerExecutionLeasesView, error) {
	if err := ctx.Err(); err != nil {
		return query.SchedulerExecutionLeasesView{}, err
	}
	if s == nil {
		return query.SchedulerExecutionLeasesView{}, errors.New("scheduler job service is nil")
	}
	now := time.Now().UTC()
	s.mu.RLock()
	items := s.schedulerExecutionLeaseSnapshotLocked()
	s.mu.RUnlock()
	return schedulerExecutionLeasesView(items, now), nil
}

func createdAtOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func utcOrZero(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC()
}

func timezoneOrUTC(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "UTC"
	}
	return value
}

func schedulerJobDiagnostics(jobs []model.SchedulerJob, filter query.SchedulerJobDiagnosticsFilter) query.SchedulerJobDiagnosticsView {
	now := time.Now().UTC()
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(filter.Timestamp)); err == nil {
		now = parsed.UTC()
	}
	dueSoonSeconds := filter.DueSoonSeconds
	if dueSoonSeconds <= 0 {
		dueSoonSeconds = 5 * 60
	}
	if dueSoonSeconds > 24*60*60 {
		dueSoonSeconds = 24 * 60 * 60
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	items := append([]model.SchedulerJob(nil), jobs...)
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.FireAt.Equal(right.FireAt) {
			return left.ID < right.ID
		}
		if left.FireAt.IsZero() {
			return false
		}
		if right.FireAt.IsZero() {
			return true
		}
		return left.FireAt.Before(right.FireAt)
	})

	view := query.SchedulerJobDiagnosticsView{
		JobsByTrigger:  map[string]int{"at": 0, "after": 0, "every": 0},
		JobsByTier:     map[string]int{"instant": 0, "soft": 0},
		JobsByChannel:  make(map[string]int),
		JobsByStatus:   map[string]int{"disabled": 0, "overdue": 0, "due_soon": 0, "future": 0},
		Recent:         make([]query.SchedulerJobDiagnosticSampleView, 0, minSchedulerJobDiagnosticLimit(limit, len(items))),
		DueSoonSeconds: dueSoonSeconds,
		SideEffect:     "none",
	}

	for _, job := range items {
		status, dueIn, overdueBy := schedulerJobTimingStatus(job, now, time.Duration(dueSoonSeconds)*time.Second)
		view.SampledJobs++
		if job.Enabled {
			view.EnabledJobs++
		} else {
			view.DisabledJobs++
		}
		if status == "overdue" {
			view.OverdueJobs++
		}
		if status == "due_soon" {
			view.DueSoonJobs++
		}
		if job.Tier == "instant" {
			view.InstantJobs++
		}
		if job.Tier == "soft" {
			view.SoftJobs++
		}
		view.JobsByTrigger[job.Trigger]++
		view.JobsByTier[job.Tier]++
		if strings.TrimSpace(job.Channel) != "" {
			view.JobsByChannel[job.Channel]++
		}
		view.JobsByStatus[status]++
		if view.NextFireAt == "" && job.Enabled && !job.FireAt.IsZero() {
			view.NextFireAt = job.FireAt.UTC().Format(time.RFC3339Nano)
		}
		if len(view.Recent) < limit {
			sample := query.SchedulerJobDiagnosticSampleView{
				ID:       job.ID,
				Trigger:  job.Trigger,
				Tier:     job.Tier,
				Channel:  job.Channel,
				ChatID:   job.ChatID,
				FireAt:   job.FireAt.UTC().Format(time.RFC3339Nano),
				Status:   status,
				RunCount: job.RunCount,
				Enabled:  job.Enabled,
			}
			if dueIn > 0 {
				sample.DueIn = dueIn
			}
			if overdueBy > 0 {
				sample.OverdueBy = overdueBy
			}
			view.Recent = append(view.Recent, sample)
		}
	}
	return view
}

func schedulerJobTimingStatus(job model.SchedulerJob, now time.Time, dueSoonWindow time.Duration) (string, int, int) {
	if !job.Enabled {
		return "disabled", 0, 0
	}
	if job.FireAt.IsZero() {
		return "future", 0, 0
	}
	fireAt := job.FireAt.UTC()
	if fireAt.Before(now) || fireAt.Equal(now) {
		return "overdue", 0, int(now.Sub(fireAt).Seconds())
	}
	dueIn := int(fireAt.Sub(now).Seconds())
	if fireAt.Before(now.Add(dueSoonWindow)) || fireAt.Equal(now.Add(dueSoonWindow)) {
		return "due_soon", dueIn, 0
	}
	return "future", dueIn, 0
}

func minSchedulerJobDiagnosticLimit(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func (s *SchedulerJobService) ensureSchedulerLeaseMapLocked() {
	if s.leases == nil {
		s.leases = make(map[string]model.SchedulerExecutionLease)
	}
}

func (s *SchedulerJobService) schedulerExecutionLeaseSnapshotLocked() []model.SchedulerExecutionLease {
	items := make([]model.SchedulerExecutionLease, 0, len(s.leases))
	for _, item := range s.leases {
		items = append(items, item)
	}
	return model.SortedSchedulerExecutionLeases(items)
}

func schedulerExecutionLeasesView(items []model.SchedulerExecutionLease, now time.Time) query.SchedulerExecutionLeasesView {
	totals := map[string]int{
		"leases":  len(items),
		"active":  0,
		"expired": 0,
	}
	for _, item := range items {
		if item.ActiveAt(now) {
			totals["active"]++
		} else {
			totals["expired"]++
		}
	}
	return query.SchedulerExecutionLeasesView{
		Leases:     assembler.ToSchedulerExecutionLeaseViews(items, now),
		Totals:     totals,
		Notes:      []string{"side_effect=runtime_state_only"},
		SideEffect: "runtime_state_only",
	}
}

func schedulerExecutionLeaseTTL(ttlSeconds int) time.Duration {
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	if ttlSeconds < 30 {
		ttlSeconds = 30
	}
	if ttlSeconds > 3600 {
		ttlSeconds = 3600
	}
	return time.Duration(ttlSeconds) * time.Second
}

func randomSchedulerExecutionLeaseToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func boolPtr(value bool) *bool {
	return &value
}
