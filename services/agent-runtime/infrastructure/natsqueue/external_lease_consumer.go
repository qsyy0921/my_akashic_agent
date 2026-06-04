package natsqueue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	inport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/in"
)

const (
	defaultExternalLeaseDurable     = "AKASHIC_EXTERNAL_LEASE_OUTBOX"
	defaultExternalLeaseAllDurable  = "AKASHIC_EXTERNAL_LEASE_ALL"
	defaultExternalLeasePollTimeout = 2 * time.Second
	defaultExternalLeaseNackDelay   = 30 * time.Second
)

type ExternalLeaseConsumerConfig struct {
	URL                                       string
	Stream                                    string
	SubjectPrefix                             string
	Durable                                   string
	WorkerID                                  string
	LeaseTTLSeconds                           int
	NackDelay                                 time.Duration
	Timeout                                   time.Duration
	ConsumerConcurrency                       int
	MaxInFlight                               int
	ChannelByAccount                          map[string]string
	AllowedStepKinds                          []string
	AllowedStepKindsByAccount                 map[string][]string
	AllowedStepKindsByAccountConversationType map[string]map[string][]string
	AllowedStepKindsByAccountConversationID   map[string]map[string]map[string][]string
	IncludeAgentJobs                          bool
}

type ExternalLeaseConsumer struct {
	conn                                      *nats.Conn
	js                                        jetStreamConsumer
	sub                                       *nats.Subscription
	stream                                    string
	subjectPrefix                             string
	durable                                   string
	workerID                                  string
	leaseTTLSeconds                           int
	nackDelay                                 time.Duration
	timeout                                   time.Duration
	consumerConcurrency                       int
	maxInFlight                               int
	channelByAccount                          map[string]string
	allowedStepKinds                          []string
	allowedStepKindsByAccount                 map[string][]string
	allowedStepKindsByAccountConversationType map[string]map[string][]string
	allowedStepKindsByAccountConversationID   map[string]map[string]map[string][]string
	includeAgentJobs                          bool
}

func NewExternalLeaseConsumer(config ExternalLeaseConsumerConfig) (*ExternalLeaseConsumer, error) {
	config = normalizeExternalLeaseConsumerConfig(config)
	if strings.TrimSpace(config.URL) == "" {
		return nil, errors.New("nats external lease consumer requires url")
	}
	conn, err := nats.Connect(config.URL, nats.Timeout(config.Timeout))
	if err != nil {
		return nil, err
	}
	js, err := conn.JetStream(nats.MaxWait(config.Timeout))
	if err != nil {
		conn.Close()
		return nil, err
	}
	consumer := &ExternalLeaseConsumer{
		conn:                      conn,
		js:                        js,
		stream:                    config.Stream,
		subjectPrefix:             config.SubjectPrefix,
		durable:                   config.Durable,
		workerID:                  config.WorkerID,
		leaseTTLSeconds:           config.LeaseTTLSeconds,
		nackDelay:                 config.NackDelay,
		timeout:                   config.Timeout,
		consumerConcurrency:       config.ConsumerConcurrency,
		maxInFlight:               config.MaxInFlight,
		channelByAccount:          cloneMap(config.ChannelByAccount),
		allowedStepKinds:          cloneStrings(config.AllowedStepKinds),
		allowedStepKindsByAccount: cloneStringSliceMap(config.AllowedStepKindsByAccount),
		allowedStepKindsByAccountConversationType: cloneStringSliceMatrix(config.AllowedStepKindsByAccountConversationType),
		allowedStepKindsByAccountConversationID:   cloneStringSliceTensor(config.AllowedStepKindsByAccountConversationID),
		includeAgentJobs:                          config.IncludeAgentJobs,
	}
	if err := consumer.ensureStream(); err != nil {
		conn.Close()
		return nil, err
	}
	sub, err := js.PullSubscribe(
		externalLeaseSubscriptionSubject(consumer.subjectPrefix, consumer.includeAgentJobs),
		consumer.durable,
		nats.BindStream(consumer.stream),
		nats.ManualAck(),
		nats.MaxAckPending(consumer.maxInFlight),
	)
	if err != nil {
		conn.Close()
		return nil, err
	}
	consumer.sub = sub
	return consumer, nil
}

func (c *ExternalLeaseConsumer) Close() {
	if c == nil || c.conn == nil {
		return
	}
	c.conn.Close()
}

