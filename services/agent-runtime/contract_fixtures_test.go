package agentruntime_test

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
	"rag_eval_job.group_memory.json",
	"knowledge_checkpoint.ragflow.qq.json",
	"inbox_replay.observe_only.qq.json",
	"outbox_delivery.qq.private.text.json",
	"outbox_delivery_event.qq.text.json",
	"media_asset_content.qq.image.json",
	"media_asset_content_access_plan.qq.image.json",
	"agent_job_event_stream.rag_ingest.json",
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
			if len(roundTripped.Extra) != len(fixture.Extra) {
				t.Fatal("extra contract fields changed during round-trip")
			}
		})
	}
}

func TestRuntimeBoundaryFixturesCoverCurrentGoOwnedContracts(t *testing.T) {
	checks := map[string][]string{
		"knowledge_checkpoint.ragflow.qq.json": {
			"checkpoint_id",
			"cursor",
		},
		"inbox_replay.observe_only.qq.json": {
			"replay",
		},
		"outbox_delivery.qq.private.text.json": {
			"delivery",
			"content",
		},
		"outbox_delivery_event.qq.text.json": {
			"delivery_id",
			"events",
		},
		"media_asset_content.qq.image.json": {
			"asset_id",
			"content_access",
		},
		"media_asset_content_access_plan.qq.image.json": {
			"asset_id",
			"content_access_plan",
		},
		"agent_job_event_stream.rag_ingest.json": {
			"job_type",
			"events",
		},
		"rag_eval_job.group_memory.json": {
			"job_type",
			"payload",
		},
	}

	dir := contractFixtureDir(t)
	for name, keys := range checks {
		t.Run(name, func(t *testing.T) {
			fixture := loadContractFixture(t, filepath.Join(dir, name))
			for _, key := range keys {
				if _, ok := fixture.Extra[key]; !ok {
					t.Fatalf("missing extra contract field %q", key)
				}
			}
		})
	}
}

func TestMediaAssetContentAccessPlanContractShape(t *testing.T) {
	fixture := loadContractFixture(t, filepath.Join(
		contractFixtureDir(t),
		"media_asset_content_access_plan.qq.image.json",
	))
	raw, ok := fixture.Extra["content_access_plan"]
	if !ok {
		t.Fatal("missing content_access_plan")
	}
	var plan map[string]any
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatalf("content_access_plan must be an object: %v", err)
	}
	if ready, ok := plan["ready"].(bool); !ok || !ready {
		t.Fatalf("unexpected ready value: %#v", plan["ready"])
	}
	assertExtraString(t, plan, "reason", "media_asset_content_ready")
	assertExtraString(t, plan, "side_effect", "none")
	assertExtraStringPrefix(t, plan, "dashboard_path", "/api/dashboard/media-assets/content-access-plan")
	assertExtraStringPrefix(t, plan, "runtime_path", "/v1/media-assets/content-access-plan")
	assertExtraStringSuffix(t, plan, "content_endpoint", "/content")
	assertExtraStringPrefix(t, plan, "content_url", "/api/dashboard/media-assets/content")
	assertStringListContains(t, plan, "required_steps", "inspect-content-diagnostics")
	assertStringListContains(t, plan, "required_steps", "open-content-endpoint")
	assertStringListContains(t, plan, "verify_steps", "rerun-content-access-plan")
	assertStringListContains(t, plan, "fallback_steps", "check-local-media-roots")
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

func loadContractFixture(t *testing.T, path string) dto.ContractFixture {
	t.Helper()
	raw, err := os.ReadFile(path)
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
	return fixture
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

func assertExtraString(t *testing.T, data map[string]any, key string, expected string) {
	t.Helper()
	value, ok := data[key].(string)
	if !ok || value != expected {
		t.Fatalf("unexpected %s: %#v", key, data[key])
	}
}

func assertExtraStringPrefix(t *testing.T, data map[string]any, key string, prefix string) {
	t.Helper()
	value, ok := data[key].(string)
	if !ok || len(value) < len(prefix) || value[:len(prefix)] != prefix {
		t.Fatalf("unexpected %s prefix: %#v", key, data[key])
	}
}

func assertExtraStringSuffix(t *testing.T, data map[string]any, key string, suffix string) {
	t.Helper()
	value, ok := data[key].(string)
	if !ok || len(value) < len(suffix) || value[len(value)-len(suffix):] != suffix {
		t.Fatalf("unexpected %s suffix: %#v", key, data[key])
	}
}

func assertStringListContains(t *testing.T, data map[string]any, key string, expected string) {
	t.Helper()
	raw, ok := data[key].([]any)
	if !ok {
		t.Fatalf("%s must be a list: %#v", key, data[key])
	}
	for _, item := range raw {
		if value, ok := item.(string); ok && value == expected {
			return
		}
	}
	t.Fatalf("%s missing %q: %#v", key, expected, data[key])
}
