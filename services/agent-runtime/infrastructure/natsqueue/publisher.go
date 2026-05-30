package natsqueue

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const (
	defaultConnectTimeout = 3 * time.Second
	defaultPublishTimeout = 3 * time.Second
)

type Config struct {
	URL           string
	Stream        string
	SubjectPrefix string
	Timeout       time.Duration
}

type Publisher struct {
	js            jetStreamPublisher
	conn          *nats.Conn
	stream        string
	subjectPrefix string
	timeout       time.Duration
}

type jetStreamPublisher interface {
	Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error)
	AddStream(cfg *nats.StreamConfig, opts ...nats.JSOpt) (*nats.StreamInfo, error)
	StreamInfo(stream string, opts ...nats.JSOpt) (*nats.StreamInfo, error)
}

type WorkNotification struct {
	SchemaVersion      string            `json:"schema_version"`
	WorkKind           string            `json:"work_kind"`
	WorkID             string            `json:"work_id"`
	AggregateID        string            `json:"aggregate_id"`
	Status             string            `json:"status"`
	Subject            string            `json:"subject"`
	Route              WorkRoute         `json:"route"`
	SourceEventIDs     []string          `json:"source_event_ids,omitempty"`
	SourceAssetIDs     []string          `json:"source_asset_ids,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	NotificationTime   string            `json:"notification_time"`
	CreatedAt          string            `json:"created_at,omitempty"`
	UpdatedAt          string            `json:"updated_at,omitempty"`
	ConsumerModel      string            `json:"consumer_model"`
	StateAuthoritative bool              `json:"state_authoritative"`
}

type WorkRoute struct {
	Kind             string `json:"kind"`
	AccountID        string `json:"account_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
}

func NewPublisher(config Config) (*Publisher, error) {
	config = normalizeConfig(config)
	if strings.TrimSpace(config.URL) == "" {
		return nil, errors.New("nats queue publisher requires url")
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
	publisher := &Publisher{
		js:            js,
		conn:          conn,
		stream:        config.Stream,
		subjectPrefix: config.SubjectPrefix,
		timeout:       config.Timeout,
	}
	if err := publisher.ensureStream(); err != nil {
		conn.Close()
		return nil, err
	}
	return publisher, nil
}

func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	p.conn.Close()
}

func (p *Publisher) PublishOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	subject := OutboxSubject(p.subjectPrefix, delivery)
	notification := OutboxNotification(subject, delivery, time.Now().UTC())
	return p.publish(subject, notification, delivery.Message.EventID)
}

func (p *Publisher) PublishAgentJob(ctx context.Context, job model.AgentJob) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	subject := AgentJobSubject(p.subjectPrefix, job)
	notification := AgentJobNotification(subject, job, time.Now().UTC())
	return p.publish(subject, notification, job.JobID)
}

func (p *Publisher) ensureStream() error {
	if p == nil || p.js == nil {
		return errors.New("nats queue publisher requires jetstream")
	}
	if _, err := p.js.StreamInfo(p.stream, nats.MaxWait(p.timeout)); err == nil {
		return nil
	}
	_, err := p.js.AddStream(&nats.StreamConfig{
		Name:     p.stream,
		Subjects: []string{p.subjectPrefix + ".>"},
		Storage:  nats.FileStorage,
	}, nats.MaxWait(p.timeout))
	return err
}

func (p *Publisher) publish(subject string, notification WorkNotification, msgID string) error {
	if p == nil || p.js == nil {
		return errors.New("nats queue publisher requires jetstream")
	}
	raw, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	_, err = p.js.Publish(subject, raw, nats.MsgId(msgID), nats.ExpectStream(p.stream), nats.AckWait(p.timeout))
	return err
}

func normalizeConfig(config Config) Config {
	config.URL = strings.TrimSpace(config.URL)
	config.Stream = safeStreamName(config.Stream)
	config.SubjectPrefix = strings.Trim(strings.TrimSpace(config.SubjectPrefix), ".")
	if config.SubjectPrefix == "" {
		config.SubjectPrefix = "akashic.work"
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultPublishTimeout
	}
	if config.Timeout < defaultConnectTimeout {
		config.Timeout = defaultConnectTimeout
	}
	return config
}

func OutboxSubject(prefix string, delivery model.OutboxDelivery) string {
	prefix = cleanSubjectPrefix(prefix)
	return strings.Join([]string{
		prefix,
		"outbox",
		safeSubjectToken(string(delivery.Message.Channel.Kind)),
		safeSubjectToken(delivery.Message.Channel.AccountID),
	}, ".")
}

func AgentJobSubject(prefix string, job model.AgentJob) string {
	prefix = cleanSubjectPrefix(prefix)
	return strings.Join([]string{
		prefix,
		"agent_job",
		safeSubjectToken(string(job.JobType)),
	}, ".")
}

func OutboxNotification(subject string, delivery model.OutboxDelivery, now time.Time) WorkNotification {
	return WorkNotification{
		SchemaVersion:      "1",
		WorkKind:           "outbox_delivery",
		WorkID:             delivery.Message.EventID,
		AggregateID:        delivery.Message.EventID,
		Status:             string(delivery.Status),
		Subject:            subject,
		Route:              routeNotification(delivery.Message.Channel),
		Metadata:           cloneMap(delivery.Message.Metadata),
		NotificationTime:   formatTime(now),
		CreatedAt:          formatTime(delivery.CreatedAt),
		UpdatedAt:          formatTime(delivery.UpdatedAt),
		ConsumerModel:      "goroutine_worker_pool",
		StateAuthoritative: true,
	}
}

func AgentJobNotification(subject string, job model.AgentJob, now time.Time) WorkNotification {
	return WorkNotification{
		SchemaVersion:      "1",
		WorkKind:           "agent_job",
		WorkID:             job.JobID,
		AggregateID:        job.JobID,
		Status:             string(job.Status),
		Subject:            subject,
		Route:              routeNotification(job.Route),
		SourceEventIDs:     append([]string(nil), job.SourceEventIDs...),
		SourceAssetIDs:     append([]string(nil), job.SourceAssetIDs...),
		Metadata:           cloneMap(job.Metadata),
		NotificationTime:   formatTime(now),
		CreatedAt:          formatTime(job.CreatedAt),
		UpdatedAt:          formatTime(job.UpdatedAt),
		ConsumerModel:      "goroutine_worker_pool",
		StateAuthoritative: true,
	}
}

func routeNotification(route model.ChannelRef) WorkRoute {
	return WorkRoute{
		Kind:             string(route.Kind),
		AccountID:        route.AccountID,
		ConversationID:   route.ConversationID,
		ConversationType: string(route.ConversationType),
	}
}

func cleanSubjectPrefix(prefix string) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), ".")
	if prefix == "" {
		return "akashic.work"
	}
	parts := strings.Split(prefix, ".")
	for i, part := range parts {
		parts[i] = safeSubjectToken(part)
	}
	return strings.Join(parts, ".")
}

func safeStreamName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "AKASHIC_WORK"
	}
	replacer := strings.NewReplacer(".", "_", "*", "_", ">", "_", "/", "_", "\\", "_")
	return safeSubjectToken(replacer.Replace(value))
}

var unsafeSubjectToken = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func safeSubjectToken(value string) string {
	value = unsafeSubjectToken.ReplaceAllString(strings.TrimSpace(value), "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "unknown"
	}
	return value
}

func cloneMap(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(items))
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

var _ outport.WorkQueuePublisher = (*Publisher)(nil)
