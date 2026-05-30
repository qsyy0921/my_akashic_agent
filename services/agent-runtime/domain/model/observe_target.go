package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ObserveTarget struct {
	TargetID     string
	Channel      ChannelRef
	ObserveOnly  bool
	ReplyAllowed bool
	RequireAt    bool
	AllowFrom    []string
	Enabled      bool
	Source       string
	Metadata     map[string]string
	UpdatedAt    time.Time
}

type ObserveTargetSpec struct {
	TargetID     string
	Channel      ChannelRef
	ObserveOnly  bool
	ReplyAllowed bool
	RequireAt    bool
	AllowFrom    []string
	Enabled      bool
	Source       string
	Metadata     map[string]string
}

func NewObserveTarget(spec ObserveTargetSpec, updatedAt time.Time) (ObserveTarget, error) {
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	target := ObserveTarget{
		TargetID:     strings.TrimSpace(spec.TargetID),
		Channel:      trimChannelRef(spec.Channel),
		ObserveOnly:  spec.ObserveOnly,
		ReplyAllowed: spec.ReplyAllowed,
		RequireAt:    spec.RequireAt,
		AllowFrom:    normalizeObserveTargetAllowFrom(spec.AllowFrom),
		Enabled:      spec.Enabled,
		Source:       strings.TrimSpace(spec.Source),
		Metadata:     cloneStringMap(spec.Metadata),
		UpdatedAt:    updatedAt.UTC(),
	}
	if target.TargetID == "" {
		target.TargetID = target.Channel.StableID()
	}
	if target.Source == "" {
		target.Source = "unknown"
	}
	if target.ObserveOnly {
		target.ReplyAllowed = false
	}
	return target, target.Validate()
}

func (t ObserveTarget) Validate() error {
	if strings.TrimSpace(t.TargetID) == "" {
		return errors.New("observe target requires target_id")
	}
	if strings.TrimSpace(string(t.Channel.Kind)) == "" {
		return errors.New("observe target requires channel kind")
	}
	if strings.TrimSpace(t.Channel.AccountID) == "" {
		return errors.New("observe target requires channel account_id")
	}
	if strings.TrimSpace(t.Channel.ConversationID) == "" {
		return errors.New("observe target requires conversation_id")
	}
	if strings.TrimSpace(string(t.Channel.ConversationType)) == "" {
		return errors.New("observe target requires conversation_type")
	}
	if t.ObserveOnly && t.ReplyAllowed {
		return errors.New("observe-only target must not allow replies")
	}
	if t.UpdatedAt.IsZero() {
		return errors.New("observe target requires updated_at")
	}
	return nil
}

func (c ChannelRef) StableID() string {
	return fmt.Sprintf(
		"%s:%s:%s:%s",
		strings.TrimSpace(string(c.Kind)),
		strings.TrimSpace(c.AccountID),
		strings.TrimSpace(string(c.ConversationType)),
		strings.TrimSpace(c.ConversationID),
	)
}

func trimChannelRef(channel ChannelRef) ChannelRef {
	return ChannelRef{
		Kind:             ChannelKind(strings.TrimSpace(string(channel.Kind))),
		AccountID:        strings.TrimSpace(channel.AccountID),
		ConversationID:   strings.TrimSpace(channel.ConversationID),
		ConversationType: ConversationType(strings.TrimSpace(string(channel.ConversationType))),
	}
}

func normalizeObserveTargetAllowFrom(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		normalized = append(normalized, item)
	}
	sort.Strings(normalized)
	return normalized
}

func cloneStringMap(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(items))
	for key, value := range items {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cloned[key] = strings.TrimSpace(value)
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}
