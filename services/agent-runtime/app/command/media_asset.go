package command

import "time"

type RegisterMediaAssetCommand struct {
	AssetID         string
	Channel         ChannelCommand
	SourceMessageID string
	SenderID        string
	Kind            string
	URL             string
	MimeType        string
	Name            string
	SizeBytes       int64
	ContentHash     string
	Retention       string
	Index           int
	Timestamp       time.Time
	Metadata        map[string]string
}

