package mediaassetstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const storeVersion = "2026-05-30.mediaassetstore.v1"

// Store persists Go-owned media registry records for observed attachments.
type Store struct {
	mu     sync.Mutex
	path   string
	assets map[string]model.MediaAsset
	order  []string
}

type persistedState struct {
	Version string             `json:"version"`
	Assets  []model.MediaAsset `json:"assets"`
}

func NewStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("media asset store path is required")
	}
	cleanPath := filepath.Clean(path)
	store := &Store{
		path:   cleanPath,
		assets: make(map[string]model.MediaAsset),
		order:  make([]string, 0),
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) SaveMediaAsset(_ context.Context, asset model.MediaAsset) error {
	if err := asset.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.assets[asset.AssetID]; !exists {
		s.order = append(s.order, asset.AssetID)
	}
	s.assets[asset.AssetID] = asset
	return s.flush()
}

func (s *Store) FindMediaAsset(_ context.Context, assetID string) (model.MediaAsset, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[assetID]
	if !ok {
		return model.MediaAsset{}, false, nil
	}
	return asset, true, nil
}

func (s *Store) ListMediaAssets(_ context.Context, filter query.MediaAssetFilter) ([]model.MediaAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items := make([]model.MediaAsset, 0, limit)
	for i := len(s.order) - 1; i >= 0 && len(items) < limit; i-- {
		assetID := s.order[i]
		if asset, ok := s.assets[assetID]; ok && matchesMediaAssetFilter(asset, filter) {
			items = append(items, asset)
		}
	}
	return items, nil
}

func (s *Store) DeleteMediaAsset(_ context.Context, assetID string) (bool, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.assets[assetID]; !ok {
		return false, nil
	}
	delete(s.assets, assetID)
	for i, item := range s.order {
		if item == assetID {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true, s.flush()
}

func matchesMediaAssetFilter(asset model.MediaAsset, filter query.MediaAssetFilter) bool {
	if filter.ChannelKind != "" && string(asset.Channel.Kind) != filter.ChannelKind {
		return false
	}
	if filter.AccountID != "" && asset.Channel.AccountID != filter.AccountID {
		return false
	}
	if filter.ConversationID != "" && asset.Channel.ConversationID != filter.ConversationID {
		return false
	}
	if filter.ConversationType != "" && string(asset.Channel.ConversationType) != filter.ConversationType {
		return false
	}
	if filter.SourceMessageID != "" && asset.SourceMessageID != filter.SourceMessageID {
		return false
	}
	if filter.SourceMessageIDSuffix != "" && !strings.HasSuffix(asset.SourceMessageID, ":"+filter.SourceMessageIDSuffix) && asset.SourceMessageID != filter.SourceMessageIDSuffix {
		return false
	}
	if filter.Kind != "" && string(asset.Kind) != filter.Kind {
		return false
	}
	return true
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}

	var state persistedState
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}
	for _, asset := range state.Assets {
		if asset.AssetID == "" {
			continue
		}
		if err := asset.Validate(); err != nil {
			continue
		}
		if _, exists := s.assets[asset.AssetID]; !exists {
			s.order = append(s.order, asset.AssetID)
		}
		s.assets[asset.AssetID] = asset
	}
	return nil
}

func (s *Store) flush() error {
	state := persistedState{
		Version: storeVersion,
		Assets:  make([]model.MediaAsset, 0, len(s.order)),
	}
	for _, assetID := range s.order {
		if asset, ok := s.assets[assetID]; ok {
			state.Assets = append(state.Assets, asset)
		}
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return err
	}
	if _, err := os.Stat(s.path); err == nil {
		if err := os.Remove(s.path); err != nil {
			return err
		}
	}
	return os.Rename(tmpPath, s.path)
}
