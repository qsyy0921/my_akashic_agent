package jobtrigger

import (
	"context"
	"errors"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type AgentJobLeaseRecoverer interface {
	RecoverExpiredLeases(ctx context.Context, cmd command.RecoverExpiredAgentJobLeasesCommand) (query.AgentJobLeaseRecoveryView, error)
}

type AgentJobLeaseRecoveryConfig struct {
	Interval   time.Duration
	Limit      int
	RunOnStart bool
	Now        func() time.Time
	Logf       func(format string, args ...any)
}

type AgentJobLeaseRecoveryRunner struct {
	recoverer  AgentJobLeaseRecoverer
	interval   time.Duration
	limit      int
	runOnStart bool
	now        func() time.Time
	logf       func(format string, args ...any)
}

func NewAgentJobLeaseRecoveryRunner(
	recoverer AgentJobLeaseRecoverer,
	config AgentJobLeaseRecoveryConfig,
) (*AgentJobLeaseRecoveryRunner, error) {
	if recoverer == nil {
		return nil, errors.New("agent job lease recovery runner requires recoverer")
	}
	if config.Interval <= 0 {
		config.Interval = 5 * time.Minute
	}
	if config.Limit <= 0 || config.Limit > 200 {
		config.Limit = 50
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	return &AgentJobLeaseRecoveryRunner{
		recoverer:  recoverer,
		interval:   config.Interval,
		limit:      config.Limit,
		runOnStart: config.RunOnStart,
		now:        config.Now,
		logf:       config.Logf,
	}, nil
}

func (r *AgentJobLeaseRecoveryRunner) Run(ctx context.Context) error {
	if r == nil {
		return errors.New("agent job lease recovery runner is nil")
	}
	if r.runOnStart {
		if err := r.RecoverOnce(ctx); err != nil {
			if ctx.Err() != nil {
				return err
			}
			r.log("agent job lease recovery failed: %v", err)
		}
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.RecoverOnce(ctx); err != nil {
				r.log("agent job lease recovery failed: %v", err)
			}
		}
	}
}

func (r *AgentJobLeaseRecoveryRunner) RecoverOnce(ctx context.Context) error {
	if r == nil {
		return errors.New("agent job lease recovery runner is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	view, err := r.recoverer.RecoverExpiredLeases(ctx, command.RecoverExpiredAgentJobLeasesCommand{
		Limit:     r.limit,
		Timestamp: r.now(),
	})
	if err != nil {
		return err
	}
	if view.Recovered > 0 || view.DeadLettered > 0 {
		r.log(
			"agent job lease recovery scanned=%d recovered=%d dead_lettered=%d",
			view.Scanned,
			view.Recovered,
			view.DeadLettered,
		)
	}
	return nil
}

func (r *AgentJobLeaseRecoveryRunner) log(format string, args ...any) {
	if r != nil && r.logf != nil {
		r.logf(format, args...)
	}
}
