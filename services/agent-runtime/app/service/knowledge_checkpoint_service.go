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
)

type KnowledgeCheckpointService struct {
	repository outport.KnowledgeCheckpointRepository
}

func NewKnowledgeCheckpointService(repository outport.KnowledgeCheckpointRepository) *KnowledgeCheckpointService {
	return &KnowledgeCheckpointService{repository: repository}
}

func (s *KnowledgeCheckpointService) Get(ctx context.Context, checkpointID string) (query.KnowledgeCheckpointView, error) {
	if s == nil || s.repository == nil {
		return query.KnowledgeCheckpointView{}, errors.New("knowledge checkpoint service requires repository")
	}
	checkpointID = strings.TrimSpace(checkpointID)
	if checkpointID == "" {
		return query.KnowledgeCheckpointView{}, errors.New("checkpoint id required")
	}
	checkpoint, ok, err := s.repository.FindKnowledgeCheckpoint(ctx, checkpointID)
	if err != nil {
		return query.KnowledgeCheckpointView{}, err
	}
	if !ok {
		return query.KnowledgeCheckpointView{}, errors.New("knowledge checkpoint not found")
	}
	return assembler.ToKnowledgeCheckpointView(checkpoint), nil
}

func (s *KnowledgeCheckpointService) Upsert(ctx context.Context, cmd command.UpsertKnowledgeCheckpointCommand) (query.KnowledgeCheckpointView, error) {
	if s == nil || s.repository == nil {
		return query.KnowledgeCheckpointView{}, errors.New("knowledge checkpoint service requires repository")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	checkpointID := strings.TrimSpace(cmd.CheckpointID)
	if checkpointID == "" {
		return query.KnowledgeCheckpointView{}, errors.New("checkpoint id required")
	}
	checkpoint, ok, err := s.repository.FindKnowledgeCheckpoint(ctx, checkpointID)
	if err != nil {
		return query.KnowledgeCheckpointView{}, err
	}
	if ok {
		if err := checkpoint.Advance(cmd.Cursor, cmd.Metadata, now); err != nil {
			return query.KnowledgeCheckpointView{}, err
		}
	} else {
		checkpoint, err = model.NewKnowledgeCheckpoint(checkpointID, cmd.Cursor, cmd.Metadata, now)
		if err != nil {
			return query.KnowledgeCheckpointView{}, err
		}
	}
	if err := s.repository.SaveKnowledgeCheckpoint(ctx, checkpoint); err != nil {
		return query.KnowledgeCheckpointView{}, err
	}
	return assembler.ToKnowledgeCheckpointView(checkpoint), nil
}
