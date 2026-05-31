package service

import (
	"testing"
	"time"
)

func TestOutboxAccountRateLimiterBlocksWithinMinInterval(t *testing.T) {
	now := time.Date(2026, 5, 31, 11, 0, 0, 0, time.UTC)
	limiter := NewOutboxAccountRateLimiter(OutboxAccountRateLimitConfig{
		MinInterval: time.Minute,
	})
	if limiter == nil {
		t.Fatal("expected limiter")
	}
	limiter.Record("qq:1049511700", now)
	if !limiter.AccountBlocked("qq:1049511700", now.Add(10*time.Second)) {
		t.Fatal("expected account blocked within min interval")
	}
	if limiter.AccountBlocked("qq:1049511700", now.Add(time.Minute)) {
		t.Fatal("expected account unblocked after min interval")
	}
}

func TestOutboxAccountRateLimiterBlocksAtWindowMaxAndPrunes(t *testing.T) {
	now := time.Date(2026, 5, 31, 11, 10, 0, 0, time.UTC)
	limiter := NewOutboxAccountRateLimiter(OutboxAccountRateLimitConfig{
		Window:                time.Minute,
		MaxDispatchesInWindow: 2,
	})
	limiter.Record("qq:1049511700", now)
	limiter.Record("qq:1049511700", now.Add(10*time.Second))
	if !limiter.AccountBlocked("qq:1049511700", now.Add(20*time.Second)) {
		t.Fatal("expected account blocked at window max")
	}
	if limiter.AccountBlocked("qq:1049511700", now.Add(2*time.Minute)) {
		t.Fatal("expected account unblocked after window prune")
	}
}

func TestOutboxAccountRateLimiterReturnsSortedBlockedKeys(t *testing.T) {
	now := time.Date(2026, 5, 31, 11, 20, 0, 0, time.UTC)
	limiter := NewOutboxAccountRateLimiter(OutboxAccountRateLimitConfig{
		MinInterval: time.Minute,
	})
	limiter.Record("telegram:bot", now)
	limiter.Record("qq:1049511700", now)

	blocked := limiter.BlockedAccountKeys(now.Add(time.Second))
	if len(blocked) != 2 || blocked[0] != "qq:1049511700" || blocked[1] != "telegram:bot" {
		t.Fatalf("unexpected blocked keys: %#v", blocked)
	}
}
