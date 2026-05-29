package agentgateway_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/api/dto"
)

var requiredContractFixtures = []string{
	"message_envelope.qq.private.text.json",
	"message_envelope.qq.group.image.json",
	"agent_inbound.allowed.private.json",
	"agent_inbound.observe_only.group.json",
	"agent_decision.reply.private.json",
	"agent_decision.create_image_job.json",
	"media_asset.qq.image.json",
	"media_asset.qq.file.json",
	"image_job.lifecycle.json",
	"memory_extract_job.group_thread.json",
	"rag_ingest_job.thread_summary.json",
	"group_thread.hardware.json",
	"group_thread.game_guide.json",
	"source_citation.message_asset.json",
}

func TestContractFixturesLoadInGo(t *testing.T) {
	dir := contractFixtureDir(t)
	for _, name := range requiredContractFixtures {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			var fixture dto.ContractFixture
			if err := json.Unmarshal(raw, &fixture); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}
			if err := fixture.Validate(); err != nil {
				t.Fatalf("validate fixture: %v", err)
			}

			encoded, err := json.Marshal(fixture)
			if err != nil {
				t.Fatalf("marshal fixture: %v", err)
			}
			var roundTripped dto.ContractFixture
			if err := json.Unmarshal(encoded, &roundTripped); err != nil {
				t.Fatalf("unmarshal round-tripped fixture: %v", err)
			}
			if err := roundTripped.Validate(); err != nil {
				t.Fatalf("validate round-tripped fixture: %v", err)
			}
			if roundTripped.StableID() != fixture.StableID() {
				t.Fatalf("stable id changed: %s != %s", roundTripped.StableID(), fixture.StableID())
			}
			if len(roundTripped.SourceMessageIDs) != len(fixture.SourceMessageIDs) {
				t.Fatal("source_message_ids changed during round-trip")
			}
			if len(roundTripped.SourceAssetIDs) != len(fixture.SourceAssetIDs) {
				t.Fatal("source_asset_ids changed during round-trip")
			}
		})
	}
}

func TestMessageEnvelopeFixturesMatchGatewayDTO(t *testing.T) {
	dir := contractFixtureDir(t)
	for _, name := range []string{
		"message_envelope.qq.private.text.json",
		"message_envelope.qq.group.image.json",
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var request dto.IngestMessageRequest
			if err := json.Unmarshal(raw, &request); err != nil {
				t.Fatalf("unmarshal ingest request: %v", err)
			}
			assertNonEmpty(t, request.EventID, "event_id")
			assertNonEmpty(t, request.Channel.Kind, "channel.kind")
			assertNonEmpty(t, request.Channel.AccountID, "channel.account_id")
			assertNonEmpty(t, request.Channel.ConversationID, "channel.conversation_id")
			assertNonEmpty(t, request.Channel.ConversationType, "channel.conversation_type")
			assertNonEmpty(t, request.Sender.ID, "sender.id")
			assertNonEmpty(t, request.Timestamp, "timestamp")
			if request.Metadata == nil {
				t.Fatal("metadata must be preserved")
			}
		})
	}
}

func contractFixtureDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Clean(filepath.Join("..", "..", "tests", "fixtures", "contracts"))
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("contract fixture dir missing: %v", err)
	}
	return dir
}

func assertNonEmpty(t *testing.T, value string, field string) {
	t.Helper()
	if value == "" {
		t.Fatalf("%s is required", field)
	}
}

