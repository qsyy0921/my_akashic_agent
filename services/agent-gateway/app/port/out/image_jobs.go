package outport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
)

type ImageJobRepository interface {
	SaveImageJob(ctx context.Context, job model.ImageJob) error
	FindImageJob(ctx context.Context, jobID string) (model.ImageJob, bool, error)
}

type ImageJobQueue interface {
	EnqueueImageJob(ctx context.Context, job model.ImageJob) error
}
