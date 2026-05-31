package service

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type OutboxAccountRateLimitConfig struct {
	MinInterval           time.Duration
	Window                time.Duration
	MaxDispatchesInWindow int
}

type OutboxAccountRateLimiter struct {
	mu            sync.Mutex
	minInterval   time.Duration
	window        time.Duration
	maxPerWindow  int
	dispatchTimes map[string][]time.Time
}

func NewOutboxAccountRateLimiter(config OutboxAccountRateLimitConfig) *OutboxAccountRateLimiter {
	if config.MinInterval <= 0 && (config.Window <= 0 || config.MaxDispatchesInWindow <= 0) {
		return nil
	}
	window := config.Window
	if window <= 0 {
		window = time.Minute
	}
	if config.MinInterval > window {
		window = config.MinInterval
	}
	return &OutboxAccountRateLimiter{
		minInterval:   config.MinInterval,
		window:        window,
		maxPerWindow:  config.MaxDispatchesInWindow,
		dispatchTimes: make(map[string][]time.Time),
	}
}

func (l *OutboxAccountRateLimiter) Enabled() bool {
	return l != nil
}

func (l *OutboxAccountRateLimiter) BlockedAccountKeys(now time.Time) []string {
	if l == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	blocked := make([]string, 0)
	for accountKey, items := range l.dispatchTimes {
		items = l.prune(items, now)
		if len(items) == 0 {
			delete(l.dispatchTimes, accountKey)
			continue
		}
		l.dispatchTimes[accountKey] = items
		if l.blocked(items, now) {
			blocked = append(blocked, accountKey)
		}
	}
	sort.Strings(blocked)
	return blocked
}

func (l *OutboxAccountRateLimiter) AccountBlocked(accountKey string, now time.Time) bool {
	if l == nil {
		return false
	}
	accountKey = strings.TrimSpace(accountKey)
	if accountKey == "" || accountKey == ":" {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	items := l.prune(l.dispatchTimes[accountKey], now)
	if len(items) == 0 {
		delete(l.dispatchTimes, accountKey)
		return false
	}
	l.dispatchTimes[accountKey] = items
	return l.blocked(items, now)
}

func (l *OutboxAccountRateLimiter) Record(accountKey string, now time.Time) {
	if l == nil {
		return
	}
	accountKey = strings.TrimSpace(accountKey)
	if accountKey == "" || accountKey == ":" {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	items := l.prune(l.dispatchTimes[accountKey], now)
	items = append(items, now)
	l.dispatchTimes[accountKey] = items
}

func (l *OutboxAccountRateLimiter) prune(items []time.Time, now time.Time) []time.Time {
	if l == nil || l.window <= 0 {
		return items
	}
	cutoff := now.Add(-l.window)
	result := items[:0]
	for _, item := range items {
		if !item.Before(cutoff) {
			result = append(result, item)
		}
	}
	return result
}

func (l *OutboxAccountRateLimiter) blocked(items []time.Time, now time.Time) bool {
	if l == nil || len(items) == 0 {
		return false
	}
	latest := items[len(items)-1]
	if l.minInterval > 0 && now.Sub(latest) < l.minInterval {
		return true
	}
	if l.window > 0 && l.maxPerWindow > 0 && len(items) >= l.maxPerWindow {
		return true
	}
	return false
}
