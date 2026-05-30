package service

import (
	"context"
	"errors"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

type RuntimeWorkerDiagnosticsService struct {
	view query.RuntimeWorkerDiagnosticsView
}

func NewRuntimeWorkerDiagnosticsService(view query.RuntimeWorkerDiagnosticsView) *RuntimeWorkerDiagnosticsService {
	return &RuntimeWorkerDiagnosticsService{view: normalizeRuntimeWorkerDiagnostics(view)}
}

func (s *RuntimeWorkerDiagnosticsService) GetRuntimeWorkers(ctx context.Context) (query.RuntimeWorkerDiagnosticsView, error) {
	if err := ctx.Err(); err != nil {
		return query.RuntimeWorkerDiagnosticsView{}, err
	}
	if s == nil {
		return query.RuntimeWorkerDiagnosticsView{}, errors.New("runtime worker diagnostics service is nil")
	}
	return s.view, nil
}

func normalizeRuntimeWorkerDiagnostics(view query.RuntimeWorkerDiagnosticsView) query.RuntimeWorkerDiagnosticsView {
	totals := map[string]int{
		"workers":  len(view.Workers),
		"enabled":  0,
		"running":  0,
		"disabled": 0,
	}
	for _, worker := range view.Workers {
		if worker.Enabled {
			totals["enabled"]++
		} else {
			totals["disabled"]++
		}
		if worker.Running {
			totals["running"]++
		}
	}
	view.Totals = totals
	return view
}
