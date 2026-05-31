package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/assembler"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/command"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type ReceiverStatusService struct {
	mu               sync.RWMutex
	receivers        map[string]model.ReceiverStatus
	leases           map[string]model.ReceiverLease
	statusRepository outport.ReceiverStatusRepository
	leaseRepository  outport.ReceiverLeaseRepository
	statusStaleAfter time.Duration
	statusClock      func() time.Time
}

func NewReceiverStatusService() *ReceiverStatusService {
	return &ReceiverStatusService{
		receivers:        make(map[string]model.ReceiverStatus),
		leases:           make(map[string]model.ReceiverLease),
		statusStaleAfter: 0,
		statusClock:      time.Now,
	}
}

func NewReceiverStatusServiceWithRepository(
	ctx context.Context,
	repository outport.ReceiverStatusRepository,
	staleAfter time.Duration,
) (*ReceiverStatusService, error) {
	return NewReceiverStatusServiceWithRepositories(ctx, repository, nil, staleAfter)
}

func NewReceiverStatusServiceWithRepositories(
	ctx context.Context,
	statusRepository outport.ReceiverStatusRepository,
	leaseRepository outport.ReceiverLeaseRepository,
	staleAfter time.Duration,
) (*ReceiverStatusService, error) {
	service := NewReceiverStatusService()
	service.statusRepository = statusRepository
	service.leaseRepository = leaseRepository
	service.statusStaleAfter = staleAfter
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if statusRepository != nil {
		items, err := statusRepository.ListReceiverStatuses(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := item.Validate(); err != nil {
				continue
			}
			service.receivers[item.ReceiverID] = item
		}
	}
	if leaseRepository != nil {
		items, err := leaseRepository.ListReceiverLeases(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := item.Validate(); err != nil {
				continue
			}
			service.leases[item.ReceiverID] = item
		}
	}
	return service, nil
}

func (s *ReceiverStatusService) ReportReceiverStatus(ctx context.Context, cmd command.ReportReceiverStatusCommand) (query.ReceiverStatusesView, error) {
	if err := ctx.Err(); err != nil {
		return query.ReceiverStatusesView{}, err
	}
	if s == nil {
		return query.ReceiverStatusesView{}, errors.New("receiver status service is nil")
	}
	timestamp := cmd.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	status, err := model.NewReceiverStatus(model.ReceiverStatusSpec{
		ReceiverID:  cmd.ReceiverID,
		Kind:        model.ChannelKind(cmd.Kind),
		ChannelName: cmd.ChannelName,
		AccountID:   cmd.AccountID,
		Endpoint:    cmd.Endpoint,
		Status:      cmd.Status,
		Reason:      cmd.Reason,
		LastError:   cmd.LastError,
		Source:      cmd.Source,
		Metadata:    cmd.Metadata,
	}, timestamp)
	if err != nil {
		return query.ReceiverStatusesView{}, err
	}

	s.mu.Lock()
	if s.receivers == nil {
		s.receivers = make(map[string]model.ReceiverStatus)
	}
	if s.statusRepository != nil {
		if err := s.statusRepository.SaveReceiverStatus(ctx, status); err != nil {
			s.mu.Unlock()
			return query.ReceiverStatusesView{}, err
		}
	}
	s.receivers[status.ReceiverID] = status
	items := s.statusSnapshotLocked()
	s.mu.Unlock()

	view := receiverStatusesView(s.applyReceiverStatusStaleness(items))
	view.Notes = append(view.Notes, "reported")
	return view, nil
}

func (s *ReceiverStatusService) ListReceiverStatuses(ctx context.Context) (query.ReceiverStatusesView, error) {
	if err := ctx.Err(); err != nil {
		return query.ReceiverStatusesView{}, err
	}
	if s == nil {
		return query.ReceiverStatusesView{}, errors.New("receiver status service is nil")
	}
	s.mu.RLock()
	items := s.statusSnapshotLocked()
	s.mu.RUnlock()
	return receiverStatusesView(s.applyReceiverStatusStaleness(items)), nil
}

