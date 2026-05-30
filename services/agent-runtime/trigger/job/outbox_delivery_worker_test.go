package jobtrigger_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	jobtrigger "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/trigger/job"
)

func TestOutboxDeliveryWorkerProcessOnceDispatchesAndMarksSucceeded(t *testing.T) {
	now := time.Date(2026, 5, 31, 4, 0, 0, 0, time.UTC)
	outbox := &fakeOutboxDeliveryManager{
		lease: query.OutboxDeliveryView{EventID: "outbox:1"},
	}
	dispatcher := &fakeOutboxDeliveryDispatcher{
		result: query.DeliveryDispatchResultView{EventID: "outbox:1", StepCount: 2},
	}
	worker, err := jobtrigger.NewOutboxDeliveryWorker(outbox, dispatcher, jobtrigger.OutboxDeliveryWorkerConfig{
		WorkerID:         "worker-1",
		LeaseTTLSeconds:  60,
		ChannelByAccount: map[string]string{"2365524513": "qq_2365524513"},
		Now:              func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	result, err := worker.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once: %v", err)
	}

	if !result.Processed || result.EventID != "outbox:1" || result.DispatchCount != 2 || result.Failed {
		t.Fatalf("unexpected result: %+v", result)
	}
	if outbox.leaseNextCalls != 1 || outbox.markDispatchingCalls != 1 || outbox.markSucceededCalls != 1 || outbox.markFailedCalls != 0 {
		t.Fatalf("unexpected outbox calls: %+v", outbox)
	}
	if outbox.lastLease.WorkerID != "worker-1" || outbox.lastLease.TTLSeconds != 60 || !outbox.lastLease.Timestamp.Equal(now) {
		t.Fatalf("unexpected lease command: %+v", outbox.lastLease)
	}
	if dispatcher.last.EventID != "outbox:1" || dispatcher.last.ChannelByAccount["2365524513"] != "qq_2365524513" {
		t.Fatalf("unexpected dispatch command: %+v", dispatcher.last)
	}
}

func TestOutboxDeliveryWorkerMarksFailedWithDispatchErrorKind(t *testing.T) {
	outbox := &fakeOutboxDeliveryManager{
		lease: query.OutboxDeliveryView{EventID: "outbox:failed"},
	}
	dispatcher := &fakeOutboxDeliveryDispatcher{
		err: fakeDeliveryDispatchError{kind: "platform_timeout", message: "platform timed out"},
	}
	worker, err := jobtrigger.NewOutboxDeliveryWorker(outbox, dispatcher, jobtrigger.OutboxDeliveryWorkerConfig{})
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	result, err := worker.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once: %v", err)
	}

	if !result.Processed || !result.Failed || result.ErrorKind != "platform_timeout" {
		t.Fatalf("unexpected failed result: %+v", result)
	}
	if outbox.markSucceededCalls != 0 || outbox.markFailedCalls != 1 {
		t.Fatalf("unexpected outbox calls: %+v", outbox)
	}
	if outbox.lastFailed.ErrorKind != "platform_timeout" || outbox.lastFailed.ErrorMessage != "platform timed out" {
		t.Fatalf("unexpected failed command: %+v", outbox.lastFailed)
	}
}

func TestOutboxDeliveryWorkerReturnsIdleWhenNoDelivery(t *testing.T) {
	outbox := &fakeOutboxDeliveryManager{
		leaseErr: errors.New("no leaseable outbox delivery"),
	}
	dispatcher := &fakeOutboxDeliveryDispatcher{}
	worker, err := jobtrigger.NewOutboxDeliveryWorker(outbox, dispatcher, jobtrigger.OutboxDeliveryWorkerConfig{})
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	result, err := worker.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once: %v", err)
	}

	if result.Processed || result.Reason != "no_delivery" {
		t.Fatalf("unexpected idle result: %+v", result)
	}
	if dispatcher.calls != 0 || outbox.markDispatchingCalls != 0 || outbox.markSucceededCalls != 0 || outbox.markFailedCalls != 0 {
		t.Fatalf("unexpected side effects: dispatcher=%d outbox=%+v", dispatcher.calls, outbox)
	}
}

type fakeOutboxDeliveryManager struct {
	lease    query.OutboxDeliveryView
	leaseErr error

	leaseNextCalls       int
	markDispatchingCalls int
	markSucceededCalls   int
	markFailedCalls      int

	lastLease  command.LeaseNextOutboxCommand
	lastFailed command.MarkOutboxFailedCommand
}

func (f *fakeOutboxDeliveryManager) LeaseNext(
	_ context.Context,
	cmd command.LeaseNextOutboxCommand,
) (query.OutboxDeliveryView, error) {
	f.leaseNextCalls++
	f.lastLease = cmd
	return f.lease, f.leaseErr
}

func (f *fakeOutboxDeliveryManager) MarkDispatching(
	context.Context,
	command.MarkOutboxDispatchingCommand,
) (query.OutboxDeliveryView, error) {
	f.markDispatchingCalls++
	return f.lease, nil
}

func (f *fakeOutboxDeliveryManager) MarkSucceeded(
	context.Context,
	command.MarkOutboxSucceededCommand,
) (query.OutboxDeliveryView, error) {
	f.markSucceededCalls++
	f.lease.Status = "succeeded"
	return f.lease, nil
}

func (f *fakeOutboxDeliveryManager) MarkFailed(
	_ context.Context,
	cmd command.MarkOutboxFailedCommand,
) (query.OutboxDeliveryView, error) {
	f.markFailedCalls++
	f.lastFailed = cmd
	f.lease.Status = "failed"
	return f.lease, nil
}

type fakeOutboxDeliveryDispatcher struct {
	calls  int
	last   command.DispatchDeliveryCommand
	result query.DeliveryDispatchResultView
	err    error
}

func (f *fakeOutboxDeliveryDispatcher) Dispatch(
	_ context.Context,
	cmd command.DispatchDeliveryCommand,
) (query.DeliveryDispatchResultView, error) {
	f.calls++
	f.last = cmd
	return f.result, f.err
}

type fakeDeliveryDispatchError struct {
	kind    string
	message string
}

func (e fakeDeliveryDispatchError) Error() string {
	return e.message
}

func (e fakeDeliveryDispatchError) DeliveryErrorKind() string {
	return e.kind
}
