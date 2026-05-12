// Package exports defines the cross-service contract for chat export requests.
//
// payment-service produces ExportRequest messages once an invoice is paid; chat-export-service
// consumes them. Keeping the payload and routing helpers here ensures both sides cannot drift
// silently when fields are added.
package exports

import (
	"fmt"
	"hash/fnv"
)

// ExportsExchange is the topic exchange for chat export requests.
const ExportsExchange = "chatdetective.exports"

// ExportShardCount mirrors the number of consumer queues. Sharding is keyed on SenderIDHash so
// every export from a single user hits the same queue (and pod), which keeps the per-user lock
// contention deterministic.
const ExportShardCount = 64

// ExportRequest is the message envelope published by payment-service after a successful payment.
//
// The bot does not have access to the original Telegram user id at this layer, so chat-export-service
// looks the user up by SenderIDHash. InterlocutorCode is the referral_code of the chat partner the
// user wants to restore. StatusChatID is the chat where the live status / progress message must
// be posted (== bot owner's private chat with the bot in normal flow, or the mirror's chat for
// mirror-bot installations).
type ExportRequest struct {
	PaymentID        int    `json:"payment_id"`
	SenderIDHash     string `json:"sender_id_hash"`
	InterlocutorCode string `json:"interlocutor_code"`
	StatusChatID     int64  `json:"status_chat_id"`
	MessagesCount    int    `json:"messages_count"`
	MirrorID         string `json:"mirror_id,omitempty"`
}

// ExportShardRoutingKey computes the queue routing key from the sender's hash. Using FNV (32-bit)
// gives a cheap, stable distribution; the exact algorithm is irrelevant as long as producers and
// consumers agree, which is enforced by the shared package boundary.
func ExportShardRoutingKey(senderHash string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(senderHash))
	shard := int(h.Sum32() % uint32(ExportShardCount))
	return fmt.Sprintf("chat_export.q%02d", shard)
}

// ExportShardQueueName is the queue name for the given shard index. Declared by chat-export-service.
func ExportShardQueueName(shard int) string {
	return fmt.Sprintf("chat_export.q%02d", shard)
}