func (c *ExternalLeaseConsumer) Run(ctx context.Context, executor inport.WorkQueueLeaseExecutor) error {
	if c == nil || c.sub == nil {
		return errors.New("nats external lease consumer requires subscription")
	}
	if executor == nil {
		return errors.New("nats external lease consumer requires executor")
	}
	messages := make(chan *nats.Msg, c.maxInFlight)
	var workers sync.WaitGroup
	for worker := 0; worker < c.consumerConcurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for msg := range messages {
				c.handleMessage(ctx, executor, msg)
			}
		}()
	}
	defer func() {
		close(messages)
		workers.Wait()
	}()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		fetched, err := c.sub.Fetch(c.fetchBatch(), nats.MaxWait(defaultExternalLeasePollTimeout))
		for _, msg := range fetched {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case messages <- msg:
			}
		}
		if errors.Is(err, nats.ErrTimeout) {
			continue
		}
		if err != nil {
			return err
		}
	}
}

func (c *ExternalLeaseConsumer) handleMessage(ctx context.Context, executor inport.WorkQueueLeaseExecutor, msg *nats.Msg) {
	cmd := WorkLeaseCommandFromNATSMessage(msg.Subject, msg.Data, time.Now().UTC())
	cmd.WorkerID = c.workerID
	cmd.LeaseTTLSeconds = c.leaseTTLSeconds
	cmd.ChannelByAccount = cloneMap(c.channelByAccount)
	cmd.AllowedStepKinds = cloneStrings(c.allowedStepKinds)
	cmd.AllowedStepKindsByAccount = cloneStringSliceMap(c.allowedStepKindsByAccount)
	cmd.AllowedStepKindsByAccountConversationType = cloneStringSliceMatrix(c.allowedStepKindsByAccountConversationType)
	cmd.AllowedStepKindsByAccountConversationID = cloneStringSliceTensor(c.allowedStepKindsByAccountConversationID)
	result, err := executor.ExecuteWorkQueueLease(ctx, cmd)
	if err != nil {
		_ = msg.Nak()
		return
	}
	switch strings.ToLower(strings.TrimSpace(result.Disposition)) {
	case "ack":
		_ = msg.Ack()
	case "term":
		_ = msg.Term()
	default:
		_ = msg.NakWithDelay(c.nackDelay)
	}
}

func (c *ExternalLeaseConsumer) fetchBatch() int {
	batch := c.maxInFlight
	if batch <= 0 {
		batch = c.consumerConcurrency
	}
	if batch > c.consumerConcurrency*4 {
		batch = c.consumerConcurrency * 4
	}
	if batch <= 0 {
		return 1
	}
	return batch
}

func (c *ExternalLeaseConsumer) ensureStream() error {
	if c == nil || c.js == nil {
		return errors.New("nats external lease consumer requires jetstream")
	}
	if _, err := c.js.StreamInfo(c.stream, nats.MaxWait(c.timeout)); err == nil {
		return nil
	}
	_, err := c.js.AddStream(&nats.StreamConfig{
		Name:     c.stream,
		Subjects: []string{c.subjectPrefix + ".>"},
		Storage:  nats.FileStorage,
	}, nats.MaxWait(c.timeout))
	return err
}

func WorkLeaseCommandFromNATSMessage(subject string, raw []byte, observedAt time.Time) command.ExecuteWorkQueueLeaseCommand {
	var notification WorkNotification
	if err := json.Unmarshal(raw, &notification); err != nil {
		return command.ExecuteWorkQueueLeaseCommand{
			WorkKind:   "invalid_payload",
			WorkID:     strings.TrimSpace(subject),
			Subject:    strings.TrimSpace(subject),
			ObservedAt: observedAt,
			Metadata:   map[string]string{"decode_error": err.Error()},
		}
	}
	subjectHint := strings.TrimSpace(notification.Subject)
	if subjectHint == "" {
		subjectHint = strings.TrimSpace(subject)
	}
	return command.ExecuteWorkQueueLeaseCommand{
		WorkKind:    notification.WorkKind,
		WorkID:      notification.WorkID,
		AggregateID: notification.AggregateID,
		Subject:     subjectHint,
		ObservedAt:  observedAt,
		Metadata:    cloneMap(notification.Metadata),
	}
}

