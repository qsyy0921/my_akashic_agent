package service

import "github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"

type ProvenanceClassifier struct {
	selfBotIDs map[string]struct{}
}

func NewProvenanceClassifier(selfBotIDs []string) ProvenanceClassifier {
	ids := make(map[string]struct{}, len(selfBotIDs))
	for _, id := range selfBotIDs {
		ids[id] = struct{}{}
	}
	return ProvenanceClassifier{selfBotIDs: ids}
}

func (c ProvenanceClassifier) Classify(message model.MessageEnvelope) model.Provenance {
	provenance := model.Provenance{
		Type:        model.ProvenanceHuman,
		ContentHash: ContentHash(message.Content),
	}

	if message.Sender.ID == message.Channel.AccountID || message.Sender.Kind == model.SenderKindSelf {
		provenance.Type = model.ProvenanceSelfEcho
		provenance.FromBotID = message.Sender.ID
	}

	if protocol, ok := ParseBotProtocolTag(message.Content); ok {
		provenance.Type = model.ProvenancePeerBot
		provenance.FromBotID = protocol.FromBotID
		provenance.Nonce = protocol.Nonce
		provenance.Hop = protocol.Hop
		provenance.HasProtocolTag = true
		return provenance
	}

	if _, ok := c.selfBotIDs[message.Sender.ID]; ok {
		provenance.Type = model.ProvenancePeerBot
		provenance.FromBotID = message.Sender.ID
		return provenance
	}

	if message.Sender.Kind == model.SenderKindBot {
		provenance.Type = model.ProvenancePeerBot
		provenance.FromBotID = message.Sender.ID
	}

	return provenance
}
