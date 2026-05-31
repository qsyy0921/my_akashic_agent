package main

import (
	"context"
	"log"
	"time"

	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	jobtrigger "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/trigger/job"
)

func startKnowledgeJobPlanner(
	agentJobs *appservice.AgentJobService,
	observeTargets *appservice.ObserveTargetService,
) (context.CancelFunc, error) {
	config, enabled, err := knowledgeJobPlannerConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}
	config.Logf = log.Printf
	planner := appservice.NewKnowledgeJobPlannerService(observeTargets, agentJobs)
	runner, err := jobtrigger.NewKnowledgeJobPlannerWorker(planner, config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := runner.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("knowledge job planner worker stopped: %v", err)
		}
	}()
	log.Printf(
		"knowledge job planner enabled interval=%s worker_id=%s agent_id=%s max_attempts=%d rag_max_messages=%d rag_parse=%t run_on_start=%t",
		config.Interval,
		config.WorkerID,
		config.AgentID,
		config.MaxAttempts,
		config.RagMaxMessages,
		config.RagParse,
		config.RunOnStart,
	)
	return cancel, nil
}

func knowledgeJobPlannerConfigFromEnv() (jobtrigger.KnowledgeJobPlannerWorkerConfig, bool, error) {
	if !boolEnv("AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED") {
		return jobtrigger.KnowledgeJobPlannerWorkerConfig{}, false, nil
	}
	intervalSeconds, err := positiveIntEnv("AKASHIC_KNOWLEDGE_JOB_PLANNER_INTERVAL_SECONDS", 60, 86400)
	if err != nil {
		return jobtrigger.KnowledgeJobPlannerWorkerConfig{}, false, err
	}
	maxAttempts, err := positiveIntEnv("AKASHIC_KNOWLEDGE_JOB_PLANNER_MAX_ATTEMPTS", 2, 10)
	if err != nil {
		return jobtrigger.KnowledgeJobPlannerWorkerConfig{}, false, err
	}
	ragMaxMessages, err := positiveIntEnv("AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_MAX_MESSAGES", 1000, 100000)
	if err != nil {
		return jobtrigger.KnowledgeJobPlannerWorkerConfig{}, false, err
	}
	return jobtrigger.KnowledgeJobPlannerWorkerConfig{
		Interval:       time.Duration(intervalSeconds) * time.Second,
		WorkerID:       envOrDefault("AKASHIC_KNOWLEDGE_JOB_PLANNER_WORKER_ID", "agent-runtime-knowledge-job-planner"),
		AgentID:        envOrDefault("AKASHIC_KNOWLEDGE_JOB_PLANNER_AGENT_ID", "akashic-python-worker"),
		MaxAttempts:    maxAttempts,
		RagMaxMessages: ragMaxMessages,
		RagParse:       boolEnvDefault("AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_PARSE", true),
		RunOnStart:     boolEnvDefault("AKASHIC_KNOWLEDGE_JOB_PLANNER_RUN_ON_START", true),
	}, true, nil
}
