package jobtrigger

import (
	"context"
	"errors"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type KnowledgeJobPlanner interface {
	PlanKnowledgeJobs(ctx context.Context, cmd command.PlanKnowledgeJobsCommand) (query.KnowledgeJobPlannerRunView, error)
}

type KnowledgeJobPlannerWorkerConfig struct {
	Interval       time.Duration
	WorkerID       string
	AgentID        string
	MaxAttempts    int
	RagMaxMessages int
	RagParse       bool
	RunOnStart     bool
	Now            func() time.Time
	Logf           func(format string, args ...any)
}

type KnowledgeJobPlannerWorker struct {
	planner        KnowledgeJobPlanner
	interval       time.Duration
	workerID       string
	agentID        string
	maxAttempts    int
	ragMaxMessages int
	ragParse       bool
	runOnStart     bool
	now            func() time.Time
	logf           func(format string, args ...any)
}

func NewKnowledgeJobPlannerWorker(
	planner KnowledgeJobPlanner,
	config KnowledgeJobPlannerWorkerConfig,
) (*KnowledgeJobPlannerWorker, error) {
	if planner == nil {
		return nil, errors.New("knowledge job planner worker requires planner")
	}
	if config.Interval <= 0 {
		config.Interval = time.Minute
	}
	if config.WorkerID == "" {
		config.WorkerID = "agent-runtime-knowledge-job-planner"
	}
	if config.AgentID == "" {
		config.AgentID = "akashic-python-worker"
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 2
	}
	if config.RagMaxMessages <= 0 {
		config.RagMaxMessages = 1000
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	return &KnowledgeJobPlannerWorker{
		planner:        planner,
		interval:       config.Interval,
		workerID:       config.WorkerID,
		agentID:        config.AgentID,
		maxAttempts:    config.MaxAttempts,
		ragMaxMessages: config.RagMaxMessages,
		ragParse:       config.RagParse,
		runOnStart:     config.RunOnStart,
		now:            config.Now,
		logf:           config.Logf,
	}, nil
}

func (w *KnowledgeJobPlannerWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("knowledge job planner worker is nil")
	}
	if w.runOnStart {
		if _, err := w.PlanOnce(ctx); err != nil {
			if ctx.Err() != nil {
				return err
			}
			w.log("knowledge job planner failed: %v", err)
		}
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := w.PlanOnce(ctx); err != nil {
				w.log("knowledge job planner failed: %v", err)
			}
		}
	}
}

func (w *KnowledgeJobPlannerWorker) PlanOnce(ctx context.Context) (query.KnowledgeJobPlannerRunView, error) {
	if w == nil {
		return query.KnowledgeJobPlannerRunView{}, errors.New("knowledge job planner worker is nil")
	}
	if err := ctx.Err(); err != nil {
		return query.KnowledgeJobPlannerRunView{}, err
	}
	view, err := w.planner.PlanKnowledgeJobs(ctx, command.PlanKnowledgeJobsCommand{
		PlannerID:       w.workerID,
		AgentID:         w.agentID,
		IntervalSeconds: int(w.interval / time.Second),
		MaxAttempts:     w.maxAttempts,
		RagMaxMessages:  w.ragMaxMessages,
		RagParse:        w.ragParse,
		Timestamp:       w.now(),
	})
	if err != nil {
		return query.KnowledgeJobPlannerRunView{}, err
	}
	if view.CreatedOrExisting > 0 || view.SuppressedByDedupe > 0 {
		w.log(
			"knowledge job planner bucket=%d targets=%d created_or_existing=%d suppressed=%d group_memory=%d rag_ingest=%d",
			view.Bucket,
			view.Targets,
			view.CreatedOrExisting,
			view.SuppressedByDedupe,
			view.GroupMemoryJobs,
			view.RagIngestJobs,
		)
	}
	return view, nil
}

func (w *KnowledgeJobPlannerWorker) log(format string, args ...any) {
	if w != nil && w.logf != nil {
		w.logf(format, args...)
	}
}
