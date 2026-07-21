package events

import "fmt"

// Shared RabbitMQ event routing constants for api-gateway publishers and consumers.
const (
	EventsExchange = "chatdetective.events"
	ShardCount     = 64

	PodTypeCommands             = "commands"
	PodTypeShipping             = "shipping"
	PodTypeBusinessEventsNew    = "business_events_new"
	PodTypeBusinessEventsEdited = "business_events_edited"
)

// ShardQueueName returns the queue/routing-key name for a pod type shard, e.g. shipping.q15.
func ShardQueueName(podType string, shard int) string {
	return fmt.Sprintf("%s.q%02d", podType, shard)
}
