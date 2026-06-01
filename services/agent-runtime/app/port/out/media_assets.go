package outport

import (
	"context"
	"errors"
	"io"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

var (
	ErrMediaAssetContentDisabled     = errors.New("media asset content access disabled")
	ErrMediaAssetContentUnavailable  = errors.New("media asset content unavailable")
	ErrMediaAssetContentForbidden    = errors.New("media asset content forbidden")
	ErrMediaAssetRecoveryUnsupported = errors.New("media asset content recovery source unsupported")
)

type MediaAssetRepository interface {
	SaveMediaAsset(ctx context.Context, asset model.MediaAsset) error
	FindMediaAsset(ctx context.Context, assetID string) (model.MediaAsset, bool, error)
	ListMediaAssets(ctx context.Context, filter query.MediaAssetFilter) ([]model.MediaAsset, error)
	DeleteMediaAsset(ctx context.Context, assetID string) (bool, error)
}

type MediaAssetContent struct {
	Name      string
	MimeType  string
	SizeBytes int64
	Body      io.ReadCloser
}

type MediaAssetContentReader interface {
	OpenMediaAssetContent(ctx context.Context, asset model.MediaAsset) (MediaAssetContent, error)
}

type RecoveredMediaAssetContent struct {
	LocalPath   string
	Name        string
	MimeType    string
	SizeBytes   int64
	ContentHash string
	SourceURL   string
}

type MediaAssetContentDownloader interface {
	RecoverMediaAssetContent(ctx context.Context, asset model.MediaAsset) (RecoveredMediaAssetContent, error)
}
