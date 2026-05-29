package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	domainservice "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/service"
)

type SendLedgerService struct {
	ledger outport.SendLedger
}

func NewSendLedgerService(ledger outport.SendLedger) *SendLedgerService {
	return &SendLedgerService{ledger: ledger}
}

func (s *SendLedgerService) Record(ctx context.Context, cmd command.RecordSendCommand) (query.SendRecordView, error) {
	if s == nil || s.ledger == nil {
		return query.SendRecordView{}, errors.New("send ledger service requires ledger")
	}
	if cmd.Timestamp.IsZero() {
		cmd.Timestamp = time.Now().UTC()
	}
	contentHash, err := contentHashFromLedgerInput(cmd.Content, cmd.ContentHash)
	if err != nil {
		return query.SendRecordView{}, err
	}
	record := model.SendRecord{
		FromBotID:      strings.TrimSpace(cmd.FromBotID),
		ConversationID: strings.TrimSpace(cmd.ConversationID),
		ContentHash:    contentHash,
		Timestamp:      cmd.Timestamp,
	}
	if err := record.Validate(); err != nil {
		return query.SendRecordView{}, err
	}
	if err := s.ledger.RecordSent(ctx, record); err != nil {
		return query.SendRecordView{}, err
	}
	return assembler.ToSendRecordView(record), nil
}

func (s *SendLedgerService) RecentlySent(ctx context.Context, cmd command.CheckRecentSendCommand) (query.RecentSendView, error) {
	if s == nil || s.ledger == nil {
		return query.RecentSendView{}, errors.New("send ledger service requires ledger")
	}
	fromBotID := strings.TrimSpace(cmd.FromBotID)
	conversationID := strings.TrimSpace(cmd.ConversationID)
	if fromBotID == "" {
		return query.RecentSendView{}, errors.New("from bot id required")
	}
	if conversationID == "" {
		return query.RecentSendView{}, errors.New("conversation id required")
	}
	contentHash, err := contentHashFromLedgerInput(cmd.Content, cmd.ContentHash)
	if err != nil {
		return query.RecentSendView{}, err
	}
	window := cmd.Window
	if window <= 0 {
		window = 15 * time.Second
	}
	recent := s.ledger.RecentlySent(fromBotID, conversationID, contentHash, window)
	return query.RecentSendView{
		Recent:         recent,
		FromBotID:      fromBotID,
		ConversationID: conversationID,
		ContentHash:    contentHash,
		WindowSeconds:  int(window.Seconds()),
	}, nil
}

func (s *SendLedgerService) List(ctx context.Context, filter query.SendRecordFilter) ([]query.SendRecordView, error) {
	if s == nil || s.ledger == nil {
		return nil, errors.New("send ledger service requires ledger")
	}
	items, err := s.ledger.ListSentRecords(ctx, filter)
	if err != nil {
		return nil, err
	}
	return assembler.ToSendRecordViews(items), nil
}

func contentHashFromLedgerInput(content string, contentHash string) (string, error) {
	contentHash = strings.TrimSpace(contentHash)
	if contentHash != "" {
		return contentHash, nil
	}
	if strings.TrimSpace(content) == "" {
		return "", errors.New("content or content_hash required")
	}
	return domainservice.ContentHash(content), nil
}
