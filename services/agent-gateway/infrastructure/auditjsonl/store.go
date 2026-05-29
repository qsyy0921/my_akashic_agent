package auditjsonl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
)

const schemaVersion = "2026-05-30.shadow-audit.v1"

type Store struct {
	mu      sync.Mutex
	path    string
	records []auditRecord
}

type auditRecord struct {
	SchemaVersion    string             `json:"schema_version"`
	EventID          string             `json:"event_id"`
	Platform         string             `json:"platform"`
	AccountID        string             `json:"account_id"`
	ConversationID   string             `json:"conversation_id"`
	ConversationType string             `json:"conversation_type"`
	SenderID         string             `json:"sender_id"`
	SenderKind       string             `json:"sender_kind"`
	Content          string             `json:"content"`
	Timestamp        string             `json:"timestamp"`
	Attachments      []attachmentRecord `json:"attachments,omitempty"`
	Metadata         map[string]string  `json:"metadata,omitempty"`
	Provenance       provenanceRecord   `json:"provenance,omitempty"`
	DecisionAction   string             `json:"decision_action"`
	DecisionReason   string             `json:"decision_reason"`
	RecordedAt       string             `json:"recorded_at"`
}

type attachmentRecord struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	URL       string `json:"url,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Name      string `json:"name,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type provenanceRecord struct {
	Type           string `json:"type,omitempty"`
	FromBotID      string `json:"from_bot_id,omitempty"`
	ContentHash    string `json:"content_hash,omitempty"`
	Nonce          string `json:"nonce,omitempty"`
	Hop            int    `json:"hop,omitempty"`
	HasProtocolTag bool   `json:"has_protocol_tag,omitempty"`
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("audit jsonl path is required")
	}
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	store := &Store{path: cleanPath}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) RecordMessageDecision(_ context.Context, envelope model.MessageEnvelope, decision model.LoopDecision) error {
	if s == nil {
		return errors.New("audit jsonl store is nil")
	}
	if err := envelope.Validate(); err != nil {
		return err
	}
	record := toAuditRecord(envelope, decision, time.Now().UTC())
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	s.records = append(s.records, record)
	return nil
}

func (s *Store) ListShadowObserved(_ context.Context, limit int) ([]outport.ShadowObservedRecord, error) {
	if s == nil {
		return nil, errors.New("audit jsonl store is nil")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	start := len(s.records) - limit
	if start < 0 {
		start = 0
	}
	records := make([]outport.ShadowObservedRecord, 0, len(s.records)-start)
	for i := len(s.records) - 1; i >= start; i-- {
		records = append(records, s.records[i].toShadowObservedRecord())
	}
	return records, nil
}

func (s *Store) load() error {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 2*1024*1024)
	for scanner.Scan() {
		var record auditRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		if record.EventID == "" {
			continue
		}
		s.records = append(s.records, record)
	}
	return scanner.Err()
}

func toAuditRecord(envelope model.MessageEnvelope, decision model.LoopDecision, recordedAt time.Time) auditRecord {
	attachments := make([]attachmentRecord, 0, len(envelope.Attachments))
	for _, attachment := range envelope.Attachments {
		attachments = append(attachments, attachmentRecord{
			ID:        attachment.ID,
			Kind:      string(attachment.Kind),
			URL:       attachment.URL,
			MimeType:  attachment.MimeType,
			Name:      attachment.Name,
			SizeBytes: attachment.SizeBytes,
		})
	}
	return auditRecord{
		SchemaVersion:    schemaVersion,
		EventID:          envelope.EventID,
		Platform:         string(envelope.Channel.Kind),
		AccountID:        envelope.Channel.AccountID,
		ConversationID:   envelope.Channel.ConversationID,
		ConversationType: string(envelope.Channel.ConversationType),
		SenderID:         envelope.Sender.ID,
		SenderKind:       string(envelope.Sender.Kind),
		Content:          envelope.Content,
		Timestamp:        envelope.Timestamp.UTC().Format(time.RFC3339Nano),
		Attachments:      attachments,
		Metadata:         copyStringMap(envelope.Metadata),
		Provenance: provenanceRecord{
			Type:           string(envelope.Provenance.Type),
			FromBotID:      envelope.Provenance.FromBotID,
			ContentHash:    envelope.Provenance.ContentHash,
			Nonce:          envelope.Provenance.Nonce,
			Hop:            envelope.Provenance.Hop,
			HasProtocolTag: envelope.Provenance.HasProtocolTag,
		},
		DecisionAction: string(decision.Action),
		DecisionReason: decision.Reason,
		RecordedAt:     recordedAt.Format(time.RFC3339Nano),
	}
}

func (r auditRecord) toShadowObservedRecord() outport.ShadowObservedRecord {
	timestamp, _ := time.Parse(time.RFC3339Nano, r.Timestamp)
	attachments := make([]model.Attachment, 0, len(r.Attachments))
	for _, attachment := range r.Attachments {
		attachments = append(attachments, model.Attachment{
			ID:        attachment.ID,
			Kind:      model.AttachmentKind(attachment.Kind),
			URL:       attachment.URL,
			MimeType:  attachment.MimeType,
			Name:      attachment.Name,
			SizeBytes: attachment.SizeBytes,
		})
	}
	return outport.ShadowObservedRecord{
		Envelope: model.MessageEnvelope{
			EventID: r.EventID,
			Channel: model.ChannelRef{
				Kind:             model.ChannelKind(r.Platform),
				AccountID:        r.AccountID,
				ConversationID:   r.ConversationID,
				ConversationType: model.ConversationType(r.ConversationType),
			},
			Sender: model.SenderRef{
				ID:   r.SenderID,
				Kind: model.SenderKind(r.SenderKind),
			},
			Content:     r.Content,
			Attachments: attachments,
			Timestamp:   timestamp,
			Metadata:    copyStringMap(r.Metadata),
			Provenance: model.Provenance{
				Type:           model.ProvenanceType(r.Provenance.Type),
				FromBotID:      r.Provenance.FromBotID,
				ContentHash:    r.Provenance.ContentHash,
				Nonce:          r.Provenance.Nonce,
				Hop:            r.Provenance.Hop,
				HasProtocolTag: r.Provenance.HasProtocolTag,
			},
		},
		Decision: model.LoopDecision{
			Action: model.LoopAction(r.DecisionAction),
			Reason: r.DecisionReason,
		},
	}
}

func copyStringMap(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
