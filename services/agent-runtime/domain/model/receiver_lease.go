package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ReceiverLease struct {
	ReceiverID  string
	Kind        ChannelKind
	ChannelName string
	AccountID   string
	HolderID    string
	LeaseToken  string
	ExpiresAt   time.Time
	AcquiredAt  time.Time
	UpdatedAt   time.Time
	Metadata    map[string]string
}

type ReceiverLeaseSpec struct {
	ReceiverID  string
	Kind        ChannelKind
	ChannelName string
	AccountID   string
	HolderID    string
	LeaseToken  string
	ExpiresAt   time.Time
	AcquiredAt  time.Time
	UpdatedAt   time.Time
	Metadata    map[string]string
}

func NewReceiverLease(spec ReceiverLeaseSpec) (ReceiverLease, error) {
	now := spec.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	acquiredAt := spec.AcquiredAt
	if acquiredAt.IsZero() {
		acquiredAt = now
	}
	lease := ReceiverLease{
		ReceiverID:  strings.TrimSpace(spec.ReceiverID),
		Kind:        ChannelKind(strings.TrimSpace(string(spec.Kind))),
		ChannelName: strings.TrimSpace(spec.ChannelName),
		AccountID:   strings.TrimSpace(spec.AccountID),
		HolderID:    strings.TrimSpace(spec.HolderID),
		LeaseToken:  strings.TrimSpace(spec.LeaseToken),
		ExpiresAt:   spec.ExpiresAt.UTC(),
		AcquiredAt:  acquiredAt.UTC(),
		UpdatedAt:   now.UTC(),
		Metadata:    cloneStringMap(spec.Metadata),
	}
	if lease.ReceiverID == "" {
		lease.ReceiverID = stableReceiverID(lease.Kind, lease.AccountID, lease.ChannelName)
	}
	return lease, lease.Validate()
}

func (l ReceiverLease) Validate() error {
	if strings.TrimSpace(l.ReceiverID) == "" {
		return errors.New("receiver lease requires receiver_id")
	}
	if strings.TrimSpace(string(l.Kind)) == "" {
		return errors.New("receiver lease requires kind")
	}
	if strings.TrimSpace(l.ChannelName) == "" {
		return errors.New("receiver lease requires channel_name")
	}
	if strings.TrimSpace(l.HolderID) == "" {
		return errors.New("receiver lease requires holder_id")
	}
	if strings.TrimSpace(l.LeaseToken) == "" {
		return errors.New("receiver lease requires lease_token")
	}
	if l.ExpiresAt.IsZero() {
		return errors.New("receiver lease requires expires_at")
	}
	if l.UpdatedAt.IsZero() {
		return errors.New("receiver lease requires updated_at")
	}
	return nil
}

func (l ReceiverLease) ActiveAt(now time.Time) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return l.ExpiresAt.After(now.UTC())
}

func (l ReceiverLease) Matches(holderID string, leaseToken string) bool {
	return strings.TrimSpace(l.HolderID) == strings.TrimSpace(holderID) &&
		strings.TrimSpace(l.LeaseToken) == strings.TrimSpace(leaseToken)
}

func (l ReceiverLease) Renew(expiresAt time.Time, updatedAt time.Time) (ReceiverLease, error) {
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	next := l
	next.ExpiresAt = expiresAt.UTC()
	next.UpdatedAt = updatedAt.UTC()
	return next, next.Validate()
}

func stableReceiverID(kind ChannelKind, accountID string, channelName string) string {
	accountID = strings.TrimSpace(accountID)
	channelName = strings.TrimSpace(channelName)
	if accountID == "" {
		accountID = channelName
	}
	return fmt.Sprintf("%s:%s:%s", strings.TrimSpace(string(kind)), accountID, channelName)
}
