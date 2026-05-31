package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	appservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/service"
	jobtrigger "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/trigger/job"
)

func startOutboxDeliveryWorker(
	outbox *appservice.OutboxService,
	dispatcher *appservice.DeliveryDispatchService,
	queueBackend query.QueueBackendView,
) (context.CancelFunc, error) {
	config, enabled, err := outboxDeliveryWorkerConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}
	if queueBackend.ExternalLease != nil && queueBackend.ExternalLease.AllowExecution {
		return nil, fmt.Errorf("AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED conflicts with external lease execution; disable local outbox worker before queue cutover")
	}
	config.Logf = log.Printf
	runner, err := jobtrigger.NewOutboxDeliveryWorker(outbox, dispatcher, config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := runner.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("outbox delivery worker stopped: %v", err)
		}
	}()
	log.Printf(
		"outbox delivery worker enabled interval=%s batch_size=%d worker_id=%s lease_ttl_seconds=%d run_on_start=%t",
		config.Interval,
		config.BatchSize,
		config.WorkerID,
		config.LeaseTTLSeconds,
		config.RunOnStart,
	)
	return cancel, nil
}

func outboxDeliveryWorkerConfigFromEnv() (jobtrigger.OutboxDeliveryWorkerConfig, bool, error) {
	if !boolEnv("AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED") {
		return jobtrigger.OutboxDeliveryWorkerConfig{}, false, nil
	}
	intervalSeconds, err := positiveIntEnv("AKASHIC_OUTBOX_DELIVERY_WORKER_INTERVAL_SECONDS", 2, 86400)
	if err != nil {
		return jobtrigger.OutboxDeliveryWorkerConfig{}, false, err
	}
	batchSize, err := positiveIntEnv("AKASHIC_OUTBOX_DELIVERY_WORKER_BATCH_SIZE", 1, 50)
	if err != nil {
		return jobtrigger.OutboxDeliveryWorkerConfig{}, false, err
	}
	leaseTTLSeconds, err := positiveIntEnv("AKASHIC_OUTBOX_DELIVERY_WORKER_LEASE_TTL_SECONDS", 300, 86400)
	if err != nil {
		return jobtrigger.OutboxDeliveryWorkerConfig{}, false, err
	}
	accountMinIntervalSeconds, err := nonNegativeIntEnv("AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MIN_INTERVAL_SECONDS", 0, 86400)
	if err != nil {
		return jobtrigger.OutboxDeliveryWorkerConfig{}, false, err
	}
	accountWindowSeconds, err := positiveIntEnv("AKASHIC_OUTBOX_DELIVERY_ACCOUNT_WINDOW_SECONDS", 60, 86400)
	if err != nil {
		return jobtrigger.OutboxDeliveryWorkerConfig{}, false, err
	}
	accountMaxPerWindow, err := nonNegativeIntEnv("AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MAX_PER_WINDOW", 0, 100000)
	if err != nil {
		return jobtrigger.OutboxDeliveryWorkerConfig{}, false, err
	}
	return jobtrigger.OutboxDeliveryWorkerConfig{
		Interval:                     time.Duration(intervalSeconds) * time.Second,
		BatchSize:                    batchSize,
		WorkerID:                     envOrDefault("AKASHIC_OUTBOX_DELIVERY_WORKER_ID", "agent-runtime-outbox-worker"),
		LeaseTTLSeconds:              leaseTTLSeconds,
		RunOnStart:                   boolEnvDefault("AKASHIC_OUTBOX_DELIVERY_WORKER_RUN_ON_START", true),
		ChannelByAccount:             keyValueCSVEnv("AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT"),
		AccountMinInterval:           time.Duration(accountMinIntervalSeconds) * time.Second,
		AccountWindow:                time.Duration(accountWindowSeconds) * time.Second,
		AccountMaxDispatchesInWindow: accountMaxPerWindow,
	}, true, nil
}
