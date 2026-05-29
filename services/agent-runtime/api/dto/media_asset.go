package dto

type RegisterMediaAssetRequest struct {
	AssetID         string            `json:"asset_id,omitempty"`
	Channel         ChannelDTO        `json:"channel"`
	SourceMessageID string            `json:"source_message_id"`
	SenderID        string            `json:"sender_id"`
	Kind            string            `json:"kind"`
	URL             string            `json:"url,omitempty"`
	MimeType        string            `json:"mime_type,omitempty"`
	Name            string            `json:"name,omitempty"`
	SizeBytes       int64             `json:"size_bytes,omitempty"`
	ContentHash     string            `json:"content_hash,omitempty"`
	Retention       string            `json:"retention,omitempty"`
	Index           int               `json:"index,omitempty"`
	Timestamp       string            `json:"timestamp,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

