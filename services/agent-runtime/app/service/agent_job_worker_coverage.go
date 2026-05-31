package service

import (
	"fmt"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func agentJobWorkerCoverageFromPressure(items []query.AgentJobTypePressureView, workers query.AgentWorkerStatusesView) []query.AgentJobWorkerCoverageView {
	if len(items) == 0 {
		return nil
	}
	workersByType := make(map[string][]query.AgentWorkerStatusView)
	for _, worker := range workers.Workers {
		workersByType[worker.WorkerType] = append(workersByType[worker.WorkerType], worker)
	}
	result := make([]query.AgentJobWorkerCoverageView, 0, len(items))
	for _, pressure := range items {
		expected := expectedWorkerTypesForAgentJobType(pressure.JobType)
		item := query.AgentJobWorkerCoverageView{
			JobType:             pressure.JobType,
			ExpectedWorkerTypes: append([]string(nil), expected...),
			HighPressure:        pressure.HighPressure,
			PressureReason:      pressure.PressureReason,
			CoverageStatus:      "muted",
		}
		if len(expected) == 0 {
			item.CoverageReason = "no_expected_worker_mapping"
			result = append(result, item)
			continue
		}
		for _, workerType := range expected {
			for _, worker := range workersByType[workerType] {
				item.WorkerCount++
				if worker.Stale {
					item.StaleWorkers++
				}
				switch worker.Status {
				case "starting", "idle":
					item.ActiveWorkers++
				case "running":
					item.ActiveWorkers++
					item.RunningWorkers++
				case "failed":
					item.FailedWorkers++
				}
			}
		}
		item.CoverageStatus, item.CoverageReason = agentJobWorkerCoverageStatus(item)
		result = append(result, item)
	}
	return result
}

func expectedWorkerTypesForAgentJobType(jobType string) []string {
	switch jobType {
	case "group_memory_extract", "rag_ingest":
		return []string{"knowledge"}
	case "rag_eval":
		return []string{"rag_eval"}
	case "image_generation":
		return []string{"image_generation"}
	default:
		return nil
	}
}

func agentJobWorkerCoverageStatus(item query.AgentJobWorkerCoverageView) (string, string) {
	if len(item.ExpectedWorkerTypes) == 0 {
		return "muted", "no_expected_worker_mapping"
	}
	if item.HighPressure && item.ActiveWorkers == 0 {
		if item.WorkerCount == 0 {
			return "danger", "high_pressure_no_registered_worker"
		}
		return "danger", "high_pressure_no_active_worker"
	}
	if item.ActiveWorkers == 0 {
		if item.WorkerCount == 0 {
			return "warn", "no_registered_worker"
		}
		if item.FailedWorkers > 0 {
			return "warn", "failed_worker_present"
		}
		if item.StaleWorkers > 0 {
			return "warn", "stale_worker_present"
		}
		return "warn", "no_active_worker"
	}
	if item.FailedWorkers > 0 {
		return "warn", "failed_worker_present"
	}
	if item.StaleWorkers > 0 {
		return "warn", "stale_worker_present"
	}
	return "ok", "active_worker_available"
}

func agentJobWorkerCoverageCount(items []query.AgentJobWorkerCoverageView, include func(query.AgentJobWorkerCoverageView) bool) int {
	total := 0
	for _, item := range items {
		if include(item) {
			total++
		}
	}
	return total
}

func runtimeAgentJobWorkerCoverageStatus(items []query.AgentJobWorkerCoverageView) string {
	if len(items) == 0 {
		return "muted"
	}
	for _, item := range items {
		if item.CoverageStatus == "danger" {
			return "danger"
		}
	}
	for _, item := range items {
		if item.CoverageStatus == "warn" {
			return "warn"
		}
	}
	for _, item := range items {
		if item.CoverageStatus == "ok" {
			return "ok"
		}
	}
	return "muted"
}

func runtimeAgentJobWorkerCoverageValue(items []query.AgentJobWorkerCoverageView) string {
	if len(items) == 0 {
		return "0/0"
	}
	uncovered := agentJobWorkerCoverageCount(items, func(item query.AgentJobWorkerCoverageView) bool {
		return len(item.ExpectedWorkerTypes) > 0 && item.ActiveWorkers == 0
	})
	return fmt.Sprintf("%d/%d", uncovered, len(items))
}
