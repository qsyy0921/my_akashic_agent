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
	defaultCompareDurable     = "AKASHIC_DUAL_READ_COMPARE"
	defaultComparePollTimeout = 2 * time.Second
)

type CompareConsumerConfig struct {
	URL                 string
	Stream              string
	SubjectPrefix       string
	Durable             string
	Timeout             time.Duration
	ConsumerConcurrency int
	MaxInFlight         int
}

type CompareConsumer struct {
	conn                *nats.Conn
	js                  jetStreamConsumer
	sub                 *nats.Subscription
	stream              string
	subjectPrefix       string
	durable             string
	timeout             time.Duration
	consumerConcurrency int
	maxInFlight         int
}

type jetStreamConsumer interface {
	AddStream(cfg *nats.StreamConfig, opts ...nats.JSOpt) (*nats.StreamInfo, error)
	StreamInfo(stream string, opts ...nats.JSOpt) (*nats.StreamInfo, error)
	PullSubscribe(subj string, durable string, opts ...nats.SubOpt) (*nats.Subscription, error)
}

func NewCompareConsumer(config CompareConsumerConfig) (*CompareConsumer, error) {
	config = normalizeCompareConsumerConfig(config)
	if strings.TrimSpace(config.URL) == "" {
		return nil, errors.New("nats compare consumer requires url")
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
	consumer := &CompareConsumer{
		conn:                conn,
		js:                  js,
		stream:              config.Stream,
		subjectPrefix:       config.SubjectPrefix,
		durable:             config.Durable,
		timeout:             config.Timeout,
		consumerConcurrency: config.ConsumerConcurrency,
		maxInFlight:         config.MaxInFlight,
	}
	if err := consumer.ensureStream(); err != nil {
		conn.Close()
		return nil, err
	}
	sub, err := js.PullSubscribe(
		consumer.subjectPrefix+".>",
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

func (c *CompareConsumer) Close() {
	if c == nil || c.conn == nil {
		return
	}
	c.conn.Close()
}

func (c *CompareConsumer) Run(ctx context.Context, comparer inport.WorkQueueCandidateComparer) error {
	if c == nil || c.sub == nil {
		return errors.New("nats compare consumer requires subscription")
	}
	if comparer == nil {
		return errors.New("nats compare consumer requires comparer")
	}
	messages := make(chan *nats.Msg, c.maxInFlight)
	var workers sync.WaitGroup
	for worker := 0; worker < c.consumerConcurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for msg := range messages {
				c.handleMessage(ctx, comparer, msg)
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
		fetched, err := c.sub.Fetch(c.fetchBatch(), nats.MaxWait(defaultComparePollTimeout))
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

func (c *CompareConsumer) handleMessage(ctx context.Context, comparer inport.WorkQueueCandidateComparer, msg *nats.Msg) {
	cmd := CompareCommandFromNATSMessage(msg.Subject, msg.Data, time.Now().UTC())
	if _, err := comparer.CompareWorkQueueCandidate(ctx, cmd); err != nil {
		_ = msg.Nak()
		return
	}
	_ = msg.Ack()
}

func (c *CompareConsumer) fetchBatch() int {
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

func (c *CompareConsumer) ensureStream() error {
	if c == nil || c.js == nil {
		return errors.New("nats compare consumer requires jetstream")
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

func CompareCommandFromNATSMessage(subject string, raw []byte, observedAt time.Time) command.CompareWorkQueueCandidateCommand {
	var notification WorkNotification
	if err := json.Unmarshal(raw, &notification); err != nil {
		return command.CompareWorkQueueCandidateCommand{
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
	return command.CompareWorkQueueCandidateCommand{
		WorkKind:    notification.WorkKind,
		WorkID:      notification.WorkID,
		AggregateID: notification.AggregateID,
		Subject:     subjectHint,
		Status:      notification.Status,
		ObservedAt:  observedAt,
		Metadata:    cloneMap(notification.Metadata),
	}
}

func normalizeCompareConsumerConfig(config CompareConsumerConfig) CompareConsumerConfig {
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
		config.Durable = defaultCompareDurable
	} else {
		replacer := strings.NewReplacer(".", "_", "*", "_", ">", "_", "/", "_", "\\", "_")
		config.Durable = safeSubjectToken(replacer.Replace(durable))
		if config.Durable == "unknown" {
			config.Durable = defaultCompareDurable
		}
	}
	if config.ConsumerConcurrency <= 0 {
		config.ConsumerConcurrency = 1
	}
	if config.MaxInFlight < config.ConsumerConcurrency {
		config.MaxInFlight = config.ConsumerConcurrency
	}
	return config
}
