package events

import "testing"

func TestShardQueueName(t *testing.T) {
	if got := ShardQueueName(PodTypeShipping, 15); got != "shipping.q15" {
		t.Fatalf("got %q want shipping.q15", got)
	}
}
