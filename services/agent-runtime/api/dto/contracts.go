package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ContractFixture struct {
	SchemaVersion    string                     `json:"schema_version"`
	Kind             string                     `json:"kind"`
	EventID          string                     `json:"event_id,omitempty"`
	JobID            string                     `json:"job_id,omitempty"`
	Platform         string                     `json:"platform"`
	AccountID        string                     `json:"account_id"`
	ConversationID   string                     `json:"conversation_id"`
	ConversationType string                     `json:"conversation_type"`
	AgentID          string                     `json:"agent_id,omitempty"`
	Channel          *ChannelDTO                `json:"channel,omitempty"`
	Timestamp        string                     `json:"timestamp"`
	SourceMessageIDs []string                   `json:"source_message_ids"`
	SourceAssetIDs   []string                   `json:"source_asset_ids"`
	Metadata         map[string]any             `json:"metadata"`
	Extra            map[string]json.RawMessage `json:"-"`
}

func (c *ContractFixture) UnmarshalJSON(data []byte) error {
	type alias ContractFixture
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var extra map[string]json.RawMessage
	if err := json.Unmarshal(data, &extra); err != nil {
		return err
	}
	for _, key := range contractFixtureKnownKeys {
		delete(extra, key)
	}

	*c = ContractFixture(decoded)
	c.Extra = extra
	return nil
}

func (c ContractFixture) MarshalJSON() ([]byte, error) {
	data := make(map[string]any, len(c.Extra)+16)
	for key, value := range c.Extra {
		data[key] = value
	}
	data["schema_version"] = c.SchemaVersion
	data["kind"] = c.Kind
	if c.EventID != "" {
		data["event_id"] = c.EventID
	}
	if c.JobID != "" {
		data["job_id"] = c.JobID
	}
	data["platform"] = c.Platform
	data["account_id"] = c.AccountID
	data["conversation_id"] = c.ConversationID
	data["conversation_type"] = c.ConversationType
	if c.AgentID != "" {
		data["agent_id"] = c.AgentID
	}
	if c.Channel != nil {
		data["channel"] = c.Channel
	}
	data["timestamp"] = c.Timestamp
	data["source_message_ids"] = c.SourceMessageIDs
	data["source_asset_ids"] = c.SourceAssetIDs
	data["metadata"] = c.Metadata
	return json.Marshal(data)
}

func (c ContractFixture) StableID() string {
	if c.EventID != "" {
		return c.EventID
	}
	return c.JobID
}

func (c ContractFixture) Validate() error {
	if err := requireContractText(c.SchemaVersion, "schema_version"); err != nil {
		return err
	}
	if err := requireContractText(c.Kind, "kind"); err != nil {
		return err
	}
	if strings.TrimSpace(c.EventID) == "" && strings.TrimSpace(c.JobID) == "" {
		return errors.New("event_id or job_id is required")
	}
	for field, value := range map[string]string{
		"platform":          c.Platform,
		"account_id":        c.AccountID,
		"conversation_id":   c.ConversationID,
		"conversation_type": c.ConversationType,
		"timestamp":         c.Timestamp,
	} {
		if err := requireContractText(value, field); err != nil {
			return err
		}
	}
	if c.Kind != "MediaAsset" {
		if err := requireContractText(c.AgentID, "agent_id"); err != nil {
			return err
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, c.Timestamp); err != nil {
		return fmt.Errorf("timestamp must include RFC3339 timezone: %w", err)
	}
	if c.Metadata == nil {
		return errors.New("metadata must be an object")
	}
	if c.SourceMessageIDs == nil {
		return errors.New("source_message_ids must be a list")
	}
	if c.SourceAssetIDs == nil {
		return errors.New("source_asset_ids must be a list")
	}
	if c.Channel != nil {
		if err := c.validateChannelRoute(*c.Channel); err != nil {
			return err
		}
	}
	return nil
}

func (c ContractFixture) validateChannelRoute(channel ChannelDTO) error {
	if channel.RoutePlatform() != c.Platform {
		return errors.New("channel platform must match top-level route")
	}
	if channel.AccountID != c.AccountID {
		return errors.New("channel account_id must match top-level route")
	}
	if channel.ConversationID != c.ConversationID {
		return errors.New("channel conversation_id must match top-level route")
	}
	if channel.ConversationType != c.ConversationType {
		return errors.New("channel conversation_type must match top-level route")
	}
	return nil
}

func requireContractText(value string, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

var contractFixtureKnownKeys = []string{
	"schema_version",
	"kind",
	"event_id",
	"job_id",
	"platform",
	"account_id",
	"conversation_id",
	"conversation_type",
	"agent_id",
	"channel",
	"timestamp",
	"source_message_ids",
	"source_asset_ids",
	"metadata",
}

