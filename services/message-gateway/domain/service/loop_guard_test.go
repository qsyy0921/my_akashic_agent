package service_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/service"
)

type nonceLedgerStub struct {
	seen bool
}

func (l nonceLedgerStub) SeenNonce(_ string, _ time.Duration) bool {
	return l.seen
}

func TestLoopGuardAllowsTaggedPeerBotWithinBudget(t *testing.T) {
	guard := service.NewLoopGuard([]string{"1049511700", "2365524513"}, time.Second, 6)
	decision := guard.Decide(model.MessageEnvelope{
		Channel: model.ChannelRef{AccountID: "1049511700", ConversationID: "2365524513"},
		Sender:  model.SenderRef{ID: "2365524513", Kind: model.SenderKindBot},
		Content: "hello",
		Provenance: model.Provenance{
			Type:           model.ProvenancePeerBot,
			FromBotID:      "2365524513",
			Nonce:          "n-1",
			Hop:            2,
			HasProtocolTag: true,
		},
	}, nil, nonceLedgerStub{})

	if decision.Action != model.LoopActionAllow {
		t.Fatalf("expected allow, got %+v", decision)
	}
}

func TestLoopGuardObservesUntaggedPeerBot(t *testing.T) {
	guard := service.NewLoopGuard([]string{"1049511700", "2365524513"}, time.Second, 6)
	decision := guard.Decide(model.MessageEnvelope{
		Channel:    model.ChannelRef{AccountID: "1049511700", ConversationID: "2365524513"},
		Sender:     model.SenderRef{ID: "2365524513", Kind: model.SenderKindBot},
		Content:    "plain bot message",
		Provenance: model.Provenance{Type: model.ProvenancePeerBot},
	}, nil, nonceLedgerStub{})

	if decision.Action != model.LoopActionObserveOnly {
		t.Fatalf("expected observe-only, got %+v", decision)
	}
}
