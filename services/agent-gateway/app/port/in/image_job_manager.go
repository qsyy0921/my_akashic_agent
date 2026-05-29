package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/command"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/query"
)

type ImageJobManager interface {
	Create(ctx context.Context, cmd command.CreateImageJobCommand) (query.ImageJobView, error)
	MarkRunning(ctx context.Context, cmd command.MarkImageJobRunningCommand) (query.ImageJobView, error)
	Complete(ctx context.Context, cmd command.CompleteImageJobCommand) (query.ImageJobView, error)
	Fail(ctx context.Context, cmd command.FailImageJobCommand) (query.ImageJobView, error)
	Get(ctx context.Context, jobID string) (query.ImageJobView, error)
}
