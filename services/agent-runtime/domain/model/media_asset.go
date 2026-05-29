package model

import (
	"errors"
	"strings"
	"time"
)

type MediaAssetKind string

const (
	MediaAssetImage   MediaAssetKind = "image"
	MediaAssetFile    MediaAssetKind = "file"
	MediaAssetAudio   MediaAssetKind = "audio"
	MediaAssetVideo   MediaAssetKind = "video"
	MediaAssetUnknown MediaAssetKind = "unknown"
)

type MediaAsset struct {
	AssetID         string
	Channel         ChannelRef
	SourceMessageID string
	SenderID        string
	Kind            MediaAssetKind
	URL             string
	MimeType        string
	Name            string
	SizeBytes       int64
	ContentHash     string
	Retention       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Metadata        map[string]string
}

type MediaAssetSpec struct {
	AssetID         string
	SourceMessageID string
	SenderID        string
	Kind            MediaAssetKind
	URL             string
	MimeType        string
	Name            string
	SizeBytes       int64
	ContentHash     string
	Retention       string
	Metadata        map[string]string
}

func NewMediaAsset(channel ChannelRef, spec MediaAssetSpec, now time.Time) (MediaAsset, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if spec.Kind == "" {
		spec.Kind = MediaAssetUnknown
	}
	if strings.TrimSpace(spec.Retention) == "" {
		spec.Retention = "default"
	}
	asset := MediaAsset{
		AssetID:         strings.TrimSpace(spec.AssetID),
		Channel:         channel,
		SourceMessageID: strings.TrimSpace(spec.SourceMessageID),
		SenderID:        strings.TrimSpace(spec.SenderID),
		Kind:            spec.Kind,
		URL:             strings.TrimSpace(spec.URL),
		MimeType:        strings.TrimSpace(spec.MimeType),
		Name:            strings.TrimSpace(spec.Name),
		SizeBytes:       spec.SizeBytes,
		ContentHash:     strings.TrimSpace(spec.ContentHash),
		Retention:       strings.TrimSpace(spec.Retention),
		CreatedAt:       now,
		UpdatedAt:       now,
		Metadata:        spec.Metadata,
	}
	if err := asset.Validate(); err != nil {
		return MediaAsset{}, err
	}
	return asset, nil
}

func (a MediaAsset) Validate() error {
	if strings.TrimSpace(a.AssetID) == "" {
		return errors.New("media asset requires asset id")
	}
	if strings.TrimSpace(string(a.Channel.Kind)) == "" {
		return errors.New("media asset requires channel kind")
	}
	if strings.TrimSpace(a.Channel.AccountID) == "" {
		return errors.New("media asset requires channel account id")
	}
	if strings.TrimSpace(a.Channel.ConversationID) == "" {
		return errors.New("media asset requires conversation id")
	}
	if strings.TrimSpace(string(a.Channel.ConversationType)) == "" {
		return errors.New("media asset requires conversation type")
	}
	if strings.TrimSpace(a.SourceMessageID) == "" {
		return errors.New("media asset requires source message id")
	}
	if strings.TrimSpace(a.SenderID) == "" {
		return errors.New("media asset requires sender id")
	}
	if a.SizeBytes < 0 {
		return errors.New("media asset size cannot be negative")
	}
	if strings.TrimSpace(a.URL) == "" && strings.TrimSpace(a.Name) == "" {
		return errors.New("media asset requires url or name")
	}
	if a.CreatedAt.IsZero() {
		return errors.New("media asset requires created_at")
	}
	if a.UpdatedAt.IsZero() {
		return errors.New("media asset requires updated_at")
	}
	switch a.Kind {
	case MediaAssetImage, MediaAssetFile, MediaAssetAudio, MediaAssetVideo, MediaAssetUnknown:
		return nil
	default:
		return errors.New("media asset has invalid kind")
	}
}

