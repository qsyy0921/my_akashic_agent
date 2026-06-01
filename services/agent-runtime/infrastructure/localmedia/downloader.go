package localmedia

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type Downloader struct {
	root     string
	client   *http.Client
	maxBytes int64
}

func NewDownloader(root string, maxBytes int64) (*Downloader, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("media asset recovery cache root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	abs = filepath.Clean(abs)
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	if maxBytes <= 0 {
		maxBytes = 100 * 1024 * 1024
	}
	return &Downloader{
		root: abs,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		maxBytes: maxBytes,
	}, nil
}

func (d *Downloader) RecoverMediaAssetContent(ctx context.Context, asset model.MediaAsset) (outport.RecoveredMediaAssetContent, error) {
	if err := ctx.Err(); err != nil {
		return outport.RecoveredMediaAssetContent{}, err
	}
	if d == nil || d.client == nil || strings.TrimSpace(d.root) == "" {
		return outport.RecoveredMediaAssetContent{}, outport.ErrMediaAssetContentDisabled
	}
	sourceURL := strings.TrimSpace(asset.Metadata["recovered_from_url"])
	if sourceURL == "" {
		sourceURL = strings.TrimSpace(asset.Metadata["source_url"])
	}
	if sourceURL == "" {
		sourceURL = strings.TrimSpace(asset.URL)
	}
	parsed, err := url.Parse(sourceURL)
	if err != nil || parsed.Scheme == "" {
		return outport.RecoveredMediaAssetContent{}, outport.ErrMediaAssetRecoveryUnsupported
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return outport.RecoveredMediaAssetContent{}, outport.ErrMediaAssetRecoveryUnsupported
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return outport.RecoveredMediaAssetContent{}, err
	}
	response, err := d.client.Do(request)
	if err != nil {
		return outport.RecoveredMediaAssetContent{}, outport.ErrMediaAssetContentUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return outport.RecoveredMediaAssetContent{}, outport.ErrMediaAssetContentUnavailable
	}
	contentLength := response.ContentLength
	if contentLength > d.maxBytes {
		return outport.RecoveredMediaAssetContent{}, outport.ErrMediaAssetContentForbidden
	}
	name := recoveredMediaFilename(asset, response.Header.Get("Content-Type"), parsed)
	target := filepath.Join(d.root, name)
	if !pathInsideRoot(d.root, target) {
		return outport.RecoveredMediaAssetContent{}, outport.ErrMediaAssetContentForbidden
	}
	tmp, err := os.CreateTemp(d.root, ".recover-*")
	if err != nil {
		return outport.RecoveredMediaAssetContent{}, err
	}
	tmpPath := tmp.Name()
	hash := sha256.New()
	limited := io.LimitReader(response.Body, d.maxBytes+1)
	written, copyErr := io.Copy(io.MultiWriter(tmp, hash), limited)
	closeErr := tmp.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return outport.RecoveredMediaAssetContent{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return outport.RecoveredMediaAssetContent{}, closeErr
	}
	if written > d.maxBytes {
		_ = os.Remove(tmpPath)
		return outport.RecoveredMediaAssetContent{}, outport.ErrMediaAssetContentForbidden
	}
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		return outport.RecoveredMediaAssetContent{}, err
	}
	mimeType := strings.TrimSpace(response.Header.Get("Content-Type"))
	if semicolon := strings.Index(mimeType, ";"); semicolon >= 0 {
		mimeType = strings.TrimSpace(mimeType[:semicolon])
	}
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(target))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return outport.RecoveredMediaAssetContent{
		LocalPath:   target,
		Name:        filepath.Base(target),
		MimeType:    mimeType,
		SizeBytes:   written,
		ContentHash: "sha256:" + hex.EncodeToString(hash.Sum(nil)),
		SourceURL:   sourceURL,
	}, nil
}

func recoveredMediaFilename(asset model.MediaAsset, contentType string, parsed *url.URL) string {
	ext := strings.TrimSpace(filepath.Ext(parsed.Path))
	if ext == "" {
		if values, _, err := mime.ParseMediaType(contentType); err == nil {
			if candidates, err := mime.ExtensionsByType(values); err == nil && len(candidates) > 0 {
				ext = candidates[0]
			}
		}
	}
	if ext == "" {
		ext = strings.TrimSpace(filepath.Ext(asset.Name))
	}
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%s%s", safeMediaAssetID(asset.AssetID), ext)
}

func safeMediaAssetID(assetID string) string {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return "media-asset"
	}
	replacer := strings.NewReplacer(":", "_", "/", "_", "\\", "_", "?", "_", "&", "_", "=", "_", " ", "_")
	value := replacer.Replace(assetID)
	if len(value) > 160 {
		sum := sha256.Sum256([]byte(assetID))
		value = value[:120] + "_" + hex.EncodeToString(sum[:])[:16]
	}
	return value
}
