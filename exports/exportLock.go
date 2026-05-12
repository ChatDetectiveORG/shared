package exports

import (
	e "github.com/ChatDetectiveORG/shared/errors"

	"github.com/gomodule/redigo/redis"
)

// Per-user export lock. The lifetime is dimensioned around: invoice deliver → payment → archive
// upload. chat-export-service refreshes the TTL while processing; if a pod dies, the lock self-heals
// after ExportLockTTLSeconds.
const ExportLockTTLSeconds = 4 * 60 * 60

func exportLockKey(senderIDHash string) string {
	return "export:lock:" + senderIDHash
}

// AcquireExportLock atomically takes the per-user export lock; returns ok=false if the user
// already has an export in flight.
func AcquireExportLock(senderIDHash string, conn redis.Conn) (bool, *e.ErrorInfo) {
	defer func() { _ = conn.Close() }()

	reply, rawErr := conn.Do("SET", exportLockKey(senderIDHash), "invoice_sent", "NX", "EX", ExportLockTTLSeconds)
	if rawErr != nil {
		return false, e.FromError(rawErr, "failed to acquire export lock").WithSeverity(e.Notice)
	}
	return reply != nil, e.Nil()
}

// ReleaseExportLock removes the per-user export lock; safe if the lock was already gone.
func ReleaseExportLock(senderIDHash string, conn redis.Conn) *e.ErrorInfo {
	defer func() { _ = conn.Close() }()
	if _, rawErr := conn.Do("DEL", exportLockKey(senderIDHash)); rawErr != nil {
		return e.FromError(rawErr, "failed to release export lock").WithSeverity(e.Notice)
	}
	return e.Nil()
}
