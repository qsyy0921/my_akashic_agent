package model

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type ControlMutationStatus string

const (
	ControlMutationPlanned    ControlMutationStatus = "planned"
	ControlMutationApplied    ControlMutationStatus = "applied"
	ControlMutationFailed     ControlMutationStatus = "failed"
	ControlMutationRolledBack ControlMutationStatus = "rolled_back"
)

type ControlMutationAudit struct {
	MutationID  string
	TargetKind  string
	TargetID    string
	Action      string
	Status      ControlMutationStatus
	OperatorID  string
	ApprovalID  string
	Reason      string
	RollbackOf  string
	RollbackRef string
	CreatedAt   time.Time
	Metadata    map[string]string
}

type ControlMutationAuditSpec struct {
	MutationID  string
	TargetKind  string
	TargetID    string
	Action      string
	Status      string
	OperatorID  string
	ApprovalID  string
	Reason      string
	RollbackOf  string
	RollbackRef string
	Metadata    map[string]string
}

func NewControlMutationAudit(spec ControlMutationAuditSpec, createdAt time.Time) (ControlMutationAudit, error) {
	item := ControlMutationAudit{
		MutationID:  strings.TrimSpace(spec.MutationID),
		TargetKind:  strings.TrimSpace(spec.TargetKind),
		TargetID:    strings.TrimSpace(spec.TargetID),
		Action:      strings.TrimSpace(spec.Action),
		Status:      normalizeControlMutationStatus(spec.Status),
		OperatorID:  strings.TrimSpace(spec.OperatorID),
		ApprovalID:  strings.TrimSpace(spec.ApprovalID),
		Reason:      strings.TrimSpace(spec.Reason),
		RollbackOf:  strings.TrimSpace(spec.RollbackOf),
		RollbackRef: strings.TrimSpace(spec.RollbackRef),
		CreatedAt:   createdAt.UTC(),
		Metadata:    cloneStringMap(spec.Metadata),
	}
	if item.MutationID == "" {
		item.MutationID = item.stableID()
	}
	return item, item.Validate()
}

func (a ControlMutationAudit) Validate() error {
	if strings.TrimSpace(a.MutationID) == "" {
		return fmt.Errorf("control mutation audit requires mutation_id")
	}
	if strings.TrimSpace(a.TargetKind) == "" {
		return fmt.Errorf("control mutation audit requires target_kind")
	}
	if strings.TrimSpace(a.TargetID) == "" {
		return fmt.Errorf("control mutation audit requires target_id")
	}
	if strings.TrimSpace(a.Action) == "" {
		return fmt.Errorf("control mutation audit requires action")
	}
	if !isKnownControlMutationStatus(a.Status) {
		return fmt.Errorf("unknown control mutation status: %s", a.Status)
	}
	if strings.TrimSpace(a.OperatorID) == "" {
		return fmt.Errorf("control mutation audit requires operator_id")
	}
	if strings.TrimSpace(a.ApprovalID) == "" {
		return fmt.Errorf("control mutation audit requires approval_id")
	}
	if a.CreatedAt.IsZero() {
		return fmt.Errorf("control mutation audit requires created_at")
	}
	if (a.Status == ControlMutationFailed || a.Status == ControlMutationRolledBack) && strings.TrimSpace(a.Reason) == "" {
		return fmt.Errorf("control mutation audit failed or rolled_back status requires reason")
	}
	if a.Status == ControlMutationRolledBack && strings.TrimSpace(a.RollbackOf) == "" && strings.TrimSpace(a.RollbackRef) == "" {
		return fmt.Errorf("control mutation audit rolled_back status requires rollback_of or rollback_ref")
	}
	return nil
}

func (a ControlMutationAudit) stableID() string {
	return fmt.Sprintf(
		"control_mutation:%s:%s:%s:%s:%d",
		safeControlMutationKey(a.TargetKind),
		safeControlMutationKey(a.TargetID),
		safeControlMutationKey(a.Action),
		safeControlMutationKey(string(a.Status)),
		a.CreatedAt.UnixNano(),
	)
}

func normalizeControlMutationStatus(value string) ControlMutationStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "plan", "planned":
		return ControlMutationPlanned
	case "apply", "applied", "success", "succeeded":
		return ControlMutationApplied
	case "fail", "failed", "error":
		return ControlMutationFailed
	case "rollback", "rolled_back", "rolledback":
		return ControlMutationRolledBack
	default:
		return ControlMutationStatus(strings.TrimSpace(value))
	}
}

func isKnownControlMutationStatus(value ControlMutationStatus) bool {
	switch value {
	case ControlMutationPlanned, ControlMutationApplied, ControlMutationFailed, ControlMutationRolledBack:
		return true
	default:
		return false
	}
}

func safeControlMutationKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9._:-]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "unknown"
	}
	return value
}

func SortedControlMutationAudits(items []ControlMutationAudit) []ControlMutationAudit {
	sorted := append([]ControlMutationAudit(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].CreatedAt.Equal(sorted[j].CreatedAt) {
			return sorted[i].MutationID > sorted[j].MutationID
		}
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})
	return sorted
}
