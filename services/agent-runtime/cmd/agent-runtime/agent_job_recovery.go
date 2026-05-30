package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	jobtrigger "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/trigger/job"
)

func startAgentJobLeaseRecovery(agentJobs *appservice.AgentJobService) (context.CancelFunc, error) {
	config, enabled, err := agentJobLeaseRecoveryConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}
	config.Logf = log.Printf
	runner, err := jobtrigger.NewAgentJobLeaseRecoveryRunner(agentJobs, config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := runner.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("agent job lease recovery runner stopped: %v", err)
		}
	}()
	log.Printf(
		"agent job lease recovery enabled interval=%s limit=%d run_on_start=%t",
		config.Interval,
		config.Limit,
		config.RunOnStart,
	)
	return cancel, nil
}

func agentJobLeaseRecoveryConfigFromEnv() (jobtrigger.AgentJobLeaseRecoveryConfig, bool, error) {
	if !boolEnv("AKASHIC_AGENT_JOB_RECOVERY_ENABLED") {
		return jobtrigger.AgentJobLeaseRecoveryConfig{}, false, nil
	}
	intervalSeconds, err := positiveIntEnv("AKASHIC_AGENT_JOB_RECOVERY_INTERVAL_SECONDS", 300, 86400)
	if err != nil {
		return jobtrigger.AgentJobLeaseRecoveryConfig{}, false, err
	}
	limit, err := positiveIntEnv("AKASHIC_AGENT_JOB_RECOVERY_LIMIT", 50, 200)
	if err != nil {
		return jobtrigger.AgentJobLeaseRecoveryConfig{}, false, err
	}
	return jobtrigger.AgentJobLeaseRecoveryConfig{
		Interval:   time.Duration(intervalSeconds) * time.Second,
		Limit:      limit,
		RunOnStart: boolEnvDefault("AKASHIC_AGENT_JOB_RECOVERY_RUN_ON_START", true),
	}, true, nil
}

func boolEnvDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return boolEnv(key)
}
