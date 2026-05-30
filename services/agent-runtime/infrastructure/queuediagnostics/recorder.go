package queuediagnostics

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type SubjectResolver struct {
	OutboxDelivery func(model.OutboxDelivery) string
	AgentJob       func(model.AgentJob) string
}

type Recorder struct {
	inner    outport.WorkQueuePublisher
	subjects SubjectResolver

	mu       sync.Mutex
	workKind map[string]*kindStats
	subject  map[string]*subjectStats
}

type kindStats struct {
	workKind        string
	attempts        int
	succeeded       int
	failed          int
	lastPublishedAt time.Time
	lastFailedAt    time.Time
	lastError       string
}

type subjectStats struct {
	subject         string
	workKind        string
	attempts        int
	succeeded       int
	failed          int
	lastPublishedAt time.Time
	lastFailedAt    time.Time
	lastError       string
}

func NewRecorder(inner outport.WorkQueuePublisher, subjects SubjectResolver) (*Recorder, error) {
	if inner == nil {
		return nil, errors.New("queue diagnostics recorder requires publisher")
	}
	if subjects.OutboxDelivery == nil || subjects.AgentJob == nil {
		return nil, errors.New("queue diagnostics recorder requires subject resolvers")
	}
	return &Recorder{
		inner:    inner,
		subjects: subjects,
		workKind: make(map[string]*kindStats),
		subject:  make(map[string]*subjectStats),
	}, nil
}

func (r *Recorder) PublishOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error {
	subject := r.subjects.OutboxDelivery(delivery)
	err := r.inner.PublishOutboxDelivery(ctx, delivery)
	r.record("outbox_delivery", subject, err, time.Now().UTC())
	return err
}

func (r *Recorder) PublishAgentJob(ctx context.Context, job model.AgentJob) error {
	subject := r.subjects.AgentJob(job)
	err := r.inner.PublishAgentJob(ctx, job)
	r.record("agent_job", subject, err, time.Now().UTC())
	return err
}

func (r *Recorder) SnapshotWorkQueuePublishDiagnostics(ctx context.Context) (query.QueueShadowPublishDiagnostics, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueShadowPublishDiagnostics{}, err
	}
	if r == nil {
		return query.QueueShadowPublishDiagnostics{}, errors.New("queue diagnostics recorder is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	diagnostics := query.QueueShadowPublishDiagnostics{Enabled: true}
	for _, stats := range r.workKind {
		item := query.QueuePublishWorkKindStats{
			WorkKind:        stats.workKind,
			AttemptCount:    stats.attempts,
			SucceededCount:  stats.succeeded,
			FailedCount:     stats.failed,
			LastPublishedAt: formatTime(stats.lastPublishedAt),
			LastFailedAt:    formatTime(stats.lastFailedAt),
			LastError:       stats.lastError,
		}
		diagnostics.WorkKinds = append(diagnostics.WorkKinds, item)
		diagnostics.AttemptTotal += stats.attempts
		diagnostics.SucceededTotal += stats.succeeded
		diagnostics.FailedTotal += stats.failed
		diagnostics.LastPublishedAt = later(diagnostics.LastPublishedAt, item.LastPublishedAt)
		diagnostics.LastFailedAt = later(diagnostics.LastFailedAt, item.LastFailedAt)
		if item.LastError != "" {
			diagnostics.LastError = item.LastError
		}
	}
	for _, stats := range r.subject {
		diagnostics.Subjects = append(diagnostics.Subjects, query.QueuePublishSubjectStats{
			Subject:         stats.subject,
			WorkKind:        stats.workKind,
			AttemptCount:    stats.attempts,
			SucceededCount:  stats.succeeded,
			FailedCount:     stats.failed,
			LastPublishedAt: formatTime(stats.lastPublishedAt),
			LastFailedAt:    formatTime(stats.lastFailedAt),
			LastError:       stats.lastError,
		})
	}
	sort.Slice(diagnostics.WorkKinds, func(i, j int) bool {
		return diagnostics.WorkKinds[i].WorkKind < diagnostics.WorkKinds[j].WorkKind
	})
	sort.Slice(diagnostics.Subjects, func(i, j int) bool {
		return diagnostics.Subjects[i].Subject < diagnostics.Subjects[j].Subject
	})
	return diagnostics, nil
}

func (r *Recorder) record(workKind string, subject string, err error, now time.Time) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	kind := r.workKind[workKind]
	if kind == nil {
		kind = &kindStats{workKind: workKind}
		r.workKind[workKind] = kind
	}
	subjectItem := r.subject[subject]
	if subjectItem == nil {
		subjectItem = &subjectStats{subject: subject, workKind: workKind}
		r.subject[subject] = subjectItem
	}

	kind.attempts++
	subjectItem.attempts++
	if err != nil {
		kind.failed++
		kind.lastFailedAt = now
		kind.lastError = err.Error()
		subjectItem.failed++
		subjectItem.lastFailedAt = now
		subjectItem.lastError = err.Error()
		return
	}
	kind.succeeded++
	kind.lastPublishedAt = now
	subjectItem.succeeded++
	subjectItem.lastPublishedAt = now
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func later(left string, right string) string {
	if right == "" {
		return left
	}
	if left == "" || right > left {
		return right
	}
	return left
}

var _ outport.WorkQueuePublisher = (*Recorder)(nil)
var _ outport.WorkQueuePublishDiagnosticReader = (*Recorder)(nil)
