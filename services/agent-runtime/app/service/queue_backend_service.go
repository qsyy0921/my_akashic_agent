package service

import (
	"context"
	"errors"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type QueueBackendService struct {
	view query.QueueBackendView
}

func NewQueueBackendService(view query.QueueBackendView) *QueueBackendService {
	return &QueueBackendService{view: view}
}

func (s *QueueBackendService) Get(ctx context.Context) (query.QueueBackendView, error) {
	if err := ctx.Err(); err != nil {
		return query.QueueBackendView{}, err
	}
	if s == nil {
		return query.QueueBackendView{}, errors.New("queue backend service is nil")
	}
	return s.view, nil
}
