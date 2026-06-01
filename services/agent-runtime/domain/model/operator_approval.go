package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type OperatorApprovalDecision string

const (
	OperatorApprovalApproved OperatorApprovalDecision = "approved"
	OperatorApprovalRejected OperatorApprovalDecision = "rejected"
	OperatorApprovalRevoked  OperatorApprovalDecision = "revoked"
)

type OperatorApproval struct {
	ApprovalID string
	TargetKind string
	TargetID   string
	Decision   OperatorApprovalDecision
	OperatorID string
	Reason     string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	Metadata   map[string]string
}

type OperatorApprovalSpec struct {
	ApprovalID string
	TargetKind string
	TargetID   string
	Decision   string
	OperatorID string
	Reason     string
	ExpiresAt  time.Time
	Metadata   map[string]string
}

func NewOperatorApproval(spec OperatorApprovalSpec, createdAt time.Time) (OperatorApproval, error) {
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	approval := OperatorApproval{
		ApprovalID: strings.TrimSpace(spec.ApprovalID),
		TargetKind: strings.TrimSpace(spec.TargetKind),
		TargetID:   strings.TrimSpace(spec.TargetID),
		Decision:   normalizeOperatorApprovalDecision(spec.Decision),
		OperatorID: strings.TrimSpace(spec.OperatorID),
		Reason:     strings.TrimSpace(spec.Reason),
		ExpiresAt:  spec.ExpiresAt.UTC(),
		CreatedAt:  createdAt.UTC(),
		Metadata:   cloneStringMap(spec.Metadata),
	}
	if approval.ApprovalID == "" {
		approval.ApprovalID = approval.stableID()
	}
	return approval, approval.Validate()
}

func (a OperatorApproval) Validate() error {
	if strings.TrimSpace(a.ApprovalID) == "" {
		return errors.New("operator approval requires approval_id")
	}
	if strings.TrimSpace(a.TargetKind) == "" {
		return errors.New("operator approval requires target_kind")
	}
	if strings.TrimSpace(a.TargetID) == "" {
		return errors.New("operator approval requires target_id")
	}
	if !isKnownOperatorApprovalDecision(a.Decision) {
		return fmt.Errorf("unknown operator approval decision: %s", a.Decision)
	}
	if strings.TrimSpace(a.OperatorID) == "" {
		return errors.New("operator approval requires operator_id")
	}
	if a.CreatedAt.IsZero() {
		return errors.New("operator approval requires created_at")
	}
	if (a.Decision == OperatorApprovalRejected || a.Decision == OperatorApprovalRevoked) && strings.TrimSpace(a.Reason) == "" {
		return errors.New("operator approval rejection or revocation requires reason")
	}
	return nil
}

func (a OperatorApproval) ActiveAt(now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if a.Decision != OperatorApprovalApproved {
		return false
	}
	return a.ExpiresAt.IsZero() || now.UTC().Before(a.ExpiresAt.UTC())
}

func (a OperatorApproval) stableID() string {
	return fmt.Sprintf(
		"operator_approval:%s:%s:%s:%d",
		safeOperatorApprovalKey(a.TargetKind),
		safeOperatorApprovalKey(a.TargetID),
		string(a.Decision),
		a.CreatedAt.UnixNano(),
	)
}

func normalizeOperatorApprovalDecision(value string) OperatorApprovalDecision {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "approved", "approve", "ack", "acknowledged":
		return OperatorApprovalApproved
	case "rejected", "reject", "deny", "denied":
		return OperatorApprovalRejected
	case "revoked", "revoke", "cancelled", "canceled":
		return OperatorApprovalRevoked
	default:
		return OperatorApprovalDecision(strings.TrimSpace(value))
	}
}

func isKnownOperatorApprovalDecision(value OperatorApprovalDecision) bool {
	switch value {
	case OperatorApprovalApproved, OperatorApprovalRejected, OperatorApprovalRevoked:
		return true
	default:
		return false
	}
}

func safeOperatorApprovalKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_")
	return replacer.Replace(value)
}

func SortedOperatorApprovals(items []OperatorApproval) []OperatorApproval {
	sorted := append([]OperatorApproval(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].CreatedAt.Equal(sorted[j].CreatedAt) {
			return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
		}
		if sorted[i].TargetKind != sorted[j].TargetKind {
			return sorted[i].TargetKind < sorted[j].TargetKind
		}
		if sorted[i].TargetID != sorted[j].TargetID {
			return sorted[i].TargetID < sorted[j].TargetID
		}
		return sorted[i].ApprovalID < sorted[j].ApprovalID
	})
	return sorted
}
