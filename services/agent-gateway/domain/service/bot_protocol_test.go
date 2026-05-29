package service_test

import (
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/domain/service"
)

func TestBotProtocolRoundTrip(t *testing.T) {
	content := service.PrependBotProtocol("hello", model.BotProtocol{
		FromBotID: "1049511700",
		Nonce:     "abc123",
		Hop:       3,
	})

	protocol, ok := service.ParseBotProtocolTag(content)
	if !ok {
		t.Fatal("expected bot protocol tag to be parsed")
	}
	if protocol.FromBotID != "1049511700" || protocol.Nonce != "abc123" || protocol.Hop != 3 {
		t.Fatalf("unexpected protocol: %+v", protocol)
	}
}