func normalizeExternalLeaseConsumerConfig(config ExternalLeaseConsumerConfig) ExternalLeaseConsumerConfig {
	normalized := normalizeConfig(Config{
		URL:           config.URL,
		Stream:        config.Stream,
		SubjectPrefix: config.SubjectPrefix,
		Timeout:       config.Timeout,
	})
	config.URL = normalized.URL
	config.Stream = normalized.Stream
	config.SubjectPrefix = normalized.SubjectPrefix
	config.Timeout = normalized.Timeout
	durable := strings.TrimSpace(config.Durable)
	if durable == "" {
		if config.IncludeAgentJobs {
			config.Durable = defaultExternalLeaseAllDurable
		} else {
			config.Durable = defaultExternalLeaseDurable
		}
	} else {
		replacer := strings.NewReplacer(".", "_", "*", "_", ">", "_", "/", "_", "\\", "_")
		config.Durable = safeSubjectToken(replacer.Replace(durable))
		if config.Durable == "unknown" {
			config.Durable = defaultExternalLeaseDurable
		}
	}
	if config.WorkerID = strings.TrimSpace(config.WorkerID); config.WorkerID == "" {
		config.WorkerID = "agent-runtime-external-lease"
	}
	if config.LeaseTTLSeconds <= 0 {
		config.LeaseTTLSeconds = 300
	}
	if config.NackDelay <= 0 {
		config.NackDelay = defaultExternalLeaseNackDelay
	}
	if config.ConsumerConcurrency <= 0 {
		config.ConsumerConcurrency = 1
	}
	if config.MaxInFlight < config.ConsumerConcurrency {
		config.MaxInFlight = config.ConsumerConcurrency
	}
	config.ChannelByAccount = cloneMap(config.ChannelByAccount)
	config.AllowedStepKinds = cloneStrings(config.AllowedStepKinds)
	config.AllowedStepKindsByAccount = cloneStringSliceMap(config.AllowedStepKindsByAccount)
	config.AllowedStepKindsByAccountConversationType = cloneStringSliceMatrix(config.AllowedStepKindsByAccountConversationType)
	config.AllowedStepKindsByAccountConversationID = cloneStringSliceTensor(config.AllowedStepKindsByAccountConversationID)
	return config
}

func externalLeaseSubscriptionSubject(subjectPrefix string, includeAgentJobs bool) string {
	subjectPrefix = cleanSubjectPrefix(subjectPrefix)
	if includeAgentJobs {
		return subjectPrefix + ".>"
	}
	return subjectPrefix + ".outbox.>"
}

func cloneStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		cloned = append(cloned, item)
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func cloneStringSliceMap(items map[string][]string) map[string][]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(items))
	for key, values := range items {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if normalized := cloneStrings(values); len(normalized) > 0 {
			cloned[key] = normalized
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func cloneStringSliceMatrix(items map[string]map[string][]string) map[string]map[string][]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]map[string][]string, len(items))
	for outerKey, byInner := range items {
		outerKey = strings.TrimSpace(outerKey)
		if outerKey == "" || len(byInner) == 0 {
			continue
		}
		innerClone := make(map[string][]string)
		for innerKey, values := range byInner {
			innerKey = strings.TrimSpace(innerKey)
			if innerKey == "" {
				continue
			}
			if normalized := cloneStrings(values); len(normalized) > 0 {
				innerClone[innerKey] = normalized
			}
		}
		if len(innerClone) > 0 {
			cloned[outerKey] = innerClone
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func cloneStringSliceTensor(items map[string]map[string]map[string][]string) map[string]map[string]map[string][]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]map[string]map[string][]string, len(items))
	for outerKey, byMiddle := range items {
		outerKey = strings.TrimSpace(outerKey)
		if outerKey == "" || len(byMiddle) == 0 {
			continue
		}
		middleClone := make(map[string]map[string][]string)
		for middleKey, byInner := range byMiddle {
			middleKey = strings.TrimSpace(middleKey)
			if middleKey == "" || len(byInner) == 0 {
				continue
			}
			innerClone := make(map[string][]string)
			for innerKey, values := range byInner {
				innerKey = strings.TrimSpace(innerKey)
				if innerKey == "" {
					continue
				}
				if normalized := cloneStrings(values); len(normalized) > 0 {
					innerClone[innerKey] = normalized
				}
			}
			if len(innerClone) > 0 {
				middleClone[middleKey] = innerClone
			}
		}
		if len(middleClone) > 0 {
			cloned[outerKey] = middleClone
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}
