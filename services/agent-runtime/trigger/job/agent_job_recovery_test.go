package jobtrigger_test

import (
	"context"
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	jobtrigger "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/trigger/job"
)

func TestAgentJobLeaseRecoveryRunnerRecoverOnceUsesConfiguredLimitAndTimestamp(t *testing.T) {
	now := time.Date(2026, 5, 31, 3, 0, 0, 0, time.UTC)
	recoverer := &fakeAgentJobRecoverer{
		view: query.AgentJobLeaseRecoveryView{
			Scanned:      2,
			Recovered:    1,
			DeadLettered: 1,
		},
	}
	logs := make([]string, 0)
	runner, err := jobtrigger.NewAgentJobLeaseRecoveryRunner(recoverer, jobtrigger.AgentJobLeaseRecoveryConfig{
		Interval: time.Minute,
		Limit:    17,
		Now:      func() time.Time { return now },
		Logf: func(format string, args ...any) {
			logs = append(logs, format)
		},
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	if err := runner.RecoverOnce(context.Background()); err != nil {
		t.Fatalf("recover once: %v", err)
	}

	if recoverer.calls != 1 {
		t.Fatalf("expected one recovery call, got %d", recoverer.calls)
	}
	if recoverer.last.Limit != 17 || !recoverer.last.Timestamp.Equal(now) {
		t.Fatalf("unexpected recovery command: %+v", recoverer.last)
	}
	if len(logs) != 1 || logs[0] != "agent job lease recovery scanned=%d recovered=%d dead_lettered=%d" {
		t.Fatalf("expected recovery summary log, got %+v", logs)
	}
}

func TestAgentJobLeaseRecoveryRunnerDefaultsAreBounded(t *testing.T) {
	recoverer := &fakeAgentJobRecoverer{}
	runner, err := jobtrigger.NewAgentJobLeaseRecoveryRunner(recoverer, jobtrigger.AgentJobLeaseRecoveryConfig{
		Limit: 999,
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	if err := runner.RecoverOnce(context.Background()); err != nil {
		t.Fatalf("recover once: %v", err)
	}
	if recoverer.last.Limit != 50 {
		t.Fatalf("expected default bounded limit 50, got %+v", recoverer.last)
	}
	if recoverer.last.Timestamp.IsZero() {
		t.Fatalf("expected non-zero default timestamp")
	}
}

type fakeAgentJobRecoverer struct {
	calls int
	last  command.RecoverExpiredAgentJobLeasesCommand
	view  query.AgentJobLeaseRecoveryView
	err   error
}

func (f *fakeAgentJobRecoverer) RecoverExpiredLeases(
	_ context.Context,
	cmd command.RecoverExpiredAgentJobLeasesCommand,
) (query.AgentJobLeaseRecoveryView, error) {
	f.calls++
	f.last = cmd
	return f.view, f.err
}
