package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
)

type SendLedger interface {
	RecentlySent(botID string, conversationID string, contentHash string, window time.Duration) bool
}

type NonceLedger interface {
	SeenNonce(nonce string, window time.Duration) bool
}

type LoopGuard struct {
	selfBotIDs     map[string]struct{}
	sendEchoWindow time.Duration
	nonceWindow    time.Duration
	maxHop         int
}

func NewLoopGuard(selfBotIDs []string, sendEchoWindow time.Duration, maxHop int) LoopGuard {
	ids := make(map[string]struct{}, len(selfBotIDs))
	for _, id := range selfBotIDs {
		ids[id] = struct{}{}
	}
	if sendEchoWindow <= 0 {
		sendEchoWindow = 15 * time.Second
	}
	if maxHop <= 0 {
		maxHop = 6
	}
	return LoopGuard{
		selfBotIDs:     ids,
		sendEchoWindow: sendEchoWindow,
		nonceWindow:    10 * time.Minute,
		maxHop:         maxHop,
	}
}

func (g LoopGuard) Decide(message model.MessageEnvelope, sendLedger SendLedger, nonceLedger NonceLedger) model.LoopDecision {
	if message.Provenance.Type == model.ProvenanceSelfEcho || message.Sender.ID == message.Channel.AccountID {
		return model.LoopDecision{Action: model.LoopActionObserveOnly, Reason: "self bot echo"}
	}

	if message.Provenance.Type == model.ProvenancePeerBot && !message.Provenance.HasProtocolTag {
		return model.LoopDecision{Action: model.LoopActionObserveOnly, Reason: "peer bot message without protocol tag"}
	}

	if message.Provenance.HasProtocolTag && message.Provenance.Hop > g.maxHop {
		return model.LoopDecision{Action: model.LoopActionDrop, Reason: "bot protocol hop limit exceeded"}
	}

	if message.Provenance.Nonce != "" && nonceLedger != nil && nonceLedger.SeenNonce(message.Provenance.Nonce, g.nonceWindow) {
		return model.LoopDecision{Action: model.LoopActionObserveOnly, Reason: "bot protocol nonce already seen"}
	}

	hash := message.Provenance.ContentHash
	if hash == "" {
		hash = ContentHash(message.Content)
	}
	if sendLedger != nil && sendLedger.RecentlySent(
		message.Channel.AccountID,
		message.Channel.ConversationID,
		hash,
		g.sendEchoWindow,
	) {
		return model.LoopDecision{Action: model.LoopActionObserveOnly, Reason: "recent outbound echo"}
	}

	return model.LoopDecision{Action: model.LoopActionAllow, Reason: "accepted"}
}

func ContentHash(content string) string {
	normalized := strings.TrimSpace(content)
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
