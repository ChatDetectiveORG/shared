// Package ratelimit implements a centralised, Redis-backed token bucket with priority queues for
// every service that talks to the Telegram Bot API.
//
// The token bucket lives in event-loop: a single goroutine ticks at the configured rate and pops
// one waiting request from the highest-priority non-empty queue. Clients block on a per-request
// response list (BLPOP) until they get a token.
//
// Why this design:
//   - Cross-service: messgae-sender and chat-export-service share the same 30 rps budget without
//     needing direct connections to each other.
//   - Priority: bot commands always get tokens before delete-message reactions or chat exports.
//   - Crash-safe: orphan response lists carry a TTL; clients that crash mid-wait don't leak memory.
//   - Outage-tolerant: clients that fail to acquire (Redis down) MAY proceed; messgae-sender logs
//     and falls through, so a flapping bucket cannot block user-visible work.
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

// Priority levels. Lower number = higher priority.
type Priority int

const (
	// PriorityCommand is for /start, /help and similar user-typed commands. The user is actively
	// waiting on the keyboard, so we give them the first available token.
	PriorityCommand Priority = 1
	// PriorityCallback is for callback-query answers and inline keyboard interactions.
	PriorityCallback Priority = 2
	// PriorityBusiness is for reactive notifications about edited / deleted business messages.
	PriorityBusiness Priority = 3
	// PriorityExport is for chat-export-service: long pipelines that should never starve commands.
	PriorityExport Priority = 4

	// DefaultPriority is what callers get when they pass 0.
	DefaultPriority = PriorityCallback
)

// Resolve normalises an int (commonly carried in JSON envelopes) into a Priority value, applying
// the default for unset (zero) and clamping unknown values to the lowest priority.
func Resolve(value int) Priority {
	if value <= 0 {
		return DefaultPriority
	}
	if value > int(PriorityExport) {
		return PriorityExport
	}
	return Priority(value)
}

const (
	// queueKey is the Redis list per priority. Keep them as separate lists rather than a sorted set
	// so atomic LPOP semantics give strict FIFO within a priority class.
	queueKeyPrefix = "tg:rl:queue:"
	responseKey    = "tg:rl:resp:"

	// responseTTLSeconds bounds the lifetime of granted tokens that nobody picked up (because the
	// requesting client crashed or its context was cancelled between RPUSH and BLPOP).
	responseTTLSeconds = 60

	// defaultBlockSeconds is the upper bound for a single BLPOP wait. The actual wait is min of
	// this and the caller's ctx deadline.
	defaultBlockSeconds = 30
)

// PriorityList returns the queue names ordered from highest to lowest priority. The dispatcher
// drains them in order on every tick.
func PriorityList() []string {
	return []string{
		queueKeyPrefix + "1",
		queueKeyPrefix + "2",
		queueKeyPrefix + "3",
		queueKeyPrefix + "4",
	}
}

func queueKey(p Priority) string {
	return fmt.Sprintf("%s%d", queueKeyPrefix, int(p))
}

func respListKey(reqID string) string {
	return responseKey + reqID
}

var (
	poolMu sync.RWMutex
	pool   *redis.Pool
)

// SetPool wires up the Redis connection pool used by both clients and the dispatcher. Each service
// initialises its own pool and calls this once during boot. Calling SetPool(nil) disables the
// rate limiter (AcquireToken returns immediately with an error that the caller may ignore).
func SetPool(p *redis.Pool) {
	poolMu.Lock()
	defer poolMu.Unlock()
	pool = p
}

func getPool() *redis.Pool {
	poolMu.RLock()
	defer poolMu.RUnlock()
	return pool
}

// AcquireToken blocks until the dispatcher grants a token for the given priority, or until the
// context expires, or until the BLPOP timeout elapses (whichever comes first).
//
// The returned error is informational. messgae-sender intentionally ignores it (logs only) so a
// failing rate limiter does not break user-facing flows; chat-export-service treats the error as
// a soft signal to slow down.
func AcquireToken(ctx context.Context, priority Priority) error {
	p := getPool()
	if p == nil {
		return errors.New("ratelimit: pool is not configured")
	}

	conn := p.Get()
	defer func() { _ = conn.Close() }()
	if err := conn.Err(); err != nil {
		return fmt.Errorf("ratelimit: redis conn: %w", err)
	}

	reqID := uuid.New().String()
	if _, err := conn.Do("RPUSH", queueKey(priority), reqID); err != nil {
		return fmt.Errorf("ratelimit: enqueue: %w", err)
	}

	// BLPOP timeout in seconds (Redis API). Trim it to remaining ctx time when caller has a deadline.
	timeout := defaultBlockSeconds
	if dl, ok := ctx.Deadline(); ok {
		remaining := int(time.Until(dl).Seconds())
		if remaining < 1 {
			remaining = 1
		}
		if remaining < timeout {
			timeout = remaining
		}
	}

	respKey := respListKey(reqID)
	if _, err := redis.Values(conn.Do("BLPOP", respKey, timeout)); err != nil {
		if err == redis.ErrNil {
			return errors.New("ratelimit: token wait timed out")
		}
		return fmt.Errorf("ratelimit: wait token: %w", err)
	}
	// We don't need the BLPOP value, but we DO want to clean up the list so it doesn't linger when
	// the dispatcher's TTL is longer than the wait.
	_, _ = conn.Do("DEL", respKey)
	return nil
}

// StartDispatcher launches a goroutine that ticks at ratePerSecond and grants one token per tick to
// the highest-priority waiter. Designed to run in event-loop only; calling it elsewhere will burn
// the rate budget.
func StartDispatcher(ctx context.Context, ratePerSecond int) {
	if ratePerSecond <= 0 {
		log.Printf("ratelimit: dispatcher disabled (ratePerSecond=%d)", ratePerSecond)
		return
	}
	if getPool() == nil {
		log.Printf("ratelimit: dispatcher cannot start without a pool")
		return
	}

	interval := time.Second / time.Duration(ratePerSecond)
	go runDispatcher(ctx, interval)
}

func runDispatcher(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("ratelimit: dispatcher stopping")
			return
		case <-ticker.C:
			grantOne()
		}
	}
}

func grantOne() {
	p := getPool()
	if p == nil {
		return
	}
	conn := p.Get()
	defer func() { _ = conn.Close() }()
	if err := conn.Err(); err != nil {
		log.Printf("ratelimit: dispatcher conn err: %v", err)
		return
	}

	var reqID string
	for _, q := range PriorityList() {
		reply, err := redis.String(conn.Do("LPOP", q))
		if err == redis.ErrNil {
			continue
		}
		if err != nil {
			log.Printf("ratelimit: dispatcher LPOP %s: %v", q, err)
			return
		}
		reqID = reply
		break
	}
	if reqID == "" {
		return
	}

	respKey := respListKey(reqID)
	if err := conn.Send("MULTI"); err != nil {
		log.Printf("ratelimit: dispatcher MULTI: %v", err)
		return
	}
	_ = conn.Send("RPUSH", respKey, "1")
	_ = conn.Send("EXPIRE", respKey, responseTTLSeconds)
	if _, err := conn.Do("EXEC"); err != nil {
		log.Printf("ratelimit: dispatcher EXEC for req=%s: %v", reqID, err)
	}
}
