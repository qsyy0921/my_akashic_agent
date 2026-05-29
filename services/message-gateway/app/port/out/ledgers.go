package outport

import (
	"context"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
)

type SendLedger interface {
	RecentlySent(botID string, conversationID string, contentHash string, window time.Duration) bool
	RecordSent(ctx context.Context, record model.SendRecord) error
}

type NonceLedger interface {
	SeenNonce(nonce string, window time.Duration) bool
	RecordNonce(ctx context.Context, nonce string, seenAt time.Time) error
}