func (s *ReceiverStatusService) AcquireReceiverLease(ctx context.Context, cmd command.AcquireReceiverLeaseCommand) (query.ReceiverLeaseView, error) {
	if err := ctx.Err(); err != nil {
		return query.ReceiverLeaseView{}, err
	}
	if s == nil {
		return query.ReceiverLeaseView{}, errors.New("receiver status service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ttl := receiverLeaseTTL(cmd.TTLSeconds)
	receiverID := receiverLeaseID(cmd.ReceiverID, cmd.Kind, cmd.AccountID, cmd.ChannelName)
	holderID := strings.TrimSpace(cmd.HolderID)
	if holderID == "" {
		return query.ReceiverLeaseView{}, errors.New("receiver lease requires holder_id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leases == nil {
		s.leases = make(map[string]model.ReceiverLease)
	}
	if current, ok := s.leases[receiverID]; ok && current.ActiveAt(now) && current.HolderID != holderID {
		view := assembler.ToReceiverLeaseView(current, now, false)
		view.Acquired = receiverLeaseAcquired(false)
		view.DeniedReason = "active_lease_held"
		return view, nil
	}
	token := ""
	acquiredAt := now
	if current, ok := s.leases[receiverID]; ok && current.ActiveAt(now) && current.HolderID == holderID {
		token = current.LeaseToken
		acquiredAt = current.AcquiredAt
	} else {
		var err error
		token, err = randomReceiverLeaseToken()
		if err != nil {
			return query.ReceiverLeaseView{}, err
		}
	}
	lease, err := model.NewReceiverLease(model.ReceiverLeaseSpec{
		ReceiverID:  receiverID,
		Kind:        model.ChannelKind(cmd.Kind),
		ChannelName: cmd.ChannelName,
		AccountID:   cmd.AccountID,
		HolderID:    holderID,
		LeaseToken:  token,
		ExpiresAt:   now.Add(ttl),
		AcquiredAt:  acquiredAt,
		UpdatedAt:   now,
		Metadata:    cmd.Metadata,
	})
	if err != nil {
		return query.ReceiverLeaseView{}, err
	}
	if s.leaseRepository != nil {
		if err := s.leaseRepository.SaveReceiverLease(ctx, lease); err != nil {
			return query.ReceiverLeaseView{}, err
		}
	}
	s.leases[lease.ReceiverID] = lease
	view := assembler.ToReceiverLeaseView(lease, now, true)
	view.Acquired = receiverLeaseAcquired(true)
	return view, nil
}

func (s *ReceiverStatusService) RenewReceiverLease(ctx context.Context, cmd command.RenewReceiverLeaseCommand) (query.ReceiverLeaseView, error) {
	if err := ctx.Err(); err != nil {
		return query.ReceiverLeaseView{}, err
	}
	if s == nil {
		return query.ReceiverLeaseView{}, errors.New("receiver status service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	receiverID := strings.TrimSpace(cmd.ReceiverID)
	if receiverID == "" {
		return query.ReceiverLeaseView{}, errors.New("receiver lease renew requires receiver_id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.leases[receiverID]
	if !ok {
		return query.ReceiverLeaseView{}, errors.New("receiver lease not found")
	}
	if !current.ActiveAt(now) {
		return query.ReceiverLeaseView{}, errors.New("receiver lease expired")
	}
	if !current.Matches(cmd.HolderID, cmd.LeaseToken) {
		return query.ReceiverLeaseView{}, errors.New("receiver lease token mismatch")
	}
	renewed, err := current.Renew(now.Add(receiverLeaseTTL(cmd.TTLSeconds)), now)
	if err != nil {
		return query.ReceiverLeaseView{}, err
	}
	if s.leaseRepository != nil {
		if err := s.leaseRepository.SaveReceiverLease(ctx, renewed); err != nil {
			return query.ReceiverLeaseView{}, err
		}
	}
	s.leases[receiverID] = renewed
	view := assembler.ToReceiverLeaseView(renewed, now, true)
	view.Acquired = receiverLeaseAcquired(true)
	return view, nil
}

func (s *ReceiverStatusService) ReleaseReceiverLease(ctx context.Context, cmd command.ReleaseReceiverLeaseCommand) (query.ReceiverLeaseView, error) {
	if err := ctx.Err(); err != nil {
		return query.ReceiverLeaseView{}, err
	}
	if s == nil {
		return query.ReceiverLeaseView{}, errors.New("receiver status service is nil")
	}
	now := cmd.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	receiverID := strings.TrimSpace(cmd.ReceiverID)
	if receiverID == "" {
		return query.ReceiverLeaseView{}, errors.New("receiver lease release requires receiver_id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.leases[receiverID]
	if !ok {
		return query.ReceiverLeaseView{}, errors.New("receiver lease not found")
	}
	if !current.Matches(cmd.HolderID, cmd.LeaseToken) {
		return query.ReceiverLeaseView{}, errors.New("receiver lease token mismatch")
	}
	if s.leaseRepository != nil {
		if err := s.leaseRepository.DeleteReceiverLease(ctx, receiverID); err != nil {
			return query.ReceiverLeaseView{}, err
		}
	}
	delete(s.leases, receiverID)
	view := assembler.ToReceiverLeaseView(current, now, false)
	view.Active = false
	view.Acquired = receiverLeaseAcquired(false)
	return view, nil
}

func (s *ReceiverStatusService) ListReceiverLeases(ctx context.Context) (query.ReceiverLeasesView, error) {
	if err := ctx.Err(); err != nil {
		return query.ReceiverLeasesView{}, err
	}
	if s == nil {
		return query.ReceiverLeasesView{}, errors.New("receiver status service is nil")
	}
	now := time.Now().UTC()
	s.mu.RLock()
	items := s.leaseSnapshotLocked()
	s.mu.RUnlock()
	return receiverLeasesView(items, now), nil
}

func (s *ReceiverStatusService) statusSnapshotLocked() []model.ReceiverStatus {
	items := make([]model.ReceiverStatus, 0, len(s.receivers))
	for _, item := range s.receivers {
		items = append(items, item)
	}
	return model.SortedReceiverStatuses(items)
}

func (s *ReceiverStatusService) applyReceiverStatusStaleness(items []model.ReceiverStatus) []model.ReceiverStatus {
	if s == nil || s.statusStaleAfter <= 0 {
		return items
	}
	now := time.Now().UTC()
	if s.statusClock != nil {
		now = s.statusClock().UTC()
	}
	result := append([]model.ReceiverStatus(nil), items...)
	for index, item := range result {
		if item.Status != model.ReceiverStatusConnected && item.Status != model.ReceiverStatusStarting {
			continue
		}
		if item.UpdatedAt.IsZero() || now.Sub(item.UpdatedAt.UTC()) <= s.statusStaleAfter {
			continue
		}
		lastStatus := item.Status
		item.Status = model.ReceiverStatusStopped
		item.Reason = "heartbeat_stale"
		if item.LastError == "" {
			item.LastError = "last receiver heartbeat exceeded stale threshold"
		}
		item.Metadata = cloneReceiverStatusMetadata(item.Metadata)
		item.Metadata["last_status"] = string(lastStatus)
		item.Metadata["stale_after_seconds"] = strconv.Itoa(int(s.statusStaleAfter.Seconds()))
		result[index] = item
	}
	return result
}

func cloneReceiverStatusMetadata(items map[string]string) map[string]string {
	cloned := make(map[string]string, len(items)+2)
	for key, value := range items {
		cloned[key] = value
	}
	return cloned
}

func (s *ReceiverStatusService) leaseSnapshotLocked() []model.ReceiverLease {
	items := make([]model.ReceiverLease, 0, len(s.leases))
	for _, item := range s.leases {
		items = append(items, item)
	}
	return model.SortedReceiverLeases(items)
}

func receiverStatusesView(items []model.ReceiverStatus) query.ReceiverStatusesView {
	totals := map[string]int{
		"receivers":  0,
		"starting":   0,
		"connected":  0,
		"suspended":  0,
		"failed":     0,
		"stopped":    0,
		"qq":         0,
		"telegram":   0,
		"other_kind": 0,
	}
	totals["receivers"] = len(items)
	for _, item := range items {
		switch item.Status {
		case model.ReceiverStatusStarting:
			totals["starting"]++
		case model.ReceiverStatusConnected:
			totals["connected"]++
		case model.ReceiverStatusSuspended:
			totals["suspended"]++
		case model.ReceiverStatusFailed:
			totals["failed"]++
		case model.ReceiverStatusStopped:
			totals["stopped"]++
		}
		switch item.Kind {
		case model.ChannelKindQQ:
			totals["qq"]++
		case model.ChannelKindTelegram:
			totals["telegram"]++
		default:
			totals["other_kind"]++
		}
	}
	return query.ReceiverStatusesView{
		Receivers:  assembler.ToReceiverStatusViews(items),
		Totals:     totals,
		Notes:      []string{"side_effect=none", "runtime_view_only"},
		SideEffect: "none",
	}
}

func receiverLeasesView(items []model.ReceiverLease, now time.Time) query.ReceiverLeasesView {
	totals := map[string]int{
		"leases":   len(items),
		"active":   0,
		"expired":  0,
		"qq":       0,
		"telegram": 0,
	}
	for _, item := range items {
		if item.ActiveAt(now) {
			totals["active"]++
		} else {
			totals["expired"]++
		}
		switch item.Kind {
		case model.ChannelKindQQ:
			totals["qq"]++
		case model.ChannelKindTelegram:
			totals["telegram"]++
		}
	}
	return query.ReceiverLeasesView{
		Leases:     assembler.ToReceiverLeaseViews(items, now),
		Totals:     totals,
		Notes:      []string{"side_effect=runtime_state_only"},
		SideEffect: "runtime_state_only",
	}
}

func receiverLeaseID(receiverID string, kind string, accountID string, channelName string) string {
	receiverID = strings.TrimSpace(receiverID)
	if receiverID != "" {
		return receiverID
	}
	accountID = strings.TrimSpace(accountID)
	channelName = strings.TrimSpace(channelName)
	if accountID == "" {
		accountID = channelName
	}
	return strings.TrimSpace(kind) + ":" + accountID + ":" + channelName
}

func receiverLeaseTTL(ttlSeconds int) time.Duration {
	if ttlSeconds <= 0 {
		ttlSeconds = 120
	}
	if ttlSeconds < 30 {
		ttlSeconds = 30
	}
	if ttlSeconds > 3600 {
		ttlSeconds = 3600
	}
	return time.Duration(ttlSeconds) * time.Second
}

func randomReceiverLeaseToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func receiverLeaseAcquired(value bool) *bool {
	return &value
}
